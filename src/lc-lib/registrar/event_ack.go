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
  "github.com/driskell/log-courier/src/lc-lib/core"
)

type AckEvent struct {
  events []*core.EventDescriptor
}

func NewAckEvent(events []*core.EventDescriptor) *AckEvent {
  return &AckEvent{
    events: events,
  }
}

func (e *AckEvent) Process(state map[core.Stream]*FileState) {
  if len(e.events) == 1 {
    log.Debug("Registrar received offsets for %d log entries", len(e.events))
  } else {
    log.Debug("Registrar received offsets for %d log entries", len(e.events))
  }

  for _, event := range e.events {
    _, is_found := state[event.Stream]
    if !is_found {
      // This is probably stdin then or a deleted file we can't resume
      continue
    }

    state[event.Stream].Offset = event.Offset
  }
}
