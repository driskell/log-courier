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

package endpoint

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/internallist"
)

// TimeoutFunc describes a callback that can be registered with the sink timer
// channel
type TimeoutFunc func()

// Timeout holds timeout information for use with the sink timeout channel
// It can be embedded into any other structure that requires timeout support
type Timeout struct {
	timeoutDue     time.Time
	timeoutElement internallist.Element
	timeoutFunc    TimeoutFunc
}

// InitTimeout initialises the timeout structure
func (t *Timeout) InitTimeout() {
	t.timeoutElement.Value = t
}

// RegisterTimeout registers a timeout structure with a timeout and timeout callback
func (f *Sink) RegisterTimeout(timeout *Timeout, duration time.Duration, timeoutFunc TimeoutFunc) {
	if timeout.timeoutFunc != nil {
		// Remove existing entry
		f.timeoutList.Remove(&timeout.timeoutElement)
	}

	timeoutDue := time.Now().Add(duration)
	timeout.timeoutDue = timeoutDue
	timeout.timeoutFunc = timeoutFunc

	// Add to the list in time order
	var existing *internallist.Element
	for existing = f.timeoutList.Front(); existing != nil; existing = existing.Next() {
		if existing.Value.(*Timeout).timeoutDue.After(timeoutDue) {
			break
		}
	}

	if existing == nil {
		f.timeoutList.PushFront(&timeout.timeoutElement)
	} else {
		f.timeoutList.InsertBefore(&timeout.timeoutElement, existing)
	}

	f.resetTimeoutTimer()
}

// ProcessTimeouts processes all pending timeouts
func (f *Sink) ProcessTimeouts() {
	next := f.timeoutList.Front()
	if next == nil {
		return
	}

	for {
		timeout := f.timeoutList.Remove(next).(*Timeout)
		if callback := timeout.timeoutFunc; callback != nil {
			timeout.timeoutFunc = nil
			callback()
		}

		next = f.timeoutList.Front()
		if next == nil || next.Value.(*Timeout).timeoutDue.After(time.Now()) {
			// No more due
			break
		}
	}

	f.resetTimeoutTimer()
}
