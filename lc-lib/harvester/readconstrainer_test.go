/*
* Copyright 2012-2021 Jason Woods
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

func TestFullRead(t *testing.T) {
	bufString := "testing\ntesting\n"
	buf := bytes.NewBufferString(bufString)
	reader := newReadConstrainer(buf, 100)
	value := make([]byte, 100)
	size, err := reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != len(bufString) {
		t.Fatalf("Unexpected read size (expected %d): %d", len(bufString), size)
	}

	size, err = reader.Read(value)
	if err != io.EOF {
		t.Fatal("Expected EOF")
	}
}

func TestUnderRead(t *testing.T) {
	bufString := "testing\ntesting\n"
	buf := bytes.NewBufferString(bufString)
	reader := newReadConstrainer(buf, 100)
	value := make([]byte, 10)
	size, err := reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != 10 {
		t.Fatalf("Unexpected read size (expected %d): %d", len(bufString), size)
	}
	expected := []byte("testing\nte")
	if bytes.Compare(value, expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
}

func TestConstrainedRead(t *testing.T) {
	bufString := "testing\ntesting\ntesting\n"
	buf := bytes.NewBufferString(bufString)
	reader := newReadConstrainer(buf, 10)
	value := make([]byte, 100)
	size, err := reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != 10 {
		t.Fatalf("Unexpected read size (expected %d): %d", 10, size)
	}
	expected := []byte("testing\nte")
	if bytes.Compare(value[:size], expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
	// Cannot read anymore as max reached
	size, err = reader.Read(value)
	if size != 0 {
		t.Fatalf("Unexpected read size (expected %d): %d", 0, size)
	}
	if err != ErrMaxDataSizeExceeded {
		t.Fatal("Expected ErrMaxDataSizeExceeded")
	}
	// Once reset, should allow more
	size = reader.setMaxRead(10)
	if size != 0 {
		t.Fatalf("Unexpected returned read size (expected %d): %d", 0, size)
	}
	size, err = reader.Read(value)
	if size != 10 {
		t.Fatalf("Unexpected read size (expected %d): %d", 10, size)
	}
	expected = []byte("sting\ntest")
	if bytes.Compare(value[:size], expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
}

func TestPartialMaxRead(t *testing.T) {
	bufString := "testing\ntesting\ntesting\n"
	buf := bytes.NewBufferString(bufString)
	reader := newReadConstrainer(buf, 10)
	value := make([]byte, 5)
	size, err := reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != 5 {
		t.Fatalf("Unexpected read size (expected %d): %d", 5, size)
	}
	expected := []byte("testi")
	if bytes.Compare(value[:size], expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
	size = reader.setMaxRead(10)
	if size != 5 {
		t.Fatalf("Unexpected returned read size (expected %d): %d", 5, size)
	}
}

func TestChangeMaxRead(t *testing.T) {
	bufString := "testing\ntesting\ntesting\n"
	buf := bytes.NewBufferString(bufString)
	reader := newReadConstrainer(buf, 5)
	value := make([]byte, 100)
	size, err := reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != 5 {
		t.Fatalf("Unexpected read size (expected %d): %d", 5, size)
	}
	expected := []byte("testi")
	if bytes.Compare(value[:size], expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
	size = reader.setMaxRead(10)
	if size != 0 {
		t.Fatalf("Unexpected returned read size (expected %d): %d", 0, size)
	}
	size, err = reader.Read(value)
	if err != nil {
		t.Fatal("Expected EOF")
	}
	if size != 10 {
		t.Fatalf("Unexpected read size (expected %d): %d", 10, size)
	}
	expected = []byte("ng\ntesting")
	if bytes.Compare(value[:size], expected) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", expected, value)
	}
}
