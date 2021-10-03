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

package es

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
)

func createTestBulkRequest(count int, first string, second string) *bulkRequest {
	events := make([]*event.Event, count)
	for i := 0; i < count; i++ {
		var timestamp time.Time
		if i < count/2 {
			timestamp, _ = time.Parse("2006-01-02", first)
		} else {
			timestamp, _ = time.Parse("2006-01-02", second)
		}
		events[i] = event.NewEvent(context.Background(), nil, map[string]interface{}{
			"@timestamp": timestamp,
			"message":    fmt.Sprintf("message %d", i),
		})
	}
	return newBulkRequest("logstash-%{+2006-01-02}", events)
}

func TestRequestCreate(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-07")
	if request.Remaining() != 3 {
		t.Errorf("Unexpected initial remaining count: %d", request.Remaining())
	}
	if request.Created() != 0 {
		t.Errorf("Unexpected initial created count: %d", request.Created())
	}
	if request.AckSequence() != 0 {
		t.Errorf("Unexpected initial ack sequence: %d", request.AckSequence())
	}
}

func TestRequestReadFull(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-07")
	result, err := ioutil.ReadAll(request)
	if err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	defaultIndex, err := request.DefaultIndex()
	if err != nil {
		t.Errorf("Failed to get default index: %s", err)
	}
	if defaultIndex != "logstash-2020-03-07" {
		t.Errorf("Unexpected default index: %s", defaultIndex)
	}
	if !bytes.Equal(
		result,
		[]byte(
			"{\"index\":{}}\n"+
				"{\"@timestamp\":\"2020-03-07T00:00:00Z\",\"message\":\"message 0\",\"tags\":[]}\n"+
				"{\"index\":{}}\n"+
				"{\"@timestamp\":\"2020-03-07T00:00:00Z\",\"message\":\"message 1\",\"tags\":[]}\n"+
				"{\"index\":{}}\n"+
				"{\"@timestamp\":\"2020-03-07T00:00:00Z\",\"message\":\"message 2\",\"tags\":[]}\n",
		),
	) {
		t.Errorf("Unexpected result: %s", string(result))
	}
}

func TestRequestReadMultiple(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-14")
	result, err := ioutil.ReadAll(request)
	if err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	defaultIndex, err := request.DefaultIndex()
	if err != nil {
		t.Errorf("Failed to get default index: %s", err)
	}
	if defaultIndex != "logstash-2020-03-07" {
		t.Errorf("Unexpected default index: %s", defaultIndex)
	}
	if !bytes.Equal(
		result,
		[]byte(
			"{\"index\":{}}\n"+
				"{\"@timestamp\":\"2020-03-07T00:00:00Z\",\"message\":\"message 0\",\"tags\":[]}\n"+
				"{\"index\":{\"_index\":\"logstash-2020-03-14\"}}\n"+
				"{\"@timestamp\":\"2020-03-14T00:00:00Z\",\"message\":\"message 1\",\"tags\":[]}\n"+
				"{\"index\":{\"_index\":\"logstash-2020-03-14\"}}\n"+
				"{\"@timestamp\":\"2020-03-14T00:00:00Z\",\"message\":\"message 2\",\"tags\":[]}\n",
		),
	) {
		t.Errorf("Unexpected result: %s", string(result))
	}
}

func TestRequestReadReset(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-14")
	if _, err := ioutil.ReadAll(request); err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	result, err := ioutil.ReadAll(request)
	if err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	if len(result) != 0 {
		t.Errorf("Unexpected result: %s", string(result))
	}
	request.Reset()
	result, err = ioutil.ReadAll(request)
	if err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	if len(result) == 0 {
		t.Error("Reset did not reset read pointer")
	}
}

func TestRequestMark(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-14")
	cursor, end := request.Mark(nil, true)
	if end || cursor == nil {
		t.Error("Unexpected end")
	}
	cursor, end = request.Mark(cursor, false)
	if end || cursor == nil {
		t.Error("Unexpected end")
	}
	cursor, end = request.Mark(cursor, true)
	if !end || cursor != nil {
		t.Error("Missing end")
	}
	if request.Remaining() != 1 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if request.Created() != 2 {
		t.Errorf("Unexpected created count: %d", request.Created())
	}
	if request.AckSequence() != 1 {
		t.Errorf("Unexpected ack sequence: %d", request.AckSequence())
	}
	request.Reset()
	result, err := ioutil.ReadAll(request)
	if err != nil {
		t.Errorf("Failed to encode: %s", err)
	}
	defaultIndex, err := request.DefaultIndex()
	if err != nil {
		t.Errorf("Failed to get default index: %s", err)
	}
	if defaultIndex != "logstash-2020-03-14" {
		t.Errorf("Unexpected default index: %s", defaultIndex)
	}
	if !bytes.Equal(
		result,
		[]byte(
			"{\"index\":{}}\n"+
				"{\"@timestamp\":\"2020-03-14T00:00:00Z\",\"message\":\"message 1\",\"tags\":[]}\n",
		),
	) {
		t.Errorf("Unexpected result: %s", string(result))
	}
	cursor, end = request.Mark(nil, true)
	if !end || cursor != nil {
		t.Error("Missing end")
	}
	if request.Remaining() != 0 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if request.Created() != 3 {
		t.Errorf("Unexpected created count: %d", request.Created())
	}
	if request.AckSequence() != 3 {
		t.Errorf("Unexpected ack sequence: %d", request.AckSequence())
	}
}

func TestRequestMarkFailTwiceInvalidPos(t *testing.T) {
	request := createTestBulkRequest(3, "2020-03-07", "2020-03-14")
	if request.Event(nil) == nil {
		t.Error("Unexpected missing event")
	}
	cursor, end := request.Mark(nil, false)
	if end || cursor == nil {
		t.Error("Unexpected end")
	}
	if request.Event(cursor) == nil {
		t.Error("Unexpected missing event")
	}
	cursor, end = request.Mark(cursor, true)
	if end || cursor == nil {
		t.Error("Unexpected end")
	}
	if request.Event(cursor) == nil {
		t.Error("Unexpected missing event")
	}
	cursor, end = request.Mark(cursor, false)
	if !end || cursor != nil {
		t.Error("Missing end")
	}
	request.Reset()
	if request.Event(nil) == nil {
		t.Error("Unexpected missing event")
	}
	cursor, end = request.Mark(nil, false)
	if end || cursor == nil {
		t.Error("Unexpected end")
	}
	if request.Event(cursor) == nil {
		t.Error("Unexpected missing event")
	}
	cursor, end = request.Mark(cursor, true)
	if !end || cursor != nil {
		t.Error("Missing end")
	}
	if request.Remaining() != 1 {
		t.Errorf("Unexpected remaining count: %d", request.Remaining())
	}
	if request.Created() != 2 {
		t.Errorf("Unexpected created count: %d", request.Created())
	}
	if request.AckSequence() != 0 {
		t.Errorf("Unexpected ack sequence: %d", request.AckSequence())
	}
}
