/*
 * Copyright 2014 Jason Woods.
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
	"github.com/driskell/log-courier/src/lc-lib/core"
)

type CodecPlainFactory struct {
}

type CodecPlain struct {
	last_offset   int64
	callback_func core.CodecCallbackFunc
}

func NewPlainCodecFactory(config *core.Config, config_path string, unused map[string]interface{}, name string) (core.CodecFactory, error) {
	if err := config.ReportUnusedConfig(config_path, unused); err != nil {
		return nil, err
	}
	return &CodecPlainFactory{}, nil
}

func (f *CodecPlainFactory) NewCodec(callback_func core.CodecCallbackFunc, offset int64) core.Codec {
	return &CodecPlain{
		last_offset:   offset,
		callback_func: callback_func,
	}
}

func (c *CodecPlain) Teardown() int64 {
	return c.last_offset
}

func (c *CodecPlain) Reset() {
}

func (c *CodecPlain) Event(start_offset int64, end_offset int64, text string) {
	c.last_offset = end_offset

	c.callback_func(start_offset, end_offset, text)
}

func (c *CodecPlain) Meter() {
}

func (c *CodecPlain) Snapshot() *core.Snapshot {
	return nil
}

// Register the codec
func init() {
	core.RegisterCodec("plain", NewPlainCodecFactory)
}
