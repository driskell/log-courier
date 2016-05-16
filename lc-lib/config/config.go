/*
 * Copyright 2014-2015 Jason Woods.
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
	"path"
)

var (
	// DefaultConfigurationFile is a path to the default configuration file to
	// load, this can be changed during init()
	DefaultConfigurationFile = ""
)

// Section is implemented by external config structures that will be
// registered with the config package
type Section interface {
	Validate(config *Config, buildMetadata bool) error
}

// SectionCreator creates new Section structures
type SectionCreator func() Section

// registeredSectionCreators contains a list of registered external Section
// creators that should be processed in all new Config structures
var registeredSectionCreators = make(map[string]SectionCreator)

// Config holds the configuration
type Config struct {
	// TODO: Needs shifting into Log Courier
	Stdin Stream `config:"stdin"`

	Sections map[string]Section `config:",dynamic"`
}

// NewConfig creates a new, empty, configuration structure
func NewConfig() *Config {
	c := &Config{
		Sections: make(map[string]Section),
	}

	for name, creator := range registeredSectionCreators {
		c.Sections[name] = creator()
	}

	return c
}

// Load the configuration from the given file
// If buildMetadata is false, factories (such as codec names or transport
// names) and other metadata are not initialised
func (c *Config) Load(path string, buildMetadata bool) (err error) {
	// Read the main config file
	rawConfig := make(map[string]interface{})
	if err = LoadFile(path, &rawConfig); err != nil {
		return
	}

	// Populate configuration - reporting errors on spelling mistakes etc.
	if err = PopulateConfig(c, rawConfig, "/"); err != nil {
		return
	}

	if err = c.Stdin.Init(c, "/stdin", buildMetadata); err != nil {
		return
	}

	// Validate the registered configurables
	for _, section := range c.Sections {
		if err = section.Validate(c, buildMetadata); err != nil {
			return
		}
	}

	return
}

// Section returns the requested dynamic configuration entry
func (c *Config) Section(name string) interface{} {
	ret, ok := c.Sections[name]
	if !ok {
		return nil
	}

	return ret
}

// LoadFile detects the extension of the given file and loads it using the
// relevant load function
func LoadFile(filePath string, rawConfig interface{}) error {
	ext := path.Ext(filePath)

	switch ext {
	case ".json":
		return loadJSONFile(filePath, rawConfig)
	case ".conf":
		return loadJSONFile(filePath, rawConfig)
	case ".yaml":
		return loadYAMLFile(filePath, rawConfig)
	}

	return fmt.Errorf("File extension '%s' is not within the known extensions: conf, json, yaml", ext)
}

// RegisterConfigSection registers a new Section creator which will be used to
// create new sections that will be available via Get() in all created Config
// structures
func RegisterConfigSection(name string, creator SectionCreator) {
	registeredSectionCreators[name] = creator
}
