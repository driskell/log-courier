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

package prospector

type prospectorInfoSnapshot struct {
	file    string
	status  int
	running bool
}

func newProspectorInfoSnapshot(info *prospectorInfo) *prospectorInfoSnapshot {
	return &prospectorInfoSnapshot{
		file:    info.file,
		status:  info.status,
		running: info.running,
	}
}

type prospectorSnapshot struct {
	files []*prospectorInfoSnapshot
}

func newProspectorSnapshot(files []*prospectorInfoSnapshot) *prospectorSnapshot {
	return &prospectorSnapshot{
		files: files,
	}
}

func (h *prospectorSnapshot) Description() string {
	return "Prospector"
}

func (h *prospectorSnapshot) NumEntries() int {
	return len(h.files)
}

func (h *prospectorSnapshot) Entry(n int) (string, string) {
	entry := h.files[n]

	var status string
	if entry.status == statusFailed {
		status = "Failed"
	} else {
		if entry.running {
			status = "Running"
		} else {
			status = "Dead"
		}
	}

	return entry.file, status
}
