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

package action

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

const (
	defaultKVActionSeparators = " "
)

type kvState int

const (
	kvStateName kvState = iota
	kvStateNameRaw
	kvStateValue
	kvStateValueRaw
	kvStateValueQuoted
	kvStateValueQuotedEsc
	kvStateValueQuotedEnd
)

type kvNode struct {
	prefix        string
	prefixPattern event.Pattern
	separators    string
	nested        map[string]struct{}
	unquotedHex   map[string]struct{}
}

var _ ast.ProcessArgumentsNode = &kvNode{}

func newKVNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &kvNode{}, nil
}

func (n *kvNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("prefix", ast.ArgumentExprDisallowed),
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentString("separators", ast.ArgumentExprDisallowed),
		ast.NewArgumentListString("nested", ast.ArgumentExprDisallowed),
		ast.NewArgumentListString("unquoted_hex", ast.ArgumentExprDisallowed),
	}
}

func (n *kvNode) Init(arguments []any) error {
	n.prefix, _ = arguments[0].(string)
	n.prefixPattern = event.NewPatternFromString(n.prefix)
	n.separators = defaultKVActionSeparators
	if arguments[2] != nil {
		n.separators = arguments[2].(string)
	}
	if n.separators == "" {
		return fmt.Errorf("separators for kv must not be empty")
	}
	if strings.ContainsAny(n.separators, "=\"'\\") {
		return fmt.Errorf("separators for kv must not contain '=', quotes or backslash")
	}
	nested, _ := arguments[3].([]string)
	n.nested = kvNameSet(nested)
	unquotedHex, _ := arguments[4].([]string)
	n.unquotedHex = kvNameSet(unquotedHex)
	return nil
}

func kvNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}

func (n *kvNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	field := arguments[1].(string)

	entry, err := subject.Resolve(field, nil)
	if err != nil {
		subject.AddError("kv", fmt.Sprintf("Field '%s' could not be resolved: %s", field, err))
		return subject
	}

	var (
		stringValue string
		ok          bool
	)
	stringValue, ok = entry.(string)
	if !ok {
		subject.AddError("kv", fmt.Sprintf("Field '%s' is not present or not a string", field))
		return subject
	}

	prefix, err := n.prefixPattern.Format(subject)
	if err != nil {
		subject.AddError("kv", fmt.Sprintf("Failed to format prefix from event: %s", n.prefix))
		return subject
	}

	n.parse(subject, stringValue, prefix)
	return subject
}

func (n *kvNode) isSeparator(char rune) bool {
	return strings.ContainsRune(n.separators, char)
}

// parse stores each pair from input with the given prefix, returning false if parsing was interrupted
func (n *kvNode) parse(subject *event.Event, input string, prefix string) bool {
	state := kvStateName
	storeValue := func(name string, value string, quoted bool) bool {
		if _, ok := n.nested[name]; ok {
			return n.parse(subject, value, prefix+event.EscapeKey(name)+"_")
		}
		if _, ok := n.unquotedHex[name]; ok && !quoted {
			value = decodeKVHex(value)
		}
		field := prefix + event.EscapeKey(name)
		_, err := subject.Resolve(field, value)
		if err != nil {
			subject.AddError("kv", fmt.Sprintf("Failed to set field '%s': %s", field, err))
			return false
		}
		return true
	}

	var (
		value      []rune = []rune(input)
		name       string
		nameStart  int
		valueStart int
		quoteStyle rune
	)
	for idx := 0; idx < len(value); idx++ {
		switch state {
		case kvStateName:
			if n.isSeparator(value[idx]) {
				continue
			}
			if value[idx] == '=' {
				subject.AddError("kv", "Parsing interrupted, encountered key with no name")
				return false
			}
			state = kvStateNameRaw
			nameStart = idx
		case kvStateNameRaw:
			if value[idx] == '=' {
				state = kvStateValue
				name = string(value[nameStart:idx])
			}
		case kvStateValue:
			if value[idx] == '"' || value[idx] == '\'' {
				state = kvStateValueQuoted
				valueStart = idx + 1
				quoteStyle = value[idx]
			} else if n.isSeparator(value[idx]) {
				if !storeValue(name, "", false) {
					return false
				}
				state = kvStateName
			} else {
				state = kvStateValueRaw
				valueStart = idx
			}
		case kvStateValueRaw:
			if n.isSeparator(value[idx]) {
				if !storeValue(name, string(value[valueStart:idx]), false) {
					return false
				}
				state = kvStateName
			}
		case kvStateValueQuoted:
			if value[idx] == quoteStyle {
				if !storeValue(name, string(value[valueStart:idx]), true) {
					return false
				}
				state = kvStateValueQuotedEnd
			} else if value[idx] == '\\' {
				state = kvStateValueQuotedEsc
			}
		case kvStateValueQuotedEsc:
			state = kvStateValueQuoted
		case kvStateValueQuotedEnd:
			if !n.isSeparator(value[idx]) {
				subject.AddError("kv", "Parsing interrupted, unexpected text after end of quoted value")
				return false
			}
			state = kvStateName
		}
	}
	switch state {
	case kvStateValueRaw:
		return storeValue(name, string(value[valueStart:]), false)
	case kvStateValue:
		return storeValue(name, "", false)
	case kvStateValueQuotedEnd:
	case kvStateName:
	default:
		subject.AddError("kv", "Parsing interrupted, unexpected end of field")
		return false
	}

	return true
}

// decodeKVHex decodes an uppercase hex value with NUL separators as spaces, returning it unchanged if it is not valid hex or UTF-8
func decodeKVHex(value string) string {
	if strings.ContainsAny(value, "abcdef") {
		return value
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || !utf8.Valid(decoded) {
		return value
	}
	return strings.ReplaceAll(string(decoded), "\x00", " ")
}

// init will register the action
func init() {
	ast.RegisterAction("kv", newKVNode)
}
