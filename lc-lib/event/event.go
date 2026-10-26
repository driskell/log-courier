/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ResolveParam is a special set value you can pass to event.Resolve
type ResolveParam int

const (
	// ResolveParamUnset will unset the value at a path given to event.Resolve
	ResolveParamUnset ResolveParam = iota
)

// MetadataKey is the top-level key of the @metadata entry, which is never encoded
const MetadataKey = "@metadata"

// Acknowledger is something that can be called with events once they ar acknowledged
type Acknowledger interface {
	Acknowledge([]*Event)
}

// Event describes an event
type Event struct {
	ctx   context.Context
	acker Acknowledger
	data  map[string]interface{}

	encoded []byte

	// metadata is the @metadata entry, held outside data as it never comes from or goes to the encoded form
	metadata Metadata

	dropped bool
}

// Drop marks the event as dropped so that it is not transmitted to any sink
// It is still acknowledged in order alongside the events around it
func (e *Event) Drop() {
	e.dropped = true
}

// Dropped returns true if the event was marked as dropped during processing
func (e *Event) Dropped() bool {
	return e.dropped
}

// Builtin is used around builtin keys to allow damage prevention
type Builtin interface {
	// VerifySetEnter checks if we can set the given key (if we're a map for example)
	VerifySetEnter(string) (map[string]interface{}, error)
	// VerifySet checks if we can be set to the given value
	VerifySet(interface{}) (interface{}, error)
}

// NewEvent creates a new event structure from the given data
func NewEvent(ctx context.Context, acker Acknowledger, data map[string]interface{}) *Event {
	ret := &Event{
		ctx:   ctx,
		acker: acker,
		data:  data,
	}
	ret.convertData()
	return ret
}

// NewEventFromBytes creates a new event structure from the given data
func NewEventFromBytes(ctx context.Context, acker Acknowledger, data []byte) *Event {
	return &Event{
		ctx:     ctx,
		acker:   acker,
		encoded: data,
	}
}

// Data returns the internal event data for reading or mutation
// The return data must NOT be mutated
func (e *Event) Data() map[string]interface{} {
	if e.data == nil {
		err := json.Unmarshal(e.encoded, &e.data)
		if err != nil {
			e.data = make(map[string]interface{})
			e.data["message"] = err.Error()
			e.data["@timestamp"] = Timestamp(time.Now())
			e.data[MetadataKey] = e.metadataEntry()
			e.data["tags"] = Tags{"_unmarshal_failure"}
		} else {
			e.convertData()
		}
	}
	return e.data
}

// metadataEntry returns the @metadata entry, creating it if needed
func (e *Event) metadataEntry() Metadata {
	if e.metadata == nil {
		e.metadata = Metadata{}
	}
	return e.metadata
}

// convertData is the internal function that enforces guaranteed types
func (e *Event) convertData() {
	// Normalize "tags" first (other resolutions ignore it)
	if entry, ok := e.data["tags"]; ok {
		switch value := entry.(type) {
		case Tags:
		case string:
			e.data["tags"] = Tags{value}
		case []interface{}:
			// From unmarshaled over the wire we get []interface{}
			tags := Tags{}
			for _, tag := range value {
				tagString, ok := tag.(string)
				if !ok {
					tags = Tags{"_tags_parse_failure"}
					e.data["tags_parse_error"] = fmt.Sprintf("tags list must contain only strings, found a %T", tag)
					break
				}
				tags = append(tags, tagString)
			}
			e.data["tags"] = tags
		default:
			e.data["tags"] = Tags{"_tags_parse_failure"}
			e.data["tags_parse_error"] = fmt.Sprintf("tags was not a string or string list, was %T", value)
		}
	} else {
		e.data["tags"] = Tags{}
	}
	// Normalize "@timestamp" to a time.Time
	if entry, ok := e.data["@timestamp"]; ok {
		switch value := entry.(type) {
		case Timestamp:
		case time.Time:
			e.data["@timestamp"] = Timestamp(value)
		case string:
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				e.data["@timestamp"] = Timestamp(time.Now())
				e.data["timestamp_parse_error"] = err
				e.AddTag("_timestamp_parse_failure")
			} else {
				e.data["@timestamp"] = Timestamp(parsed)
			}
		default:
			e.data["@timestamp"] = Timestamp(time.Now())
			e.data["timestamp_parse_error"] = fmt.Sprintf("@timestamp was not a string, was %T", value)
			e.AddTag("_timestamp_parse_failure")
		}
	} else {
		e.data["@timestamp"] = Timestamp(time.Now())
	}
	// Normalize "@metadata"
	e.data[MetadataKey] = e.metadataEntry()
}

