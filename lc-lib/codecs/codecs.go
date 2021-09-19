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
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/config"
)

// Codec is the generic interface that all codecs implement
type Codec interface {
	Teardown() int64
	Reset()
	ProcessEvent(int64, int64, map[string]interface{}) error
	Meter()
	APIEncodable() api.Encodable
}

// CallbackFunc is a callback function that a codec will call for each of its
// "output" events. It could be called at any time by any routine (not
// necessarily the routine providing the "input" events.)
type CallbackFunc func(int64, int64, map[string]interface{}) error

// codecFactory is the interface that all codec factories implement. The codec
// factory should store the codec's configuration and, when NewCodec is called,
// return an instance of the codec that obeys that configuration
type codecFactory interface {
	NewCodec(CallbackFunc, int64) Codec
}

// NewCodec returns a Codec interface initialised from the given Factory
func NewCodec(factory interface{}, callbackFunc CallbackFunc, offset int64) Codec {
	return factory.(codecFactory).NewCodec(callbackFunc, offset)
}

// CodecRegistrarFunc is a callback that can be registered that will validate
// the configuration settings for a codec registered via RegisterCodec
type CodecRegistrarFunc func(*config.Parser, string, map[string]interface{}, string) (interface{}, error)

var registeredCodecs = make(map[string]CodecRegistrarFunc)

// Register registers a new codec with the configuration module, with a callback
// that can be used to validate its configuration
func Register(codec string, registrarFunc CodecRegistrarFunc) {
	registeredCodecs[codec] = registrarFunc
}

// Available returns the list of registered codecs available for use
func Available() (ret []string) {
	ret = make([]string, 0, len(registeredCodecs))
	for k := range registeredCodecs {
		ret = append(ret, k)
	}
	return
}

// init registers this module provider
func init() {
	config.RegisterAvailable("codecs", Available)
}
