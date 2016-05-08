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

package prospector

import (
	"fmt"
	"path/filepath"

	"github.com/driskell/log-courier/lc-lib/config"
)

// FileConfig holds the configuration for a set of paths that share the same
// stream configuration
type FileConfig struct {
	Paths         []string `config:"paths"`
	config.Stream `config:",embed"`
}

// IncludeConfig holds additional files that need to be loaded into the
// configuration
type IncludeConfig []string

// Config holds the prospector files configuration
type Config []FileConfig

// Validate the configuration and process the includes
func (ic IncludeConfig) Validate(cfg *config.Config, buildMetadata bool) (err error) {
	// Iterate includes
	for _, glob := range ic {
		// Glob the path
		var matches []string
		if matches, err = filepath.Glob(glob); err != nil {
			return
		}

		for _, include := range matches {
			// Read the include
			var rawInclude []interface{}
			if err = config.LoadFile(include, &rawInclude); err != nil {
				return
			}

			// Append to files configuration
			if err = config.PopulateSlice(cfg.Section("files").([]FileConfig), rawInclude, fmt.Sprintf("%s/", include)); err != nil {
				return
			}
		}
	}

	return
}

// Validate validates the config structure
func (c Config) Validate(cfg *config.Config, buildMetadata bool) (err error) {
	for k := range c {
		if len(c[k].Paths) == 0 {
			err = fmt.Errorf("No paths specified for /files[%d]/", k)
			return
		}

		if err = c[k].Stream.Init(cfg, fmt.Sprintf("/files[%d]", k), buildMetadata); err != nil {
			return
		}
	}

	return
}

func init() {
	config.RegisterConfigSection("files", func() config.Section {
		return Config{}
	})

	config.RegisterConfigSection("includes", func() config.Section {
		return &IncludeConfig{}
	})
}
