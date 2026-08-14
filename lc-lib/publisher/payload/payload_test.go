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

package payload

import (
	"context"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

type testContext string

const (
	testContextOffset testContext = "offset"
	testNonce                     = "12345678901234567890123456"
)

func createTestPayload(t *testing.T, numEvents int) *Payload {
	testEvents := make([]*event.Event, numEvents)
	for idx := range testEvents {
		ctx := context.WithValue(context.Background(), testContextOffset, idx)
		testEvents[idx] = event.NewEvent(ctx, nil, map[string]interface{}{})
	}

	ret := NewPayload(testEvents)
	ret.Cache = []byte("Dummy")
	return ret
}

func createTestPayloadWithDrops(t *testing.T, numEvents int, dropped map[int]bool) *Payload {
	testEvents := make([]*event.Event, numEvents)
	for idx := range testEvents {
		ctx := context.WithValue(context.Background(), testContextOffset, idx)
		testEvents[idx] = event.NewEvent(ctx, nil, map[string]interface{}{})
		if dropped[idx] {
			testEvents[idx].Drop()
		}
	}

	ret := NewPayload(testEvents)
	ret.Cache = []byte("Dummy")
	return ret
}

func verifyAck(t *testing.T, payload *Payload, n int, expLines int, expFull bool) {
	lines, full := payload.Ack(n)
	if lines != expLines {
		t.Errorf("Ack returned event count is wrong, got: %d, expected: %d", lines, expLines)
	}
	if full != expFull {
		t.Errorf("Ack full signal is wrong, got: %t, expected: %t", full, expFull)
	}
}

func verifyPayload(t *testing.T, payload *Payload, ack bool, complete bool, numEvents int, startEvent int) {
	if got := payload.HasAck(); got != ack {
		t.Errorf("Payload has ack flag wrong, got: %t, expected: %t", got, ack)
	}

	if ack && payload.Cache != nil {
		t.Errorf("Payload has ack but cache was not cleared")
	}

	if got := payload.Complete(); got != complete {
		t.Errorf("Payload has completed flag wrong, got: %t, expected: %t", got, complete)
	}

	events := payload.Rollup()
	if len(events) != numEvents {
		t.Errorf("Payload rollup event count wrong, got: %d, expected: %d", len(events), numEvents)
	}

	// Ignore events if we already failed
	if t.Failed() {
		return
	}

	for _, event := range events {
		offset := event.Context().Value(testContextOffset).(int)
		if offset != startEvent {
			t.Errorf("Payload rollup event offset wrong, got: %d, expected: %d", offset, startEvent)
		}
		startEvent++
	}
}

func TestPayloadNew(t *testing.T) {
	payload := createTestPayload(t, 1024)

	verifyPayload(t, payload, false, false, 0, 0)
}

func TestPayloadFullAck(t *testing.T) {
	payload := createTestPayload(t, 1024)

	verifyAck(t, payload, 1024, 1024, true)

	t.Log("First check")
	verifyPayload(t, payload, true, true, 1024, 0)
	t.Log("Second check")
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadPartialAck(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Initial partial ack")
	verifyAck(t, payload, 64, 64, false)
	verifyPayload(t, payload, true, false, 64, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Second partial ack")
	verifyAck(t, payload, 132, 68, false)
	verifyPayload(t, payload, true, false, 68, 64)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Repeated partial ack")
	verifyAck(t, payload, 132, 0, false)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Double ack")
	verifyAck(t, payload, 148, 16, false)
	verifyAck(t, payload, 192, 44, false)
	verifyPayload(t, payload, true, false, 60, 132)

	t.Log("Final ack")
	verifyAck(t, payload, 1024, 832, true)
	verifyPayload(t, payload, true, true, 832, 192)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadResend(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Initial partial ack")
	verifyAck(t, payload, 512, 512, false)
	verifyPayload(t, payload, true, false, 512, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	payload.ResetSequence()

	t.Log("Initial partial ack on new sequence")
	verifyAck(t, payload, 256, 256, false)
	verifyPayload(t, payload, true, false, 256, 512)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Final ack on new sequence")
	verifyAck(t, payload, 512, 256, true)
	verifyPayload(t, payload, true, true, 256, 768)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadEdgeCases(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Invalid sequence < 0")
	verifyAck(t, payload, -1024, 0, false)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Sequence revert - initial ack")
	verifyAck(t, payload, 500, 500, false)
	verifyPayload(t, payload, true, false, 500, 0)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Sequence revert - reverted ack")
	verifyAck(t, payload, 246, 0, false)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Sequence revert - next ack")
	verifyAck(t, payload, 512, 12, false)
	verifyPayload(t, payload, true, false, 12, 500)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Sequence past end")
	verifyAck(t, payload, 2048, 512, true)
	verifyPayload(t, payload, true, true, 512, 512)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadDroppedLeading(t *testing.T) {
	payload := createTestPayloadWithDrops(t, 4, map[int]bool{0: true})

	if got := len(payload.Events()); got != 3 {
		t.Errorf("Events() included a dropped event, got: %d, expected: 3", got)
	}

	t.Log("Ack sweeps up the dropped event that leads the acked boundary")
	verifyAck(t, payload, 1, 1, false)
	verifyPayload(t, payload, true, false, 2, 0)
	verifyPayload(t, payload, false, false, 0, 0)
}

func TestPayloadDroppedTrailing(t *testing.T) {
	payload := createTestPayloadWithDrops(t, 4, map[int]bool{2: true})

	if got := len(payload.Events()); got != 3 {
		t.Errorf("Events() included a dropped event, got: %d, expected: 3", got)
	}

	t.Log("Ack sweeps up the dropped event that trails the acked boundary")
	verifyAck(t, payload, 2, 2, false)
	verifyPayload(t, payload, true, false, 3, 0)
	verifyPayload(t, payload, false, false, 0, 0)
}

func TestPayloadDroppedInterleaved(t *testing.T) {
	// [A, drop, B, drop, drop, C, D] - 4 sendable events amongst 3 drops
	payload := createTestPayloadWithDrops(t, 7, map[int]bool{1: true, 3: true, 4: true})

	if got := len(payload.Events()); got != 4 {
		t.Errorf("Events() included a dropped event, got: %d, expected: 4", got)
	}

	t.Log("First partial ack sweeps up one dropped event")
	verifyAck(t, payload, 1, 1, false)
	verifyPayload(t, payload, true, false, 2, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Second partial ack sweeps up two dropped events")
	verifyAck(t, payload, 2, 1, false)
	verifyPayload(t, payload, true, false, 3, 2)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Final ack completes the payload")
	verifyAck(t, payload, 4, 2, true)
	verifyPayload(t, payload, true, true, 2, 5)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadAllDropped(t *testing.T) {
	// Built directly, rather than via createTestPayloadWithDrops, since a
	// payload that never transmits anything never populates a wire cache -
	// unlike the other tests there is no Ack() call here to clear one
	testEvents := make([]*event.Event, 3)
	for idx := range testEvents {
		ctx := context.WithValue(context.Background(), testContextOffset, idx)
		testEvents[idx] = event.NewEvent(ctx, nil, map[string]interface{}{})
		testEvents[idx].Drop()
	}

	payload := NewPayload(testEvents)

	if !payload.HasAck() {
		t.Error("Payload with every event dropped was not immediately acknowledged")
	}
	if !payload.Complete() {
		t.Error("Payload with every event dropped was not immediately complete")
	}
	if payload.Cache != nil {
		t.Error("Payload with every event dropped did not clear its cache")
	}
	if got := payload.Len(); got != 0 {
		t.Errorf("Payload with every event dropped has a non-zero sendable length: %d", got)
	}
	if got := len(payload.Events()); got != 0 {
		t.Errorf("Payload with every event dropped unexpectedly has events to transmit: %d", got)
	}

	events := payload.Rollup()
	if len(events) != 3 {
		t.Errorf("Payload rollup event count wrong, got: %d, expected: 3", len(events))
	}
	for idx, evnt := range events {
		offset := evnt.Context().Value(testContextOffset).(int)
		if offset != idx {
			t.Errorf("Payload rollup event offset wrong, got: %d, expected: %d", offset, idx)
		}
	}
}

func TestPayloadDroppedResend(t *testing.T) {
	// [A, drop, B, C, drop, D] - 4 sendable events amongst 2 drops
	payload := createTestPayloadWithDrops(t, 6, map[int]bool{1: true, 4: true})

	t.Log("Initial partial ack covers two sendable events and their neighbouring drop")
	verifyAck(t, payload, 2, 2, false)
	verifyPayload(t, payload, true, false, 3, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	payload.ResetSequence()

	if got := payload.Len(); got != 2 {
		t.Errorf("ResetSequence did not exclude the remaining dropped event from the sendable length: %d", got)
	}
	if got := len(payload.Events()); got != 2 {
		t.Errorf("Events() after resend included a dropped event: %d", got)
	}

	t.Log("Resent ack sweeps up the remaining dropped event")
	verifyAck(t, payload, 1, 1, false)
	verifyPayload(t, payload, true, false, 2, 3)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Final ack completes the resend")
	verifyAck(t, payload, 2, 1, true)
	verifyPayload(t, payload, true, true, 1, 5)
	verifyPayload(t, payload, false, true, 0, 0)
}
