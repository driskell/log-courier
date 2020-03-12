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
	"context"
	"fmt"
	"testing"
	"time"
)

func createTestEvent() *Event {
	return NewEvent(context.Background(), nil, map[string]interface{}{
		"message": "Hello World",
		"space":   " ",
		"friend":  "Jane",
		"sub": map[string]interface{}{
			"inside": 567,
		},
	})
}

func createDateTestEvent() *Event {
	timestamp, err := time.Parse("2006-01-02", "2020-08-03")
	if err != nil {
		panic("Unexpected error")
	}

	return NewEvent(context.Background(), nil, map[string]interface{}{
		"message":    "Hello World",
		"@timestamp": timestamp,
	})
}

func TestCreateType(t *testing.T) {
	result := NewPatternFromString("I say words")
	if _, ok := result.(staticPattern); !ok {
		t.Errorf("Unexpected pattern type: %T", result)
	}
	result = NewPatternFromString("%\\{testing\\}")
	if _, ok := result.(staticPattern); !ok {
		t.Errorf("Unexpected pattern type: %T", result)
	}
	result = NewPatternFromString("%{testing}")
	if _, ok := result.(variablePattern); !ok {
		t.Errorf("Unexpected pattern type: %T", result)
	}
	result = NewPatternFromString("Even more %{testing}")
	if _, ok := result.(variablePattern); !ok {
		t.Errorf("Unexpected pattern type: %T", result)
	}
	result = NewPatternFromString("Final %{testin still variable")
	if _, ok := result.(variablePattern); !ok {
		t.Errorf("Unexpected pattern type: %T", result)
	}
}

func TestFormatStatic(t *testing.T) {
	result, err := NewPatternFromString("I say words").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say words" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableStart(t *testing.T) {
	result, err := NewPatternFromString("I say to you, %{message}").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, Hello World" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableEnd(t *testing.T) {
	result, err := NewPatternFromString("%{message}, I say to you").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World, I say to you" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMiddle(t *testing.T) {
	result, err := NewPatternFromString("I say to you, \"%{message}\", as loud as I will").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "I say to you, \"Hello World\", as loud as I will" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMultiple(t *testing.T) {
	result, err := NewPatternFromString("%{message}%{space}%{friend}").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Hello World Jane" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableMissing(t *testing.T) {
	result, err := NewPatternFromString("This is %{nothere} not there").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "This is  not there" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatVariableKey(t *testing.T) {
	result, err := NewPatternFromString("We have %{sub[inside]} events").Format(createTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "We have 567 events" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestampMissing(t *testing.T) {
	day := time.Now().Format("2006-01-02")

	// Event defaults to autopopulate a timestamp so forcefully remove it
	event := createDateTestEvent()
	delete(event.Data(), "@timestamp")

	result, err := NewPatternFromString("Value at %{+2006-01-02} should be current day").Format(event)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != fmt.Sprintf("Value at %s should be current day", day) {
		t.Errorf("Unexpected result: [%s]", result)
	}
}

func TestFormatDateTimestamp(t *testing.T) {
	result, err := NewPatternFromString("Value at %{+2006-01-02} should be event day").Format(createDateTestEvent())
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "Value at 2020-08-03 should be event day" {
		t.Errorf("Unexpected result: [%s]", result)
	}
}
