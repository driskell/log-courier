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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
	"github.com/driskell/log-courier/lc-lib/processor/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// astToken holds the type of a token during parsing
type astToken string

const (
	astTokenAction astToken = "action"
	astTokenIf     astToken = "if"
	astTokenElseIf astToken = "else if"
	astTokenElse   astToken = "else"
)

// ASTEntry is an entry in the syntax tree that processes events
type ASTEntry interface {
	Process(*event.Event) *event.Event
}

// astLogic processes an event through a conditional branch
type astLogic struct {
	IfExpr         string        `config:"if"` // should match astTokenIf
	Then           *LegacyConfig `config:"then"`
	ElseIfBranches []*logicBranchElseIf
	ElseBranch     *logicBranchElse

	ifProgram cel.Program
}

// Init the branch
func (l *astLogic) Init(p *config.Parser, path string) (err error) {
	if l.ifProgram, err = expr.ParseExpression(l.IfExpr); err != nil {
		return fmt.Errorf("Condition failed to parse at %s: [%s] -> %s", path, l.IfExpr, err)
	}
	return nil
}

// Process handles logic for the event
func (l *astLogic) Process(subject *event.Event) *event.Event {
	var next []ast.ProcessNode
	if evalLogicBranchProgram(l.ifProgram, l.IfExpr, subject) {
		next = l.Then.AST
	} else {
		if len(l.ElseIfBranches) != 0 {
			for _, elseIfBranch := range l.ElseIfBranches {
				if evalLogicBranchProgram(elseIfBranch.elseIfProgram, elseIfBranch.ElseIfExpr, subject) {
					next = elseIfBranch.Then.AST
					break
				}
			}
		}
		if next == nil {
			if l.ElseBranch != nil {
				next = l.ElseBranch.Else.AST
			} else {
				return subject
			}
		}
	}
	for _, entry := range next {
		subject = entry.Process(subject)
	}
	return subject
}

// logicBranchElseIf branch
type logicBranchElseIf struct {
	ElseIfExpr string        `config:"else if"` // should match astTokenElseIf
	Then       *LegacyConfig `config:"then"`

	elseIfProgram cel.Program
}

// Init the branch
func (l *logicBranchElseIf) Init(p *config.Parser, path string) (err error) {
	if l.elseIfProgram, err = expr.ParseExpression(l.ElseIfExpr); err != nil {
		return fmt.Errorf("Condition failed to parse at %s: [%s] -> %s", path, l.ElseIfExpr, err)
	}
	return nil
}

// logicBranchElse branch
type logicBranchElse struct {
	Else *LegacyConfig `config:"else"`
}

// evalLogicBranchProgram runs the condition program and returns true or false
func evalLogicBranchProgram(program cel.Program, source string, subject *event.Event) bool {
	val, _, err := program.Eval(map[string]interface{}{"event": subject.Data()})
	if err != nil {
		log.Warningf("Failed to evaluate if: [%s] -> %s", source, err)
		return false
	}
	return val.ConvertToType(types.BoolType) == types.True
}
