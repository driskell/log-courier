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

package config

// CodecRegistrarFunc is a callback that can be registered that will validate
// the configuration settings for a codec registered via RegisterCodec
type CodecRegistrarFunc func(*Config, string, map[string]interface{}, string) (interface{}, error)

var registeredCodecs = make(map[string]CodecRegistrarFunc)

// RegisterCodec registers a new codec with the configuration module, with a
// callback that can be used to validate its configuration
func RegisterCodec(codec string, registrarFunc CodecRegistrarFunc) {
	registeredCodecs[codec] = registrarFunc
}

// AvailableCodecs returns the list of registered codecs available for use
func AvailableCodecs() (ret []string) {
	ret = make([]string, 0, len(registeredCodecs))
	for k := range registeredCodecs {
		ret = append(ret, k)
	}
	return
}
