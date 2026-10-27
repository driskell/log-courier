package receiver

import (
	"context"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

func newFairQueueTestEvents(count int) []*event.Event {
	events := make([]*event.Event, count)
	for idx := range events {
		events[idx] = event.NewEvent(context.Background(), nil, map[string]interface{}{})
	}
	return events
}

// drainedSlice records a drained slice and the payload it completed, if any
type drainedSlice struct {
	events    []*event.Event
	size      int
	completed bool
}

func drainFairQueue(t *testing.T, queue *fairQueue) []drainedSlice {
	t.Helper()
	var drained []drainedSlice
	for !queue.empty() {
		slice := queue.peek()
		if slice == nil {
			t.Fatal("Queue returned no slice while not empty")
		}
		size, completed := queue.pop()
		drained = append(drained, drainedSlice{events: slice.events, size: size, completed: completed})
	}
	if queue.peek() != nil {
		t.Fatal("Queue returned a slice once empty")
	}
	return drained
}

func TestFairQueueDrainsEachConnectionInOrderExactlyOnce(t *testing.T) {
	queue := newFairQueue()
	pushed := map[string][]*event.Event{}
	for _, name := range []string{"a", "b", "a", "c", "b", "a"} {
		events := newFairQueueTestEvents(300)
		queue.push(name, events, 300*400)
		pushed[name] = append(pushed[name], events...)
	}

	owner := map[*event.Event]string{}
	position := map[*event.Event]int{}
	for name, events := range pushed {
		for idx, evnt := range events {
			owner[evnt] = name
			position[evnt] = idx
		}
	}
	seen := map[*event.Event]bool{}
	next := map[string]int{}
	for _, slice := range drainFairQueue(t, queue) {
		for _, evnt := range slice.events {
			if seen[evnt] {
				t.Fatal("Event drained more than once")
			}
			seen[evnt] = true
			name := owner[evnt]
			if position[evnt] != next[name] {
				t.Fatalf("Connection %s drained event %d when event %d was next", name, position[evnt], next[name])
			}
			next[name]++
		}
	}
	if len(seen) != len(owner) {
		t.Fatalf("Drained %d events, expected %d", len(seen), len(owner))
	}
}

func TestFairQueueServesLightConnectionAheadOfHeavyBacklog(t *testing.T) {
	queue := newFairQueue()
	for i := 0; i < 10; i++ {
		queue.push("heavy", newFairQueueTestEvents(1024), 1024*400)
	}
	light := newFairQueueTestEvents(10)
	queue.push("light", light, 10*400)

	for idx, slice := range drainFairQueue(t, queue) {
		if slice.events[0] == light[0] {
			if idx > 1 {
				t.Fatalf("Light payload drained at position %d, behind the heavy backlog", idx)
			}
			return
		}
	}
	t.Fatal("Light payload was never drained")
}

func TestFairQueueSplitsLargePayloadsIntoSlices(t *testing.T) {
	queue := newFairQueue()
	queue.push("conn", newFairQueueTestEvents(1000), 1001)

	drained := drainFairQueue(t, queue)
	expected := []int{256, 256, 256, 232}
	if len(drained) != len(expected) {
		t.Fatalf("Expected %d slices, got %d", len(expected), len(drained))
	}
	total := 0
	for idx, slice := range drained {
		if len(slice.events) != expected[idx] {
			t.Fatalf("Slice %d had %d events, expected %d", idx, len(slice.events), expected[idx])
		}
		if slice.completed != (idx == len(drained)-1) {
			t.Fatalf("Slice %d completed %v, expected only the last to complete the payload", idx, slice.completed)
		}
		total += slice.size
	}
	if total != 1001 {
		t.Fatalf("Slice sizes totalled %d, expected the payload size 1001", total)
	}
}

func TestFairQueueRemovesEmptiedConnectionWithItsCredit(t *testing.T) {
	queue := newFairQueue()
	queue.push("conn", newFairQueueTestEvents(1), 100)
	drainFairQueue(t, queue)
	if len(queue.ring) != 0 || len(queue.connections) != 0 {
		t.Fatalf("Emptied connection remained queued: ring %d, connections %d", len(queue.ring), len(queue.connections))
	}

	queue.push("conn", newFairQueueTestEvents(1), 100)
	if deficit := queue.connections["conn"].deficit; deficit != 0 {
		t.Fatalf("Requeued connection kept credit %d from before it emptied", deficit)
	}
}

func TestFairQueueDrainsSliceCostingManyQuanta(t *testing.T) {
	queue := newFairQueue()
	queue.push("huge", newFairQueueTestEvents(1), 1<<30)
	queue.push("small", newFairQueueTestEvents(1), 100)

	drained := drainFairQueue(t, queue)
	if len(drained) != 2 {
		t.Fatalf("Expected 2 slices, got %d", len(drained))
	}
}

func TestFairQueueDrainsEmptyPayload(t *testing.T) {
	queue := newFairQueue()
	queue.push("conn", nil, 10)

	drained := drainFairQueue(t, queue)
	if len(drained) != 1 || len(drained[0].events) != 0 || !drained[0].completed || drained[0].size != 10 {
		t.Fatalf("Expected one empty slice completing the payload with its size, got %+v", drained)
	}
}
