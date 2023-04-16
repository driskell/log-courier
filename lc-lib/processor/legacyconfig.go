package processor

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

// LegacyConfig contains legacy configuration for a processor pipeline
// Users should use `pipeline` instead of `pipelines` which is Config
type LegacyConfig struct {
	Pipeline []*LegacyConfigASTEntry `config:",embed_slice" json:",omitempty"`

	AST []ast.ProcessNode
}

// LegacyConfigASTEntry is a configuration entry we need to parse into an ASTEntry
type LegacyConfigASTEntry struct {
	Unused map[string]interface{}
	Path   string
}

// Init the pipeline configuration
func (c *LegacyConfig) Init(p *config.Parser, path string) error {
	c.AST = make([]ast.ProcessNode, 0, len(c.Pipeline))

	var (
		ifEntry, elseEntry *LegacyConfigASTEntry
		elseIfEntries      []*LegacyConfigASTEntry
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
func (c *LegacyConfig) initAction(p *config.Parser, entry *LegacyConfigASTEntry) (ast.ProcessNode, error) {
	action, ok := entry.Unused["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'name' at %s", entry.Path)
	}

	// Convert unused to a map of values to pass to legacy constructor which will
	// validate them and return a resolver node
	values := make(map[string]ast.ValueNode)
	for key, value := range entry.Unused {
		values[key] = ast.LegacyLiteral(value)
	}

	node, err := ast.LegacyFetchAction(action, values)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// initLogic creates and returns a new ASTLogic entry
func (c *LegacyConfig) initLogic(p *config.Parser, ifEntry *LegacyConfigASTEntry, elseIfEntries []*LegacyConfigASTEntry, elseEntry *LegacyConfigASTEntry) (ast.ProcessNode, error) {
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

// FetchLegacyConfig returns the processor configuration from a Config structure
func FetchLegacyConfig(cfg *config.Config) *LegacyConfig {
	return cfg.Section("pipelines").(*LegacyConfig)
}
