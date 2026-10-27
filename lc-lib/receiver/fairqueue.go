/*
 * Copyright 2026 Jason Woods and contributors
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

package receiver

import (
	"math"

	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	// fairQueueQuantum is the byte credit a connection earns each time the drain reaches it
	fairQueueQuantum = 64 * 1024
	// fairQueueSliceEvents is the most events drained from a payload at once, so large payloads share the drain
	fairQueueSliceEvents = 256
)

// fairQueueEntry is a received payload waiting to be drained
type fairQueueEntry struct {
	events []*event.Event
	size   int
	// offset and sent are the events and bytes already drained
	offset int
	sent   int
}

// fairQueueConnection holds one connection's queued payloads in the order they were received
type fairQueueConnection struct {
	key     interface{}
	entries []*fairQueueEntry
	deficit int
	// credited is set once the connection has earned its quantum on the drain's current visit
	credited bool
}

// slice returns the next events to drain from the connection's oldest payload, with their share of its size
func (c *fairQueueConnection) slice() *fairQueueSlice {
	entry := c.entries[0]
	count := min(fairQueueSliceEvents, len(entry.events)-entry.offset)
	last := entry.offset+count == len(entry.events)
	size := entry.size - entry.sent
	if !last {
		size = entry.size * count / len(entry.events)
	}
	return &fairQueueSlice{events: entry.events[entry.offset : entry.offset+count], size: size, last: last}
}

// fairQueueSlice is the next events to drain
type fairQueueSlice struct {
	events []*event.Event
	size   int
	// last is set when the slice completes its payload
	last bool
}

// fairQueue drains payloads queued by many connections using deficit round robin, so a connection with a
// large backlog cannot hold back a connection with little queued
type fairQueue struct {
	connections map[interface{}]*fairQueueConnection
	// ring holds the connections with payloads queued, and cursor is the one the drain is visiting
	ring   []*fairQueueConnection
	cursor int
	// next is the slice chosen by peek, kept until pop so it does not change while waiting to be sent
	next *fairQueueSlice
}

func newFairQueue() *fairQueue {
	return &fairQueue{connections: make(map[interface{}]*fairQueueConnection)}
}

// push queues a payload received from a connection
func (q *fairQueue) push(connection interface{}, events []*event.Event, size int) {
	queue, ok := q.connections[connection]
	if !ok {
		queue = &fairQueueConnection{key: connection}
		q.connections[connection] = queue
		q.ring = append(q.ring, queue)
	}
	queue.entries = append(queue.entries, &fairQueueEntry{events: events, size: size})
}

// queued returns true when the connection has events waiting to be drained
func (q *fairQueue) queued(connection interface{}) bool {
	_, ok := q.connections[connection]
	return ok
}

// empty returns true when nothing is queued
func (q *fairQueue) empty() bool {
	return len(q.ring) == 0
}

// peek returns the next slice to drain without removing it, or nil when nothing is queued
func (q *fairQueue) peek() *fairQueueSlice {
	if q.next != nil || len(q.ring) == 0 {
		return q.next
	}

	unserved, minRounds := 0, math.MaxInt
	for {
		if q.cursor >= len(q.ring) {
			q.cursor = 0
		}
		queue := q.ring[q.cursor]
		if !queue.credited {
			queue.deficit += fairQueueQuantum
			queue.credited = true
		}
		slice := queue.slice()
		if slice.size <= queue.deficit {
			q.next = slice
			return slice
		}

		minRounds = min(minRounds, (slice.size-queue.deficit+fairQueueQuantum-1)/fairQueueQuantum)
		queue.credited = false
		q.cursor++
		unserved++
		if unserved == len(q.ring) {
			// Nothing could be served in a full round, so credit the rounds needed until something can be
			for _, each := range q.ring {
				each.deficit += (minRounds - 1) * fairQueueQuantum
			}
			unserved, minRounds = 0, math.MaxInt
		}
	}
}

// pop removes the slice returned by peek, returning its size and whether it completed a payload
func (q *fairQueue) pop() (int, bool) {
	slice := q.next
	q.next = nil
	queue := q.ring[q.cursor]
	entry := queue.entries[0]
	entry.offset += len(slice.events)
	entry.sent += slice.size
	queue.deficit -= slice.size

	if slice.last {
		queue.entries[0] = nil
		queue.entries = queue.entries[1:]
		if len(queue.entries) == 0 {
			// An emptied connection leaves the drain with its credit, so it cannot bank credit while idle
			q.ring = append(q.ring[:q.cursor], q.ring[q.cursor+1:]...)
			delete(q.connections, queue.key)
		}
	}
	return slice.size, slice.last
}
