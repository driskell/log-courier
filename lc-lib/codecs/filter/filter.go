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
	"fmt"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
)

// CodecFilterFactory holds the configuration for a filter codec
type CodecFilterFactory struct {
	Patterns []string `config:"patterns"`
	Match    string   `config:"match"`

	patterns        codecs.PatternCollection
	requiredMatches int
}

// CodecFilter is an instance of a filter codec that is used by the Harvester
// for filtering
type CodecFilter struct {
	config        *CodecFilterFactory
	lastOffset    int64
	filteredLines uint64
	callbackFunc  codecs.CallbackFunc
	meterFiltered uint64
}

// NewFilterCodecFactory creates a new FilterCodecFactory for a codec definition
// in the configuration file. This factory can be used to create instances of a
// filter codec for use by harvesters
func NewFilterCodecFactory(p *config.Parser, configPath string, unused map[string]interface{}, name string) (interface{}, error) {
	var err error

	result := &CodecFilterFactory{}
	if err = p.Populate(result, unused, configPath, true); err != nil {
		return nil, err
	}

	if err = result.patterns.Set(result.Patterns, result.Match); err != nil {
		return nil, fmt.Errorf("Invalid patterns for filter codec at %s: %s", configPath, err)
	}

	return result, nil
}

// NewCodec returns a new codec instance that will send events to the callback
// function provided upon completion of processing
func (f *CodecFilterFactory) NewCodec(callbackFunc codecs.CallbackFunc, offset int64) codecs.Codec {
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
	// Only flush the event if it matches
	matched := c.config.patterns.Match(text)

	if matched {
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

// APIEncodable is called to get the codec status for the API
func (c *CodecFilter) APIEncodable() api.Encodable {
	apiKV := &api.KeyValue{}
	apiKV.SetEntry("filtered_lines", api.Number(c.meterFiltered))
	return apiKV
}

// Register the codec
func init() {
	codecs.Register("filter", NewFilterCodecFactory)
}
