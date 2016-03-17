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

package core

import (
	"math"
	"time"
)

// DefaultExpFactor is the default factor for expontential backoff
const DefaultExpFactor = 1.25

// MaxExpFactor is the maximum backoff factor allowed
const MaxExpFactor = 60

// DefaultDelay is the default delay for an ExpBackoff structure when it is
// not specified
const DefaultDelay = 1 * time.Second

// ExpBackoff implements an exponential backoff helper
// The default delay is 1 second
type ExpBackoff struct {
	requiresInit bool
	defaultDelay time.Duration
	expFactor    float64
	expCount     float64
}

// NewExpBackoff creates a new ExpBackoff structure with the given default delay
func NewExpBackoff(defaultDelay time.Duration) *ExpBackoff {
	return &ExpBackoff{
		requiresInit: false,
		defaultDelay: defaultDelay,
		expFactor:    DefaultExpFactor,
	}
}

// Trigger informs the ExpBackoff that backoff needs to happen and returns the
// next delay to use
func (e *ExpBackoff) Trigger() time.Duration {
	if e.requiresInit {
		e.defaultDelay = DefaultDelay
		e.expFactor = DefaultExpFactor
	}

	// Calculate next delay factor - it starts at 1 due to starting expCount of 0
	factor := math.Pow(e.expFactor, e.expCount)
	if factor < MaxExpFactor {
		// Increase exponential delay but only if factor not hit max
		e.expCount++
	} else {
		factor = MaxExpFactor
	}

	nextDelay := time.Duration(float64(e.defaultDelay) * factor)
	log.Debug("Backoff: %v (factor: %f default: %v)", nextDelay, factor, e.defaultDelay)

	return nextDelay
}

// Reset resets the exponential backoff to default values
func (e *ExpBackoff) Reset() {
	e.expCount = 0.
}

func CalculateSpeed(duration time.Duration, speed float64, count float64, seconds_no_change *int) float64 {
	if count == 0 {
		*seconds_no_change++
	} else {
		*seconds_no_change = 0
	}

	if speed == 0. {
		return count
	}

	if *seconds_no_change >= 5 {
		*seconds_no_change = 0
		return 0.
	}

	// Calculate a moving average over 5 seconds - use similiar weight as load average
	return count + math.Exp(float64(duration)/float64(time.Second)/-5.)*(speed-count)
}
