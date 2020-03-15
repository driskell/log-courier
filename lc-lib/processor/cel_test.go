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

package processor

import (
	"testing"
)

func TestCELFieldAccess(t *testing.T) {
	program, err := ParseExpression("event.test")
	if err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	val, _, err := program.Eval(map[string]interface{}{"event": map[string]interface{}{"test": 123}})
	if err != nil {
		t.Fatalf("Unexpected eval error: %s", err)
	}
	if val.Value().(int64) != 123 {
		t.Fatalf("Unexpected value: %v (error %s)", val, err)
	}
}

func TestCELDeepFieldAccess(t *testing.T) {
	program, err := ParseExpression("event.test.final")
	if err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	val, _, err := program.Eval(map[string]interface{}{"event": map[string]interface{}{"test": map[string]interface{}{"final": "hello"}}})
	if err != nil {
		t.Fatalf("Unexpected eval error: %s", err)
	}
	if val.Value().(string) != "hello" {
		t.Fatalf("Unexpected value: %v (error %s)", val, err)
	}
}

func TestCELMacroHas(t *testing.T) {
	program, err := ParseExpression("has(event.test.final)")
	if err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	val, _, err := program.Eval(map[string]interface{}{"event": map[string]interface{}{"test": map[string]interface{}{"final": "hello"}}})
	if err != nil {
		t.Fatalf("Unexpected eval error: %s", err)
	}
	if !val.Value().(bool) {
		t.Fatalf("Unexpected value: %v (error %s)", val, err)
	}
}

func TestCELMacroHasNot(t *testing.T) {
	program, err := ParseExpression("has(event.not)")
	if err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	val, _, err := program.Eval(map[string]interface{}{"event": map[string]interface{}{"test": 123}})
	if err != nil {
		t.Fatalf("Unexpected eval error: %s", err)
	}
	if val.Value().(bool) {
		t.Fatalf("Unexpected value: %v (error %s)", val, err)
	}
}

func TestCELMacroHasNotDeep(t *testing.T) {
	program, err := ParseExpression("has(event.miss) && has(event.miss.not)")
	if err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	val, _, err := program.Eval(map[string]interface{}{"event": map[string]interface{}{"test": 123}})
	if err != nil {
		t.Fatalf("Unexpected eval error: %s", err)
	}
	if val.Value().(bool) {
		t.Fatalf("Unexpected value: %v (error %s)", val, err)
	}
}
