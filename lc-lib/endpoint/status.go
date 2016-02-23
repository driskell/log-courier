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

// status holds an Endpoint status
type status int

// Endpoint statuses
// Ordering is important due to use of >= etc.
const (
	// Not yet used
	endpointStatusIdle status = iota

	// Active
	endpointStatusActive

	// Could receive events but too many are oustanding
	endpointStatusFull

	// Do not use this endpoint, it has failed
	endpointStatusFailed

	// The endpoint is about to shutdown once pending payloads are complete
	endpointStatusClosing
)

func (s status) String() string {
	switch s {
	case endpointStatusIdle:
		return "Idle"
	case endpointStatusActive:
		return "Active"
	case endpointStatusFull:
		return "Full"
	case endpointStatusFailed:
		return "Failed"
	case endpointStatusClosing:
		return "Shutting down"
	}
	return "Unknown"
}

// IsIdle returns true if this Endpoint is idle (newly created and unused)
func (e *Endpoint) IsIdle() bool {
	return e.status == endpointStatusIdle
}

// IsActive returns true if this Endpoint is active
func (e *Endpoint) IsActive() bool {
	return e.status == endpointStatusActive
}

// IsFull returns true if this endpoint has been marked as full
func (e *Endpoint) IsFull() bool {
	return e.status == endpointStatusFull
}

// IsFailed returns true if this endpoint has been marked as failed
func (e *Endpoint) IsFailed() bool {
	return e.status == endpointStatusFailed
}

// IsClosing returns true if this Endpoint is closing down
func (e *Endpoint) IsClosing() bool {
	return e.status == endpointStatusClosing
}

// IsNotFull returns true if this endpoint is alive and not full
func (e *Endpoint) IsNotFull() bool {
	return e.status < endpointStatusFailed
}

// IsAlive returns true if this endpoint is not failed or closing
func (e *Endpoint) IsAlive() bool {
	return e.status < endpointStatusFailed
}
