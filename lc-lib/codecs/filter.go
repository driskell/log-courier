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

package codecs

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

// codecFilterPatternInstance holds the regular expression matcher for a
// single pattern in the configuration file, along with any pattern specific
// configurations
type codecFilterPatternInstance struct {
	matcher *regexp.Regexp
	negate  bool
}

// CodecFilterFactory holds the configuration for a filter codec
type CodecFilterFactory struct {
	Patterns []string `config:"patterns"`

	patterns []*codecFilterPatternInstance
}

// CodecFilter is an instance of a filter codec that is used by the Harvester
// for filtering
type CodecFilter struct {
	config        *CodecFilterFactory
	lastOffset    int64
	filteredLines uint64
	callbackFunc  CallbackFunc
	meterFiltered uint64
}

// NewFilterCodecFactory creates a new FilterCodecFactory for a codec definition
// in the configuration file. This factory can be used to create instances of a
// filter codec for use by harvesters
func NewFilterCodecFactory(config *config.Config, configPath string, unused map[string]interface{}, name string) (interface{}, error) {
	var err error

	result := &CodecFilterFactory{}
	if err = config.PopulateConfig(result, unused, configPath); err != nil {
		return nil, err
	}

	if len(result.Patterns) == 0 {
		return nil, errors.New("Filter codec pattern must be specified.")
	}

	result.patterns = make([]*codecFilterPatternInstance, len(result.Patterns))
	for k, pattern := range result.Patterns {
		patternInstance := &codecFilterPatternInstance{}

		switch pattern[0] {
		case '!':
			patternInstance.negate = true
			pattern = pattern[1:]
		case '=':
			pattern = pattern[1:]
		}

		patternInstance.matcher, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("Failed to compile filter codec pattern, '%s'.", err)
		}

		result.patterns[k] = patternInstance
	}

	return result, nil
}

// NewCodec returns a new codec instance that will send events to the callback
// function provided upon completion of processing
func (f *CodecFilterFactory) NewCodec(callbackFunc CallbackFunc, offset int64) Codec {
	return &CodecFilter{
		config:       f,
		lastOffset:   offset,
		callbackFunc: callbackFunc,
	}
}

// Teardown ends the codec and returns the last offset shipped to the callback
func (c *CodecFilter) Teardown() int64 {
	return c.lastOffset
}

// Reset restores the codec to a blank state so it can be reused on a new file
// stream
func (c *CodecFilter) Reset() {
}

// Event is called by a Harvester when a new line event occurs on a file.
// Filtering takes place and only accepted lines are shipped to the callback
func (c *CodecFilter) Event(startOffset int64, endOffset int64, text string) {
	// Only flush the event if it matches a filter
	var matchFailed bool
	for _, pattern := range c.config.patterns {
		if matchFailed = pattern.negate == pattern.matcher.MatchString(text); !matchFailed {
			break
		}
	}

	if !matchFailed {
		c.callbackFunc(startOffset, endOffset, text)
	} else {
		c.filteredLines++
	}

	c.lastOffset = endOffset
}

// Meter is called by the Harvester to request accounting
func (c *CodecFilter) Meter() {
	c.meterFiltered = c.filteredLines
}

// Snapshot is called when lc-admin tool requests a snapshot and the accounting
// data is returned in a snapshot structure
func (c *CodecFilter) Snapshot() *core.Snapshot {
	snap := core.NewSnapshot("Filter Codec")
	snap.AddEntry("Filtered lines", c.meterFiltered)
	return snap
}

// Register the codec
func init() {
	config.RegisterCodec("filter", NewFilterCodecFactory)
}
