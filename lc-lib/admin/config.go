/*
 * Copyright 2014-2015 Jason Woods.
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

package admin

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

var (
	defaultAdminEnabled = false

	// DefaultAdminBind is the default bind address to use when admin is enabled
	// and can be modified during init()
	DefaultAdminBind = "tcp:127.0.0.1:12345"
)

// Config holds the admin configuration
// It also holds the root of the API which pipeline segments can attach to in
// order to provide action functions and status returns
type Config struct {
	Enabled bool   `config:"enabled"`
	Bind    string `config:"listen address"`

	apiRoot APINavigatable
}

// Validate validates the config structure
func (c *Config) Validate(config *config.Config, buildMetadata bool) (err error) {
	if c.Enabled && c.Bind == "" {
		err = fmt.Errorf("/admin/listen address must be specified if /admin/enabled is true")
		return
	}

	return
}

// SetEntry sets a new root API entry
func (c *Config) SetEntry(path string, entry APINavigatable) {
	c.apiRoot.(*apiRoot).SetEntry(path, entry)
}

// ConfigFromApp returns the config from the given application config
func ConfigFromApp(app *core.App) *Config {
	return app.Config().Section("admin").(*Config)
}

func init() {
	config.RegisterConfigSection("admin", func() config.Section {
		return &Config{
			Enabled: defaultAdminEnabled,
			Bind:    DefaultAdminBind,
		}
	})
}
