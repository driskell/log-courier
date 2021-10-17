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
// (Exponential moving average)
func CalculateRunningAverage(period float64, totalPeriods float64, average float64, measurement float64) float64 {
	if average == 0. {
		return measurement
	}

	exp := math.Exp(period / -totalPeriods)
	return (1-exp)*measurement + exp*(average)
}
