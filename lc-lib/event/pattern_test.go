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

package event

import (
	"fmt"
	"testing"
	"time"
)

func createTestMap() map[string]interface{} {
	return map[string]interface{}{
		"message": "Hello World",
		"space":   " ",
		"friend":  "Jane",
		"sub": map[string]interface{}{
			"inside": 567,
			"deeper": map[string]interface{}{
				"last": true,
			},
		},
	}
}

func createDateTestMap() map[string]interface{} {
	timestamp, err := time.Parse("2006-01-02", "2020-08-03")
	if err != nil {
		panic("Unexpected error")
	}

	return map[string]interface{}{
		"message":    "Hello World",
		"@timestamp": timestamp,
	}
}

func TestFormatVariableStart(t *testing.T) {
	result, err := FormatPattern("I say to you, %{message}", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, Hello World" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableEnd(t *testing.T) {
	result, err := FormatPattern("%{message}, I say to you", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World, I say to you" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMiddle(t *testing.T) {
	result, err := FormatPattern("I say to you, \"%{message}\", as loud as I will", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, \"Hello World\", as loud as I will" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMultiple(t *testing.T) {
	result, err := FormatPattern("%{message}%{space}%{friend}", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World Jane" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMissing(t *testing.T) {
	result, err := FormatPattern("This is %{nothere} not there", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "This is  not there" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKey(t *testing.T) {
	result, err := FormatPattern("We have %{sub[inside]} events", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have 567 events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKeyMultiple(t *testing.T) {
	result, err := FormatPattern("We have %{sub[deeper][last]} events", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have true events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKeyInvalid(t *testing.T) {
	result, err := FormatPattern("This %{sub[} will fail", createTestMap())
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = FormatPattern("This %{su]b} will fail", createTestMap())
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = FormatPattern("This %{sub[inside]more} will fail", createTestMap())
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = FormatPattern("This %{su[]} will fail", createTestMap())
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}
}

func TestFormatVariableKeyMissing(t *testing.T) {
	result, err := FormatPattern("We have %{sub[missing]} not found events", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have  not found events" {
		t.Errorf("Unexpected result: [%s]", result)
	}

	result, err = FormatPattern("We have %{missing[sub]} not got events", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have  not got events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestampMissing(t *testing.T) {
	day := time.Now().Format("2006-01-02")

	result, err := FormatPattern("Value at %{+2006-01-02} should be current day", createTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != fmt.Sprintf("Value at %s should be current day", day) {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestamp(t *testing.T) {
	result, err := FormatPattern("Value at %{+2006-01-02} should be event day", createDateTestMap())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Value at 2020-08-03 should be event day" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}
