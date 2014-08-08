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

package harvester

import "fmt"

type HarvesterSnapshot struct {
	file  string
	speed float64
	count int64
}

func NewHarvesterSnapshot(file string, speed float64, count int64) *HarvesterSnapshot {
	return &HarvesterSnapshot{
		file:  file,
		speed: speed,
		count: count,
	}
}

func (h *HarvesterSnapshot) Description() string {
	return fmt.Sprintf("Harvester for %s", h.file)
}

func (h *HarvesterSnapshot) NumEntries() int {
	return 2
}

func (h *HarvesterSnapshot) Entry(n int) (string, string) {
	if n == 0 {
		return "Speed", fmt.Sprintf("%.2f", h.speed)
	} else {
		return "Count", fmt.Sprintf("%d", h.count)
	}
}
