/*
 * Copyright 2014-2016 Jason Woods.
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

import (
	"bytes"
	"encoding/json"
	"io"
)

// JSONReader is a read interface that reads JSON maps
type JSONReader struct {
	rd      *readConstrainer
	dec     *json.Decoder
	maxSize int
	split   bool
}

// NewJSONReader returns a new JSONReader for the specified reader.
// Size does not set the size of the persistent buffer, rather, it sets the size
// at which point a decoder buffer should be discard and a new one created to
// ensure memory is freed after excessively long entries. Maximum size sets the
// maximum allowed JSON object size, if an object exceeds this size it is left
// as raw JSON and split across multiple events
func NewJSONReader(rd io.Reader, size int, maxSize int) *JSONReader {
	ret := &JSONReader{
		rd:      newReadConstrainer(rd, maxSize),
		dec:     json.NewDecoder(rd),
		maxSize: maxSize,
	}

	return ret
}

// refreshDecoder creates a new decoder using the buffer from the previous and
// the reader, so we can clear any errors and get rid of any oversized buffers
func (jr *JSONReader) refreshDecoder() {
	jr.dec = json.NewDecoder(io.MultiReader(jr.dec.Buffered(), jr.rd))
}

// Reset the linereader, still using the same io.Reader, but as if it had just
// being constructed. This will cause any currently buffered data to be lost
func (jr *JSONReader) Reset() {
	jr.dec = json.NewDecoder(jr.rd)
}

// BufferedLen returns the current number of bytes sitting in the buffer
// awaiting a new line
func (jr *JSONReader) BufferedLen() int {
	// Not sure if this cast is future-compatible but hopefully it is
	return jr.dec.Buffered().(*bytes.Reader).Len()
}

// ReadItem returns the next JSON structure from the file
// Returns ErrMaxDataSizeExceed if the data was cut short because it was longer
// than the maximum data length allowed. In this case, the data will be raw in a
// single field called "data". Subsequent returned data will also be raw and a
// continuation of the cut data, with the final segment returning nil error
func (jr *JSONReader) ReadItem() (map[string]interface{}, int, error) {
	if jr.split {
		return jr.readRaw()
	}

	var event map[string]interface{}

	err := jr.dec.Decode(&event)
	if err == io.EOF {
		// We can never use this decoder again as it stores the EOF error
		// Create a new one with the old buffer
		jr.refreshDecoder()
		return nil, 0, err
	}

	if err != nil {
		if err == ErrMaxDataSizeExceeded {
			jr.split = true
			jr.refreshDecoder()
			return jr.readRaw()
		}
		return nil, 0, err
	}

	// Reset max read size
	length := jr.maxSize - jr.rd.setMaxRead(jr.maxSize)

	// If the JSON buffer grew beyond our preferred size, renew it
	if length > jr.maxSize {
		jr.refreshDecoder()
	}

	return event, length, nil
}

func (jr *JSONReader) readRaw() (map[string]interface{}, int, error) {
	// TODO: Finish
	return nil, 0, io.EOF
}
