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
	"testing"
)

func checkLine(t *testing.T, reader *LineReader, expected []byte, expectedErr error) {
	line, err := reader.ReadSlice()
	if line != nil || expected != nil {
		if line == nil {
			t.Error("No line returned")
		} else if expected == nil {
			t.Errorf("Line data was not expected: [% X]", line)
		} else if !bytes.Equal(line, expected) {
			t.Errorf("Line data incorrect: [% X]", line)
		}
	}
	if err != expectedErr {
		t.Errorf("Unexpected error: %s", err)
	}
}

func checkBufferedLen(t *testing.T, reader *LineReader, expected int) {
	if reader.BufferedLen() != expected {
		t.Errorf("Incorrected buffered length: found %d != expected %d", reader.BufferedLen(), expected)
	}
}

func TestLineRead(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n12345678901234567890\n")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 0)
}

func TestLineReadEmpty(t *testing.T) {
	data := bytes.NewBufferString("\n12345678901234567890\n")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 0)
}

func TestLineReadIncomplete(t *testing.T) {
	data := bytes.NewBufferString("\n12345678901234567890\n123456")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 6)
}

func TestLineReadOverflow(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with 21 bytes buffer but 100 max line to trigger overflow
	reader := NewLineReader(data, 21, 100)

	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, []byte("123456789012345678901234567890\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 0)
}

func TestLineReadOverflowTooLong(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with 10 bytes buffer and 21 max line length
	// This test regression tests a really old bug where when a too long line
	// was received that overflowed in a way specific to these values it would
	// corrupt the entry
	reader := NewLineReader(data, 10, 21)

	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, []byte("123456789012345678901"), ErrLineTooLong)
	checkLine(t, reader, []byte("234567890\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 0)
}

func TestLineReadTooLong(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with ample buffer and 21 max line length
	reader := NewLineReader(data, 100, 21)

	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, []byte("123456789012345678901"), ErrLineTooLong)
	checkLine(t, reader, []byte("234567890\n"), nil)
	checkLine(t, reader, []byte("12345678901234567890\n"), nil)
	checkLine(t, reader, nil, io.EOF)
	checkBufferedLen(t, reader, 0)
}
