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

type CodecRegistrar interface {
  NewFactory(string, map[string]interface{}) (CodecFactory, error)
}

type CodecFactory interface {
  NewCodec(string, *FileConfig, *ProspectorInfo, int64, chan<- *FileEvent) Codec
}

type Codec interface {
  Teardown() int64
  Event(int64, int64, uint64, *string) bool
}

var codecRegistry map[string]CodecRegistrar = make(map[string]CodecRegistrar)

func RegisterCodec(registrar CodecRegistrar, name string) {
  codecRegistry[name] = registrar
}

func NewCodecFactory(config_path string, name string, config map[string]interface{}) (CodecFactory, error) {
  if registrar, ok := codecRegistry[name]; ok {
    return registrar.NewFactory(config_path, config)
  }
  return nil, nil
}
