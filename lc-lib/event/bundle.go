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

package event

// Bundle represents a bundle of events and associated "marks"
// Marks are used to add additional context to a bundle
// For example, the Sequencer uses the marks to enforce ordering
// of bundles through parallel processing or remote transmission
type Bundle struct {
	events []*Event
	marks  map[interface{}]interface{}
}

// NewBundle creates a new bundle from the given events
func NewBundle(events []*Event) *Bundle {
	return &Bundle{
		events: events,
		marks:  make(map[interface{}]interface{}),
	}
}

// Mark saves a value against the given key
// Keys should be treated like context.Context and each module
// have their own type for them, to ensure no conflict
func (b *Bundle) Mark(key interface{}, value interface{}) {
	b.marks[key] = value
}

// Value fetches the value for the given key
func (b *Bundle) Value(key interface{}) interface{} {
	if v, ok := b.marks[key]; ok {
		return v
	}
	return nil
}

// Len returns the number of events in this bundle
func (b *Bundle) Len() int {
	return len(b.events)
}

// Events returns the events
func (b *Bundle) Events() []*Event {
	return b.events
}
