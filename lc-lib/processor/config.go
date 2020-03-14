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
	"errors"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Config contains configuration for a processor pipeline
type Config struct {
	Pipeline []*ConfigASTEntry `config:",embed_slice"`

	AST []ASTEntry
}

// ConfigASTEntry is a configuration entry we need to parse into an ASTEntry
type ConfigASTEntry struct {
	Unused map[string]interface{}
}

// ConfigASTLogic processes configuration for a logical branch
type ConfigASTLogic struct {
	IfExpr string  `config:"if_expr"`
	Then   *Config `config:"then"`
	Else   *Config `config:"else"`
}

// Init the pipeline configuration
func (c *Config) Init(p *config.Parser, path string) (err error) {
	c.AST = make([]ASTEntry, 0, len(c.Pipeline))

	for idx, entry := range c.Pipeline {
		// Slip an index before the / so users know which entry we're examining if error occurs
		entryPath := fmt.Sprintf("%s[%d]/", path[:len(path)-1], idx)

		var ast ASTEntry
		if _, ok := entry.Unused["name"]; ok {
			if action, ok := entry.Unused["name"].(string); ok {
				ast, err = c.initAction(p, entryPath, entry, action)
			} else {
				err = errors.New("Action 'name' must be a string")
				return
			}
		} else {
			ast, err = c.initLogic(p, entryPath, entry)
		}
		if err != nil {
			return
		}
		c.AST = append(c.AST, ast)
	}
	return
}

// initAction creates and returns a new action entry
func (c *Config) initAction(p *config.Parser, path string, entry *ConfigASTEntry, action string) (ASTEntry, error) {
	registrarFunc, ok := registeredActions[action]
	if !ok {
		return nil, fmt.Errorf("Unrecognised action '%s'", action)
	}

	// Action registrars will not consume "name" so remove it before passing
	delete(entry.Unused, "name")
	ast, err := registrarFunc(p, path, entry.Unused, action)
	if err != nil {
		return nil, err
	}

	return ast, nil
}

// initLogic creates and returns a new ASTLogic entry
func (c *Config) initLogic(p *config.Parser, path string, entry *ConfigASTEntry) (ASTEntry, error) {
	ast := &ConfigASTLogic{}
	if err := p.Populate(ast, entry.Unused, path, true); err != nil {
		return nil, err
	}
	return newASTLogicFromConfig(ast)
}

// FetchConfig returns the processor configuration from a Config structure
func FetchConfig(cfg *config.Config) *Config {
	return cfg.Section("pipelines").(*Config)
}

// init registers this module provider
func init() {
	config.RegisterSection("pipelines", func() interface{} {
		return &Config{}
	})

	config.RegisterAvailable("actions", AvailableActions)
}
