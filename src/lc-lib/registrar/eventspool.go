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
  "lc-lib/core"
)

type RegistrarEvent interface {
  Process(state map[core.Stream]*FileState)
}

type RegistrarEventSpool struct {
  registrar *Registrar
  events    []RegistrarEvent
}

func newRegistrarEventSpool(r *Registrar) *RegistrarEventSpool {
  ret := &RegistrarEventSpool{
    registrar: r,
  }
  ret.reset()
  return ret
}

func (r *RegistrarEventSpool) Close() {
  r.registrar.dereferenceSpooler()
}

func (r *RegistrarEventSpool) Add(event RegistrarEvent) {
  r.events = append(r.events, event)
}

func (r *RegistrarEventSpool) Send() {
  if len(r.events) != 0 {
    r.registrar.registrar_chan <- r.events
    r.reset()
  }
}

func (r *RegistrarEventSpool) reset() {
  r.events = make([]RegistrarEvent, 0, 0)
}
