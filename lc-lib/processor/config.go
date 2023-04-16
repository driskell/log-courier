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

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
	"github.com/driskell/log-courier/lc-lib/processor/gen"
)

// astState holds the FSM state when parsing the processor configuration
type astState int

const (
	astStatePipeline astState = iota
	astStateIf

	defaultGeneralProcessorRoutines    = 4
	defaultGeneralProcessorDebugEvents = false
)

// General contains general configuration values
type General struct {
	ProcessorRoutines int  `config:"processor routines"`
	DebugEvents       bool `config:"debug events"`
}

// Validate the additional general configuration
func (gc *General) Validate(p *config.Parser, path string) (err error) {
	if gc.ProcessorRoutines > 128 {
		err = fmt.Errorf("%sprocessor routines can not be greater than 128", path)
		return
	}

	return
}

// Config contains processor pipeline script
type Config struct {
	Source string `config:",embed_string" json:",omitempty"`

	AST []ast.ProcessNode
}

// Init the pipeline configuration
func (c *Config) Init(p *config.Parser, path string) error {
	inputStream := antlr.NewInputStream(c.Source)
	lexer := gen.NewLogCarverLexer(inputStream)
	tokenStream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	errorListenerImpl := &errorListener{}
	parser := gen.NewLogCarverParser(tokenStream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListenerImpl)
	program := parser.Program()
	visitor := ast.NewVisitor(errorListenerImpl)
	c.AST = visitor.Visit(program).([]ast.ProcessNode)
	if len(errorListenerImpl.errors) != 0 {
		return fmt.Errorf("failed to parse processor pipeline script:\n%s", errors.Join(errorListenerImpl.errors...))
	}
	return nil
}

// FetchConfig returns the processor configuration from a Config structure
func FetchConfig(cfg *config.Config) *Config {
	return cfg.Section("pipeline").(*Config)
}

// init registers this module provider
func init() {
	config.RegisterGeneral("processor", func() interface{} {
		return &General{
			ProcessorRoutines: defaultGeneralProcessorRoutines,
			DebugEvents:       defaultGeneralProcessorDebugEvents,
		}
	})

	// Deprecated YAML format language
	config.RegisterSection("pipelines", func() interface{} {
		return &LegacyConfig{}
	})

	// New script format
	config.RegisterSection("pipeline", func() interface{} {
		return &Config{}
	})
}
