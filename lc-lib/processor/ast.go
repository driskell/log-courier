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

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// ASTEntry is an entry in the syntax tree
type ASTEntry interface {
	Process(*event.Event) *event.Event
}

// ASTLogic contains configuration for a logical branch
type ASTLogic struct {
	ifProgram cel.Program
	ifSource  string
	thenAST   []ASTEntry
	elseAST   []ASTEntry
}

// newASTLogicFromConfig creates an ASTEntry for a logic branch from the configuration
func newASTLogicFromConfig(config *ConfigASTLogic) (*ASTLogic, error) {
	ifProgram, err := ParseExpression(config.IfExpr)
	if err != nil {
		return nil, fmt.Errorf("Condition failed to parse: [%s] -> %s", config.IfExpr, err)
	}

	return &ASTLogic{
		ifProgram: ifProgram,
		ifSource:  config.IfExpr,
		thenAST:   config.Then.AST,
		elseAST:   config.Else.AST,
	}, nil
}

// Process handles logic for the event
func (a *ASTLogic) Process(evnt *event.Event) *event.Event {
	val, _, err := a.ifProgram.Eval(map[string]interface{}{"event": evnt.Data()})
	if err != nil {
		log.Warningf("Failed to evaluate if_expr: [%s] -> %s", a.ifSource, err)
		return evnt
	}
	var next []ASTEntry
	if val.ConvertToType(types.BoolType) == types.True {
		next = a.thenAST
	} else {
		next = a.elseAST
	}
	for _, entry := range next {
		evnt = entry.Process(evnt)
	}
	return evnt
}
