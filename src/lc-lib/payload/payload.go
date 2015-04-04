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
	"github.com/driskell/log-courier/src/lc-lib/internallist"
)

// Payload holds the data and acknowledged status of a spool of events
// and provides methods for processing acknowledgements so that a future resend
// of the payload does not resend acknowledged events
type Payload struct {
	events        []*core.EventDescriptor
	lastSequence  int
	sequenceLen   int
	ackEvents     int
	processed     int
	payload       []byte

	Nonce         string
	Resending     bool
	Element       internallist.Element
	ResendElement internallist.Element
}

// NewPayload initialises a new payload structure from the given spool of events
func NewPayload(events []*core.EventDescriptor) *Payload {
	ret := &Payload{
		events:      events,
		sequenceLen: len(events),
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

// Events returns the unacknowledged set of events in the payload
func (pp *Payload) Events() []*core.EventDescriptor {
	return pp.events[pp.ackEvents:]
}

// Ack processes an acknowledgement sequence, marking events as sent and
// preventing resends from sending those events
// Returns the number of events acknowledged, with the second return value true
// if the payload is now completely acknowledged
func (pp *Payload) Ack(sequence int) (int, bool) {
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
func (pp *Payload) ResetSequence() {
	pp.lastSequence = 0
	pp.sequenceLen = len(pp.events) - pp.ackEvents
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
func (pp *Payload) Rollup() []*core.EventDescriptor {
	pp.processed += pp.ackEvents
	rollup := pp.events[:pp.ackEvents]
	pp.events = pp.events[pp.ackEvents:]
	pp.ackEvents = 0
	pp.ResetSequence()
	return rollup
}