// MustResolve is the same as Resolve but will panic if an error occurs
//
// This should never be used with user input, and only used with hard-coded field names that are guaranteed to be correct syntax
func (e *Event) MustResolve(path string, set interface{}) interface{} {
	output, err := e.Resolve(path, set)
	if err != nil {
		panic(err)
	}
	return output
}

// Resolve will return (and optionally set) the value at the given path (field), using a[b][c] syntax
// where a backslash escapes the next character, so a\[b\] is the single name "a[b]"
//
// # It will return an error if the path is invalid, and return nil if the path does not exist
//
// If you specify non-nil in the set parameter - the path will be set to that value and
// the old value will be returned - and any non-map parts of the path will be converted to empty maps
//
// # Remember ClearCache is required to flush any cached representations if you set or unset a value
//
// The following paths are guaranteed to exist and be of a specific type:
//
//	@timestamp: time.Time
//	    If no value currently exists, it is created with the current time.
//	    If the existing value is invalid, the _timestamp_parse_failure tag is added
//	    and the error is stored inside the timestamp_parse_error field
//	@metadata: Metadata (Empty Metadata instance)
//	    Always empty, and overrides any value that came in over the wire.
//	    It is purely for processing metadata and is removed prior to any sending
//	tags: Tags (Empty Tags instance)
//	    If no value currently exists, it is created empty with no tags.
//	    If the existing value is invalid, the _tags_parse_failure tag is added
//	    and the error is stored inside the tags_parse_error field
//
// If any error occured during unmarshaling of an event received over the network
// then a new event with only a message key (and the above defaults) will be created
// where the message is the error message, and the "_unmarshal_failure" tag is added
func (e *Event) Resolve(path string, set interface{}) (output interface{}, err error) {
	if strings.HasPrefix(path, MetadataKey) {
		if len(path) == len(MetadataKey) {
			// The @metadata entry itself keeps its builtin protection against being set or unset
			return e.resolvePath(map[string]interface{}{MetadataKey: e.metadataEntry()}, path, set)
		}
		if path[len(MetadataKey)] == '[' {
			return e.ResolveMetadata(path[len(MetadataKey):], set)
		}
	}
	return e.resolvePath(e.Data(), path, set)
}

// MustResolveMetadata is the same as ResolveMetadata but will panic if an error occurs
//
// This should never be used with user input, and only used with hard-coded field names that are guaranteed to be correct syntax
func (e *Event) MustResolveMetadata(path string, set interface{}) interface{} {
	output, err := e.ResolveMetadata(path, set)
	if err != nil {
		panic(err)
	}
	return output
}

// ResolveMetadata is the same as Resolve for a path within @metadata, so "receiver" resolves
// "@metadata[receiver]", without decoding the event as @metadata never comes from the encoded form
func (e *Event) ResolveMetadata(path string, set interface{}) (output interface{}, err error) {
	return e.resolvePath(e.metadataEntry(), path, set)
}

