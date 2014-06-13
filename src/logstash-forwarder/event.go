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

package main

import "os"

type Event map[string]interface{}

type FileEvent struct {
  ProspectorInfo *ProspectorInfo
  Offset         int64
  Event          Event
}

type RegistrarEvent interface {
  Process(state map[*ProspectorInfo]*FileState)
}

type NewFileEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         string
  Offset         int64
  fileinfo       os.FileInfo
}

type DeletedEvent struct {
  ProspectorInfo *ProspectorInfo
}

type RenamedEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         string
}

type EventsEvent struct {
  Events []*FileEvent
}

func NewEvent(fields map[string]string, file *string, offset int64, line uint64, message *string) Event {
  event := Event{
    "file":    file,
    "offset":  offset,
    "message": message,
  }
  for k, v := range fields {
    event[k] = &v
  }
  return event
}
