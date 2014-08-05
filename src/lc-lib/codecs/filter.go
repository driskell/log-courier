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
  "errors"
  "fmt"
  "lc-lib/core"
  "regexp"
)

type CodecFilterFactory struct {
  Pattern string `config:"pattern"`
  Negate  bool   `config:"negate"`

  matcher *regexp.Regexp
}

type CodecFilter struct {
  config        *CodecFilterFactory
  last_offset   int64
  callback_func core.CodecCallbackFunc
}

func NewFilterCodecFactory(config *core.Config, config_path string, unused map[string]interface{}, name string) (core.CodecFactory, error) {
  var err error

  result := &CodecFilterFactory{}
  if err = config.PopulateConfig(result, config_path, unused); err != nil {
    return nil, err
  }

  if result.Pattern == "" {
    return nil, errors.New("Filter codec pattern must be specified.")
  }

  result.matcher, err = regexp.Compile(result.Pattern)
  if err != nil {
    return nil, fmt.Errorf("Failed to compile filter codec pattern, '%s'.", err)
  }

  return result, nil
}

func (f *CodecFilterFactory) NewCodec(callback_func core.CodecCallbackFunc, offset int64) core.Codec {
  return &CodecFilter{
    config:        f,
    last_offset:   offset,
    callback_func: callback_func,
  }
}

func (c *CodecFilter) Teardown() int64 {
  return c.last_offset
}

func (c *CodecFilter) Event(start_offset int64, end_offset int64, line uint64, text string) {
  // Only flush the event if it doesn't match the filter
  if c.config.Negate == c.config.matcher.MatchString(text) {
    c.callback_func(start_offset, end_offset, line, text)
  }
}

// Register the codec
func init() {
  core.RegisterCodec("filter", NewFilterCodecFactory)
}
