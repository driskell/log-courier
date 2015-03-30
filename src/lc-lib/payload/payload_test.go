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
	"time"
)

const (
	test_nonce = "12345678901234567890123456"
)

func createTestPayload(t *testing.T, num_events int) *Payload {
	test_events := make([]*core.EventDescriptor, num_events)
	for idx := range test_events {
		test_events[idx] = &core.EventDescriptor{
			Stream: nil,
			Offset: int64(idx),
			Event:  []byte(""),
		}
	}

	payload, err := NewPayload(test_events, test_nonce, time.Second)
	if err != nil {
		t.Log("Failed to create pending payload structure")
		t.FailNow()
	}

	return payload
}

func verifyPayload(t *testing.T, payload *Payload, ack bool, complete bool, num_events int, start_event int) {
	if got := payload.HasAck(); got != ack {
		t.Logf("Payload has ack flag wrong, got: %t, should be: %t", got, ack)
		t.FailNow()
	}

	if got := payload.Complete(); got != complete {
		t.Logf("Payload has completed flag wrong, got: %t, should be: %t", got, complete)
		t.FailNow()
	}

	events := payload.Rollup()
	if len(events) != num_events {
		t.Logf("Payload rollup event count wrong, got: %d, should be: %d", len(events), num_events)
		t.FailNow()
	}

	for _, event := range events {
		if event.Offset != int64(start_event) {
			t.Logf("Payload rollup event offset wrong, got: %d, should be: %d", event.Offset, start_event)
			t.FailNow()
		}
		start_event++
	}
}

func TestPayloadNew(t *testing.T) {
	payload := createTestPayload(t, 1024)

	verifyPayload(t, payload, false, false, 0, 0)
}

func TestPayloadFullAck(t *testing.T) {
	payload := createTestPayload(t, 1024)

	payload.Ack(1024)
	verifyPayload(t, payload, true, false, 1024, 0)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadPartialAck(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Initial partial ack")
	payload.Ack(64)
	verifyPayload(t, payload, true, false, 64, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Second partial ack")
	payload.Ack(132)
	verifyPayload(t, payload, true, false, 68, 64)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Repeated partial ack")
	payload.Ack(132)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Double ack")
	payload.Ack(148)
	payload.Ack(192)
	verifyPayload(t, payload, true, false, 60, 132)

	t.Log("Final ack")
	payload.Ack(1024)
	verifyPayload(t, payload, true, false, 832, 192)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadResend(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Initial partial ack")
	payload.Ack(512)
	verifyPayload(t, payload, true, false, 512, 0)
	verifyPayload(t, payload, false, false, 0, 0)

	payload.Generate()

	t.Log("Initial partial ack on new sequence")
	payload.Ack(256)
	verifyPayload(t, payload, true, false, 256, 512)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Final ack on new sequence")
	payload.Ack(512)
	verifyPayload(t, payload, true, false, 256, 768)
	verifyPayload(t, payload, false, true, 0, 0)
}

func TestPayloadEdgeCases(t *testing.T) {
	payload := createTestPayload(t, 1024)

	t.Log("Invalid sequence < 0")
	payload.Ack(-1024)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Sequence revert - initial ack")
	payload.Ack(500)
	verifyPayload(t, payload, true, false, 500, 0)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Sequence revert - reverted ack")
	payload.Ack(246)
	verifyPayload(t, payload, false, false, 0, 0)
	t.Log("Sequence revert - next ack")
	payload.Ack(512)
	verifyPayload(t, payload, true, false, 12, 500)
	verifyPayload(t, payload, false, false, 0, 0)

	t.Log("Sequence past end")
	payload.Ack(2048)
	verifyPayload(t, payload, true, false, 512, 512)
	verifyPayload(t, payload, false, true, 0, 0)
}
