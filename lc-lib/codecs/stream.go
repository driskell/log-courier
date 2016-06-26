/*
 * Copyright 2014-2015 Jason Woods.
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

package codecs

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	defaultStreamCodec          string = "plain"
	defaultStreamAddOffsetField bool   = true
)

// Stub holds an unknown codec configuration
// After initial parsing of configuration, these Stubs are turned into
// real configuration blocks for the codec given by their Name field
type Stub struct {
	Name    string `config:"name"`
	Unused  map[string]interface{}
	Factory interface{}
}

// StreamConfig holds the configuration for a log stream that supports codecs
type StreamConfig struct {
	event.StreamConfig `config:",embed"`
	AddOffsetField     bool   `config:"add offset field"`
	Codecs             []Stub `config:"codecs"`
}

// Stream represents a single stream of events that involves a codec chain
type Stream struct {
	streamConfig *StreamConfig
	firstCodec   Codec
	codecChain   []Codec
	eventFunc    EventFunc
}

// EventFunc is called by a Stream when an Event is produced by the
// codec chain
type EventFunc func(int64, int64, map[string]interface{})

// Defaults sets the default codec stream configuration
func (sc *StreamConfig) Defaults() {
	sc.AddOffsetField = defaultStreamAddOffsetField
}

// Validate does nothing for a codec stream
// This is here to prevent double validation of event.StreamConfig whose
// validation function would otherwise be inherited
func (sc *StreamConfig) Validate(p *config.Parser, path string) (err error) {
	return nil
}

// Init initialises a stream configuration with codecs by creating the necessary
// codec factories that will be required
func (sc *StreamConfig) Init(p *config.Parser, path string) (err error) {
	if len(sc.Codecs) == 0 {
		sc.Codecs = []Stub{Stub{Name: defaultStreamCodec}}
	}

	for i := 0; i < len(sc.Codecs); i++ {
		codec := &sc.Codecs[i]
		if registrarFunc, ok := registeredCodecs[codec.Name]; ok {
			if codec.Factory, err = registrarFunc(p, path, codec.Unused, codec.Name); err != nil {
				return
			}
		} else {
			return fmt.Errorf("Unrecognised codec '%s' for %s", codec.Name, path)
		}
	}

	// TODO: EDGE CASE: Event transmit length is uint32, if fields length is rediculous we will fail

	return nil
}

// NewStream creates a new stream with a set of codec processors using the
// current stream configuration and the given start offset
func (sc *StreamConfig) NewStream(eventFunc EventFunc, startOffset int64) *Stream {
	stream := &Stream{
		streamConfig: sc,
		codecChain:   make([]Codec, len(sc.Codecs)-1),
		eventFunc:    eventFunc,
	}

	// Build the codec chain
	var entry Codec
	eventCallback := stream.eventCallback
	for i := len(sc.Codecs) - 1; i >= 0; i-- {
		entry = NewCodec(sc.Codecs[i].Factory, eventCallback, startOffset)
		eventCallback = entry.Event
		if i != 0 {
			stream.codecChain[i-1] = entry
		}
	}
	stream.firstCodec = entry

	return stream
}

// ProcessEvent receives events that should be processed by the codec stream
func (cs *Stream) ProcessEvent(startOffset int64, endOffset int64, text string) {
	cs.firstCodec.Event(startOffset, endOffset, text)
}

// eventCallback receives events from the final codec and ships them to the output
func (cs *Stream) eventCallback(startOffset int64, endOffset int64, text string) {
	data := map[string]interface{}{
		"message": text,
	}

	if cs.streamConfig.AddOffsetField {
		data["offset"] = startOffset
	}

	cs.eventFunc(startOffset, endOffset, data)
}

// Reset resets the stream and all codecs
func (cs *Stream) Reset() {
	for _, codec := range cs.codecChain {
		codec.Reset()
	}

	cs.firstCodec.Reset()
}

// Meter causes the stream to calculate metrics for itself and all codecs
func (cs *Stream) Meter() {
	for _, codec := range cs.codecChain {
		codec.Meter()
	}

	cs.firstCodec.Meter()
}

// Close shuts down all codecs in the order they are used and closes the stream
// and returns the last completed (non-pending) offset in the stream that was
// shipped to the returnCallback
func (cs *Stream) Close() int64 {
	for _, codec := range cs.codecChain {
		codec.Teardown()
	}

	return cs.firstCodec.Teardown()
}

// APIEntry returns API information for the stream and all codecs
func (cs *Stream) APIEntry() api.Navigatable {
	codecs := &api.Array{}
	i := 0
	if encodable := cs.firstCodec.APIEncodable(); encodable != nil {
		codecs.AddEntry(cs.streamConfig.Codecs[0].Name, api.NewDataEntry(encodable))
	}
	for _, codec := range cs.codecChain {
		if encodable := codec.APIEncodable(); encodable != nil {
			i++
			codecs.AddEntry(cs.streamConfig.Codecs[i].Name, api.NewDataEntry(encodable))
		}
	}
	return codecs
}
