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
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

var (
	validationReady = 0
	validationWait  = 2

	// DefaultGeneralPersistDir is a path to the default directory to store
	DefaultGeneralPersistDir = ""
)

const (
	defaultStreamAddOffsetField bool          = true
	defaultStreamAddPathField   bool          = true
	defaultStreamDeadTime       time.Duration = 1 * time.Hour

	defaultGeneralProspectInterval time.Duration = 10 * time.Second
)

// FileConfig holds the configuration for a set of paths that share the same
// stream configuration
type FileConfig struct {
	harvester.StreamConfig `config:",embed"`

	DeadTime time.Duration `config:"dead time"`
	Paths    []string      `config:"paths"`
}

// IncludeConfig holds additional files that need to be loaded into the
// configuration
type IncludeConfig []string

// Config holds the prospector files configuration
type Config []*FileConfig

// Defaults sets up the FileConfig defaults prior to population
func (fc *FileConfig) Defaults() {
	fc.AddOffsetField = defaultStreamAddOffsetField
	fc.AddPathField = defaultStreamAddPathField
	fc.DeadTime = defaultStreamDeadTime
}

// Validate does nothing for a prospector stream
// This is here to prevent double validation of event.StreamConfig whose
// validation function would otherwise be inherited
func (fc *FileConfig) Validate(p *config.Parser, path string) (err error) {
	return nil
}

// Validate the configuration and process the includes
func (ic IncludeConfig) Validate(p *config.Parser, path string) (err error) {
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
			if err = p.PopulateSlice(p.Config().Section("files").(*Config), rawInclude, fmt.Sprintf("%s/%s", path, include)); err != nil {
				return
			}
		}
	}

	// Wait for Config to be processed
	validationReady++
	if validationReady == validationWait {
		err = validateFileConfigs(p)
	}

	return
}

// Validate the Config structure
func (c Config) Validate(p *config.Parser, path string) (err error) {
	// Wait for Includes to be processed
	validationReady++
	if validationReady == validationWait {
		err = validateFileConfigs(p)
	}

	return
}

// General contains extra general section configuration values for the
// prospector and registrar
type General struct {
	PersistDir       string        `config:"persist directory"`
	ProspectInterval time.Duration `config:"prospect interval"`
}

// Validate the additional general configuration
func (gc *General) Validate(p *config.Parser, path string) (err error) {
	if gc.PersistDir == "" {
		err = fmt.Errorf("%s/persist directory must be specified", path)
		return
	}

	return
}

// Validate validates all config structures and initialises streams
func validateFileConfigs(p *config.Parser) (err error) {
	c := p.Config().Section("files").(Config)
	for k := range c {
		if len(c[k].Paths) == 0 {
			err = fmt.Errorf("No paths specified for /files[%d]/", k)
			return
		}

		// Init the harvester config
		if err = c[k].Init(p, fmt.Sprintf("/files[%d]", k)); err != nil {
			return
		}
	}

	return
}

func init() {
	config.RegisterSection("files", func() interface{} {
		return Config{}
	})

	config.RegisterSection("includes", func() interface{} {
		return IncludeConfig{}
	})

	config.RegisterGeneral("prospector", func() interface{} {
		return &General{
			PersistDir:       DefaultGeneralPersistDir,
			ProspectInterval: defaultGeneralProspectInterval,
		}
	})
}
