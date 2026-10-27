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

package transports

import (
	"context"
	"sync"
)

// EventsSegment is a run of events from one payload, which a connection may hand over as several segments
type EventsSegment struct {
	Nonce *string
	// Offset is the number of the payload's events in the segments before this one
	Offset uint32
	// Skipped is the number of events skipped at the start of the segment, which come before its events
	Skipped uint32
	// Raw holds undecoded events, and Decoded holds events from sources that decode as they read
	Raw     [][]byte
	Decoded []map[string]interface{}
	Size    int
	// Final is set on the last segment of a payload
	Final bool
}

// Count returns the number of events in the segment, excluding those skipped
func (s *EventsSegment) Count() uint32 {
	return uint32(len(s.Raw) + len(s.Decoded))
}

// ReceiveWindow holds the events a connection has read until the receiver pool takes them, so a connection reads
// ahead without waiting for as long as the events held fit within the window size. The pool is asked to take the
// events once they fill the window to a given fraction, or a payload ends, or the connection must wait for room.
type ReceiveWindow struct {
	context context.Context
	size    int
	// requestAt is how much must be held before the receiver pool is asked to take it
	requestAt int
	mutex     sync.Mutex
	segments  []*EventsSegment
	held      int
	// payload is the nonce of the payload being received, and offset the number of its events so far
	payload *string
	offset  uint32
	// requested is set while the receiver pool has been asked to take the held events
	requested bool
	taken     chan struct{}
}

// NewReceiveWindow creates a receive window of the given size for a connection, which asks for its events to be
// taken once they fill the given fraction of it
func NewReceiveWindow(context context.Context, size int, fill float64) *ReceiveWindow {
	return &ReceiveWindow{
		context:   context,
		size:      size,
		requestAt: max(1, int(float64(size)*fill)),
		taken:     make(chan struct{}, 1),
	}
}

// Context returns the connection context
func (w *ReceiveWindow) Context() context.Context {
	return w.context
}

// Size returns the most event data the window holds
func (w *ReceiveWindow) Size() int {
	return w.size
}

// Fits returns true if an event of the given size can be held now
func (w *ReceiveWindow) Fits(size int) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.held == 0 || w.held+size <= w.size
}

// Held returns the size of the event data held
func (w *ReceiveWindow) Held() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.held
}

// Empty returns true when nothing is held
func (w *ReceiveWindow) Empty() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return len(w.segments) == 0
}

// Taken returns a channel signalled each time the receiver pool takes the held events
func (w *ReceiveWindow) Taken() <-chan struct{} {
	return w.taken
}

// AddRaw holds an undecoded event from the payload with the given nonce, returning true if the receiver pool must
// now be asked to take the held events
func (w *ReceiveWindow) AddRaw(nonce *string, data []byte) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	segment := w.eventSegment(nonce)
	segment.Raw = append(segment.Raw, data)
	segment.Size += len(data)
	w.held += len(data)
	return w.held >= w.requestAt && w.request()
}

// Add holds a decoded event of the given size from the payload with the given nonce, returning true if the receiver
// pool must now be asked to take the held events
func (w *ReceiveWindow) Add(nonce *string, data map[string]interface{}, size int) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	segment := w.eventSegment(nonce)
	segment.Decoded = append(segment.Decoded, data)
	segment.Size += size
	w.held += size
	return w.held >= w.requestAt && w.request()
}

// Request returns true if the receiver pool must now be asked to take the held events, as the connection cannot
// continue until it does
func (w *ReceiveWindow) Request() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return len(w.segments) != 0 && w.request()
}

// Skip records an event from the payload with the given nonce that was not read, which the receiver pool learns of
// when it next takes the held events
func (w *ReceiveWindow) Skip(nonce *string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.start(nonce)
	segment := w.open()
	if segment == nil || segment.Count() != 0 {
		// Skipped events come before a segment's events, so a skip after events starts a new segment
		segment = w.newSegment()
	}
	segment.Skipped++
	w.offset++
}

// End marks the payload with the given nonce as complete, returning true if the receiver pool must now be asked to
// take the held events
func (w *ReceiveWindow) End(nonce *string) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.start(nonce)
	segment := w.open()
	if segment == nil {
		segment = w.newSegment()
	}
	segment.Final = true
	w.payload = nil
	w.offset = 0
	return w.request()
}

// Take removes and returns the held segments, in the order they were received
func (w *ReceiveWindow) Take() []*EventsSegment {
	w.mutex.Lock()
	segments := w.segments
	w.segments = nil
	w.held = 0
	w.requested = false
	w.mutex.Unlock()
	select {
	case w.taken <- struct{}{}:
	default:
	}
	return segments
}

// start begins a payload with the given nonce if none is being received
func (w *ReceiveWindow) start(nonce *string) {
	if w.payload == nil {
		w.payload = nonce
		w.offset = 0
	}
}

// open returns the held segment still receiving events for the current payload, or nil if there is none
func (w *ReceiveWindow) open() *EventsSegment {
	if len(w.segments) == 0 {
		return nil
	}
	segment := w.segments[len(w.segments)-1]
	if segment.Final {
		return nil
	}
	return segment
}

// eventSegment returns the segment the next event of the payload with the given nonce is added to
func (w *ReceiveWindow) eventSegment(nonce *string) *EventsSegment {
	w.start(nonce)
	segment := w.open()
	if segment == nil {
		segment = w.newSegment()
	}
	w.offset++
	return segment
}

// newSegment starts a new held segment for the current payload
func (w *ReceiveWindow) newSegment() *EventsSegment {
	segment := &EventsSegment{Nonce: w.payload, Offset: w.offset}
	w.segments = append(w.segments, segment)
	return segment
}

// request returns true if the receiver pool has not yet been asked to take the held events, marking it asked
func (w *ReceiveWindow) request() bool {
	if w.requested {
		return false
	}
	w.requested = true
	return true
}
