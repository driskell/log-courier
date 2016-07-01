/*
 * Copyright 2016 Jason Woods.
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

package simplejson

import (
	"encoding/json"
	"reflect"
	"testing"
)

var (
	input = map[string]interface{}{
		"first":  "value",
		"second": "identical value",
		"third": map[string]interface{}{
			"embedded":     "dictionary",
			"long entry":   "with lots of text, just to make things a bit more juicy for the tests",
			"potentially":  "what we parse could contain many many many items",
			"following":    "are some special types",
			"float":        float64(5.99),
			"anotherfloat": float32(2.34),
			"integer":      int(-1),
		},
		"slicing":  []interface{}{"test", int8(127), int16(-31000), int32(2100000000), int64(-50000000000)},
		"unsigned": []interface{}{uint(6), uint8(250), uint16(60000), uint32(2900000000), uint64(999999999999)},
		"a nil":    nil,
		"fourth":   "and the last entry",
	}
	inputResult = map[string]interface{}{
		"first":  "value",
		"second": "identical value",
		"third": map[string]interface{}{
			"embedded":     "dictionary",
			"long entry":   "with lots of text, just to make things a bit more juicy for the tests",
			"potentially":  "what we parse could contain many many many items",
			"following":    "are some special types",
			"float":        float64(5.99),
			"anotherfloat": float64(2.34),
			"integer":      float64(-1),
		},
		"slicing":  []interface{}{"test", float64(127), float64(-31000), float64(2100000000), float64(-50000000000)},
		"unsigned": []interface{}{float64(6), float64(250), float64(60000), float64(2900000000), float64(999999999999)},
		"a nil":    nil,
		"fourth":   "and the last entry",
	}
	embed = map[string]interface{}{
		"first":    "value",
		"embedded": InlineJSON("{\"this\":\"embeds directly\"}"),
		"again":    NewInlineJSON("\"pointed\""),
	}
	embedResult = map[string]interface{}{
		"first": "value",
		"embedded": map[string]interface{}{
			"this": "embeds directly",
		},
		"again": "pointed",
	}
)

func compareFailure(t *testing.T, before, after interface{}) {
	t.Logf("Received: %v", after)
	t.Logf("Expected: %v", before)
	t.Errorf("Encode result did not match expected")
}

func checkResult(t *testing.T, before map[string]interface{}, afterRaw []byte) {
	var after map[string]interface{}
	if err := json.Unmarshal(afterRaw, &after); err != nil {
		t.Logf("Result: %s", afterRaw)
		t.Errorf("Decoding failed: %s", err)
		return
	}

	if !reflect.DeepEqual(before, after) {
		compareFailure(t, before, after)
	}
}

func TestMarshal(t *testing.T) {
	result, err := Marshal(input)
	if err != nil {
		t.Logf("Encode error: %s", err)
		t.FailNow()
	}

	checkResult(t, inputResult, result)
}

func TestEncodeEmbed(t *testing.T) {
	result, err := Marshal(embed)
	if err != nil {
		t.Errorf("Encode error: %s", err)
	}

	checkResult(t, embedResult, result)
}

func BenchmarkPrepare(b *testing.B) {
	e := &encoder{}
	for i := 0; i < b.N; i++ {
		err := e.prepareBuffer(input)
		if err != nil {
			continue
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	e := &encoder{dst: make([]byte, 1024)}
	for i := 0; i < b.N; i++ {
		e.p = 0
		err := e.encodeValue(input)
		if err != nil {
			continue
		}
	}
}

func BenchmarkMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Marshal(input)
	}
}

func BenchmarkJsonPkg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json.Marshal(input)
	}
}
