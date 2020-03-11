/*
 * Copyright 2014-2016 Jason Woods.
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
	"github.com/google/cel-go/cel"
)

// Config is the top level section configuration, and is an array of processor pipelines
type Config []*PipelineConfig

// PipelineConfig contains configuration for a processor pipeline
type PipelineConfig struct {
	ConditionExpr string          `config:"condition_expr"`
	Actions       []*ActionConfig `config:"actions"`

	conditionProgram cel.Program
}

// ActionConfig contains configuration of a single action/directive
type ActionConfig struct {
	Name   string `config:"name"`
	Unused map[string]interface{}

	Handler Action
}

// Init the pipeline configuration
func (c *PipelineConfig) Init(p *config.Parser, path string) (err error) {
	c.conditionProgram, err = ParseExpression(c.ConditionExpr)
	if err != nil {
		return fmt.Errorf("Condition at %s failed to parse: %s", path, err)
	}
	return
}

// Init the action configuration
func (c *ActionConfig) Init(p *config.Parser, path string) (err error) {
	registrarFunc, ok := registeredActions[c.Name]
	if !ok {
		err = fmt.Errorf("Unrecognised action '%s'", c.Name)
		return
	}

	c.Handler, err = registrarFunc(p, path, c.Unused, c.Name)
	return
}

// FetchConfig returns the processor configuration from a Config structure
func FetchConfig(cfg *config.Config) Config {
	return cfg.Section("pipelines").(Config)
}

// init registers this module provider
func init() {
	config.RegisterSection("pipelines", func() interface{} {
		return Config{}
	})

	config.RegisterAvailable("actions", AvailableActions)

	RegisterAction("add_field", newAddFieldAction)
	RegisterAction("add_tag", newAddTagAction)
	RegisterAction("date", newDateAction)
	RegisterAction("grok", newGrokAction)
	RegisterAction("remove_field", newRemoveFieldAction)
	RegisterAction("remove_tag", newRemoveTagAction)
}
