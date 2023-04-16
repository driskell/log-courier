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

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/grok"
	"github.com/driskell/log-courier/lc-lib/processor"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

type grokNode struct {
	localPatterns map[string]string

	compiled []grok.Pattern
}

var _ ast.ProcessArgumentsNode = &grokNode{}

func newGrokNode() (ast.ProcessArgumentsNode, error) {
	return &grokNode{}, nil
}

func (n *grokNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentListString("patterns", ast.ArgumentRequired&ast.ArgumentExprDisallowed),
		ast.NewArgumentString("field", ast.ArgumentRequired),
		ast.NewArgumentBool("remove", ast.ArgumentOptional),
		// TODO: local patterns is optional and a map?
	}
}

func (n *grokNode) Init(arguments []any) error {
	patterns := arguments[0].([]string)
	// TODO: load it?
	grokConfig := &processor.GrokConfig{}
	n.compiled = make([]grok.Pattern, 0, len(patterns))
	for _, pattern := range patterns {
		compiled, err := grokConfig.Grok.CompilePattern(pattern, n.localPatterns)
		if err != nil {
			return fmt.Errorf("Failed to initialise grok patterns: %s", err)
		}
		n.compiled = append(n.compiled, compiled)
	}
	return nil
}

func (n *grokNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	field := arguments[1].(string)
	remove := false
	if arguments[2] != nil {
		remove = arguments[2].(bool)
	}

	entry, err := subject.Resolve(field, nil)
	if err != nil {
		subject.AddError("grok", fmt.Sprintf("Field '%s' failed to resolve: %s", field, err))
		return subject
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		subject.AddError("grok", fmt.Sprintf("Field '%s' is not present or not a string", field))
		return subject
	}

	for _, pattern := range n.compiled {
		err := pattern.Apply(value, func(name string, value interface{}) error {
			_, err := subject.Resolve(name, value)
			return err
		})
		if err != nil {
			if err == grok.ErrNoMatch {
				continue
			}
			subject.AddError("grok", fmt.Sprintf("Grok failure: %s", err))
			return subject
		}
		if remove {
			_, err := subject.Resolve(field, event.ResolveParamUnset)
			if err != nil {
				subject.AddError("grok", fmt.Sprintf("Failed to remove field '%s': %s", field, err))
			}
		}
		return subject
	}

	subject.AddError("grok", fmt.Sprintf("Field '%s' was not matched by any of the given patterns", field))
	return subject
}

func init() {
	ast.RegisterAction("grok", newGrokNode)
}
