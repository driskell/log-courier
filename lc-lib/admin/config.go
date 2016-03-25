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

const (
	defaultGeneralAdminEnabled bool = false
)

// Config holds the admin configuration
// It also holds the root of the API which pipeline segments can attach to in
// order to provide action functions and status returns
type Config struct {
	Enabled bool   `config:"enabled"`
	Bind    string `config:"listen address"`

	APINode
}

// Validate validates the config structure
func (c *Config) Validate() (err error) {
	if c.Enabled && c.Bind == "" {
		err = fmt.Errorf("/admin/listen address must be specified if /admin/enabled is true")
		return
	}

	c.APINode.SetEntry("version", NewAPIDataEntry(APIString(core.LogCourierVersion)))

	return
}

func init() {
	config.RegisterConfigSection("admin", func() config.Section {
		c := &Config{}
		c.Enabled = defaultGeneralAdminEnabled
		c.Bind = "" //DefaultGeneralAdminBind
		return c
	})
}
