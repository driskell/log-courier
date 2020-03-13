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
	"os"

	"gopkg.in/op/go-logging.v1"
)

// registeredGeneralCreators contains a list of registered external Section
// creators that should be processed in all new General structures
var registeredGeneralCreators = make(map[string]SectionCreator)

const (
	defaultGeneralHost      string        = "localhost.localdomain"
	defaultGeneralLogLevel  logging.Level = logging.INFO
	defaultGeneralLogStdout bool          = true
	defaultGeneralLogSyslog bool          = false
)

// General holds the general configuration
type General struct {
	GlobalFields map[string]interface{} `config:"global fields"`
	Host         string                 `config:"host"`
	LogFile      string                 `config:"log file"`
	LogLevel     logging.Level          `config:"log level"`
	LogStdout    bool                   `config:"log stdout"`
	LogSyslog    bool                   `config:"log syslog"`

	Custom map[string]interface{} `config:",embed_dynamic"`
}

// Validate the configuration
func (gc *General) Validate(p *Parser, path string) (err error) {
	if gc.Host == "" {
		ret, hostErr := os.Hostname()
		if hostErr == nil {
			gc.Host = ret
		} else {
			gc.Host = defaultGeneralHost
			log.Warning("Failed to determine the FQDN: %s", hostErr)
			log.Warning("Falling back to using default hostname: %s", gc.Host)
		}
	}

	// Ensure all GlobalFields are map[string]interface{}
	if err = p.FixMapKeys(path+"/global fields", gc.GlobalFields); err != nil {
		return
	}

	return
}

// General returns the general configuration
func (c *Config) General() *General {
	return c.Sections["general"].(*General)
}

// GeneralPart returns the named custom general configuration registered through
// RegisterGeneral()
func (c *Config) GeneralPart(name string) interface{} {
	return c.Sections["general"].(*General).Custom[name]
}

// RegisterGeneral registers additional configuration values to be read from
// the general section of the configuration file. It works exactly the same
// as RegisterSection except that the structure is populated from the general
// section instead of a dedicated named section. The name given is purely for
// accessing the structure directly using GeneralPart()
//
// It is useful for packages to register small sets of rarely used configuration
// values where a dedicated section in the configuration file seems unnecessary.
func RegisterGeneral(name string, creator SectionCreator) {
	registeredGeneralCreators[name] = creator
}

func init() {
	RegisterSection("general", func() interface{} {
		c := &General{
			LogLevel:  defaultGeneralLogLevel,
			LogStdout: defaultGeneralLogStdout,
			LogSyslog: defaultGeneralLogSyslog,
			Custom:    make(map[string]interface{}),
		}

		for k, creator := range registeredGeneralCreators {
			c.Custom[k] = creator()
		}

		return c
	})
}
