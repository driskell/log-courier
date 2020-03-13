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

package codecs

import (
	"errors"
	"fmt"
	"regexp"
)

// patternInstance holds the regular expression matcher for a single pattern in
// the configuration file, along with any pattern specific configurations
type patternInstance struct {
	matcher *regexp.Regexp
	negate  bool
}

// PatternCollection holds a list of patterns that can be matched against text
type PatternCollection struct {
	patterns        []*patternInstance
	requiredMatches int
}

// Set the pattern list to use and whether to match "any" or "all"
func (c *PatternCollection) Set(patterns []string, match string) error {
	if len(patterns) == 0 {
		return errors.New("At least one pattern must be specified.")
	}

	var err error

	c.patterns = make([]*patternInstance, len(patterns))
	for k, pattern := range patterns {
		patternInstance := &patternInstance{}

		switch pattern[0] {
		case '!':
			patternInstance.negate = true
			pattern = pattern[1:]
		case '=':
			pattern = pattern[1:]
		}

		patternInstance.matcher, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("Failed to compile pattern, '%s': %s", pattern, err)
		}

		c.patterns[k] = patternInstance
	}

	if match == "" || match == "any" {
		c.requiredMatches = 1
	} else if match == "all" {
		c.requiredMatches = len(patterns)
	} else {
		return fmt.Errorf("Unknown \"match\" value for multiline codec, '%s'.", match)
	}

	return nil
}

// Match attempts to match the given text against the set patterns
func (c *PatternCollection) Match(text string) bool {
	if c.patterns == nil {
		panic("Patterns not set")
	}

	var matchCount int
	for _, pattern := range c.patterns {
		if matchFailed := pattern.negate == pattern.matcher.MatchString(text); matchFailed {
			continue
		}
		matchCount++
		if matchCount == c.requiredMatches {
			return true
		}
	}

	return false
}
