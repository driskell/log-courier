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

package core

import (
	"math"
	"time"
)

// expFactor is the factor for expontential backoff
const expFactor = 2

// DefaultDelay is the default delay for an ExpBackoff structure when it is
// not specified, or when it is zero (in which case this becomes the delay after
// the first immediate retry)
const DefaultDelay = 1 * time.Second

// ExpBackoff implements an exponential backoff helper
// The default delay is 1 second
type ExpBackoff struct {
	name         string
	requiresInit bool
	defaultDelay time.Duration
	maxDelay     time.Duration
	expCount     float64
}

// NewExpBackoff creates a new ExpBackoff structure with the given default delay
func NewExpBackoff(name string, defaultDelay time.Duration, maxDelay time.Duration) *ExpBackoff {
	return &ExpBackoff{
		name:         name,
		requiresInit: false,
		defaultDelay: defaultDelay,
		maxDelay:     maxDelay,
	}
}

// Trigger informs the ExpBackoff that backoff needs to happen and returns the
// next delay to use
func (e *ExpBackoff) Trigger() time.Duration {
	if e.requiresInit {
		e.defaultDelay = DefaultDelay
	}

	// If 0 initial delay, retry immediately if first failure
	delay := e.defaultDelay
	if delay == 0 {
		if e.expCount == 0. {
			e.expCount++
			return 0
		}

		// 0 initial delay, but not first retry, use defaultDelay
		delay = DefaultDelay
	}

	// Calculate next delay factor - it starts at 1 due to starting expCount of 0
	factor := math.Pow(expFactor, e.expCount)
	nextDelay := time.Duration(float64(delay) * factor)
	log.Debug("[%s] Backoff: %v (factor: %f default: %v)", e.name, nextDelay, factor, e.defaultDelay)

	if nextDelay < e.maxDelay {
		// Increase exponential delay but only if max delay not hit
		e.expCount++
	} else {
		nextDelay = e.maxDelay
	}

	return nextDelay
}

// Reset resets the exponential backoff to default values
func (e *ExpBackoff) Reset() {
	e.expCount = 0.
}

// CalculateSpeed returns a running average for a speed using variable time
// periods over 5 seconds. If all measurements are 0 in a 5 second period it
// will auto-reset
func CalculateSpeed(duration time.Duration, average float64, measurement float64, secondsNoChange *int) float64 {
	if measurement == 0 {
		*secondsNoChange += int(math.Ceil(float64(duration) / float64(time.Second)))
	} else {
		*secondsNoChange = 0
	}

	if *secondsNoChange >= 5 {
		*secondsNoChange = 0
		return 0.
	}

	// Calculate a moving average over 5 seconds - use similiar weight as load average
	return CalculateRunningAverage(float64(duration)/float64(time.Second), 5, average, measurement)
}

// CalculateRunningAverage returns a running average
// On the first call, where the existing average is 0, it will return the
// measurement unchanged
func CalculateRunningAverage(period float64, totalPeriods float64, average float64, measurement float64) float64 {
	if average == 0. {
		return measurement
	}

	return measurement + math.Exp(period/-totalPeriods)*(average-measurement)
}
