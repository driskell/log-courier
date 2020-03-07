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

func createTestEvent() *Event {
	data := map[string]interface{}{
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

	return NewEvent(nil, data, nil)
}

func createDateTestEvent() *Event {
	timestamp, err := time.Parse("2006-01-02", "2020-08-03")
	if err != nil {
		panic("Unexpected error")
	}

	data := map[string]interface{}{
		"message":    "Hello World",
		"@timestamp": timestamp,
	}

	return NewEvent(nil, data, nil)
}

func TestFormatVariableStart(t *testing.T) {
	result, err := createTestEvent().Format("I say to you, %{message}")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, Hello World" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableEnd(t *testing.T) {
	result, err := createTestEvent().Format("%{message}, I say to you")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World, I say to you" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMiddle(t *testing.T) {
	result, err := createTestEvent().Format("I say to you, \"%{message}\", as loud as I will")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, \"Hello World\", as loud as I will" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMultiple(t *testing.T) {
	result, err := createTestEvent().Format("%{message}%{space}%{friend}")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World Jane" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMissing(t *testing.T) {
	result, err := createTestEvent().Format("This is %{nothere} not there")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "This is  not there" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKey(t *testing.T) {
	result, err := createTestEvent().Format("We have %{sub[inside]} events")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have 567 events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKeyMultiple(t *testing.T) {
	result, err := createTestEvent().Format("We have %{sub[deeper][last]} events")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have true events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKeyInvalid(t *testing.T) {
	result, err := createTestEvent().Format("This %{sub[} will fail")
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = createTestEvent().Format("This %{su]b} will fail")
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = createTestEvent().Format("This %{sub[inside]more} will fail")
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}

	result, err = createTestEvent().Format("This %{su[]} will fail")
	if err == nil {
		t.Errorf("Unexpected successful result: %s", result)
	}
}

func TestFormatVariableKeyMissing(t *testing.T) {
	result, err := createTestEvent().Format("We have %{sub[missing]} not found events")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have  not found events" {
		t.Errorf("Unexpected result: [%s]", result)
	}

	result, err = createTestEvent().Format("We have %{missing[sub]} not got events")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have  not got events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestampMissing(t *testing.T) {
	day := time.Now().Format("2006-01-02")

	result, err := createTestEvent().Format("Value at %{+2006-01-02} should be current day")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != fmt.Sprintf("Value at %s should be current day", day) {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestamp(t *testing.T) {
	result, err := createDateTestEvent().Format("Value at %{+2006-01-02} should be event day")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Value at 2020-08-03 should be event day" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}
