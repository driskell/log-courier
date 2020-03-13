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

package core

import (
	"math"
	"time"
)

// expFactor is the factor for expontential backoff
const expFactor = 2

// DefaultBaseDelay is the default delay for an ExpBackoff structure when it is
// not specified, or when it is zero (in which case this becomes the delay after
// the first immediate retry)
const DefaultBaseDelay = 1 * time.Second

// ExpBackoff implements an exponential backoff helper
// The default delay is 1 second
type ExpBackoff struct {
	// Constructor
	name      string
	baseDelay *time.Duration
	maxDelay  time.Duration

	// Internal
	expCount    int
	lastTrigger time.Time
}

// NewExpBackoff creates a new ExpBackoff structure with the given default delay
func NewExpBackoff(name string, baseDelay time.Duration, maxDelay time.Duration) *ExpBackoff {
	return &ExpBackoff{
		name:      name,
		baseDelay: &baseDelay,
		maxDelay:  maxDelay,
	}
}

// Trigger informs the ExpBackoff that backoff needs to happen and returns the next delay to use
// The delays will increase over time, and will only reset if the last Trigger call was as long ago as the next delay would be
func (e *ExpBackoff) Trigger() time.Duration {
	if e.baseDelay == nil {
		e.baseDelay = new(time.Duration)
		*e.baseDelay = DefaultBaseDelay
	}

	// Calculate next delay, and if time since last trigger is larger, reset
	// This enforces the "time to recover" that resets the back off clock to be longer than the delay that led to recovery
	// It prevents immediately retrying backoff timers hitting situation where they never back off
	nextDelay := e.calculateDelay(e.expCount)

	// Did we recover for long enough? Reset delay
	if e.expCount != 0 && time.Since(e.lastTrigger) > nextDelay {
		log.Debug("[%s] Backoff had recovered, resetting failured count", e.name)
		nextDelay = e.calculateDelay(0)
		e.expCount = 0
	}

	e.lastTrigger = time.Now()

	// Increase next delay (and the required recovery time) and enforce maximum delay
	e.expCount++
	if nextDelay > e.maxDelay {
		nextDelay = e.maxDelay
	}

	log.Debug("[%s] Backoff (%d failures): %v", e.name, e.expCount, nextDelay)
	return nextDelay
}

// calculateDelay returns the delay for the specified failure count
func (e *ExpBackoff) calculateDelay(expCount int) time.Duration {
	// If this is an immediately retry, return 0 if first retry, otherwise use default delay
	baseDelay := *e.baseDelay
	if baseDelay == 0 {
		if expCount == 0 {
			return 0
		}
		baseDelay = DefaultBaseDelay
	}

	// Calculate next delay factor - it starts at 1 due to starting expCount of 0
	factor := math.Pow(expFactor, float64(expCount))
	return time.Duration(float64(baseDelay) * factor)
}
