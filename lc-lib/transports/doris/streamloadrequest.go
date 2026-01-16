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

package doris

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/driskell/log-courier/lc-lib/event"
)

type streamLoadRequest struct {
	// Constructor
	events         []*event.Event
	columnDefs     map[string]string // column name -> type
	restJSONColumn string

	// Internal
	currentIndex int
	currentBytes []byte
	len          int
}

func newStreamLoadRequest(columnDefs map[string]string, restJSONColumn string, events []*event.Event) *streamLoadRequest {
	return &streamLoadRequest{
		events:         events,
		columnDefs:     columnDefs,
		restJSONColumn: restJSONColumn,
		currentIndex:   0,
	}
}

// EventCount returns the total number of events in this request
func (p *streamLoadRequest) EventCount() int {
	return len(p.events)
}

// Reset allows the request to be Read again
func (p *streamLoadRequest) Reset() {
	p.currentIndex = 0
	p.currentBytes = nil
	p.len = 0
}

// Len returns the total number of bytes read so far
func (p *streamLoadRequest) Len() int {
	return p.len
}

// Read implements io.Reader and returns a JSON array of events for Doris stream load
func (p *streamLoadRequest) Read(dst []byte) (n int, err error) {
	for len(dst) > 0 {
		if p.currentBytes == nil {
			if p.currentIndex >= len(p.events) {
				return n, io.EOF
			}

			// Convert event to JSON object with mapped columns
			eventData := p.events[p.currentIndex].Data()
			mappedEvent := make(map[string]interface{})
			restData := make(map[string]interface{})

			// Single loop: map columns and collect unmapped fields
			for key, value := range eventData {
				if _, isMapped := p.columnDefs[key]; isMapped {
					// Column is mapped - format and add to mappedEvent
					if key != p.restJSONColumn {
						mappedEvent[key] = value
					}
				} else if key != "@metadata" {
					// Unmapped field - add to rest data (exclude @ prefixed metadata fields)
					restData[key] = value
				}
			}

			// Add nil values for missing mapped columns
			for colName := range p.columnDefs {
				if colName != p.restJSONColumn {
					if _, exists := mappedEvent[colName]; !exists {
						mappedEvent[colName] = nil
					}
				}
			}

			// Add rest JSON column
			if len(restData) > 0 {
				mappedEvent[p.restJSONColumn] = restData
			} else {
				mappedEvent[p.restJSONColumn] = nil
			}

			jsonBytes, err := json.Marshal(mappedEvent)
			if err != nil {
				return n, fmt.Errorf("failed to marshal event: %s", err)
			}

			p.currentBytes = append(jsonBytes, '\n')
			p.currentIndex++
		}

		copied := copy(dst, p.currentBytes)
		if copied >= len(p.currentBytes) {
			p.currentBytes = nil
		} else {
			p.currentBytes = p.currentBytes[copied:]
		}

		n += copied
		p.len += copied
		dst = dst[copied:]
	}

	return
}
