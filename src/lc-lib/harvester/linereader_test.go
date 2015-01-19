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
	"testing"
)

func checkLine(t *testing.T, reader *LineReader, expected []byte) {
	line, err := reader.ReadSlice()
	if line == nil {
		t.Log("No line returned")
		t.FailNow()
	}
	if !bytes.Equal(line, expected) {
		t.Logf("Line data incorrect: [% X]", line)
		t.FailNow()
	}
	if err != nil {
		t.Logf("Unexpected error: %s", err)
		t.FailNow()
	}
}

func checkLineTooLong(t *testing.T, reader *LineReader, expected []byte) {
	line, err := reader.ReadSlice()
	if line == nil {
		t.Log("No line returned")
		t.FailNow()
	}
	if !bytes.Equal(line, expected) {
		t.Logf("Line data incorrect: [% X]", line)
		t.FailNow()
	}
	if err != ErrLineTooLong {
		t.Logf("Unexpected error: %s", err)
		t.FailNow()
	}
}

func checkEnd(t *testing.T, reader *LineReader) {
	line, err := reader.ReadSlice()
	if line != nil {
		t.Log("Unexpected line returned")
		t.FailNow()
	}
	if err == nil {
		t.Log("Expected error")
		t.FailNow()
	}
}

func TestLineRead(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n12345678901234567890\n")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}

func TestLineReadEmpty(t *testing.T) {
	data := bytes.NewBufferString("\n12345678901234567890\n")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}

func TestLineReadIncomplete(t *testing.T) {
	data := bytes.NewBufferString("\n12345678901234567890\n123456")

	// New line read with 100 bytes, enough for the above
	reader := NewLineReader(data, 100, 100)

	checkLine(t, reader, []byte("\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}

func TestLineReadOverflow(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with 21 bytes buffer but 100 max line to trigger overflow
	reader := NewLineReader(data, 21, 100)

	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkLine(t, reader, []byte("123456789012345678901234567890\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}

func TestLineReadOverflowTooLong(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with 10 bytes buffer and 21 max line length
	reader := NewLineReader(data, 10, 21)

	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkLineTooLong(t, reader, []byte("123456789012345678901"))
	checkLine(t, reader, []byte("234567890\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}

func TestLineReadTooLong(t *testing.T) {
	data := bytes.NewBufferString("12345678901234567890\n123456789012345678901234567890\n12345678901234567890\n")

	// New line read with ample buffer and 21 max line length
	reader := NewLineReader(data, 100, 21)

	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkLineTooLong(t, reader, []byte("123456789012345678901"))
	checkLine(t, reader, []byte("234567890\n"))
	checkLine(t, reader, []byte("12345678901234567890\n"))
	checkEnd(t, reader)
}
