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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/grok"
)

type grokAction struct {
	Field         string            `config:"field"`
	LocalPatterns map[string]string `config:"local_patterns"`
	Patterns      []string          `config:"patterns"`

	compiled []grok.Pattern
}

func newGrokAction(p *config.Parser, configPath string, unused map[string]interface{}, name string) (Action, error) {
	var err error
	action := &grokAction{}
	if err = p.Populate(action, unused, configPath, true); err != nil {
		return nil, err
	}
	return action, nil
}

func (g *grokAction) Validate(p *config.Parser, configPath string) error {
	grokConfig := FetchGrokConfig(p.Config())
	g.compiled = make([]grok.Pattern, 0, len(g.Patterns))
	for _, pattern := range g.Patterns {
		compiled, err := grokConfig.Grok.CompilePattern(pattern, g.LocalPatterns)
		if err != nil {
			return fmt.Errorf("Failed to compile grok pattern '%s': %s", pattern, err)
		}
		g.compiled = append(g.compiled, compiled)
	}
	return nil
}

func (g *grokAction) Process(event *event.Event) *event.Event {
	entry, err := event.Resolve(g.Field, nil)
	if err != nil {
		event.AddError("grok", fmt.Sprintf("Field '%s' failed to resolve: %s", g.Field, err))
		return event
	}

	var (
		value string
		ok    bool
	)
	if value, ok = entry.(string); !ok {
		event.AddError("grok", fmt.Sprintf("Field '%s' is not present or not a string", g.Field))
		return event
	}

	for _, pattern := range g.compiled {
		err := pattern.Apply(value, func(name string, value interface{}) error {
			_, err := event.Resolve(name, value)
			return err
		})
		if err != nil {
			if err == grok.ErrNoMatch {
				continue
			}
			event.AddError("grok", fmt.Sprintf("Grok failure: %s", err))
		}
		return event
	}

	event.AddError("grok", fmt.Sprintf("Field '%s' was not matched by any of the given patterns", g.Field))
	return event
}