// resolvePath implements Resolve against the given top-level map
func (e *Event) resolvePath(currentMap map[string]interface{}, path string, set interface{}) (output interface{}, err error) {
	// Begin with simple lookup - if we're not looking up something[xxx] and there are no escapes
	// then we can avoid splitting the path
	if strings.IndexAny(path, "[]\\") == -1 {
		if set != nil && e.isSameMap(currentMap, set) {
			return nil, fmt.Errorf("Cannot set '%s': would create circular reference", path)
		}
		output, err = e.simpleResolve(currentMap, path, set)
		return
	}
	names, ok := splitPath(path)
	if !ok {
		return nil, fmt.Errorf("Invalid field: %s", path)
	}
	for j, name := range names {
		if j == len(names)-1 {
			// Last item, so will always be a value
			output, err = e.simpleResolve(currentMap, name, set)
		} else {
			// Calculate next inner
			// Can't use non-map, ignore as if it didn't exist, and keep validating
			// (However, if we are setting, overwrite it with empty map)
			if value, ok := currentMap[name].(map[string]interface{}); ok {
				currentMap = value
			} else if builtin, ok := currentMap[name].(Builtin); ok {
				// Block entry to set if this is a builtin that does not want it
				enter, err := builtin.VerifySetEnter(name)
				if err != nil {
					return nil, err
				}
				currentMap = enter
			} else if set != nil {
				// Convert path to empty map
				newMap := map[string]interface{}{}
				currentMap[name] = newMap
				currentMap = newMap
			} else {
				// Doesn't exist so there's no value but keep validating
				currentMap = nil
			}
			if set != nil && currentMap != nil && e.isSameMap(currentMap, set) {
				return nil, fmt.Errorf("Cannot set '%s': would create circular reference", path)
			}
		}
	}
	return
}

// splitPath splits a path in a[b][c] syntax into its names, where a backslash escapes the next character
func splitPath(path string) ([]string, bool) {
	const (
		stateName = iota
		stateBracket
		stateAfterBracket
	)
	var (
		names []string
		name  strings.Builder
		state = stateName
	)
	for idx := 0; idx < len(path); idx++ {
		char := path[idx]
		switch state {
		case stateName, stateBracket:
			if char == '\\' {
				idx++
				if idx == len(path) {
					return nil, false
				}
				name.WriteByte(path[idx])
			} else if char == '[' {
				if state == stateBracket {
					return nil, false
				}
				if name.Len() != 0 {
					names = append(names, name.String())
					name.Reset()
				}
				state = stateBracket
			} else if char == ']' {
				if state == stateName || name.Len() == 0 {
					return nil, false
				}
				names = append(names, name.String())
				name.Reset()
				state = stateAfterBracket
			} else {
				name.WriteByte(char)
			}
		case stateAfterBracket:
			if char != '[' {
				return nil, false
			}
			state = stateBracket
		}
	}
	switch state {
	case stateBracket:
		return nil, false
	case stateName:
		names = append(names, name.String())
	}
	return names, true
}

var keyEscaper = strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]")

// EscapeKey escapes a key so that Resolve treats it as a single literal name
func EscapeKey(key string) string {
	return keyEscaper.Replace(key)
}

// isSameMap checks if the given value is the same map as the host map to prevent circular references
func (e *Event) isSameMap(host map[string]interface{}, value interface{}) bool {
	if valueMap, ok := value.(map[string]interface{}); ok {
		if fmt.Sprintf("%p", valueMap) == fmt.Sprintf("%p", host) {
			return true
		}
	}
	return false
}

// simpleResolve is used by Resolve for the last leg (or the only leg if a simple top-level lookup)
func (e *Event) simpleResolve(currentMap map[string]interface{}, path string, set interface{}) (output interface{}, err error) {
	if value, ok := currentMap[path]; ok {
		output = value
	}
	if set != nil {
		if builtin, ok := currentMap[path].(Builtin); ok {
			if set == ResolveParamUnset {
				return nil, fmt.Errorf("Builtin entry '%s' cannot be unset", path)
			}
			result, err := builtin.VerifySet(set)
			if err != nil {
				return nil, err
			}
			currentMap[path] = result
		} else {
			if set == ResolveParamUnset {
				delete(currentMap, path)
			} else {
				currentMap[path] = set
			}
		}
	}
	return
}

