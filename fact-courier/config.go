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

package main

import (
	"os"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	defaultMuninConfigBase    string = "/etc/munin"
	defaultMuninConfigFile    string = "${munin config base}/munin-node.conf"
	defaultMuninConfigPluginD string = "${munin config base}/plugin-conf.d"
	defaultMuninPluginBase    string = "${munin config base}/plugins"
)

// Config holds the Fact Courier configuration, and the Stream configuration
type Config struct {
	event.StreamConfig `config:",embed"`

	MuninConfigBase    string `config:"munin config base"`
	MuninConfigFile    string `config:"munin config file"`
	MuninConfigPluginD string `config:"munin config plugind"`
	MuninPluginBase    string `config:"munin plugin base"`
}

// Defaults populates any default configurations
// Ensure we override the one from event.StreamConfig
func (c *Config) Defaults() {
}

// Init initialises the configuration
// Ensure we override the one from event.StreamConfig
func (c *Config) Init() {
}

// Validate will check the configuration and expand variables
// Ensure we override the one from event.StreamConfig
func (c *Config) Validate(p *config.Parser, path string) (err error) {
	mapper := func(name string) string {
		switch name {
		case "munin config base":
			return c.MuninConfigBase
		}
		return ""
	}

	c.MuninConfigFile = os.Expand(c.MuninConfigFile, mapper)
	c.MuninConfigPluginD = os.Expand(c.MuninConfigPluginD, mapper)
	c.MuninPluginBase = os.Expand(c.MuninPluginBase, mapper)

	return nil
}

func init() {
	config.RegisterSection("facts", func() interface{} {
		return &Config{
			MuninConfigBase:    defaultMuninConfigBase,
			MuninConfigFile:    defaultMuninConfigFile,
			MuninConfigPluginD: defaultMuninConfigPluginD,
			MuninPluginBase:    defaultMuninPluginBase,
		}
	})
}
