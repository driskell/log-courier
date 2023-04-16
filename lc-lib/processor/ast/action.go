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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/google/cel-go/common/types"
)

type actionFactoryFunc func() (ProcessArgumentsNode, error)

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

func newActionNode(name string) (ProcessArgumentsNode, error) {
	factory, ok := registeredActions[name]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	return factory()
}

func LegacyFetchAction(name string, values map[string]ValueNode) (ProcessNode, error) {
	factory, ok := registeredActions[name]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	node, err := factory()
	if err != nil {
		return nil, err
	}
	return newArgumentResolverNode(node, values)
}

func LegacyLiteral(value interface{}) ValueNode {
	switch typedValue := value.(type) {
	case string:
		return &literalNode{value: types.String(typedValue)}
	case []byte:
		return &literalNode{value: types.String(typedValue)}
	case bool:
		return &literalNode{value: types.Bool(typedValue)}
	case []string:
		return &literalNode{value: types.NewStringList(types.DefaultTypeAdapter, typedValue)}
	}
	return &literalNode{value: types.Unknown([]int64{})}
}

func init() {
	config.RegisterAvailable("actions", AvailableActions)
}
