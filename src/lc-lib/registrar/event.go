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
  "os"
)

type RegistrarEvent interface {
  Process(state map[core.Stream]*FileState)
}

type DiscoverEvent struct {
  stream   core.Stream
  source   string
  offset   int64
  fileinfo os.FileInfo
}

func NewDiscoverEvent(stream core.Stream, source string, offset int64, fileinfo os.FileInfo) *DiscoverEvent {
  return &DiscoverEvent{
    stream:   stream,
    source:   source,
    offset:   offset,
    fileinfo: fileinfo,
  }
}

type DeletedEvent struct {
  stream core.Stream
}

func NewDeletedEvent(stream core.Stream) *DeletedEvent {
  return &DeletedEvent{
    stream: stream,
  }
}

type RenamedEvent struct {
  stream core.Stream
  source string
}

func NewRenamedEvent(stream core.Stream, source string) *RenamedEvent {
  return &RenamedEvent{
    stream: stream,
    source: source,
  }
}

type EventsEvent struct {
  events []*core.EventDescriptor
}

func NewEventsEvent(events []*core.EventDescriptor) *EventsEvent {
  return &EventsEvent{
    events: events,
  }
}
