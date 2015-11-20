/*
* Copyright 2014 Jason Woods.
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
	"github.com/driskell/log-courier/src/lc-lib/core"
	"testing"
)

const (
	testNonce = "12345678901234567890123456"
)

func createTestPayload(t *testing.T, numEvents int) *Payload {
	testEvents := make([]*core.EventDescriptor, numEvents)
	for idx := range testEvents {
		testEvents[idx] = &core.EventDescriptor{
			Stream: nil,
			Offset: int64(idx),
			Event:  []byte(""),
		}
	}

	return NewPayload(testEvents)
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
		if event.Offset != int64(startEvent) {
			t.Errorf("Payload rollup event offset wrong, got: %d, expected: %d", event.Offset, startEvent)
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
