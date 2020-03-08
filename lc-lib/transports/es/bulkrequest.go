/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

package es

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/driskell/log-courier/lc-lib/event"
)

type bulkRequestCursor struct {
	pos   []*event.Event
	moved bool
}

type bulkRequest struct {
	// Constructor
	events       []*event.Event
	markCursor   *bulkRequestCursor
	readCursor   *bulkRequestCursor
	created      int
	remaining    int
	ackSequence  uint32
	segmentQueue [][]byte
	indexPattern event.Pattern

	// Internal
	defaultIndex string
	currentBytes []byte
}

func newBulkRequest(indexPattern string, events []*event.Event) *bulkRequest {
	eventsClone := append(events[:0:0], events...)

	return &bulkRequest{
		// events will be mutated and nil holes punched for successful events
		// and anything non-nil remains oustanding
		events:       eventsClone,
		markCursor:   &bulkRequestCursor{pos: eventsClone},
		readCursor:   &bulkRequestCursor{pos: eventsClone},
		created:      0,
		remaining:    len(events),
		ackSequence:  0,
		segmentQueue: make([][]byte, 2),
		indexPattern: event.NewPatternFromString(indexPattern),
	}
}

// DefaultIndex returns the index to use in the URL
func (p *bulkRequest) DefaultIndex() (string, error) {
	if p.defaultIndex == "" {
		defaultIndex, err := p.indexPattern.Format(p.markCursor.pos[0])
		if err != nil {
			return "", err
		}
		p.defaultIndex = defaultIndex
	}
	return p.defaultIndex, nil
}

// Created returns the number of events that have been successfully created
func (p *bulkRequest) Created() int {
	return p.created
}

// Remaining returns the number of events left to send in this request
func (p *bulkRequest) Remaining() int {
	return p.remaining
}

// AckSequence returns the sequence marking the end of the contiguous events that we can ack
func (p *bulkRequest) AckSequence() uint32 {
	return p.ackSequence
}

// Mark sets the status of the first outstanding event based on the given successful value
// It returns a cursor which can then be passed in to mark the next item, and repeat
// Pass in nil cursor to start from the beginning
// Returns true when the cursor reaches the end
func (p *bulkRequest) Mark(cursor *bulkRequestCursor, successful bool) (*bulkRequestCursor, bool) {
	var currentCursor *bulkRequestCursor
	if cursor == nil {
		currentCursor = &bulkRequestCursor{pos: p.markCursor.pos}
	} else {
		currentCursor = cursor
	}

	if !successful {
		currentCursor.pos = currentCursor.pos[1:]
		currentCursor.moved = true
		if len(currentCursor.pos) == 0 {
			return nil, true
		}
		return currentCursor, false
	}

	currentCursor.pos[0] = nil
	p.remaining--
	p.created++
	for currentCursor.pos[0] == nil {
		currentCursor.pos = currentCursor.pos[1:]
		if !currentCursor.moved {
			// A cursor is moved when it no longer matches the original
			p.ackSequence++
			p.markCursor.pos = currentCursor.pos
			p.Reset()
		}
		if len(currentCursor.pos) == 0 {
			return nil, true
		}
	}

	return currentCursor, false
}

// Reset allows the request to be Read again
func (p *bulkRequest) Reset() {
	p.readCursor.pos = p.markCursor.pos
}

// Read implements io.Reader and returns a Bulk request body for the attached events
func (p *bulkRequest) Read(dst []byte) (n int, err error) {
	defaultIndex, err := p.DefaultIndex()
	if err != nil {
		return 0, err
	}

	for len(dst) > 0 {
		if len(p.segmentQueue) == 0 {
			if len(p.readCursor.pos) == 0 {
				return n, io.EOF
			}

			index, err := p.indexPattern.Format(p.readCursor.pos[0])
			if err != nil {
				return n, err
			}

			var indexLine []byte
			if index == defaultIndex {
				indexLine = []byte("{\"index\":{}}\n")
			} else {
				jsonIndex, err := json.Marshal(index)
				if err != nil {
					return n, err
				}
				indexLine = []byte(fmt.Sprintf("{\"index\":{\"_index\":%s}}\n", jsonIndex))
			}

			p.segmentQueue = [][]byte{
				indexLine,
				p.readCursor.pos[0].Bytes(),
				[]byte("\n"),
			}

			p.readCursor.pos = p.readCursor.pos[1:]
			for len(p.readCursor.pos) != 0 && p.readCursor.pos[0] == nil {
				p.readCursor.pos = p.readCursor.pos[1:]
			}
		}

		if p.currentBytes == nil {
			p.currentBytes = p.segmentQueue[0]
			p.segmentQueue = p.segmentQueue[1:]
		}

		copied := copy(dst, p.currentBytes)
		if copied >= len(p.currentBytes) {
			p.currentBytes = nil
		} else {
			p.currentBytes = p.currentBytes[copied:]
		}

		n += copied
		dst = dst[copied:]
	}

	return
}
