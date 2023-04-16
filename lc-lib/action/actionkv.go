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
	"fmt"
	"strings"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
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
	prefixPattern event.Pattern
}

var _ ast.ProcessArgumentsNode = &kvNode{}

func newKVNode() (ast.ProcessArgumentsNode, error) {
	return &kvNode{}, nil
}

func (n *kvNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("prefix", ast.ArgumentRequired&ast.ArgumentExprDisallowed),
		ast.NewArgumentString("field", ast.ArgumentRequired),
	}
}

func (n *kvNode) Init(arguments []any) error {
	n.prefixPattern = event.NewPatternFromString(arguments[0].(string))
	return nil
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
		subject.AddError("kv", fmt.Sprintf("Failed to format prefix from event: %s", prefix))
		return subject
	}

	state := kvStateName
	storeValue := func(name string, value string) bool {
		field := prefix + strings.ReplaceAll(strings.ReplaceAll(name, "[", ""), "]", "")
		_, err := subject.Resolve(field, value)
		if err != nil {
			subject.AddError("kv", fmt.Sprintf("Failed to set field '%s': %s", field, err))
			return false
		}
		return true
	}

	var (
		value      []rune = []rune(stringValue)
		name       string
		nameStart  int
		valueStart int
		quoteStyle rune
	)
	for idx := 0; idx < len(value); idx++ {
		switch state {
		case kvStateName:
			if value[idx] == '=' {
				subject.AddError("kv", "Parsing interrupted, encountered key with no name")
				return subject
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
			} else {
				state = kvStateValueRaw
				valueStart = idx
			}
		case kvStateValueRaw:
			if value[idx] == ' ' {
				if !storeValue(name, string(value[valueStart:idx])) {
					return subject
				}
				state = kvStateName
			}
		case kvStateValueQuoted:
			if value[idx] == quoteStyle {
				if !storeValue(name, string(value[valueStart:idx])) {
					return subject
				}
				state = kvStateValueQuotedEnd
			} else if value[idx] == '\\' {
				state = kvStateValueQuotedEsc
			}
		case kvStateValueQuotedEsc:
			state = kvStateValueQuoted
		case kvStateValueQuotedEnd:
			if value[idx] != ' ' {
				subject.AddError("kv", "Parsing interrupted, unexpected text after end of quoted value")
				return subject
			}
			state = kvStateName
		}
	}
	switch state {
	case kvStateValueRaw:
		if !storeValue(name, string(value[valueStart:])) {
			return subject
		}
	case kvStateValueQuotedEnd:
	case kvStateName:
	default:
		subject.AddError("kv", "Parsing interrupted, unexpected end of field")
		return subject
	}

	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("kv", newKVNode)
}
