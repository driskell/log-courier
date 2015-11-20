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
	"github.com/driskell/log-courier/lc-lib/core"
)

// Codec is the generic interface that all codecs implement
type Codec interface {
	Teardown() int64
	Reset()
	Event(int64, int64, string)
	Meter()
	Snapshot() *core.Snapshot
}

// CallbackFunc is a callback function that a codec will call for each of its
// "output" events. It could be called at any time by any routine (not
// necessarily the routine providing the "input" events.)
type CallbackFunc func(int64, int64, string)

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
