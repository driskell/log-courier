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

package grok

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPatternApply(t *testing.T) {
	pattern := &compiledPattern{
		pattern: "(?P<text>[a-z]+) with (?P<numbers>[0-9]+) and (?P<floats>[0-9.]+)",
		types:   map[string]TypeHint{},
	}
	err := pattern.init()
	if err != nil {
		t.Fatalf("Failed to init pattern: %s", err)
	}
	results := map[string]interface{}{}
	err = pattern.Apply("something with 8765 and 56.7453", func(name string, value interface{}) error {
		results[name] = value
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to apply pattern: %s", err)
	}
	resultsJSON, _ := json.Marshal(results)
	if string(resultsJSON) != `{"floats":"56.7453","numbers":"8765","text":"something"}` {
		t.Fatalf("Unexpected results: %s", resultsJSON)
	}
}

func TestPatternApplyTypes(t *testing.T) {
	pattern := &compiledPattern{
		pattern: "(?P<text>[a-z]+) with (?P<numbers>[0-9]+) and (?P<floats>[0-9.]+)",
		types:   map[string]TypeHint{"numbers": TypeHintInt, "floats": TypeHintFloat},
	}
	err := pattern.init()
	if err != nil {
		t.Fatalf("Failed to init pattern: %s", err)
	}
	results := map[string]interface{}{}
	err = pattern.Apply("something with 8765 and 56.7453", func(name string, value interface{}) error {
		results[name] = value
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to apply pattern: %s", err)
	}
	resultsJSON, _ := json.Marshal(results)
	if string(resultsJSON) != `{"floats":56.7453,"numbers":8765,"text":"something"}` {
		t.Fatalf("Unexpected results: %s", resultsJSON)
	}
}

func TestPatternApplyNoMatch(t *testing.T) {
	pattern := &compiledPattern{
		pattern: "(?P<text>[a-z]+)",
		types:   map[string]TypeHint{},
	}
	err := pattern.init()
	if err != nil {
		t.Fatalf("Failed to init pattern: %s", err)
	}
	err = pattern.Apply("SOMETHING", func(name string, value interface{}) error {
		return nil
	})
	if err != ErrNoMatch {
		t.Fatal("Unexpected pattern match")
	}
}

func TestPatternApplyFailure(t *testing.T) {
	pattern := &compiledPattern{
		pattern: "(?invalid",
		types:   map[string]TypeHint{},
	}
	err := pattern.init()
	if err == nil {
		t.Fatalf("Unexpected init success: %s", pattern.re.String())
	}
}

func TestPatternApplyCallbackFailure(t *testing.T) {
	pattern := &compiledPattern{
		pattern: "(?P<text>[a-z]+)",
		types:   map[string]TypeHint{},
	}
	err := pattern.init()
	if err != nil {
		t.Fatalf("Failed to init pattern: %s", err)
	}
	callbackErr := errors.New("Error test")
	err = pattern.Apply("something", func(name string, value interface{}) error {
		return callbackErr
	})
	if err != callbackErr {
		t.Fatalf("Unexpected err result: %v", err)
	}
}
