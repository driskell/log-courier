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
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	matcher = regexp.MustCompile(`%\{([^}]+)\}`)

	errIncompletePattern = errors.New("Pattern is incomplete")
)

// Grok parses messages into fields using regular expressions
type Grok struct {
	compiled map[string]*compiledPattern
	pending  map[string][]*compilationState
}

type compiledPattern struct {
	pattern string
	types   map[string]string
}

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

func newCompiledPattern(pattern string) *compiledPattern {
	return &compiledPattern{
		pattern: pattern,
		types:   map[string]string{},
	}
}

func newCompiledPatternFromState(state *compilationState) *compiledPattern {
	return &compiledPattern{
		pattern: state.output,
		types:   state.types,
	}
}

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

// AddPattern adds a new pattern definition to the Grok instance
// This pattern can be used within other patterns
// If this pending uses other patterns that have not yet being added, it
// will remain available until those other patterns are added
func (g *Grok) AddPattern(name string, pattern string) error {
	if strings.Contains(pattern, "%{") {
		state := newCompilationState(name, pattern)
		err := g.compilePattern(state)
		if err != nil {
			if err == errIncompletePattern {
				g.compiled[name] = newCompiledPattern("")
				return nil
			}
			return err
		}
		g.compiled[name] = newCompiledPatternFromState(state)
	} else {
		g.compiled[name] = newCompiledPattern(pattern)
	}
	return g.rebuildIncompleteUsing(name)
}

func (g *Grok) compilePattern(state *compilationState) error {
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

		var (
			partials []*compilationState
			ok       bool
		)
		if partials, ok = g.pending[spec[0]]; !ok {
			partials = nil
		}

		g.pending[spec[0]] = append(partials, state)
		return errIncompletePattern
	}

	state.output += state.pattern[state.lastOffset:]

	return nil
}

func (g *Grok) rebuildIncompleteUsing(name string) error {
	incomplete, ok := g.pending[name]
	if !ok {
		return nil
	}

	newIncomplete := incomplete[:0]
	for _, state := range incomplete {
		err := g.compilePattern(state)
		if err != nil {
			if err == errIncompletePattern {
				newIncomplete = append(newIncomplete, state)
				continue
			}
			return err
		}

		g.compiled[state.name] = newCompiledPatternFromState(state)
		err = g.rebuildIncompleteUsing(state.name)
		if err != nil {
			return err
		}
	}

	if len(newIncomplete) == 0 {
		delete(g.pending, name)
	} else if len(newIncomplete) != len(incomplete) {
		g.pending[name] = newIncomplete
	}

	return nil
}
