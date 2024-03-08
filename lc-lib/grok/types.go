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
	"fmt"
	"strconv"

	"github.com/driskell/log-courier/lc-lib/event"
)

// TypeHint is a type hint specified in a grok pattern
type TypeHint string

const (
	// TypeHintString is the default, the value is output as a string
	TypeHintString TypeHint = "string"
	// TypeHintInt will convert the value to an integer
	TypeHintInt = "int"
	// TypeHintFloat will convert the value to a float
	TypeHintFloat = "float"
)

// parseType takes a string and attempts to convert to a TypeHint
func parseType(typeHint string) (TypeHint, error) {
	switch typeHint {
	case string(TypeHintString):
		return TypeHintString, nil
	case string(TypeHintInt):
		return TypeHintInt, nil
	case string(TypeHintFloat):
		return TypeHintFloat, nil
	}
	return "", fmt.Errorf("invalid type hint: %s", typeHint)
}

// convertToType converts the value to the given type
// Conversions never fail, and return zero value if they cannot parse a valid value
func convertToType(value string, typeHint TypeHint) interface{} {
	switch typeHint {
	case TypeHintString:
		panic("Do not call convertToType with TypeHintString")
	case TypeHintInt:
		// Atoi is equivilant to ParseInt(value, 10, 0), but has a fast path for base-10
		result, _ := strconv.Atoi(value)
		return result
	case TypeHintFloat:
		result, _ := strconv.ParseFloat(value, 64)
		return event.FloatValue64(result)
	}
	return nil
}
