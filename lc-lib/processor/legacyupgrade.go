package processor

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

func LegacyUpgradeScript(cfg *config.Config, c *LegacyConfig) (string, error) {
	for idx, entry := range c.Pipeline {
		entry.path = fmt.Sprintf("pipelines[%d]/", idx)
	}

	var buf bytes.Buffer
	if err := writeStatements(&buf, cfg, c.Pipeline, "", true); err != nil {
		return "", err
	}

	result := buf.String()
	if _, err := compileScript(cfg, result); err != nil {
		return "", fmt.Errorf("compiled script validation failed: %s", err)
	}
	return result, nil
}

// writeAction writes a single action statement to the buffer
func writeAction(buf *bytes.Buffer, cfg *config.Config, entry *LegacyConfigASTEntry, indent string) error {
	// Get the action name
	action, ok := entry.Logic["name"].(string)
	if !ok {
		return fmt.Errorf("invalid or missing 'name' at %s", entry.path)
	}

	buf.WriteString(action)

	// Write all parameters except 'name'
	values := make(map[string]ast.ValueNode)
	keys := make([]string, 0, len(entry.Logic)-1)
	for key := range entry.Logic {
		if key != "name" {
			keys = append(keys, key)
			values[key] = ast.LegacyLiteral(entry.Logic[key])
		}
	}
	sort.Strings(keys)

	if _, err := ast.LegacyFetchAction(cfg, action, values); err != nil {
		return fmt.Errorf("invalid parameters for action '%s' at %s: %s", action, entry.path, err)
	}

	if len(keys) > 0 {
		buf.WriteString(" ")
		for i, key := range keys {
			if i > 0 {
				buf.WriteString(",\n")
				buf.WriteString(indent)
				buf.WriteString("    ")
			}
			buf.WriteString(key)
			buf.WriteString("=")
			if err := writeValue(buf, entry.Logic[key], indent); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeStatements walks a legacy pipeline using a small FSM to mirror legacycompiler
// behaviour while emitting script output.
func writeStatements(buf *bytes.Buffer, cfg *config.Config, entries []*LegacyConfigASTEntry, indent string, allowSpacing bool) error {
	var (
		ifEntry       *LegacyConfigASTEntry
		elseEntry     *LegacyConfigASTEntry
		elseIfEntries []*LegacyConfigASTEntry
		state         = astStatePipeline
		idx           = 0
	)

	writeLogic := func(nextIdx int) error {
		if err := writeIfStatement(buf, cfg, ifEntry, elseIfEntries, elseEntry, indent); err != nil {
			return err
		}
		if allowSpacing && nextIdx < len(entries) {
			nextToken := tokenForEntry(entries[nextIdx])
			if nextToken != astTokenElseIf && nextToken != astTokenElse {
				buf.WriteString("\n")
			}
		}
		return nil
	}

	for idx < len(entries) {
		entry := entries[idx]
		idx++
		token := tokenForEntry(entry)

	StateMachine:
		switch state {
		case astStatePipeline:
			if token == astTokenAction {
				if err := writeActionLine(buf, cfg, entry, indent); err != nil {
					return err
				}
				break
			}
			if token == astTokenIf {
				ifEntry = entry
				elseIfEntries = nil
				elseEntry = nil
				state = astStateIf
				break
			}
			return fmt.Errorf("Unexpected '%s' at %s", token, entry.path)
		case astStateIf:
			if token == astTokenElseIf {
				elseIfEntries = append(elseIfEntries, entry)
				break
			}
			if token == astTokenElse {
				elseEntry = entry
			}
			if err := writeLogic(idx); err != nil {
				return err
			}
			state = astStatePipeline
			if token != astTokenElse {
				idx--
				entry = entries[idx]
				token = tokenForEntry(entry)
				goto StateMachine
			}
		}
	}

	if state == astStateIf {
		if err := writeLogic(idx); err != nil {
			return err
		}
	}

	return nil
}

// writeActionLine emits an action with trailing separator and indent
func writeActionLine(buf *bytes.Buffer, cfg *config.Config, entry *LegacyConfigASTEntry, indent string) error {
	buf.WriteString(indent)
	if err := writeAction(buf, cfg, entry, indent); err != nil {
		return err
	}
	buf.WriteString(";\n")
	return nil
}

// tokenForEntry identifies the AST token represented by a legacy entry
func tokenForEntry(entry *LegacyConfigASTEntry) astToken {
	if _, ok := entry.Logic[string(astTokenIf)]; ok {
		return astTokenIf
	}
	if _, ok := entry.Logic[string(astTokenElseIf)]; ok {
		return astTokenElseIf
	}
	if _, ok := entry.Logic[string(astTokenElse)]; ok {
		return astTokenElse
	}
	return astTokenAction
}

// writeValue writes a value in script format with proper indentation for multi-line values
func writeValue(buf *bytes.Buffer, value interface{}, indent string) error {
	switch v := value.(type) {
	case string:
		buf.WriteString(strconv.Quote(v))
	case bool:
		buf.WriteString(fmt.Sprintf("%v", v))
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			buf.WriteString(fmt.Sprintf("%d", int64(v)))
		} else {
			buf.WriteString(fmt.Sprintf("%v", v))
		}
	case []interface{}:
		// Check if this is a list of strings that should be formatted multi-line
		if len(v) > 0 {
			isStringList := true
			for _, item := range v {
				if _, ok := item.(string); !ok {
					isStringList = false
					break
				}
			}
			if isStringList {
				buf.WriteString("[\n")
				for i, item := range v {
					if i > 0 {
						buf.WriteString(",\n")
					}
					buf.WriteString(indent)
					buf.WriteString("    ")
					if err := writeValue(buf, item, indent); err != nil {
						return err
					}
				}
				buf.WriteString("\n")
				buf.WriteString(indent)
				buf.WriteString("]")
				return nil
			}
		}
		// Non-string lists use compact format
		buf.WriteString("[")
		for i, item := range v {
			if i > 0 {
				buf.WriteString(", ")
			}
			if err := writeValue(buf, item, indent); err != nil {
				return err
			}
		}
		buf.WriteString("]")
	case []string:
		buf.WriteString("[\n")
		for i, item := range v {
			if i > 0 {
				buf.WriteString(",\n")
			}
			buf.WriteString(indent)
			buf.WriteString("    ")
			buf.WriteString(strconv.Quote(item))
		}
		buf.WriteString("\n")
		buf.WriteString(indent)
		buf.WriteString("]")
	default:
		return fmt.Errorf("unsupported value type %T", value)
	}
	return nil
}

// writeIfStatement writes an if/else chain with the supplied indentation.
func writeIfStatement(buf *bytes.Buffer, cfg *config.Config, ifEntry *LegacyConfigASTEntry, elseIfEntries []*LegacyConfigASTEntry, elseEntry *LegacyConfigASTEntry, indent string) error {
	ifCond, ok := ifEntry.Logic[string(astTokenIf)].(string)
	if !ok {
		return fmt.Errorf("invalid or missing 'if' condition at %s", ifEntry.path)
	}

	buf.WriteString(indent)
	buf.WriteString("if (")
	buf.WriteString(ifCond)
	buf.WriteString(") {\n")

	thenItems, ok := ifEntry.Logic["then"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid or missing 'then' block at %s", ifEntry.path)
	}
	if len(thenItems) > 0 {
		if err := writeBlock(buf, cfg, thenItems, indent+"    "); err != nil {
			return err
		}
	}
	buf.WriteString(indent)
	buf.WriteString("}\n")

	for _, elseIfEntry := range elseIfEntries {
		elseIfCond, ok := elseIfEntry.Logic[string(astTokenElseIf)].(string)
		if !ok {
			return fmt.Errorf("invalid or missing 'else if' condition at %s", elseIfEntry.path)
		}

		buf.WriteString(indent)
		buf.WriteString("else if (")
		buf.WriteString(elseIfCond)
		buf.WriteString(") {\n")

		thenItems, ok := elseIfEntry.Logic["then"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid or missing 'then' block at %s", elseIfEntry.path)
		}
		if len(thenItems) > 0 {
			if err := writeBlock(buf, cfg, thenItems, indent+"    "); err != nil {
				return err
			}
		}
		buf.WriteString(indent)
		buf.WriteString("}\n")
	}

	if elseEntry != nil {
		buf.WriteString(indent)
		buf.WriteString("else {\n")

		elseItems, ok := elseEntry.Logic["else"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid or missing 'else' block at %s", elseEntry.path)
		}
		if len(elseItems) > 0 {
			if err := writeBlock(buf, cfg, elseItems, indent+"    "); err != nil {
				return err
			}
		}
		buf.WriteString(indent)
		buf.WriteString("}\n")
	}

	return nil
}

// writeBlock renders a nested block by reusing the state machine without
// inserting blank lines between statements.
func writeBlock(buf *bytes.Buffer, cfg *config.Config, items []interface{}, indent string) error {
	entries := make([]*LegacyConfigASTEntry, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid block item type %T, expected map[string]interface{}", item)
		}
		entries = append(entries, &LegacyConfigASTEntry{Logic: m})
	}

	return writeStatements(buf, cfg, entries, indent, false)
}
