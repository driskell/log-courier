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
	"fmt"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

type grokAction struct {
	Field    string   `config:"field"`
	Patterns []string `config:"patterns"`

	compiled      []*regexp.Regexp
	compiledNames [][]string
}

func newGrokAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &grokAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	action.compiled = make([]*regexp.Regexp, 0, len(action.Patterns))
	action.compiledNames = make([][]string, 0, len(action.Patterns))
	for _, pattern := range action.Patterns {
		compiled, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			return nil, err
		}
		action.compiled = append(action.compiled, compiled)
		action.compiledNames = append(action.compiledNames, compiled.SubexpNames())
	}
	return action, nil
}

func (g *grokAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(g.Field, nil)
	if value, ok := entry.(string); err != nil && ok {
		for reidx, re := range g.compiled {
			result := re.FindStringSubmatch(value)
			if result == nil {
				continue
			}
			for nameidx, name := range g.compiledNames[reidx][1:] {
				event.Resolve(name, result[nameidx+1])
			}
			return event
		}
	}
	event.AddTag("_grok_parse_failure")
	return event
}