// AddError adds an error tag and an error message field for a specific action that has failed
func (e *Event) AddError(action string, message string) {
	e.Data()[fmt.Sprintf("_%s_error", action)] = message
	e.AddTag(fmt.Sprintf("_%s_failure", action))
}

// AddTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) AddTag(tag string) {
	data := e.Data()
	tags := data["tags"].(Tags)
	length := len(tags)
	idx := sort.SearchStrings(tags, tag)
	if idx >= length {
		data["tags"] = append(tags, tag)
	} else if tags[idx] != tag && length < 1024 {
		// Max 1024 tags - this also prevents CWE-190 from CodeQL (Size computation for allocation may overflow) due to length+1 without bound check
		if length+1 > cap(tags) {
			oldTags := tags
			tags = make(Tags, length+1)
			copy(tags, oldTags[:idx])
			copy(tags[idx+1:], oldTags[idx:])
		} else {
			tags = tags[:length+1]
			copy(tags[idx+1:], tags[idx:])
		}
		tags[idx] = tag
		data["tags"] = tags
	}
}

// RemoveTag adds a tag to the event
// Remember ClearCache is required to flush any cached representations
func (e *Event) RemoveTag(tag string) {
	data := e.Data()
	tags := data["tags"].(Tags)
	length := len(tags)
	idx := sort.SearchStrings(tags, tag)
	if idx < length && tags[idx] == tag {
		copy(tags[idx:], tags[idx+1:])
		data["tags"] = tags[:length-1]
	}
}

// ClearCache clears any cached representations, always call it if the event is changed
func (e *Event) ClearCache() {
	// An event never decoded holds its data only in the encoded form
	e.Data()
	e.encoded = nil
}

// Bytes returns the encoded event bytes
// The @metadata key is completely ignored
// The returned slice should not be modified and be treated immutable
// There is currently no way to modify it. To change the event, use Data(),
// and then use ClearCache to clear the Bytes() cache so it regenerates
func (e *Event) Bytes() []byte {
	if e.encoded == nil {
		var err error
		prunedData := make(map[string]interface{})
		for key, value := range e.data {
			if key == MetadataKey {
				continue
			}
			prunedData[key] = value
		}
		if e.encoded, err = json.Marshal(prunedData); err != nil {
			e.encoded = []byte("{\"tags\":[\"_encode_failure\"]}")
		}
	}
	return e.encoded
}

// Context returns the stream context for this Event - and can be used to
// distinguish events from different sources
func (e *Event) Context() context.Context {
	return e.ctx
}

// DispatchAck processes a bulk of events and calls the required acknowledgement
// callbacks
func DispatchAck(events []*Event) {
	if len(events) == 0 {
		return
	}

	// Grab first ackFunc
	e := 0
	acker := events[0].acker
	for _, event := range events[1:] {
		e++

		// Different acker?
		if event.acker != acker {
			// Multiple event acks in the works, split up the bulk and pass to the
			// relevant callbacks
			dispatchAckForMultipleFuncs(acker, append([]*Event(nil), events[0:e]...), events[e:])
			return
		}
	}

	if acker == nil {
		// Skip where no acker
		return
	}

	// Single event type - call it's acknowledger
	acker.Acknowledge(events)
}

// dispatchAckForMultipleFuncs handles the case where a bulk of events to be
// acknowledged contains events for different processors by splitting them apart
// and calling each processor
func dispatchAckForMultipleFuncs(firstAcker Acknowledger, firstAckerEvents []*Event, events []*Event) {
	ackMap := map[Acknowledger][]*Event{}
	if firstAcker != nil {
		ackMap[firstAcker] = firstAckerEvents
	}

	for _, event := range events {
		if event.acker != nil {
			ackMap[event.acker] = append(ackMap[event.acker], event)
		}
	}

	for acker, ackEvents := range ackMap {
		acker.Acknowledge(ackEvents)
	}
}
