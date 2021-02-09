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

// DeletedEvent informs the registrar of a file deletion so it can remove
// unnecessary states from the state file
type DeletedEvent struct {
	entry Entry
}

// NewDeletedEvent creates a new deletion event
func NewDeletedEvent(entry Entry) *DeletedEvent {
	return &DeletedEvent{
		entry: entry,
	}
}

// process persists the deletion event into the state
func (e *DeletedEvent) process(state map[Entry]*FileState) {
	if _, ok := state[e.entry]; ok {
		log.Debug("Registrar received a deletion event for %s", *state[e.entry].Source)
	} else {
		log.Warning("Registrar received a deletion event for an unknown context")
	}

	// Purge the registrar entry - means the file is deleted so we can't resume
	// This keeps the state clean so it doesn't build up after thousands of log files
	delete(state, e.entry)
}
