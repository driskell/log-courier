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
	"sort"
	"strings"
)

const controlHexDigits = "0123456789ABCDEF"

// controlLength returns the number of bytes of the control character at idx, or 0 if there is none
//
// Control characters are C0 other than tab and newline, DEL, and C1 encoded as UTF-8
func controlLength(value string, idx int) int {
	char := value[idx]
	switch {
	case char < 0x20:
		if char == '\t' || char == '\n' {
			return 0
		}
		return 1
	case char == 0x7f:
		return 1
	case char == 0xc2 && idx+1 < len(value) && value[idx+1] >= 0x80 && value[idx+1] <= 0x9f:
		return 2
	}
	return 0
}

// EscapeControl returns the value with control characters escaped as \xHH, and whether any were found
//
// The value is returned unchanged without allocation if it has no control characters, otherwise
// backslashes are also escaped as \\ so the escaped value can be reversed
func EscapeControl(value string) (string, bool) {
	idx := 0
	for ; idx < len(value); idx++ {
		if controlLength(value, idx) != 0 {
			break
		}
	}
	if idx == len(value) {
		return value, false
	}

	var builder strings.Builder
	builder.Grow(len(value) + 16)
	for idx = 0; idx < len(value); idx++ {
		if value[idx] == '\\' {
			builder.WriteString(`\\`)
			continue
		}
		length := controlLength(value, idx)
		if length == 0 {
			builder.WriteByte(value[idx])
			continue
		}
		for end := idx + length; idx < end; idx++ {
			builder.WriteString(`\x`)
			builder.WriteByte(controlHexDigits[value[idx]>>4])
			builder.WriteByte(controlHexDigits[value[idx]&0xf])
		}
		idx--
	}
	return builder.String(), true
}

// EscapeControlValue escapes control characters in a string, or recursively in place within a map
// (including its keys), slice or tags, returning the result and whether any were found
//
// Other types, including metadata and timestamps, are returned unchanged
func EscapeControlValue(value interface{}) (interface{}, bool) {
	// Clean values return the original interface so strings are not boxed again
	switch typed := value.(type) {
	case string:
		if escaped, ok := EscapeControl(typed); ok {
			return escaped, true
		}
		return value, false
	case map[string]interface{}:
		return value, escapeControlMap(typed)
	case []interface{}:
		found := false
		for idx, entry := range typed {
			if escaped, ok := EscapeControlValue(entry); ok {
				typed[idx] = escaped
				found = true
			}
		}
		return value, found
	case Tags:
		found := false
		for idx, tag := range typed {
			if escaped, ok := EscapeControl(tag); ok {
				typed[idx] = escaped
				found = true
			}
		}
		if found {
			sort.Strings(typed)
		}
		return value, found
	}
	return value, false
}

func escapeControlMap(entries map[string]interface{}) bool {
	var (
		found   bool
		renamed map[string]interface{}
	)
	for key, entry := range entries {
		if escaped, ok := EscapeControlValue(entry); ok {
			entries[key] = escaped
			found = true
		}
		if escapedKey, ok := EscapeControl(key); ok {
			if renamed == nil {
				renamed = make(map[string]interface{})
			}
			renamed[escapedKey] = entries[key]
			delete(entries, key)
			found = true
		}
	}
	for key, entry := range renamed {
		entries[key] = entry
	}
	return found
}

const controlCharsTag = "_control_chars"

// EscapeControlChars escapes control characters throughout the event, tagging each top-level field that had any
func (e *Event) EscapeControlChars() {
	data := e.Data()
	for key, value := range data {
		escapedKey, keyFound := EscapeControl(key)
		escaped, valueFound := EscapeControlValue(value)
		if !keyFound && !valueFound {
			continue
		}
		if keyFound {
			delete(data, key)
		}
		data[escapedKey] = escaped
		e.TagControlChars(escapedKey)
	}
}

// TagControlChars tags the event as having had control characters escaped within the given field
func (e *Event) TagControlChars(field string) {
	fieldTag, _ := EscapeControl(field)
	e.AddTag(controlCharsTag)
	e.AddTag(controlCharsTag + "_" + fieldTag)
}
