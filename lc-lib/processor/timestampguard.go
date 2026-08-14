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

package processor

import (
	"sync/atomic"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
)

// timestampGuard drops events whose @timestamp falls outside a configured age
// window, counting how many were dropped for each bound
// It is safe for concurrent use, as Check is called from every processor
// routine, and the limits and counters are read by the admin API
type timestampGuard struct {
	maximumAge       atomic.Int64 // nanoseconds, 0 disables the check
	maximumFutureAge atomic.Int64 // nanoseconds, 0 disables the check

	droppedTooOld atomic.Int64
	droppedFuture atomic.Int64
}

// SetLimits updates the age bounds enforced by the guard
func (g *timestampGuard) SetLimits(maximumAge, maximumFutureAge time.Duration) {
	g.maximumAge.Store(int64(maximumAge))
	g.maximumFutureAge.Store(int64(maximumFutureAge))
}

// MaximumAge returns the currently configured maximum event age
func (g *timestampGuard) MaximumAge() time.Duration {
	return time.Duration(g.maximumAge.Load())
}

// MaximumFutureAge returns the currently configured maximum future event age
func (g *timestampGuard) MaximumFutureAge() time.Duration {
	return time.Duration(g.maximumFutureAge.Load())
}

// Check marks the event as dropped if its @timestamp falls outside the
// configured bounds, relative to now, and returns true if it was dropped
func (g *timestampGuard) Check(evnt *event.Event, now time.Time) bool {
	maximumAge := g.MaximumAge()
	maximumFutureAge := g.MaximumFutureAge()
	if maximumAge == 0 && maximumFutureAge == 0 {
		return false
	}

	timestamp, ok := evnt.MustResolve("@timestamp", nil).(event.Timestamp)
	if !ok {
		return false
	}
	value := time.Time(timestamp)

	if maximumAge != 0 && value.Before(now.Add(-maximumAge)) {
		g.droppedTooOld.Add(1)
		evnt.Drop()
		return true
	}
	if maximumFutureAge != 0 && value.After(now.Add(maximumFutureAge)) {
		g.droppedFuture.Add(1)
		evnt.Drop()
		return true
	}

	return false
}

// Dropped returns the number of events dropped so far for each bound
func (g *timestampGuard) Dropped() (tooOld, future int64) {
	return g.droppedTooOld.Load(), g.droppedFuture.Load()
}
