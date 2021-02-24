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
	"encoding/json"
	"io"
	"testing"
)

func TestJsonRead(t *testing.T) {
	jsonData := []byte("{\"number\":500,\"test\":\"value\"}")
	data := bytes.NewBuffer(jsonData)

	reader := NewJSONReader(data, 1024, 1024)
	value, size, err := reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData), size)
	}
	jsonValue, _ := json.Marshal(value)
	if bytes.Compare(jsonData, jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData, jsonValue)
	}
}

func TestJsonReadMultiple(t *testing.T) {
	jsonData1 := []byte("{\"number\":500,\"test\":\"value\"}")
	jsonData2 := []byte("{\"number\":200,\"test\":\"value2\"}")
	jsonData := bytes.Join([][]byte{jsonData1, jsonData2}, []byte{})
	data := bytes.NewBuffer(jsonData)

	reader := NewJSONReader(data, 1024, 1024)
	value, size, err := reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData1) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData1), size)
	}
	jsonValue, _ := json.Marshal(value)
	if bytes.Compare(jsonData1, jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData1, jsonValue)
	}

	value, size, err = reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData2) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData2), size)
	}
	jsonValue, _ = json.Marshal(value)
	if bytes.Compare(jsonData2, jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData2, jsonValue)
	}
}

func TestJsonReadMultipleWhitespace(t *testing.T) {
	jsonData1 := []byte("{\"number\":500,\"test\":\"value\"}")
	jsonData2 := []byte("\n \t{\"number\":200,\"test\":\"value2\"}")
	jsonData2Trim := bytes.TrimLeft(jsonData2, "\n \t")
	jsonData := bytes.Join([][]byte{jsonData1, jsonData2}, []byte{})
	data := bytes.NewBuffer(jsonData)

	reader := NewJSONReader(data, 1024, 1024)
	value, size, err := reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData1) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData1), size)
	}
	jsonValue, _ := json.Marshal(value)
	if bytes.Compare(jsonData1, jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData1, jsonValue)
	}

	value, size, err = reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData2) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData2), size)
	}
	jsonValue, _ = json.Marshal(value)
	if bytes.Compare(jsonData2Trim, jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData2Trim, jsonValue)
	}
}

func TestJsonReadOverflow(t *testing.T) {
	jsonData1 := []byte("{\"number\":500,\"test\":\"value_too_long\"}")
	jsonData2 := []byte("{\"number\":200,\"test\":\"value\"}")
	jsonData := bytes.Join([][]byte{jsonData1, jsonData2}, []byte{})
	data := bytes.NewBuffer(jsonData)

	reader := NewJSONReader(data, 1024, 30)
	_, _, err := reader.ReadItem()
	if err != ErrMaxDataSizeExceeded {
		t.Fatal("Expected ErrMaxDataSizeExceeded")
	}
}

func TestJsonReadEofRetry(t *testing.T) {
	jsonData1 := []byte("{\"number\":500,")
	jsonData2 := []byte("\"test\":\"value_too_long\"}")
	data := bytes.NewBuffer(jsonData1)

	reader := NewJSONReader(data, 1024, 1024)
	value, size, err := reader.ReadItem()
	if err != io.EOF {
		t.Fatalf("Expected EOF. Actually read %d: %v (%s)", size, value, err)
	}
	data.Write(jsonData2)
	value, size, err = reader.ReadItem()
	if err != nil {
		t.Fatalf("Unexpected read error: %s", err)
	}
	if size != len(jsonData1)+len(jsonData2) {
		t.Fatalf("Unexpected read length (expected %d): %d", len(jsonData2), size)
	}
	jsonValue, _ := json.Marshal(value)
	if bytes.Compare(bytes.Join([][]byte{jsonData1, jsonData2}, []byte{}), jsonValue) != 0 {
		t.Fatalf("Unexpected value (expected %s): %s", jsonData2, jsonValue)
	}
}
