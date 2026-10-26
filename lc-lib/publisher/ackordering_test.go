package publisher

import (
	"context"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
)

type testContextKey string

const testContextName testContextKey = "name"

// recordingAcker records the names of acknowledged events in the order they were dispatched
type recordingAcker struct {
	names []string
}

func (a *recordingAcker) Acknowledge(events []*event.Event) {
	for _, evnt := range events {
		a.names = append(a.names, evnt.Context().Value(testContextName).(string))
	}
}

func newTestOrderingPayload(acker event.Acknowledger, names ...string) *payload.Payload {
	events := make([]*event.Event, len(names))
	for idx, name := range names {
		events[idx] = event.NewEvent(context.WithValue(context.Background(), testContextName, name), acker, map[string]interface{}{})
	}
	return payload.NewPayload(events)
}

func assertDispatched(t *testing.T, acker *recordingAcker, expected ...string) {
	t.Helper()
	if len(expected) == 0 {
		expected = nil
	}
	if !reflect.DeepEqual(acker.names, expected) {
		t.Fatalf("Expected events %v to be dispatched, got %v", expected, acker.names)
	}
}

func assertCompleted(t *testing.T, completed int64, expected int64) {
	t.Helper()
	if completed != expected {
		t.Fatalf("Expected %d payloads completed, got %d", expected, completed)
	}
}

func TestOrderedAcksHoldsLaterPayloadUntilEarlierCompletes(t *testing.T) {
	acker := &recordingAcker{}
	ordering := OrderedAcks()
	first := newTestOrderingPayload(acker, "a1", "a2")
	second := newTestOrderingPayload(acker, "b1", "b2")
	ordering.add(first)
	ordering.add(second)

	second.Ack(2)
	assertCompleted(t, ordering.acknowledged(second, true), 0)
	assertDispatched(t, acker)

	first.Ack(2)
	assertCompleted(t, ordering.acknowledged(first, true), 2)
	assertDispatched(t, acker, "a1", "a2", "b1", "b2")
}

func TestUnorderedAcksDispatchesEachPayloadImmediately(t *testing.T) {
	acker := &recordingAcker{}
	ordering := UnorderedAcks()
	first := newTestOrderingPayload(acker, "a1", "a2")
	second := newTestOrderingPayload(acker, "b1", "b2")
	ordering.add(first)
	ordering.add(second)

	second.Ack(2)
	assertCompleted(t, ordering.acknowledged(second, true), 1)
	assertDispatched(t, acker, "b1", "b2")

	first.Ack(2)
	assertCompleted(t, ordering.acknowledged(first, true), 1)
	assertDispatched(t, acker, "b1", "b2", "a1", "a2")
}

func TestUnorderedAcksDispatchesPartialAcknowledgements(t *testing.T) {
	acker := &recordingAcker{}
	ordering := UnorderedAcks()
	pending := newTestOrderingPayload(acker, "a1", "a2", "a3")
	ordering.add(pending)

	pending.Ack(2)
	assertCompleted(t, ordering.acknowledged(pending, true), 0)
	assertDispatched(t, acker, "a1", "a2")

	pending.Ack(3)
	assertCompleted(t, ordering.acknowledged(pending, false), 1)
	assertDispatched(t, acker, "a1", "a2", "a3")
}

func TestUnorderedAcksCompletesPayloadOnce(t *testing.T) {
	acker := &recordingAcker{}
	ordering := UnorderedAcks()
	pending := newTestOrderingPayload(acker, "a1")
	ordering.add(pending)

	pending.Ack(1)
	assertCompleted(t, ordering.acknowledged(pending, true), 1)
	pending.Ack(1)
	assertCompleted(t, ordering.acknowledged(pending, false), 0)
	assertDispatched(t, acker, "a1")
}

func TestAckOrderingsCompleteFullyDroppedPayload(t *testing.T) {
	for name, ordering := range map[string]AckOrdering{"ordered": OrderedAcks(), "unordered": UnorderedAcks()} {
		t.Run(name, func(t *testing.T) {
			acker := &recordingAcker{}
			events := []*event.Event{event.NewEvent(context.WithValue(context.Background(), testContextName, "dropped"), acker, map[string]interface{}{})}
			events[0].Drop()
			pending := payload.NewPayload(events)
			ordering.add(pending)

			assertCompleted(t, ordering.acknowledged(pending, true), 1)
			assertDispatched(t, acker, "dropped")
		})
	}
}
