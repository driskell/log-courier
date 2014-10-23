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

import (
  "bytes"
  "errors"
  "io"
)

var (
  ErrBufferFull = errors.New("LineReader: buffer full")
)

// A read interface to tail
type LineReader struct {
  rd    io.Reader
  buf   []byte
  start int
  end   int
  err   error
}

func NewLineReader(rd io.Reader, size int64) *LineReader {
  lr := &LineReader{
    rd:  rd,
    buf: make([]byte, size),
  }

  return lr
}

func (lr *LineReader) Reset() {
  lr.start = 0
}

func (lr *LineReader) ReadSlice() ([]byte, error) {
  var err error

  if lr.end == 0 {
    err = lr.fill()
  }

  for {
    if n := bytes.IndexByte(lr.buf[lr.start:lr.end], '\n'); n >= 0 {
      line := lr.buf[lr.start:lr.start+n+1]
      lr.start += n + 1
      return line, nil
    }

    if err != nil {
      return nil, err
    }

    if lr.end - lr.start >= len(lr.buf) {
      break
    }

    err = lr.fill()
  }

  lr.start, lr.end = 0, 0
  return lr.buf, ErrBufferFull
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
