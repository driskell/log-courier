/*
 * Copyright 2014-2016 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

import (
	"fmt"
	"time"
)

const (
	defaultStreamAddHostField     bool          = true
	defaultStreamAddOffsetField   bool          = true
	defaultStreamAddPathField     bool          = true
	defaultStreamAddTimezoneField bool          = false
	defaultStreamCodec            string        = "plain"
	defaultStreamDeadTime         time.Duration = 1 * time.Hour
)

// CodecStub holds an unknown codec configuration
// After initial parsing of configuration, these CodecStubs are turned into
// real configuration blocks for the codec given by their Name field
type CodecStub struct {
	Name    string `config:"name"`
	Unused  map[string]interface{}
	Factory interface{}
}

// Stream holds the configuration for a log stream
// TODO: Currently this is controlled by harvester but we should take it out
//       into a stream library along with Codecs
type Stream struct {
	AddHostField     bool                   `config:"add host field"`
	AddOffsetField   bool                   `config:"add offset field"`
	AddPathField     bool                   `config:"add path field"`
	AddTimezoneField bool                   `config:"add timezone field"`
	Codecs           []CodecStub            `config:"codecs"`
	DeadTime         time.Duration          `config:"dead time"`
	Fields           map[string]interface{} `config:"fields"`
}

// InitDefaults initialises the default configuration for a log stream
func (sc *Stream) InitDefaults() {
	sc.AddHostField = defaultStreamAddHostField
	sc.AddOffsetField = defaultStreamAddOffsetField
	sc.AddPathField = defaultStreamAddPathField
	sc.AddTimezoneField = defaultStreamAddTimezoneField
	sc.DeadTime = defaultStreamDeadTime
}

// Init initialises a stream configuration by creating the necessary
// codec factories the harvesters will require
func (sc *Stream) Init(cfg *Config, path string, buildMetadata bool) (err error) {
	if !buildMetadata {
		// Currently only codec factory is initialised, so skip if we're not doing that
		return nil
	}

	if len(sc.Codecs) == 0 {
		sc.Codecs = []CodecStub{CodecStub{Name: defaultStreamCodec}}
	}

	for i := 0; i < len(sc.Codecs); i++ {
		codec := &sc.Codecs[i]
		if registrarFunc, ok := registeredCodecs[codec.Name]; ok {
			if codec.Factory, err = registrarFunc(cfg, path, codec.Unused, codec.Name); err != nil {
				return
			}
		} else {
			return fmt.Errorf("Unrecognised codec '%s' for %s", codec.Name, path)
		}
	}

	// TODO: EDGE CASE: Event transmit length is uint32, if fields length is rediculous we will fail

	return nil
}
