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

// Validate the configuration
func (gc *General) Validate(p *Parser, path string) (err error) {
	if gc.PersistDir == "" {
		err = fmt.Errorf("%s/persist directory must be specified", path)
		return
	}

	// Enforce maximum of 2 GB since event transmit length is uint32
	if gc.SpoolMaxBytes > 2*1024*1024*1024 {
		err = fmt.Errorf("%s/spool max bytes can not be greater than 2 GiB", path)
		return
	}

	if gc.LineBufferBytes < 1 {
		err = fmt.Errorf("%s/line buffer bytes must be greater than 1", path)
		return
	}

	// Max line bytes can not be larger than spool max bytes
	if gc.MaxLineBytes > gc.SpoolMaxBytes {
		err = fmt.Errorf("%s/max line bytes can not be greater than %s/spool max bytes", path, path)
		return
	}

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

func init() {
	RegisterSection("general", func() interface{} {
		return &General{
			LineBufferBytes:  defaultGeneralLineBufferBytes,
			LogLevel:         defaultGeneralLogLevel,
			LogStdout:        defaultGeneralLogStdout,
			LogSyslog:        defaultGeneralLogSyslog,
			MaxLineBytes:     defaultGeneralMaxLineBytes,
			PersistDir:       DefaultGeneralPersistDir,
			ProspectInterval: defaultGeneralProspectInterval,
			SpoolSize:        defaultGeneralSpoolSize,
			SpoolMaxBytes:    defaultGeneralSpoolMaxBytes,
			SpoolTimeout:     defaultGeneralSpoolTimeout,
		}
	})
}
