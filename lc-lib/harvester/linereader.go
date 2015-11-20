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

package harvester

import (
	"bytes"
	"errors"
	"io"
)

var (
	ErrLineTooLong = errors.New("LineReader: line too long")
)

// A read interface to tail
type LineReader struct {
	rd       io.Reader
	buf      []byte
	overflow [][]byte
	size     int
	max_line int
	cur_max  int
	start    int
	end      int
	err      error
}

func NewLineReader(rd io.Reader, size int, max_line int) *LineReader {
	lr := &LineReader{
		rd:       rd,
		buf:      make([]byte, size),
		size:     size,
		max_line: max_line,
		cur_max:  max_line,
	}

	return lr
}

func (lr *LineReader) Reset() {
	lr.start = 0
}

func (lr *LineReader) ReadSlice() ([]byte, error) {
	var err error
	var line []byte

	if lr.end == 0 {
		err = lr.fill()
	}

	for {
		if n := bytes.IndexByte(lr.buf[lr.start:lr.end], '\n'); n >= 0 && n < lr.cur_max {
			line = lr.buf[lr.start : lr.start+n+1]
			lr.start += n + 1
			err = nil
			break
		}

		if err != nil {
			return nil, err
		}

		if lr.end-lr.start >= lr.cur_max {
			line = lr.buf[lr.start : lr.start+lr.cur_max]
			lr.start += lr.cur_max
			err = ErrLineTooLong
			break
		}

		if lr.end-lr.start >= len(lr.buf) {
			lr.start, lr.end = 0, 0
			if lr.overflow == nil {
				lr.overflow = make([][]byte, 0, 1)
			}
			lr.overflow = append(lr.overflow, lr.buf)
			lr.cur_max -= len(lr.buf)
			lr.buf = make([]byte, lr.size)
		}

		err = lr.fill()
	}

	if lr.overflow != nil {
		lr.overflow = append(lr.overflow, line)
		line = bytes.Join(lr.overflow, []byte{})
		lr.overflow = nil
		lr.cur_max = lr.max_line
	}
	return line, err
}

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
