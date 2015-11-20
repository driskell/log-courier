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
