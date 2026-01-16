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
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

type addTagNode struct {
}

var _ ast.ProcessArgumentsNode = &addTagNode{}

func newAddTagNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &addTagNode{}, nil
}

func (n *addTagNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("tag", ast.ArgumentRequired),
	}
}

func (n *addTagNode) Init([]any) error {
	return nil
}

func (n *addTagNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	subject.AddTag(arguments[0].(string))
	return subject
}

type removeTagNode struct {
}

var _ ast.ProcessArgumentsNode = &removeTagNode{}

func newRemoveTagNode(*config.Config) (ast.ProcessArgumentsNode, error) {
	return &removeTagNode{}, nil
}

func (n *removeTagNode) Arguments() []ast.Argument {
	return []ast.Argument{
		ast.NewArgumentString("tag", ast.ArgumentRequired),
	}
}

func (n *removeTagNode) Init([]any) error {
	return nil
}

func (n *removeTagNode) ProcessWithArguments(subject *event.Event, arguments []any) *event.Event {
	subject.RemoveTag(arguments[0].(string))
	return subject
}

// init will register the action
func init() {
	ast.RegisterAction("add_tag", newAddTagNode)
	ast.RegisterAction("remove_tag", newRemoveTagNode)
}
