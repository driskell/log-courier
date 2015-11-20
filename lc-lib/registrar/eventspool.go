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

package registrar

import (
	"github.com/driskell/log-courier/lc-lib/core"
)

type EventProcessor interface {
	Process(state map[core.Stream]*FileState)
}

type EventSpooler interface {
	Close()
	Add(EventProcessor)
	Send()
}

type EventSpool struct {
	registrar *Registrar
	events    []EventProcessor
}

func newEventSpool(r *Registrar) *EventSpool {
	ret := &EventSpool{
		registrar: r,
	}
	ret.reset()
	return ret
}

func (r *EventSpool) Close() {
	r.registrar.dereferenceSpooler()
	r.registrar = nil
}

func (r *EventSpool) Add(event EventProcessor) {
	r.events = append(r.events, event)
}

func (r *EventSpool) Send() {
	if len(r.events) != 0 {
		r.registrar.registrar_chan <- r.events
		r.reset()
	}
}

func (r *EventSpool) reset() {
	r.events = make([]EventProcessor, 0, 0)
}
