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
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// FormatPattern renders map data into a format string
// Supports %{name} syntax for referencing entries in the event
// Also supports %{+Mon 2 Jan 2006} time formatting
func FormatPattern(pattern string, data map[string]interface{}) (string, error) {
	matcher := regexp.MustCompile(`%\{([^}]+)\}`)
	keyMatcher := regexp.MustCompile(`^([^\[\]]+)|\[([^\[\]]+)\]`)

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
			currentMap := data
			lastIndex := 0
			results := keyMatcher.FindAllStringSubmatchIndex(variable, -1)
			for j := 0; j < len(results); j++ {
				// Must always join together
				if results[j][0] != lastIndex {
					return "", fmt.Errorf("Invalid variable: %s", variable)
				}
				lastIndex = results[j][1]
				nameStart, nameEnd := results[j][2], results[j][3]
				if nameStart < 0 {
					nameStart, nameEnd = results[j][4], results[j][5]
				}
				name := variable[nameStart:nameEnd]
				fmt.Printf("Name: %s -> %s\n", variable, name)
				if j == len(results)-1 {
					// Last item, so will always be a value
					if value, ok := currentMap[name]; ok {
						switch valueTyped := value.(type) {
						case string:
							output += valueTyped
							break
						default:
							valueEncoded, err := json.Marshal(valueTyped)
							if err != nil {
								return "", err
							}
							output += string(valueEncoded)
							break
						}
					}
				} else {
					// Calculate next inner
					if value, ok := currentMap[name]; ok {
						if nextMap := value.(map[string]interface{}); ok {
							currentMap = nextMap
						} else {
							// Can't use non-map, ignore as if it didn't exist, and keep validating
							currentMap = nil
						}
					} else {
						// Doesn't exist so there's no value but keep validating
						currentMap = nil
					}
				}
			}
			if lastIndex != len(variable) {
				return "", fmt.Errorf("Invalid variable: %s", variable)
			}
		}
	}
	return output + pattern[lastOffset:], nil
}
