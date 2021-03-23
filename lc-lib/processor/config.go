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

// Config contains configuration for a processor pipeline
type Config struct {
	Pipeline []*ConfigASTEntry `config:",embed_slice" json:",omitempty"`

	AST []ASTEntry
}

// ConfigASTEntry is a configuration entry we need to parse into an ASTEntry
type ConfigASTEntry struct {
	Unused map[string]interface{}
	Path   string
}

// Validate the additional general configuration
func (gc *General) Validate(p *config.Parser, path string) (err error) {
	if gc.ProcessorRoutines > 128 {
		err = fmt.Errorf("%sprocessor routines can not be greater than 128", path)
		return
	}

	return
}

// Init the pipeline configuration
func (c *Config) Init(p *config.Parser, path string) error {
	c.AST = make([]ASTEntry, 0, len(c.Pipeline))

	var (
		ifEntry, elseEntry *ConfigASTEntry
		elseIfEntries      []*ConfigASTEntry
		state              = astStatePipeline
		idx                = 0
	)
	constructLogic := func() error {
		ast, err := c.initLogic(p, ifEntry, elseIfEntries, elseEntry)
		if err != nil {
			return err
		}
		c.AST = append(c.AST, ast)
		return nil
	}
	for idx < len(c.Pipeline) {
		// Slip an index before the / so users know which entry we're examining if error occurs
		entry := c.Pipeline[idx]
		entry.Path = fmt.Sprintf("%s[%d]/", path[:len(path)-1], idx)
		idx++

		// Tokenise
		var entryToken astToken
		if _, ok := entry.Unused[string(astTokenIf)]; ok {
			entryToken = astTokenIf
		} else if _, ok := entry.Unused[string(astTokenElseIf)]; ok {
			entryToken = astTokenElseIf
		} else if _, ok := entry.Unused[string(astTokenElse)]; ok {
			entryToken = astTokenElse
		} else {
			entryToken = astTokenAction
		}

	StateMachine:
		switch state {
		case astStatePipeline:
			if entryToken == astTokenAction {
				ast, err := c.initAction(p, entry)
				if err != nil {
					return err
				}
				c.AST = append(c.AST, ast)
				break
			}
			if entryToken == astTokenIf {
				ifEntry = entry
				elseIfEntries = nil
				elseEntry = nil
				state = astStateIf
				break
			}
			return fmt.Errorf("Unexpected '%s' at %s", entryToken, entry.Path)
		case astStateIf:
			if entryToken == astTokenElseIf {
				elseIfEntries = append(elseIfEntries, entry)
				break
			}
			if entryToken == astTokenElse {
				elseEntry = entry
			}
			if err := constructLogic(); err != nil {
				return err
			}
			state = astStatePipeline
			if entryToken != astTokenElse {
				// We didn't use the token, process it now
				goto StateMachine
			}
		}
	}
	if state == astStateIf {
		if err := constructLogic(); err != nil {
			return err
		}
	}

	// Cannot expose this to final configuration output as it won't be renderable
	// due to YAML decoding actually introducing map[interface{}]interface{}
	// Let the AST member output instead
	c.Pipeline = nil

	return nil
}

// initAction creates and returns a new action entry
func (c *Config) initAction(p *config.Parser, entry *ConfigASTEntry) (ASTEntry, error) {
	action, ok := entry.Unused["name"].(string)
	if !ok {
		return nil, fmt.Errorf("Invalid or missing 'name' at %s", entry.Path)
	}

	registrarFunc, ok := registeredActions[action]
	if !ok {
		return nil, fmt.Errorf("Unrecognised action '%s' at %s", action, entry.Path)
	}

	// Action registrars will not consume "name" so remove it before passing
	delete(entry.Unused, "name")
	ast, err := registrarFunc(p, entry.Path, entry.Unused, action)
	if err != nil {
		return nil, err
	}

	return ast, nil
}

// initLogic creates and returns a new ASTLogic entry
func (c *Config) initLogic(p *config.Parser, ifEntry *ConfigASTEntry, elseIfEntries []*ConfigASTEntry, elseEntry *ConfigASTEntry) (ASTEntry, error) {
	// First create the initial "if" AST entry
	ifAST := &astLogic{}
	if err := p.Populate(ifAST, ifEntry.Unused, ifEntry.Path, true); err != nil {
		return nil, err
	}

	// Next, create all the "else if" branches
	if len(elseIfEntries) != 0 {
		ifAST.ElseIfBranches = make([]*logicBranchElseIf, 0, len(elseIfEntries))
		for _, entry := range elseIfEntries {
			elseIfAST := &logicBranchElseIf{}
			if err := p.Populate(elseIfAST, entry.Unused, entry.Path, true); err != nil {
				return nil, err
			}
			ifAST.ElseIfBranches = append(ifAST.ElseIfBranches, elseIfAST)
		}
	}

	// Lastly create the ending "else" AST entry list
	if elseEntry != nil {
		ifAST.ElseBranch = &logicBranchElse{}
		if err := p.Populate(ifAST.ElseBranch, elseEntry.Unused, elseEntry.Path, true); err != nil {
			return nil, err
		}
	}

	return ifAST, nil
}

// FetchConfig returns the processor configuration from a Config structure
func FetchConfig(cfg *config.Config) *Config {
	return cfg.Section("pipelines").(*Config)
}

// init registers this module provider
func init() {
	config.RegisterGeneral("processor", func() interface{} {
		return &General{
			ProcessorRoutines: defaultGeneralProcessorRoutines,
			DebugEvents:       defaultGeneralProcessorDebugEvents,
		}
	})

	config.RegisterSection("pipelines", func() interface{} {
		return &Config{}
	})

	config.RegisterAvailable("actions", AvailableActions)
}
