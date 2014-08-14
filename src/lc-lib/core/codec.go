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

package core

type Codec interface {
  Teardown() int64
  Event(int64, int64, uint64, string)
  Snapshot() *Snapshot
}

type CodecCallbackFunc func(int64, int64, uint64, string)

type CodecFactory interface {
  NewCodec(CodecCallbackFunc, int64) Codec
}

type CodecRegistrarFunc func(*Config, string, map[string]interface{}, string) (CodecFactory, error)

var registered_Codecs map[string]CodecRegistrarFunc = make(map[string]CodecRegistrarFunc)

func RegisterCodec(codec string, registrar_func CodecRegistrarFunc) {
  registered_Codecs[codec] = registrar_func
}

func AvailableCodecs() (ret []string) {
  ret = make([]string, 0, len(registered_Codecs))
  for k := range registered_Codecs {
    ret = append(ret, k)
  }
  return
}
