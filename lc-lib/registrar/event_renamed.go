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

// RenamedEvent informs the registrar of a file rename that needs to be
// reflected within the state file
type RenamedEvent struct {
	ctx    context.Context
	source string
}

// NewRenamedEvent creates a new rename event
func NewRenamedEvent(ctx context.Context, source string) *RenamedEvent {
	return &RenamedEvent{
		ctx:    ctx,
		source: source,
	}
}

func (e *RenamedEvent) process(state map[context.Context]*FileState) {
	_, isFound := state[e.ctx]
	if !isFound {
		// This is probably stdin or a deleted file we can't resume
		return
	}

	log.Debug("Registrar received a rename event for %s -> %s", state[e.ctx].Source, e.source)

	// Update the stored file name
	state[e.ctx].Source = &e.source
}
