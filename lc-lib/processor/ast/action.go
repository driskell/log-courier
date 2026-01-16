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

package ast

import (
	"fmt"
	"reflect"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/google/cel-go/common/types"
)

type actionFactoryFunc func(*config.Config) (ProcessArgumentsNode, error)

var registeredActions = make(map[string]actionFactoryFunc)

// RegisterAction registers an action
func RegisterAction(name string, factory actionFactoryFunc) {
	registeredActions[name] = factory
}

// AvailableActions returns the list of registered actions available for use
func AvailableActions() (ret []string) {
	ret = make([]string, 0, len(registeredActions))
	for k := range registeredActions {
		ret = append(ret, k)
	}
	return
}

func newActionNode(config *config.Config, name string) (ProcessArgumentsNode, error) {
	factory, ok := registeredActions[name]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	return factory(config)
}

func LegacyFetchAction(config *config.Config, name string, values map[string]ValueNode) (ProcessNode, error) {
	factory, ok := registeredActions[name]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	node, err := factory(config)
	if err != nil {
		return nil, err
	}
	return newArgumentResolverNode(node, values)
}

func LegacyLiteral(value interface{}) ValueNode {
	switch typedValue := value.(type) {
	case string:
		return newLiteralNode(types.String(typedValue))
	case []byte:
		return newLiteralNode(types.String(typedValue))
	case bool:
		return newLiteralNode(types.Bool(typedValue))
	case []string:
		return newLiteralNode(types.NewStringList(types.DefaultTypeAdapter, typedValue))
	case []interface{}:
		if len(typedValue) != 0 {
			// Is it a slice of strings? Use reflection
			kind := reflect.ValueOf(typedValue[0]).Kind()
			if kind == reflect.String {
				strs := make([]string, 0, len(typedValue))
				for _, v := range typedValue {
					strs = append(strs, v.(string))
				}
				return newLiteralNode(types.NewStringList(types.DefaultTypeAdapter, strs))
			}
			return newUnknownNode(fmt.Sprintf("unknown literal slice type: %T of %v", value, kind))
		}
	}
	return newUnknownNode(fmt.Sprintf("unknown literal type: %T", value))
}

func init() {
	config.RegisterAvailable("actions", AvailableActions)
}
