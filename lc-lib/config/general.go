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
	"os"
	"time"

	"gopkg.in/op/go-logging.v1"
)

var (
	// DefaultGeneralPersistDir is a path to the default directory to store
	DefaultGeneralPersistDir = ""
)

const (
	defaultGeneralHost             string        = "localhost.localdomain"
	defaultGeneralLogLevel         logging.Level = logging.INFO
	defaultGeneralLogStdout        bool          = true
	defaultGeneralLogSyslog        bool          = false
	defaultGeneralLineBufferBytes  int64         = 16384
	defaultGeneralMaxLineBytes     int64         = 1048576
	defaultGeneralProspectInterval time.Duration = 10 * time.Second
	defaultGeneralSpoolMaxBytes    int64         = 10485760
	defaultGeneralSpoolSize        int64         = 1024
	defaultGeneralSpoolTimeout     time.Duration = 5 * time.Second
)

// General holds the general configuration
type General struct {
	GlobalFields  map[string]interface{} `config:"global fields"`
	Host          string                 `config:"host"`
	LogFile       string                 `config:"log file"`
	LogLevel      logging.Level          `config:"log level"`
	LogStdout     bool                   `config:"log stdout"`
	LogSyslog     bool                   `config:"log syslog"`
	SpoolSize     int64                  `config:"spool size"`
	SpoolMaxBytes int64                  `config:"spool max bytes"`
	SpoolTimeout  time.Duration          `config:"spool timeout"`

	// TODO: Log Courier specific fields - have a dynamic area to General? Or do
	// they deserve their own section? Own section would break compatibility though
	LineBufferBytes  int64         `config:"line buffer bytes"`
	MaxLineBytes     int64         `config:"max line bytes"`
	PersistDir       string        `config:"persist directory"`
	ProspectInterval time.Duration `config:"prospect interval"`
}

// InitDefaults initialises default values for the general configuration
func (gc *General) InitDefaults() {
	gc.LineBufferBytes = defaultGeneralLineBufferBytes
	gc.LogLevel = defaultGeneralLogLevel
	gc.LogStdout = defaultGeneralLogStdout
	gc.LogSyslog = defaultGeneralLogSyslog
	gc.MaxLineBytes = defaultGeneralMaxLineBytes
	gc.PersistDir = DefaultGeneralPersistDir
	gc.ProspectInterval = defaultGeneralProspectInterval
	gc.SpoolSize = defaultGeneralSpoolSize
	gc.SpoolMaxBytes = defaultGeneralSpoolMaxBytes
	gc.SpoolTimeout = defaultGeneralSpoolTimeout
	// NOTE: Empty string for Host means calculate it automatically, so leave it
}

// Validate the configuration
func (gc *General) Validate(config *Config, buildMetadata bool) (err error) {
	if gc.PersistDir == "" {
		err = fmt.Errorf("/general/persist directory must be specified")
		return
	}

	// Enforce maximum of 2 GB since event transmit length is uint32
	if gc.SpoolMaxBytes > 2*1024*1024*1024 {
		err = fmt.Errorf("/general/spool max bytes can not be greater than 2 GiB")
		return
	}

	if gc.LineBufferBytes < 1 {
		err = fmt.Errorf("/general/line buffer bytes must be greater than 1")
		return
	}

	// Max line bytes can not be larger than spool max bytes
	if gc.MaxLineBytes > gc.SpoolMaxBytes {
		err = fmt.Errorf("/general/max line bytes can not be greater than /general/spool max bytes")
		return
	}

	if gc.Host == "" {
		ret, err := os.Hostname()
		if err == nil {
			gc.Host = ret
		} else {
			gc.Host = defaultGeneralHost
			log.Warning("Failed to determine the FQDN; using '%s'.", gc.Host)
		}
	}

	return
}

// General returns the general configuration
func (c *Config) General() *General {
	return c.Sections["general"].(*General)
}

func init() {
	RegisterConfigSection("general", func() Section {
		return &General{}
	})
}
