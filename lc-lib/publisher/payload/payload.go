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
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/internallist"
)

// Payload holds the data and acknowledged status of a spool of events
// and provides methods for processing acknowledgements so that a future resend
// of the payload does not resend acknowledged events
// Events marked as dropped during processing remain part of the payload for
// acknowledgement purposes but are never offered for transmission
type Payload struct {
	events       []*event.Event
	lastSequence int
	sequenceLen  int
	ackEvents    int
	processed    int
	hasDropped   bool

	Nonce         string
	Resending     bool
	Element       internallist.Element
	ResendElement internallist.Element
	Cache         interface{}
}

// NewPayload initialises a new payload structure from the given spool of events
func NewPayload(events []*event.Event) *Payload {
	ret := &Payload{
		events: events,
	}

	for _, evnt := range events {
		if evnt.Dropped() {
			ret.hasDropped = true
			break
		}
	}

	ret.sequenceLen = ret.sendableCount(events)
	if ret.sequenceLen == 0 {
		// Every event was dropped during processing, so there is nothing to
		// transmit - treat the payload as immediately and fully acknowledged
		ret.ackEvents = len(events)
		ret.Cache = nil
	}

	ret.Init()

	return ret
}

// Init initialises the internallist elements and anything else requiring
// initialisation
func (pp *Payload) Init() {
	pp.Element.Value = pp
	pp.ResendElement.Value = pp
}

// Len returns the number of events in this payload that require transmission,
// excluding any that were dropped during processing
func (pp *Payload) Len() int {
	return pp.sequenceLen
}

// Events returns the unacknowledged set of events in the payload that require
// transmission, excluding any that were dropped during processing
func (pp *Payload) Events() []*event.Event {
	pending := pp.events[pp.ackEvents:]
	if !pp.hasDropped {
		return pending
	}

	sendable := make([]*event.Event, 0, len(pending))
	for _, evnt := range pending {
		if !evnt.Dropped() {
			sendable = append(sendable, evnt)
		}
	}
	return sendable
}

// sendableCount returns the number of the given events that require
// transmission, excluding any that were dropped during processing
func (pp *Payload) sendableCount(events []*event.Event) int {
	if !pp.hasDropped {
		return len(events)
	}

	count := 0
	for _, evnt := range events {
		if !evnt.Dropped() {
			count++
		}
	}
	return count
}

// mapSequence converts a number of newly acknowledged sendable events into the
// number of raw events - including any dropped events interleaved amongst them
// - that they cover from the front of the currently pending window
// Dropped events are swept along with the sendable event that immediately
// precedes them, since they are never separately transmitted or acknowledged
func (pp *Payload) mapSequence(count int) int {
	pending := pp.events[pp.ackEvents:]
	if !pp.hasDropped {
		return count
	}

	found := 0
	for idx, evnt := range pending {
		if evnt.Dropped() {
			continue
		}
		found++
		if found != count {
			continue
		}
		idx++
		for idx < len(pending) && pending[idx].Dropped() {
			idx++
		}
		return idx
	}
	// Defensive: should be unreachable, as callers only ever request a count
	// that the pending window is known to satisfy
	return len(pending)
}

// Ack processes an acknowledgement sequence, marking events as sent and
// preventing resends from sending those events
// The sequence is relative to the sendable events returned by Events(), not
// the raw event count, so dropped events are not separately acknowledged
// Returns the number of sendable events acknowledged, with the second return
// value true if the payload is now completely acknowledged
func (pp *Payload) Ack(sequence int) (int, bool) {
	if sequence <= pp.lastSequence {
		// No change
		return 0, false
	} else if sequence >= pp.sequenceLen {
		// Full ACK
		lines := pp.sequenceLen - pp.lastSequence
		pp.ackEvents = len(pp.events)
		pp.lastSequence = sequence
		pp.Cache = nil
		return lines, true
	}

	lines := sequence - pp.lastSequence
	pp.ackEvents += pp.mapSequence(lines)
	pp.lastSequence = sequence
	pp.Cache = nil
	return lines, false
}

// ResetSequence makes the first unacknowledged sendable event have a sequence
// ID of 1
// This should be called before resending to ensure the ACK messages returned
// (which will use an ID of 1 for the first unacknowledged event) are understood
// correctly
func (pp *Payload) ResetSequence() {
	pp.lastSequence = 0
	pp.sequenceLen = pp.sendableCount(pp.events[pp.ackEvents:])
	if pp.sequenceLen == 0 {
		// Defensive: everything remaining was dropped, so there is nothing
		// left to transmit - treat the payload as immediately and fully
		// acknowledged
		pp.ackEvents = len(pp.events)
	}
}

// HasAck returns true if the payload has had at least one event acknowledged
func (pp *Payload) HasAck() bool {
	return pp.ackEvents != 0
}

// Complete returns true if all events in this payload have been acknowledged
func (pp *Payload) Complete() bool {
	return len(pp.events[pp.ackEvents:]) == 0
}

// Rollup removes acknowledged events from the payload and returns them so they
// may be passed onto the Registrar
func (pp *Payload) Rollup() []*event.Event {
	pp.processed += pp.ackEvents
	rollup := pp.events[:pp.ackEvents]
	pp.events = pp.events[pp.ackEvents:]
	pp.ackEvents = 0
	return rollup
}
