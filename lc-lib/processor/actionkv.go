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
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
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

type kvAction struct {
	Field       string   `config:"field"`
	Prefix      string   `config:"prefix"`
	Separators  string   `config:"separators"`
	Nested      []string `config:"nested"`
	UnquotedHex []string `config:"unquoted hex"`

	prefixPattern event.Pattern
	nested        map[string]struct{}
	unquotedHex   map[string]struct{}
}

func newKVAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (ASTEntry, error) {
	var err error
	action := &kvAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (k *kvAction) Defaults() {
	k.Separators = defaultKVActionSeparators
}

func (k *kvAction) Validate(p *config.Parser, configPath string) error {
	if k.Separators == "" {
		return fmt.Errorf("Separators for kv at %s must not be empty", configPath)
	}
	if strings.ContainsAny(k.Separators, "=\"'\\") {
		return fmt.Errorf("Separators for kv at %s must not contain '=', quotes or backslash", configPath)
	}
	k.prefixPattern = event.NewPatternFromString(k.Prefix)
	k.nested = kvNameSet(k.Nested)
	k.unquotedHex = kvNameSet(k.UnquotedHex)
	return nil
}

func kvNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
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

	k.parse(event, stringValue, prefix)
	return event
}

func (k *kvAction) isSeparator(char rune) bool {
	return strings.ContainsRune(k.Separators, char)
}

// parse stores each pair from input with the given prefix, returning false if parsing was interrupted
func (k *kvAction) parse(evnt *event.Event, input string, prefix string) bool {
	state := kvStateName
	storeValue := func(name string, value string, quoted bool) bool {
		if _, ok := k.nested[name]; ok {
			return k.parse(evnt, value, prefix+event.EscapeKey(name)+"_")
		}
		if _, ok := k.unquotedHex[name]; ok && !quoted {
			value = decodeKVHex(value)
		}
		field := prefix + event.EscapeKey(name)
		_, err := evnt.Resolve(field, value)
		if err != nil {
			evnt.AddError("kv", fmt.Sprintf("Failed to set field '%s': %s", field, err))
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
			if k.isSeparator(value[idx]) {
				continue
			}
			if value[idx] == '=' {
				evnt.AddError("kv", "Parsing interrupted, encountered key with no name")
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
			} else if k.isSeparator(value[idx]) {
				if !storeValue(name, "", false) {
					return false
				}
				state = kvStateName
			} else {
				state = kvStateValueRaw
				valueStart = idx
			}
		case kvStateValueRaw:
			if k.isSeparator(value[idx]) {
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
			if !k.isSeparator(value[idx]) {
				evnt.AddError("kv", "Parsing interrupted, unexpected text after end of quoted value")
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
		evnt.AddError("kv", "Parsing interrupted, unexpected end of field")
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
	RegisterAction("kv", newKVAction)
}
