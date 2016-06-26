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
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
)

// CodecPlainFactory holds the configuration, it is responsible for generating
// instances as required when new log files are opened
type CodecPlainFactory struct {
}

// CodecPlain is an instance of this codec, in use by a single harvester
type CodecPlain struct {
	lastOffset   int64
	callbackFunc codecs.CallbackFunc
}

// NewPlainCodecFactory creates a new factory structure from the configuration
// data in the configuration file.
func NewPlainCodecFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (interface{}, error) {
	// At this point the Log Courier configuration only knows the name of the
	// codec and that is has (or does not have) a set of key-value configuration
	// options. The factory should use config.PopulateConfig to populate its
	// structure from those options (see Multiline for a good example), and it
	// should use ReportUnusedConfig to flag errors if not all of the
	// configuration data was used. This helps tell the user when they made a typo
	if err := p.ReportUnusedConfig(unUsed, configPath); err != nil {
		return nil, err
	}
	return &CodecPlainFactory{}, nil
}

// NewCodec creates a new codec instance starting at the given offset
func (f *CodecPlainFactory) NewCodec(callbackFunc codecs.CallbackFunc, offset int64) codecs.Codec {
	return &CodecPlain{
		lastOffset:   offset,
		callbackFunc: callbackFunc,
	}
}

// Teardown shuts down the codec and it should return the last offset sent
// by the codec. This is used by the harvester as the resume point.
func (c *CodecPlain) Teardown() int64 {
	return c.lastOffset
}

// Reset is called when a log file is truncated, and it should cause the codec
// to reset itself as if it was only just created
func (c *CodecPlain) Reset() {
}

// Event is called for every log event, the resulting log event(s) to be
// transmitted should be passed through the codec callback when ready
func (c *CodecPlain) Event(startOffset int64, endOffset int64, text string) {
	c.lastOffset = endOffset

	c.callbackFunc(startOffset, endOffset, text)
}

// Meter is called by the harvester periodically to allow the codec to calculate
// statistics if necessary
// It is called from the harvester routine so synchronisation is unnecessary
func (c *CodecPlain) Meter() {
}

// APIEncodable is called to get the status for the API
// Meter and APIEncodable are already synchronised by the Harvester, so you
// do not need to worry about race conditions between them
func (c *CodecPlain) APIEncodable() api.Encodable {
	return nil
}

// Register the codec with Log Courier
func init() {
	codecs.Register("plain", NewPlainCodecFactory)
}
