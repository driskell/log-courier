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
	"bytes"
	"context"
	"testing"
	"time"
)

func TestNewEventEmpty(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{})
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if time.Since(time.Time(timestamp)) > time.Second {
			t.Fatalf("Wrong timestamp in empty event: %v", event.Data())
		}
	} else {
		t.Fatalf("Missing timestamp in empty event: %v", event.Data())
	}
	if tags, ok := event.Data()["tags"].(Tags); ok {
		if len(tags) != 0 {
			t.Fatalf("Invalid empty tags: %d", len(tags))
		}
	} else {
		t.Fatalf("Missing tags in empty event: %v", event.Data())
	}
}

func TestNewEventInvalidTimestamp(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"@timestamp": "Invalid"})
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if time.Since(time.Time(timestamp)) > time.Second {
			t.Fatalf("Wrong timestamp in invalid event: %v", event.Data())
		}
	} else {
		t.Fatalf("Missing @timestamp invalid event: %v", event.Data())
	}
	if tags, ok := event.Data()["tags"].(Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_timestamp_parse_failure\"]")) {
			t.Fatalf("Invalid tags for failed timestamp: %v (error: %v)", tags, err)
		}
	} else {
		t.Fatalf("Missing tags in invalid event: %v", event.Data())
	}
}

func TestNewEventWrongTypeTimestamp(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"@timestamp": map[string]int{"invalid": 999}})
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if time.Since(time.Time(timestamp)) > time.Second {
			t.Fatalf("Wrong timestamp in invalid event: %v", event.Data())
		}
	} else {
		t.Fatalf("Missing @timestamp in invalid event: %v", event.Data())
	}
	if tags, ok := event.Data()["tags"].(Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_timestamp_parse_failure\"]")) {
			t.Fatalf("Invalid tags for failed timestamp: %v (error: %v)", tags, err)
		}
	} else {
		t.Fatalf("Missing tags in invalid event: %v", event.Data())
	}
}

func TestNewEventValidTimestamp(t *testing.T) {
	example := "2020-05-05T13:00:12.123Z"
	event := NewEvent(context.Background(), nil, map[string]interface{}{"@timestamp": example})
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		timestampParsed, _ := time.Parse("2006-01-02T15:04:05Z", example)
		if !time.Time(timestamp).Equal(timestampParsed) {
			t.Fatalf("Wrong timestamp in event: %v; expected %v", event.Data(), timestampParsed)
		}
	} else {
		t.Fatalf("Missing timestamp in event: %v", event.Data())
	}
}

func TestNewEventTimestampExisting(t *testing.T) {
	example := "2020-05-05T13:00:12.123Z"
	timestampParsed, _ := time.Parse("2006-01-02T15:04:05Z", example)
	event := NewEvent(context.Background(), nil, map[string]interface{}{"@timestamp": timestampParsed})
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if !time.Time(timestamp).Equal(timestampParsed) {
			t.Fatalf("Wrong timestamp in event: %v; expected %v", event.Data(), timestampParsed)
		}
	} else {
		t.Fatalf("Missing timestamp in event: %v", event.Data())
	}
}

func TestNewEventInvalidTags(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"tags": map[string]int{"Invalid": 999}})
	if tags, ok := event.Data()["tags"].(Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_tags_parse_failure\"]")) {
			t.Fatalf("Invalid tags for failed tags: %v (error: %v)", tags, err)
		}
	} else {
		t.Fatalf("Missing tags in invalid event: %v", event.Data())
	}
}

func TestNewEventStringTag(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"tags": "_string_tag"})
	if tags, ok := event.Data()["tags"].(Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_string_tag\"]")) {
			t.Fatalf("Invalid tags for string tag: %v (error: %v)", tags, err)
		}
	} else {
		t.Fatalf("Missing tags in event: %v", event.Data())
	}
}

func TestNewEventValidTags(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"tags": []interface{}{"_one_tag", "_two_tag"}})
	if tags, ok := event.Data()["tags"].(Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_one_tag\",\"_two_tag\"]")) {
			t.Fatalf("Invalid tags: %s (error: %v)", value, err)
		}
	} else {
		t.Fatalf("Missing tags in event: %v", event.Data())
	}
}

func TestNewEventBytes(t *testing.T) {
	event := NewEventFromBytes(context.Background(), nil, []byte("{\"message\":\"basic event\"}"))
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if time.Since(time.Time(timestamp)) > time.Second {
			t.Fatalf("Wrong timestamp in basic event: %v", event.Data())
		}
	} else {
		t.Fatalf("Missing timestamp in basic event: %v", event.Data())
	}
	if tags, ok := event.Data()["tags"].(Tags); ok {
		if len(tags) != 0 {
			t.Fatalf("Invalid tags for basic event: %v", tags)
		}
	} else {
		t.Fatalf("Missing tags in basic event: %v", event.Data())
	}
}

