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

package event

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

var (
	matcher = regexp.MustCompile(`%\{([^}]+)\}`)
)

// Pattern represents a pattern string that can be rendered using an event
type Pattern interface {
	// Format renders map data into a format string
	// Supports %{name} syntax for referencing entries in the event
	// Also supports %{+Mon 2 Jan 2006} time formatting//
	Format(event *Event) (string, error)
}

// A pattern string
type variablePattern string

// A static string with no variables
type staticPattern string

// NewPatternFromString creates a new Pattern
func NewPatternFromString(pattern string) Pattern {
	if strings.Contains(pattern, "%{") {
		return variablePattern(pattern)
	}
	return staticPattern(pattern)
}

// Format implementation for pattern strings
func (p variablePattern) Format(event *Event) (string, error) {
	data := event.Data()
	pattern := string(p)

	results := matcher.FindAllStringSubmatchIndex(pattern, -1)
	if results == nil {
		return pattern, nil
	}

	output := ""
	lastOffset := 0
	for i := 0; i < len(results); i++ {
		output += pattern[lastOffset:results[i][0]]
		lastOffset = results[i][1]
		variable := pattern[results[i][2]:results[i][3]]
		if variable[0] == '+' {
			// Date pattern from event timestamp
			if timestamp, ok := data["@timestamp"]; ok {
				output += timestamp.(time.Time).Format(variable[1:])
			} else {
				// No date field so just use current time
				output += time.Now().Format(variable[1:])
			}
		} else {
			value, err := event.Resolve(variable, nil)
			if err != nil {
				return "", err
			}
			if value != nil {
				switch valueTyped := value.(type) {
				case string:
					output += valueTyped
				default:
					valueEncoded, err := json.Marshal(valueTyped)
					if err != nil {
						return "", err
					}
					output += string(valueEncoded)
				}
			}
		}
	}
	return output + pattern[lastOffset:], nil
}

// String implementation
func (p variablePattern) String() string {
	return string(p)
}

// Format implementation for static strings, just returning itself
func (p staticPattern) Format(event *Event) (string, error) {
	return string(p), nil
}

// String implementation
func (p staticPattern) String() string {
	return string(p)
}
