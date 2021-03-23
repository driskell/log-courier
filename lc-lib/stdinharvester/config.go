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

package stdinharvester

import (
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

// StreamConfig is the stream configuration for the stdin stream
type StreamConfig struct {
	*harvester.StreamConfig `config:",embed"`
}

// Defaults sets default values
// Ensure we override the one from harvester.StreamConfig
func (sc *StreamConfig) Defaults() {
}

// Init initialises the configuration
// Ensure we override the one from harvester.StreamConfig
func (sc *StreamConfig) Init(p *config.Parser, path string) error {
	return nil
}

// Validate validates the stdin stream configuration
// Ensure we override the one from harvester.StreamConfig
func (sc *StreamConfig) Validate(p *config.Parser, path string) error {
	return nil
}

func init() {
	config.RegisterSection("stdin", func() interface{} {
		return &StreamConfig{}
	})
}
