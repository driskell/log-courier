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

package publisher

import (
	"github.com/driskell/log-courier/src/lc-lib/core"
)

// PendingPayload holds the data and acknowledged status of a spool of events
// and provides methods for processing acknowledgements so that a future resend
// of the payload does not resend acknowledged events
type PendingPayload struct {
	Nonce         string

	nextPayload   *PendingPayload
	events        []*core.EventDescriptor
	lastSequence  int
	sequenceLen   int
	ackEvents     int
	processed     int
	payload       []byte
}

// NewPendingPayload creates a new structure from the given spool of events
func NewPendingPayload(events []*core.EventDescriptor) (*PendingPayload, error) {
	payload := &PendingPayload{
		events:      events,
		sequenceLen: len(events),
	}

	return payload, nil
}

// Events returns the unacknowledged set of events in the payload
func (pp *PendingPayload) Events() []*core.EventDescriptor {
	return pp.events[pp.ackEvents:]
}

// Ack processes an acknowledgement sequence, marking events as sent and
// preventing resends from sending those events
// Returns the number of events acknowledged, with the second return value true
// if the payload is now completely acknowledged
func (pp *PendingPayload) Ack(sequence int) (int, bool) {
	if sequence <= pp.lastSequence {
		// No change
		return 0, false
	} else if sequence >= pp.sequenceLen {
		// Full ACK
		lines := pp.sequenceLen - pp.lastSequence
		pp.ackEvents = len(pp.events)
		pp.lastSequence = sequence
		pp.payload = nil
		return lines, true
	}

	lines := sequence - pp.lastSequence
	pp.ackEvents += lines
	pp.lastSequence = sequence
	pp.payload = nil
	return lines, false
}

// ResetSequence makes the first unacknowledged event have a sequence ID of 1
// This should be called before resending to ensure the ACK messages returned
// (which will use an ID of 1 for the first unacknowledged event) are understood
// correctly
func (pp *PendingPayload) ResetSequence() {
	pp.lastSequence = 0
	pp.sequenceLen = len(pp.events) - pp.ackEvents
}

// HasAck returns true if the payload has had at least one event acknowledged
func (pp *PendingPayload) HasAck() bool {
	return pp.ackEvents != 0
}

// Complete returns true if all events in this payload have been acknowledged
func (pp *PendingPayload) Complete() bool {
	return len(pp.events) == 0
}

// Rollup removes acknowledged events from the payload and returns them so they
// may be passed onto the Registrar
func (pp *PendingPayload) Rollup() []*core.EventDescriptor {
	pp.processed += pp.ackEvents
	rollup := pp.events[:pp.ackEvents]
	pp.events = pp.events[pp.ackEvents:]
	pp.ackEvents = 0
	pp.ResetSequence()
	return rollup
}
