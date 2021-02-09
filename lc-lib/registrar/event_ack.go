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

package registrar

import (
	"github.com/driskell/log-courier/lc-lib/event"
)

// AckEvent is a registrar ack event which triggers an update to the saved
// resume offsets for a file
type AckEvent struct {
	events []*event.Event
}

// NewAckEvent creates a new registrar ack event
func NewAckEvent(events []*event.Event) *AckEvent {
	return &AckEvent{
		events: events,
	}
}

// process persists the ack event into the registrar state by storing the offset
func (e *AckEvent) process(state map[Entry]*FileState) {
	if len(e.events) == 1 {
		log.Debug("Registrar received offsets for %d log entries", len(e.events))
	} else {
		log.Debug("Registrar received offsets for %d log entries", len(e.events))
	}

	var lastEntry Entry
	var lastState *FileState
	var isFound bool
	for _, event := range e.events {
		ctx := event.Context()
		entry := ctx.Value(ContextEntry).(Entry)
		log.Debug("%v", entry)
		if lastEntry == nil || lastEntry != entry {
			lastEntry = entry
			lastState, isFound = state[entry]
		}
		if !isFound {
			// This is probably stdin then or a deleted file we can't resume
			continue
		}

		endOffset := ctx.Value(ContextEndOffset).(int64)
		if lastState.Offset > endOffset {
			log.Debug("Registrar is reverting the offset for %s from %d to %d", *lastState.Source, lastState.Offset, endOffset)
		}

		lastState.Offset = endOffset
	}
}
