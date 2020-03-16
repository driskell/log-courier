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

// TimeoutChan returns a channel which will receive the current time when
// the next endpoint hits its registered timeout
// TODO: Can be replaced by a TimeoutEvent sent on EventChan
func (s *Sink) TimeoutChan() <-chan time.Time {
	return s.timeoutTimer.C
}

// resetTimeoutTimer resets the TimeoutTimer() channel for the next timeout
func (s *Sink) resetTimeoutTimer() {
	if s.timeoutList.Len() == 0 {
		if !s.timeoutTimer.Stop() {
			<-s.timeoutTimer.C
		}
		return
	}

	timeout := s.timeoutList.Front().Value.(*Timeout)
	log.Debug("Timeout timer reset - due at %v", timeout.timeoutDue)
	s.timeoutTimer.Reset(timeout.timeoutDue.Sub(time.Now()))
}

// RegisterTimeout registers a timeout structure with a timeout and timeout callback
func (s *Sink) RegisterTimeout(timeout *Timeout, duration time.Duration, timeoutFunc TimeoutFunc) {
	s.ClearTimeout(timeout)

	timeoutDue := time.Now().Add(duration)
	timeout.timeoutDue = timeoutDue
	timeout.timeoutFunc = timeoutFunc

	// Add to the list in time order
	// TODO: Need a sorted set to simplify this
	var existing, previous *internallist.Element
	for existing = s.timeoutList.Front(); existing != nil; existing = existing.Next() {
		if existing.Value.(*Timeout).timeoutDue.After(timeoutDue) {
			break
		}
		previous = existing
	}

	if previous == nil {
		s.timeoutList.PushFront(&timeout.timeoutElement)
	} else {
		s.timeoutList.InsertAfter(&timeout.timeoutElement, previous)
	}

	s.resetTimeoutTimer()
}

// ClearTimeout removes a timeout structure
func (s *Sink) ClearTimeout(timeout *Timeout) {
	if timeout.timeoutFunc == nil {
		return
	}

	// Remove existing entry
	s.timeoutList.Remove(&timeout.timeoutElement)
}

// ProcessTimeouts processes all pending timeouts
func (s *Sink) ProcessTimeouts() {
	next := s.timeoutList.Front()
	if next == nil {
		return
	}

	for {
		timeout := s.timeoutList.Remove(next).(*Timeout)
		if callback := timeout.timeoutFunc; callback != nil {
			timeout.timeoutFunc = nil
			callback()
		}

		next = s.timeoutList.Front()
		if next == nil || next.Value.(*Timeout).timeoutDue.After(time.Now()) {
			// No more due
			break
		}
	}

	s.resetTimeoutTimer()
}
