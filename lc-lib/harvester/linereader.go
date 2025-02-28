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
	"io"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// LineReader is a read interface that tails and returns lines
type LineReader struct {
	rd             io.Reader
	buf            []byte
	overflow       [][]byte
	size           int
	maxLine        int
	curMax         int
	start          int
	end            int
	err            error
	isContinuation bool
}

// NewLineReader creates a new line reader structure reading from the given
// io.Reader with the given size persistent buffer and the given maximum line
// size. If a line exceeds the maxmium line size it will be cut into segments.
// If the maximum line size is greater than the given buffer size, lines that
// are larger than the buffer will overflow into additional memory allocations
// which are later discarded. Therefore, the buffer size should be sized to
// handle the most common line lengths.
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

// ReadItem returns a line event from the file
// Returns ErrMaxDataSizeTruncation if the line was cut short because it was longer
// than the maximum line length allowed. Subsequent returned lines will be a
// continuation of the cut line, with the final segment returning nil error
func (lr *LineReader) ReadItem() (map[string]interface{}, int, error) {
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
			return nil, 0, err
		}

		if lr.end-lr.start >= lr.curMax {
			line = lr.buf[lr.start : lr.start+lr.curMax]
			lr.start += lr.curMax
			err = ErrMaxDataSizeTruncation
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

	var event map[string]interface{}
	length := len(line)
	if err == ErrMaxDataSizeTruncation {
		event = map[string]interface{}{
			"message": string(line),
		}
		lr.isContinuation = true
	} else {
		// Line will always end in LF, but check also for CR
		var newLine int
		if length > 1 && line[length-2] == '\r' {
			newLine = 2
		} else {
			newLine = 1
		}
		event = map[string]interface{}{
			"message": string(line[:length-newLine]),
		}
		// If this is the continuation from a previously cut line - also return max data exceeded just so it can be tagged accordingly
		if lr.isContinuation {
			lr.isContinuation = false
			err = ErrMaxDataSizeTruncation
		}
	}

	return event, length, err
}

// fill reads from the reader and fills the buffer, shifting all unread bytes to
// the front of the buffer to make room
func (lr *LineReader) fill() error {
	if lr.start != 0 {
		copy(lr.buf, lr.buf[lr.start:lr.end])
		lr.end -= lr.start
		lr.start = 0
	}
	if lr.err != nil {
		// Return existing error - we will have errored with received data and
		// returned a few times without error from that buffer, but now we need
		// to propogate that error
		// Clear the error though - so we can attempt another read if needed
		// (for example if it was EWOULDBLOCK)
		lr.err = nil
		return lr.err
	}

	// Loop until we receive data or an error occurs
	// Avoids the outer loop in ReadItem which would otherwise do unnecessary checks
	var n int
	var err error
	for {
		n, err = lr.rd.Read(lr.buf[lr.end:])
		if err == tcp.ErrIOWouldBlock {
			// Ignore incomplete reads - we will try again
			err = nil
		}
		if n > 0 || err != nil {
			break
		}
	}

	lr.end += n
	if err != nil {
		// Remember last error - so we can continue processing current buffer and once
		// we hit the end of it we return the existing error
		lr.err = err
	}

	return err
}
