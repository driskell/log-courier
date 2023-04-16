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

package processor

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/grok"
)

const (
	defaultLoadDefaults = true
)

// GrokConfig contains configuration for grok
type GrokConfig struct {
	LoadDefaults bool     `config:"load defaults"`
	PatternFiles []string `config:"pattern files"`

	Grok *grok.Grok
}

// Defaults sets defaults
func (c *GrokConfig) Defaults() {
	c.LoadDefaults = defaultLoadDefaults
}

// Init the grok configuration
func (c *GrokConfig) Init(p *config.Parser, path string) error {
	c.Grok = grok.NewGrok(c.LoadDefaults)
	for _, path := range c.PatternFiles {
		err := c.Grok.LoadPatternsFromFile(path)
		if err != nil {
			return fmt.Errorf("Failed to load patterns from %s: %s", path, err)
		}
	}
	return nil
}

// FetchGrokConfig returns the grok configuration from a Config structure
func FetchGrokConfig(cfg *config.Config) *GrokConfig {
	return cfg.Section("grok").(*GrokConfig)
}

// init registers grok config and its action
func init() {
	config.RegisterSection("grok", func() interface{} {
		return &GrokConfig{}
	})
}
