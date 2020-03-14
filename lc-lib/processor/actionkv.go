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
	"fmt"
	"strings"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
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

type kvAction struct {
	Field  string `config:"field"`
	Prefix string `config:"prefix"`

	prefixPattern event.Pattern
}

func newKVAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &kvAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (k *kvAction) Validate(p *config.Parser, configPath string) error {
	k.prefixPattern = event.NewPatternFromString(k.Prefix)
	return nil
}

func (k *kvAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(k.Field, nil)
	if err != nil {
		event.AddError("kv", fmt.Sprintf("Field '%s' could not be resolved: %s", k.Field, err))
		return event
	}

	var (
		stringValue string
		ok          bool
	)
	stringValue, ok = entry.(string)
	if !ok {
		event.AddError("kv", fmt.Sprintf("Field '%s' is not present or not a string", k.Field))
		return event
	}

	prefix, err := k.prefixPattern.Format(event)
	if err != nil {
		event.AddError("kv", fmt.Sprintf("Failed to format prefix from event: %s", k.Prefix))
		return event
	}

	state := kvStateName
	storeValue := func(name string, value string) bool {
		field := prefix + strings.ReplaceAll(strings.ReplaceAll(name, "[", ""), "]", "")
		_, err := event.Resolve(field, value)
		if err != nil {
			event.AddError("kv", fmt.Sprintf("Failed to set field '%s': %s", field, err))
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
				event.AddError("kv", "Parsing interrupted, encountered key with no name")
				return event
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
					return event
				}
				state = kvStateName
			}
		case kvStateValueQuoted:
			if value[idx] == quoteStyle {
				if !storeValue(name, string(value[valueStart:idx])) {
					return event
				}
				state = kvStateValueQuotedEnd
			} else if value[idx] == '\\' {
				state = kvStateValueQuotedEsc
			}
		case kvStateValueQuotedEsc:
			state = kvStateValueQuoted
		case kvStateValueQuotedEnd:
			if value[idx] != ' ' {
				event.AddError("kv", "Parsing interrupted, unexpected text after end of quoted value")
				return event
			}
			state = kvStateName
		}
	}
	switch state {
	case kvStateValueRaw:
		if !storeValue(name, string(value[valueStart:])) {
			return event
		}
	case kvStateValueQuotedEnd:
	case kvStateName:
	default:
		event.AddError("kv", "Parsing interrupted, unexpected end of field")
		return event
	}

	return event
}

// init will register the action
func init() {
	RegisterAction("kv", newKVAction)
}
