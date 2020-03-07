/*
 * Copyright 2014-2015 Jason Woods.
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
	"context"
)

// EventProcessor is implemented by all register events
type EventProcessor interface {
	process(state map[context.Context]*FileState)
}

// EventSpooler buffers registrar events for bulk sends
type EventSpooler struct {
	registrar *Registrar
	events    []EventProcessor
}

// NewEventSpooler creates a new EventSpooler
func NewEventSpooler(r *Registrar) *EventSpooler {
	ret := &EventSpooler{
		registrar: r,
	}
	ret.reset()
	return ret
}

// Add a registrar event to the spooler
func (r *EventSpooler) Add(event EventProcessor) {
	r.events = append(r.events, event)
}

// Send the buffered registrar events to the registrar
func (r *EventSpooler) Send() {
	if len(r.events) != 0 {
		r.registrar.registrarChan <- r.events
		r.reset()
	}
}

func (r *EventSpooler) reset() {
	r.events = make([]EventProcessor, 0, 0)
}
