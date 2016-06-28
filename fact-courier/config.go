/*
 * Copyright 2014-2016 Jason Woods.
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

import (
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

// Config holds the Fact Courier configuration, and the Stream configuration
type Config struct {
	event.StreamConfig `config:",embed"`
}

// Defaults populates any default configurations
func (c *Config) Defaults() {
}

// Validate does nothing for a fact-courier stream
// This is here to prevent double validation of event.StreamConfig whose
// validation function would otherwise be inherited
func (c *Config) Validate(p *config.Parser, path string) (err error) {
	return nil
}

func init() {
	config.RegisterSection("facts", func() interface{} {
		return &Config{}
	})
}
