/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/cel"
)

type addTagAction struct {
	Tag       string `config:"tag"`
	ValueExpr string `config:"value_expr"`

	valueProgram cel.Program
}

func newAddTagAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &addTagAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.valueProgram, err = ParseExpression(action.ValueExpr)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (f *addTagAction) Process(event *event.Event) *event.Event {
	event.AddTag(f.Tag)
	return event
}

type removeTagAction struct {
	Tag string `config:"tag"`
}

func newRemoveTagAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &removeTagAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (f *removeTagAction) Process(event *event.Event) *event.Event {
	event.RemoveTag(f.Tag)
	return event
}
