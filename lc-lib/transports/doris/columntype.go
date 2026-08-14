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

package doris

import (
	"sort"
	"strings"
)

// columnMismatch is a defined column whose existing type does not match its definition
type columnMismatch struct {
	name         string
	existingType string
	expectedType string
}

// mismatchedColumns returns defined columns whose existing type differs from its definition, in name order
func mismatchedColumns(existingCols map[string]string, columnDefs map[string]string) []columnMismatch {
	var mismatches []columnMismatch
	for _, colName := range sortedKeys(columnDefs) {
		existingType, exists := existingCols[colName]
		if !exists {
			continue
		}
		expectedType := columnDefs[colName]
		if canonicalType(existingType) != canonicalType(expectedType) {
			mismatches = append(mismatches, columnMismatch{name: colName, existingType: existingType, expectedType: expectedType})
		}
	}
	return mismatches
}

// missingColumns returns the defined columns the table does not yet have, in name order
func missingColumns(existingCols map[string]string, columnDefs map[string]string) []string {
	var missing []string
	for _, colName := range sortedKeys(columnDefs) {
		if _, exists := existingCols[colName]; !exists {
			missing = append(missing, colName)
		}
	}
	return missing
}

// sortedKeys returns the keys of m in ascending order
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// parsedType is a Doris type split into its base name and top-level argument list
type parsedType struct {
	name string   // upper-cased and aliased base type name
	open byte     // bracket that opened the argument list, zero when there is none
	args []string // top-level arguments, whitespace already compacted
}

// canonicalType renders a Doris type in a stable form so that equivalent spellings compare equal
func canonicalType(t string) string {
	return canonicalNode(compact(t))
}

// canonicalNode renders a type that has already had insignificant whitespace removed
func canonicalNode(t string) string {
	p := parseType(t)

	switch p.name {
	case "VARIANT":
		p.args = withoutProperties(p.args)
	case "TINYINT", "SMALLINT", "INT", "BIGINT", "LARGEINT", "BOOLEAN":
		// Display widths carry no meaning for these types, the name fixes the storage width
		p.args = nil
	case "DATETIME", "TIME":
		// Zero is the default fractional second scale, a non-zero scale is a different type
		if len(p.args) == 1 && p.args[0] == "0" {
			p.args = nil
		}
	}

	if len(p.args) == 0 {
		return p.name
	}

	args := make([]string, len(p.args))
	for i, arg := range p.args {
		args[i] = canonicalArg(arg)
	}
	return p.name + string(p.open) + strings.Join(args, ",") + string(closeOf(p.open))
}

// canonicalArg renders one type argument, which may be a length, a nested type, or a variant path definition
func canonicalArg(arg string) string {
	if arg == "" || !isQuote(arg[0]) {
		return canonicalNode(arg)
	}
	// A variant path is a case sensitive JSON path and is kept exactly as reported
	end := skipQuoted(arg, 0)
	if end+1 >= len(arg) {
		return arg
	}
	rest := arg[end+1:]
	if strings.HasPrefix(rest, ":") {
		return arg[:end+1] + ":" + canonicalNode(rest[1:])
	}
	return arg
}

// parseType splits a compacted type into its base name and top-level arguments
func parseType(t string) parsedType {
	end := 0
	for end < len(t) && t[end] != '(' && t[end] != '<' {
		end++
	}
	p := parsedType{name: canonicalName(strings.ToUpper(t[:end]))}
	if end == len(t) {
		return p
	}
	p.open = t[end]
	p.args = splitArgs(scanBracketed(t, end))
	return p
}

// canonicalName maps interchangeable Doris type names onto a single spelling
func canonicalName(name string) string {
	switch name {
	case "TEXT":
		return "STRING"
	case "INTEGER":
		return "INT"
	case "BOOL":
		return "BOOLEAN"
	}
	return name
}

// withoutProperties drops storage property clauses, which Doris reports even when they are defaults
func withoutProperties(args []string) []string {
	kept := args[:0]
	for _, arg := range args {
		if strings.HasPrefix(strings.ToUpper(arg), "PROPERTIES(") {
			continue
		}
		kept = append(kept, arg)
	}
	return kept
}

// scanBracketed returns the contents between the bracket at start and its match
func scanBracketed(t string, start int) string {
	open, close := t[start], closeOf(t[start])
	depth := 0
	for i := start; i < len(t); i++ {
		switch c := t[i]; {
		case isQuote(c):
			i = skipQuoted(t, i)
		case c == open:
			depth++
		case c == close:
			if depth--; depth == 0 {
				return t[start+1 : i]
			}
		}
	}
	return t[start+1:]
}

// splitArgs splits an argument list on commas that are neither nested nor quoted
func splitArgs(body string) []string {
	var args []string
	depth, start := 0, 0
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case isQuote(c):
			i = skipQuoted(body, i)
		case c == '(' || c == '<':
			depth++
		case c == ')' || c == '>':
			depth--
		case c == ',' && depth == 0:
			args = append(args, body[start:i])
			start = i + 1
		}
	}
	if body[start:] != "" || len(args) != 0 {
		args = append(args, body[start:])
	}
	return args
}

// compact removes whitespace that falls outside of quoted sections
func compact(t string) string {
	var b strings.Builder
	b.Grow(len(t))
	for i := 0; i < len(t); i++ {
		c := t[i]
		switch {
		case isQuote(c):
			end := skipQuoted(t, i)
			b.WriteString(t[i : end+1])
			i = end
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// skipQuoted returns the index of the quote closing the one at start
func skipQuoted(t string, start int) int {
	for i := start + 1; i < len(t); i++ {
		if t[i] == '\\' {
			i++
			continue
		}
		if t[i] == t[start] {
			return i
		}
	}
	return len(t) - 1
}

func isQuote(c byte) bool {
	return c == '\'' || c == '"' || c == '`'
}

func closeOf(open byte) byte {
	if open == '<' {
		return '>'
	}
	return ')'
}
