package processor

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

func compileLegacyConfig(p *config.Parser, c *LegacyConfig) error {
	c.ast = make([]ast.ProcessNode, 0, len(c.Pipeline))

	var (
		ifEntry, elseEntry *LegacyConfigASTEntry
		elseIfEntries      []*LegacyConfigASTEntry
		state              = astStatePipeline
		idx                = 0
	)
	constructLogic := func() error {
		ast, err := initLogic(p, ifEntry, elseIfEntries, elseEntry)
		if err != nil {
			return err
		}
		c.ast = append(c.ast, ast)
		return nil
	}
	for idx < len(c.Pipeline) {
		// Slip an index before the / so users know which entry we're examining if error occurs
		entry := c.Pipeline[idx]
		entry.path = fmt.Sprintf("pipelines[%d]/", idx)
		idx++

		// Tokenise
		var entryToken astToken
		if _, ok := entry.Logic[string(astTokenIf)]; ok {
			entryToken = astTokenIf
		} else if _, ok := entry.Logic[string(astTokenElseIf)]; ok {
			entryToken = astTokenElseIf
		} else if _, ok := entry.Logic[string(astTokenElse)]; ok {
			entryToken = astTokenElse
		} else {
			entryToken = astTokenAction
		}

	StateMachine:
		switch state {
		case astStatePipeline:
			if entryToken == astTokenAction {
				ast, err := initAction(entry, p.Config())
				if err != nil {
					return err
				}
				c.ast = append(c.ast, ast)
				break
			}
			if entryToken == astTokenIf {
				ifEntry = entry
				elseIfEntries = nil
				elseEntry = nil
				state = astStateIf
				break
			}
			return fmt.Errorf("Unexpected '%s' at %s", entryToken, entry.path)
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

	return nil
}

// initAction creates and returns a new action entry
func initAction(entry *LegacyConfigASTEntry, cfg *config.Config) (ast.ProcessNode, error) {
	action, ok := entry.Logic["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'name' at %s", entry.path)
	}

	// Convert unused to a map of values to pass to legacy constructor which will
	// validate them and return a resolver node
	values := make(map[string]ast.ValueNode)
	for key, value := range entry.Logic {
		if key == "name" {
			continue
		}
		values[key] = ast.LegacyLiteral(value)
	}

	node, err := ast.LegacyFetchAction(cfg, action, values)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// initLogic creates and returns a new ASTLogic entry
func initLogic(p *config.Parser, ifEntry *LegacyConfigASTEntry, elseIfEntries []*LegacyConfigASTEntry, elseEntry *LegacyConfigASTEntry) (ast.ProcessNode, error) {
	// First create the initial "if" AST entry
	ifAST := &astLogic{}
	if err := p.Populate(ifAST, ifEntry.Logic, ifEntry.path, true); err != nil {
		return nil, err
	}

	// Next, create all the "else if" branches
	if len(elseIfEntries) != 0 {
		ifAST.ElseIfBranches = make([]*logicBranchElseIf, 0, len(elseIfEntries))
		for _, entry := range elseIfEntries {
			elseIfAST := &logicBranchElseIf{}
			if err := p.Populate(elseIfAST, entry.Logic, entry.path, true); err != nil {
				return nil, err
			}
			ifAST.ElseIfBranches = append(ifAST.ElseIfBranches, elseIfAST)
		}
	}

	// Lastly create the ending "else" AST entry list
	if elseEntry != nil {
		ifAST.ElseBranch = &logicBranchElse{}
		if err := p.Populate(ifAST.ElseBranch, elseEntry.Logic, elseEntry.path, true); err != nil {
			return nil, err
		}
	}

	return ifAST, nil
}
