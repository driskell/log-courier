/*
 * Copyright 2014-2015 Jason Woods.
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

package grok

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	matcher = regexp.MustCompile(`%\{([^}]+)\}`)
)

// Grok parses messages into fields using regular expressions
type Grok struct {
	compiled map[string]*compiledPattern
	pending  map[string][]*compilationState
}

// ErrorIncompletePattern is returned when a pattern cannot be used as it
// references other patterns that are not currently available
// The name of the missing pattern is set inside the Missing field
type ErrorIncompletePattern struct {
	Missing string
}

// Error returns an error message for the missing pattern
func (e *ErrorIncompletePattern) Error() string {
	return fmt.Sprintf("Referenced pattern was not found: %s", e.Missing)
}

// compilationState tracks the compilation progress of a pattern
// It allows compilation to pause when a missing pattern is encountered
// and for it to then resume when that pattern is loaded
type compilationState struct {
	name    string
	pattern string

	output string
	types  map[string]string

	results    [][]int
	index      int
	lastOffset int
}

// NewGrok returns a new Grok instance
func NewGrok() *Grok {
	return &Grok{
		compiled: make(map[string]*compiledPattern),
		pending:  make(map[string][]*compilationState),
	}
}

// newCompilationState creates a blank compilation state for the given named pattern
func newCompilationState(name string, pattern string) *compilationState {
	return &compilationState{
		name:    name,
		pattern: pattern,
		types:   make(map[string]string),
	}
}

// LoadPatternsFromFile loads patterns from the requested file
// Each line of the file should be in the format: "NAME PATTERN"
// For optimisation, PATTERNS that include other named patterns should appear
// after the named patterns they require
func (g *Grok) LoadPatternsFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		file.Close()
	}()

	return g.loadPatternsFromReader(file)
}

// AddPattern adds a new pattern definition to the Grok instance
// This pattern can be used within other patterns
// If this pending uses other patterns that have not yet being added, it
// will remain available until those other patterns are added
func (g *Grok) AddPattern(name string, pattern string) error {
	if strings.Contains(pattern, "%{") {
		state := newCompilationState(name, pattern)
		err := g.compilePatternWithState(state)
		if err != nil {
			if missingErr, ok := err.(*ErrorIncompletePattern); ok {
				g.addIncompleteToPending(missingErr, state)
				return nil
			}
			return err
		}
		g.compiled[name] = newCompiledPatternFromState(state)
	} else {
		g.compiled[name] = newCompiledPattern(pattern)
	}
	return g.continueIncompleteUsing(name)
}

// CompilePattern compiles the given pattern, expanding any pattern
// references using the loaded patterns, and returns a Pattern interface
func (g *Grok) CompilePattern(pattern string) (Pattern, error) {
	var compiled *compiledPattern
	if strings.Contains(pattern, "%{") {
		state := newCompilationState("", pattern)
		err := g.compilePatternWithState(state)
		if err != nil {
			return nil, err
		}
		compiled = newCompiledPatternFromState(state)
	} else {
		compiled = newCompiledPattern(pattern)
	}
	err := compiled.init()
	if err != nil {
		return nil, err
	}
	return compiled, nil
}

// MissingPatterns will return a list of pattern names that are referenced
// by blocked pattern compilations which cannot complete
// This should be called and reported after any pattern loading operations
// Any blocked pattern compilations will not be usable by new patterns
func (g *Grok) MissingPatterns() []string {
	missing := make([]string, 0, len(g.pending))
	for pending := range g.pending {
		missing = append(missing, pending)
	}
	return missing
}

// loadPatternsFromReader loads patterns from a reader, see LoadPatternsFromFile
func (g *Grok) loadPatternsFromReader(reader io.Reader) error {
	bufferedReader := bufio.NewReader(reader)
	for {
		line, err := bufferedReader.ReadString('\n')
		if len(line) != 0 && line[1:] != "" {
			line = strings.TrimSpace(line)
			if line[0] == '#' {
				continue
			}

			split := strings.SplitN(line, " ", 2)
			if len(split) != 2 || split[0] == "" || split[1] == "" {
				return fmt.Errorf("Invalid pattern definition: %s", line)
			}

			err := g.AddPattern(split[0], split[1])
			if err != nil {
				return fmt.Errorf("Invalid pattern definition (%s): %s", err, line)
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}
	}

	return nil
}

// compilePatternWithState attempts to compile a new grok pattern
// it updates the state and returns either an error or nil
// If nil is returned, the state is complete and its output can be used
// If errIncompletePattern is returned, the pattern is still waiting for other patterns
// and will be automatically compiled when those dependency patterns are compiled
func (g *Grok) compilePatternWithState(state *compilationState) error {
	if state.results == nil {
		state.results = matcher.FindAllStringSubmatchIndex(state.pattern, -1)
		if state.results == nil {
			state.output = state.pattern
			return nil
		}
	}

	for state.index < len(state.results) {
		result := state.results[state.index]
		spec := strings.SplitN(state.pattern[result[2]:result[3]], ":", 3)

		if existing, ok := g.compiled[spec[0]]; ok {
			state.output += state.pattern[state.lastOffset:result[0]]
			state.lastOffset = result[1]

			if len(spec) > 1 {
				state.output += fmt.Sprintf("(?P<%s>%s)", spec[1], existing.pattern)
				if len(spec) > 2 {
					state.types[spec[1]] = spec[2]
				}
			} else {
				state.output += existing.pattern
			}

			state.index++
			continue
		}

		return &ErrorIncompletePattern{Missing: spec[0]}
	}

	state.output += state.pattern[state.lastOffset:]

	return nil
}

// addIncompleteToPending adds the given state to the pending list
// for the pattern it is missing
func (g *Grok) addIncompleteToPending(missingErr *ErrorIncompletePattern, state *compilationState) {
	var (
		partials []*compilationState
		ok       bool
	)
	if partials, ok = g.pending[missingErr.Missing]; !ok {
		partials = nil
	}
	g.pending[missingErr.Missing] = append(partials, state)
}

// rebuildIncompleteUsing will continue the compilation of any pending patterns
// that depended on the given pattern (which should now be compiled)
// This function will recurse until all relevant pattern compilations have moved
// along and either completed or blocked at another missing pattern
func (g *Grok) continueIncompleteUsing(name string) error {
	incomplete, ok := g.pending[name]
	if !ok {
		return nil
	}

	delete(g.pending, name)

	for _, state := range incomplete {
		err := g.compilePatternWithState(state)
		if err != nil {
			if missingErr, ok := err.(*ErrorIncompletePattern); ok {
				g.addIncompleteToPending(missingErr, state)
				continue
			}
			return err
		}

		g.compiled[state.name] = newCompiledPatternFromState(state)
		err = g.continueIncompleteUsing(state.name)
		if err != nil {
			return err
		}
	}

	return nil
}
