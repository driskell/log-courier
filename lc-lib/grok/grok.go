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
	name          string
	pattern       string
	localCompiled map[string]*compiledPattern

	output string
	types  map[string]TypeHint

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
func newCompilationState(name string, pattern string, localCompiled map[string]*compiledPattern) *compilationState {
	return &compilationState{
		name:          name,
		pattern:       pattern,
		localCompiled: localCompiled,
		types:         make(map[string]TypeHint),
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
		state := newCompilationState(name, pattern, nil)
		err := compilePatternWithState(g.compiled, state)
		if err != nil {
			if missingErr, ok := err.(*ErrorIncompletePattern); ok {
				addIncompleteToPending(g.pending, missingErr, state)
				return nil
			}
			return err
		}
		g.compiled[name] = newCompiledPatternFromState(state)
	} else {
		g.compiled[name] = newCompiledPattern(pattern)
	}
	return continueIncompleteUsing(g.pending, g.compiled, g.compiled, name)
}

// CompilePattern compiles the given pattern, expanding any pattern
// references using the loaded patterns, and returns a Pattern interface
// You can optionally specify a non-nil local map of patterns to be used for this compile
// only in the local parameter to save you from adding them globally unnecessarily
func (g *Grok) CompilePattern(pattern string, local map[string]string) (Pattern, error) {
	var compiled *compiledPattern
	if strings.Contains(pattern, "%{") {
		var localCompiled map[string]*compiledPattern
		if local != nil && len(local) != 0 {
			var err error
			localCompiled, err = g.compileLocal(local)
			if err != nil {
				return nil, err
			}
		}
		state := newCompilationState("", pattern, localCompiled)
		err := compilePatternWithState(g.compiled, state)
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
// by blocked added pattern compilations which cannot complete
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

// compileLocal performs a local compilation that does not add to the internal list
// of patterns. It can be used for type hinting local patterns.
func (g *Grok) compileLocal(local map[string]string) (map[string]*compiledPattern, error) {
	localCompiled := make(map[string]*compiledPattern)
	pending := make(map[string][]*compilationState)
	for localName, localPattern := range local {
		state := newCompilationState(localName, localPattern, localCompiled)
		err := compilePatternWithState(g.compiled, state)
		if err != nil {
			if missingErr, ok := err.(*ErrorIncompletePattern); ok {
				addIncompleteToPending(pending, missingErr, state)
				continue
			}
			return nil, err
		}
		localCompiled[localName] = newCompiledPatternFromState(state)
		err = continueIncompleteUsing(pending, g.compiled, localCompiled, localName)
		if err != nil {
			return nil, err
		}
	}
	if len(pending) != 0 {
		for reference := range pending {
			return nil, fmt.Errorf("Unable to compile pattern due to missing reference: %s", reference)
		}
	}
	return localCompiled, nil
}

// compilePatternWithState attempts to compile a new grok pattern
// it updates the state and returns either an error or nil
// If nil is returned, the state is complete and its output can be used
// If errIncompletePattern is returned, the pattern is still waiting for other patterns
// and will be automatically compiled when those dependency patterns are compiled
func compilePatternWithState(compiled map[string]*compiledPattern, state *compilationState) error {
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

		var existing *compiledPattern
		if found, ok := compiled[spec[0]]; ok {
			existing = found
		}
		if state.localCompiled != nil {
			if found, ok := state.localCompiled[spec[0]]; ok {
				existing = found
			}
		}
		if existing != nil {
			state.output += state.pattern[state.lastOffset:result[0]]
			state.lastOffset = result[1]

			if len(spec) > 1 {
				state.output += fmt.Sprintf("(?P<%s>%s)", spec[1], existing.pattern)
				if len(spec) > 2 {
					typeHint, err := parseType(spec[2])
					if err != nil {
						return err
					}
					if typeHint != TypeHintString {
						// Default output is string so we only need to note non-string hints
						state.types[spec[1]] = typeHint
					}
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
func addIncompleteToPending(pending map[string][]*compilationState, missingErr *ErrorIncompletePattern, state *compilationState) {
	var (
		partials []*compilationState
		ok       bool
	)
	if partials, ok = pending[missingErr.Missing]; !ok {
		partials = nil
	}
	pending[missingErr.Missing] = append(partials, state)
}

// continueIncompleteUsing will continue the compilation of any pending patterns
// that depended on the given pattern (which should now be compiled)
// This function will recurse until all relevant pattern compilations have moved
// along and either completed or blocked at another missing pattern
func continueIncompleteUsing(pending map[string][]*compilationState, compiled map[string]*compiledPattern, target map[string]*compiledPattern, name string) error {
	incomplete, ok := pending[name]
	if !ok {
		return nil
	}

	delete(pending, name)

	for _, state := range incomplete {
		err := compilePatternWithState(compiled, state)
		if err != nil {
			if missingErr, ok := err.(*ErrorIncompletePattern); ok {
				addIncompleteToPending(pending, missingErr, state)
				continue
			}
			return err
		}

		target[state.name] = newCompiledPatternFromState(state)
		err = continueIncompleteUsing(pending, compiled, target, state.name)
		if err != nil {
			return err
		}
	}

	return nil
}
