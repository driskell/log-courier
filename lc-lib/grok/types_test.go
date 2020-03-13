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

package grok

import "testing"

func TestParseType(t *testing.T) {
	if typeHint, err := parseType(string(TypeHintString)); err != nil || typeHint != TypeHintString {
		t.Fatal("Failed to parse type")
	}
	if typeHint, err := parseType(string(TypeHintInt)); err != nil || typeHint != TypeHintInt {
		t.Fatal("Failed to parse type")
	}
	if typeHint, err := parseType(string(TypeHintFloat)); err != nil || typeHint != TypeHintFloat {
		t.Fatal("Failed to parse type")
	}
}

func TestParseUnknownType(t *testing.T) {
	if typeHint, err := parseType("unknown"); err == nil {
		t.Fatalf("Unexpected success parsing unknown type: %s", typeHint)
	}
}

func TestStringType(t *testing.T) {
	defer func() {
		recover()
	}()
	convertToType("value", TypeHintString)
	t.Fatal("Unexpected successful conversion to string")
}

func TestIntType(t *testing.T) {
	result := convertToType("value", TypeHintInt)
	if result != 0 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("0123tr", TypeHintInt)
	if result != 0 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("0123", TypeHintInt)
	if result != 123 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("prefix0123", TypeHintInt)
	if result != 0 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("45666666", TypeHintInt)
	if result != 45666666 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
}

func TestFloatType(t *testing.T) {
	result := convertToType("value", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("0123tr", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("0123", TypeHintFloat)
	if result != 123. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("prefix0123", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("45666666", TypeHintFloat)
	if result != 45666666. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("4566.6666", TypeHintFloat)
	if result != 4566.6666 {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("tr1.4", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("1.4oth", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("1.4e4", TypeHintFloat)
	if result != 14000. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
	result = convertToType("1.4e4e", TypeHintFloat)
	if result != 0. {
		t.Fatalf("Unexpected conversion: %s", result)
	}
}
