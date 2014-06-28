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

package main

type CodecPlainRegistrar struct {
}

type CodecPlainFactory struct {
}

type CodecPlain struct {
  path        string
  fileconfig  *FileConfig
  info        *ProspectorInfo
  last_offset int64
  output      chan<- *FileEvent
}

func (r *CodecPlainRegistrar) NewFactory(config_path string, config map[string]interface{}) (CodecFactory, error) {
  if err := ReportUnusedConfig(config_path, config); err != nil {
    return nil, err
  }
  return &CodecPlainFactory{}, nil
}

func (f *CodecPlainFactory) NewCodec(path string, fileconfig *FileConfig, info *ProspectorInfo, offset int64, output chan<- *FileEvent) Codec {
  return &CodecPlain{
    path:        path,
    fileconfig:  fileconfig,
    info:        info,
    last_offset: offset,
    output:      output,
  }
}

func (c *CodecPlain) Teardown() int64 {
  return c.last_offset
}

func (c *CodecPlain) Event(start_offset int64, end_offset int64, line uint64, text *string) {
  c.last_offset = end_offset

  // Ship downstream
  c.output <- &FileEvent{
    ProspectorInfo: c.info,
    Offset:         end_offset,
    Event:          NewEvent(c.fileconfig.Fields, &c.path, start_offset, line, text),
  }
}

// Register the codec
func init() {
  RegisterCodec(&CodecPlainRegistrar{}, "plain")
}
