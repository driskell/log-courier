/*
 * Copyright 2022 Jason Woods and contributors
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

package ext

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func checkExpression(t *testing.T, expr string, vars interface{}) ref.Val {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("event", decls.NewMapType(decls.String, decls.Any)),
		),
		JsonEncoder(),
	)
	if err != nil {
		t.Fatalf("Failed to initialise environment: %s", err.Error())
	}

	// Parse using the environment
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		t.Fatalf("Failed to parse: %s", issues.Err().Error())
	}

	// Likely this does nothing at the moment as we don't prepare any declarations
	// But keep it here in case we improve the environment
	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		t.Fatalf("Failed to check: %s", issues.Err().Error())
	}

	program, err := env.Program(checked)
	if err != nil {
		t.Fatalf("Failed to program: %s", err.Error())
	}

	result, _, err := program.Eval(vars)
	if err != nil {
		t.Fatalf("Failed to evaluate: %s", err.Error())
	}

	return result
}

func TestEncode(t *testing.T) {
	result := checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": 10}})
	expected := types.Bytes("10")
	if result.Equal(expected) != types.True {
		t.Errorf("Failed to encode 10: %v != %v", result, expected)
	}

	result = checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": false}})
	expected = types.Bytes("false")
	if result.Equal(expected) != types.True {
		t.Errorf("Failed to encode false: %v != %v", result, expected)
	}

	result = checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": 1.5}})
	expected = types.Bytes("1.5")
	if result.Equal(expected) != types.True {
		t.Errorf("Failed to encode 1.5: %v != %v", result, expected)
	}

	result = checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": "a_string"}})
	expected = types.Bytes("\"a_string\"")
	if result.Equal(expected) != types.True {
		t.Errorf("Failed to encode \"a_string\": %v != %v", result, expected)
	}

	result = checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": map[string]interface{}{"first": "value", "second": 2}}})
	expected = types.Bytes("{\"first\":\"value\",\"second\":2}")
	if result.Equal(expected) != types.True {
		t.Errorf("Failed to encode map: %v != %v", result, expected)
	}

	result = checkExpression(t, "json.encode(event.value)", map[string]interface{}{"event": map[string]interface{}{"value": map[string]interface{}{"first": "value", "second": 2}}})
	expected = types.Bytes("{\"first\":\"not_this_value\",\"second\":2}")
	if result.Equal(expected) != types.False {
		t.Errorf("Failed to fail to encode incorrect value: %v == %v", result, expected)
	}
}

func TestDecode(t *testing.T) {
	result := checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "10"}})
	expectedDouble := types.Double(10)
	if result.Equal(expectedDouble) != types.True {
		t.Errorf("Failed to decode 10: %v != %v", result, expectedDouble)
	}

	result = checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "false"}})
	expectedBool := types.False
	if result.Equal(expectedBool) != types.True {
		t.Errorf("Failed to decode false: %v != %v", result, expectedBool)
	}

	result = checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "1.5"}})
	expectedDouble = types.Double(1.5)
	if result.Equal(expectedDouble) != types.True {
		t.Errorf("Failed to decode false: %v != %v", result, expectedDouble)
	}

	result = checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "\"a_string\""}})
	expectedString := types.String("a_string")
	if result.Equal(expectedString) != types.True {
		t.Errorf("Failed to decode false: %v != %v", result, expectedString)
	}

	result = checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "{\"first\":\"value\",\"second\":2}"}})
	expectedObject := types.DefaultTypeAdapter.NativeToValue(map[string]interface{}{"first": "value", "second": float64(2)})
	if result.Equal(expectedObject) != types.True {
		t.Errorf("Failed to decode map: %v != %v", result, expectedObject)
	}

	result = checkExpression(t, "json.decode(bytes(event.value))", map[string]interface{}{"event": map[string]interface{}{"value": "{\"first\":\"value\",\"second\":2}"}})
	expectedObjectFail := types.DefaultTypeAdapter.NativeToValue(map[string]interface{}{"first": "not_this_value", "second": float64(2)})
	if result.Equal(expectedObjectFail) != types.False {
		t.Errorf("Failed to fail when decoding incorrect value: %v == %v", result, expectedObjectFail)
	}
}
