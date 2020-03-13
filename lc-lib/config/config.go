/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"sort"
)

var (
	// DefaultConfigurationFile is a path to the default configuration file to
	// load, this can be changed during init()
	DefaultConfigurationFile = ""
)

// SectionCreator creates new structures to hold configuration
// When a configuration section is registered a creator is provided that should
// return an empty structure with default values
type SectionCreator func() interface{}

// registeredSectionCreators contains a list of registered external Section
// creators that should be processed in all new Config structures
var registeredSectionCreators = make(map[string]SectionCreator)

// AvailableCallback should return a []string containing available modules
type AvailableCallback func() []string

// registeredAvailableCallbacks contains a list of registered module providers
// which can return a list of supported modules
var registeredAvailableCallbacks = make(map[string]AvailableCallback)

// Config holds the configuration
type Config struct {
	Sections map[string]interface{} `config:",dynamic"`
}

// NewConfig creates a new, empty, configuration structure
func NewConfig() *Config {
	c := &Config{
		Sections: make(map[string]interface{}),
	}

	for name, creator := range registeredSectionCreators {
		c.Sections[name] = creator()
	}

	return c
}

// Load the configuration from the given file
// If reportUnused is false, unknown top level sections will be ignored and will
// not raise a configuration error
func (c *Config) Load(path string, reportUnused bool) (err error) {
	// Read the main config file
	rawConfig := make(map[string]interface{})
	if err = LoadFile(path, &rawConfig); err != nil {
		return
	}

	// Populate configuration - reporting errors on spelling mistakes etc.
	if err = parseConfiguration(c, rawConfig, reportUnused); err != nil {
		return
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
	case ".yml":
		return loadYAMLFile(filePath, rawConfig)
	}

	return fmt.Errorf("File extension '%s' is not within the known extensions: conf, json, yaml, yml", ext)
}

// RegisterSection registers a new configuration section with a SectionCreator
// function, that should return a new structure containing defaults only when
// called. The structure returned can implement several methods to aid in
// configuration building and validation:
//
//   Init(p *Parser, path string)
//     This is called during configuration population and can be used to
//     population dynamic sub-structures. This is used for network transports
//     so that the options available can be changed depending on what the
//     transport field is set to. As long as you have a field called Unused of
//     type map[string]interface{} any not yet parsed options are kept there.
//     This can then be passed to p.Populate() to populate another structure.
//   Validate(p *Parser, path string)
//     This is called after the entire configuration is populated. It can be
//     used to compare values between different sections for validity. Call
//     p.Config() to get the full completed configuration.
//   Defaults()
//     Called before the structure is populated so it can set up defaults.
//     This is helpful for child structures within a section. The defaults
//     for a section itself can be set by the section creator.
func RegisterSection(name string, creator SectionCreator) {
	registeredSectionCreators[name] = creator
}

// RegisterAvailable registers a callback that will be called during
// list-supported so we can list out louded modules that are available
func RegisterAvailable(name string, callback AvailableCallback) {
	registeredAvailableCallbacks[name] = callback
}

// FetchAvailable returns list of module providers and their modules
// The lists are sorted alphabetically
func FetchAvailable() map[string][]string {
	result := make(map[string][]string)
	for name, callback := range registeredAvailableCallbacks {
		list := callback()
		sort.Strings(list)
		result[name] = list
	}
	return result
}
