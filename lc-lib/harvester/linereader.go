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

package harvester

import (
	"bytes"
	"errors"
	"io"
)

var (
	// ErrLineTooLong is returned when the read line was longer than the maximum
	// allowed buffered length
	ErrLineTooLong = errors.New("LineReader: line too long")
)

// LineReader is a read interface that tails and returns lines
type LineReader struct {
	rd       io.Reader
	buf      []byte
	overflow [][]byte
	size     int
	maxLine  int
	curMax   int
	start    int
	end      int
	err      error
}

// NewLineReader creates a new line reader structure reading from the given
// io.Reader with the given size buffer and the given maximum line size
// If a line exceeds the maxmium line size it will be cut into segments, see
// ReadSlice. If the maximum line size is greater than the given buffer size,
// lines that are larger than the buffer will overflow into additional
// memory allocations. Therefore, the buffer size should be sized to handle the
// most common line lengths.
func NewLineReader(rd io.Reader, size int, maxLine int) *LineReader {
	lr := &LineReader{
		rd:      rd,
		buf:     make([]byte, size),
		size:    size,
		maxLine: maxLine,
		curMax:  maxLine,
	}

	return lr
}

// Reset the linereader, still using the same io.Reader, but as if it had just
// being constructed. This will cause any currently buffered data to be lost
func (lr *LineReader) Reset() {
	lr.start = 0
	lr.end = 0
}

// BufferedLen returns the current number of bytes sitting in the buffer
// awaiting a new line
func (lr *LineReader) BufferedLen() int {
	return lr.end - lr.start
}

// ReadSlice returns a line as a byte slice
// Returns ErrLineTooLong if the line was cut short because it was longer than
// the maximum line length allowed. Subsequent returned lines will be a
// continuation of the cut line, with the final segment returning nil error
func (lr *LineReader) ReadSlice() ([]byte, error) {
	var err error
	var line []byte

	if lr.end == 0 {
		err = lr.fill()
	}

	for {
		if n := bytes.IndexByte(lr.buf[lr.start:lr.end], '\n'); n >= 0 && n < lr.curMax {
			line = lr.buf[lr.start : lr.start+n+1]
			lr.start += n + 1
			err = nil
			break
		}

		if err != nil {
			return nil, err
		}

		if lr.end-lr.start >= lr.curMax {
			line = lr.buf[lr.start : lr.start+lr.curMax]
			lr.start += lr.curMax
			err = ErrLineTooLong
			break
		}

		if lr.end-lr.start >= len(lr.buf) {
			lr.start, lr.end = 0, 0
			if lr.overflow == nil {
				lr.overflow = make([][]byte, 0, 1)
			}
			lr.overflow = append(lr.overflow, lr.buf)
			lr.curMax -= len(lr.buf)
			lr.buf = make([]byte, lr.size)
		}

		err = lr.fill()
	}

	if lr.overflow != nil {
		lr.overflow = append(lr.overflow, line)
		line = bytes.Join(lr.overflow, []byte{})
		lr.overflow = nil
		lr.curMax = lr.maxLine
	}
	return line, err
}

// fill reads from the reader and fills the buffer, shifting all unread bytes to
// the front of the buffer to make room
func (lr *LineReader) fill() error {
	if lr.start != 0 {
		copy(lr.buf, lr.buf[lr.start:lr.end])
		lr.end -= lr.start
		lr.start = 0
	}

	n, err := lr.rd.Read(lr.buf[lr.end:])
	lr.end += n

	return err
}