func TestNewEventBytesInvalid(t *testing.T) {
	event := NewEventFromBytes(context.Background(), nil, []byte("invalid bytes"))
	if timestamp, ok := event.Data()["@timestamp"].(Timestamp); ok {
		if time.Since(time.Time(timestamp)) > time.Second {
			t.Fatalf("Wrong timestamp in invalid event from bytes: %v", event.Data())
		}
	} else {
		t.Fatalf("Missing timestamp in invalid event from bytes: %v", event.Data())
	}
	if tags, ok := event.Data()["tags"].(*Tags); ok {
		value, err := tags.MarshalJSON()
		if err != nil || !bytes.Equal(value, []byte("[\"_unmarshal_failure\"]")) {
			t.Fatalf("Invalid tags for failed unmarshal: %v (error: %v)", tags, err)
		}
	} else {
		t.Fatalf("Missing tags in invalid event from bytes: %v", event.Data())
	}
}

func TestEventBytes(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"message": "Test message", "@timestamp": "2020-02-01T13:00:00.000Z"})
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
}

func TestEventAddRemoveTag(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"message": "Test message", "@timestamp": "2020-02-01T13:00:00.000Z"})
	event.AddTag("_testing")
	event.ClearCache()
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[\"_testing\"]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
	event.AddTag("_testing")
	event.ClearCache()
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[\"_testing\"]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
	event.RemoveTag("_testing")
	event.ClearCache()
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
	event.RemoveTag("_testing")
	event.ClearCache()
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
}

func TestEventCache(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"message": "Test message", "@timestamp": "2020-02-01T13:00:00.000Z"})
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[]}")) {
		t.Fatalf("Invalid event bytes: %s", string(event.Bytes()))
	}
	event.Data()["more"] = "value"
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"tags\":[]}")) {
		t.Fatalf("Event bytes were not cached: %s", string(event.Bytes()))
	}
	event.ClearCache()
	if !bytes.Equal(event.Bytes(), []byte("{\"@timestamp\":\"2020-02-01T13:00:00Z\",\"message\":\"Test message\",\"more\":\"value\",\"tags\":[]}")) {
		t.Fatalf("Event bytes cache did not clear: %s", string(event.Bytes()))
	}
}

func TestResolveKey(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"message": "Hello world",
	})
	result, err := event.Resolve("message", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != "Hello world" {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveKeyShallow(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"deeper": 123,
		},
	})
	result, err := event.Resolve("sub[deeper]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != 123 {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveKeyDeep(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"deeper": map[string]interface{}{
				"last": true,
			},
		},
	})
	result, err := event.Resolve("sub[deeper][last]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != true {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveKeyNonMap(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": "Message",
	})
	result, err := event.Resolve("sub[deeper][last]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveKeyInvalid(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"message": "",
		},
	})
	result, err := event.Resolve("sub[", nil)
	if err == nil {
		t.Fatalf("Unexpected successful result: %s", result)
	}

	result, err = event.Resolve("su]b", nil)
	if err == nil {
		t.Fatalf("Unexpected successful result: %s", result)
	}

	result, err = event.Resolve("sub[inside]more", nil)
	if err == nil {
		t.Fatalf("Unexpected successful result: %s", result)
	}

	result, err = event.Resolve("sub[inside]nogap[more]", nil)
	if err == nil {
		t.Fatalf("Unexpected successful result: %s", result)
	}

	result, err = event.Resolve("su[]", nil)
	if err == nil {
		t.Fatalf("Unexpected successful result: %s", result)
	}
}

func TestResolveKeyMissing(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"message": "",
		},
	})
	result, err := event.Resolve("sub[missing]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}

	result, err = event.Resolve("missing[sub]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveSet(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"message": "",
		},
	})
	result, err := event.Resolve("sub[missing]", "value")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
	result, err = event.Resolve("sub[missing]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != "value" {
		t.Fatalf("Unexpected result: [%v]", result)
	}

	result, err = event.Resolve("missing[sub]", 123)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
	result, err = event.Resolve("missing[sub]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != 123 {
		t.Fatalf("Unexpected result: [%v]", result)
	}

	result, err = event.Resolve("sub[message][test]", true)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
	result, err = event.Resolve("sub[message][test]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != true {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestResolveUnset(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{
		"sub": map[string]interface{}{
			"message": "Hello",
		},
	})
	result, err := event.Resolve("sub[message]", ResolveParamUnset)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != "Hello" {
		t.Fatalf("Unexpected result: [%v]", result)
	}
	result, err = event.Resolve("sub[message]", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if result != nil {
		t.Fatalf("Unexpected result: [%v]", result)
	}
}

func TestMustResolvePanic(t *testing.T) {
	event := NewEvent(context.Background(), nil, map[string]interface{}{"message": "Hello"})
	result := event.MustResolve("message", nil)
	if result != "Hello" {
		t.Fatalf("Incorrect result from must resolve: %s", result)
	}

	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()
	result = event.MustResolve("message[", nil)
	t.Error("MustResolve did not panic")
}

// TODO: validateMutation

// TODO: Bytes() encoding error

// TODO: DispatchAck

// TODO: Context
