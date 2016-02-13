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
	"path"
	"path/filepath"
	"reflect"
	"time"

	logging "github.com/driskell/log-courier/Godeps/_workspace/src/github.com/op/go-logging"
)

const (
	defaultGeneralConfigAdminEnabled       bool          = false
	defaultGeneralConfigAdminBind          string        = "tcp:127.0.0.1:1234"
	defaultGeneralConfigProspectInterval   time.Duration = 10 * time.Second
	defaultGeneralConfigSpoolSize          int64         = 1024
	defaultGeneralConfigSpoolMaxBytes      int64         = 10485760
	defaultGeneralConfigSpoolTimeout       time.Duration = 5 * time.Second
	defaultGeneralConfigLineBufferBytes    int64         = 16384
	defaultGeneralConfigMaxLineBytes       int64         = 1048576
	defaultGeneralConfigLogLevel           logging.Level = logging.INFO
	defaultGeneralConfigLogStdout          bool          = true
	defaultGeneralConfigLogSyslog          bool          = false
	defaultNetworkConfigMethod             string        = "failover"
	defaultNetworkConfigTransport          string        = "tls"
	defaultNetworkConfigRfc2782Srv         bool          = true
	defaultNetworkConfigRfc2782Service     string        = "courier"
	defaultNetworkConfigTimeout            time.Duration = 15 * time.Second
	defaultNetworkConfigReconnect          time.Duration = 1 * time.Second
	defaultNetworkConfigMaxPendingPayloads int64         = 4
	defaultStreamConfigCodec               string        = "plain"
	defaultStreamConfigDeadTime            time.Duration = 24 * time.Hour
	defaultStreamConfigAddHostField        bool          = true
	defaultStreamConfigAddOffsetField      bool          = true
	defaultStreamConfigAddPathField        bool          = true
	defaultStreamConfigAddTimezoneField    bool          = false
)

var (
	defaultGeneralConfigHost = "localhost.localdomain"
)

// General holds the general configuration
type General struct {
	AdminEnabled     bool                   `config:"admin enabled"`
	AdminBind        string                 `config:"admin listen address"`
	PersistDir       string                 `config:"persist directory"`
	ProspectInterval time.Duration          `config:"prospect interval"`
	SpoolSize        int64                  `config:"spool size"`
	SpoolMaxBytes    int64                  `config:"spool max bytes"`
	SpoolTimeout     time.Duration          `config:"spool timeout"`
	LineBufferBytes  int64                  `config:"line buffer bytes"`
	MaxLineBytes     int64                  `config:"max line bytes"`
	LogLevel         logging.Level          `config:"log level"`
	LogStdout        bool                   `config:"log stdout"`
	LogSyslog        bool                   `config:"log syslog"`
	LogFile          string                 `config:"log file"`
	Host             string                 `config:"host"`
	GlobalFields     map[string]interface{} `config:"global fields"`
}

// InitDefaults initialises default values for the general configuration
func (gc *General) InitDefaults() {
	gc.AdminEnabled = defaultGeneralConfigAdminEnabled
	gc.AdminBind = defaultGeneralConfigAdminBind
	gc.ProspectInterval = defaultGeneralConfigProspectInterval
	gc.SpoolSize = defaultGeneralConfigSpoolSize
	gc.SpoolMaxBytes = defaultGeneralConfigSpoolMaxBytes
	gc.SpoolTimeout = defaultGeneralConfigSpoolTimeout
	gc.LineBufferBytes = defaultGeneralConfigLineBufferBytes
	gc.MaxLineBytes = defaultGeneralConfigMaxLineBytes
	gc.LogLevel = defaultGeneralConfigLogLevel
	gc.LogStdout = defaultGeneralConfigLogStdout
	gc.LogSyslog = defaultGeneralConfigLogSyslog
	// NOTE: Empty string for Host means calculate it automatically, so leave it
}

// Network holds network related configuration
type Network struct {
	Transport          string        `config:"transport"`
	Servers            []string      `config:"servers"`
	Method             string        `config:"method"`
	Rfc2782Srv         bool          `config:"rfc 2782 srv"`
	Rfc2782Service     string        `config:"rfc 2782 service"`
	Timeout            time.Duration `config:"timeout"`
	Reconnect          time.Duration `config:"reconnect"`
	MaxPendingPayloads int64         `config:"max pending payloads"`
	Factory            interface{}
	Unused             map[string]interface{}
}

// InitDefaults initiases default values for the network configuration
func (nc *Network) InitDefaults() {
	nc.Rfc2782Srv = defaultNetworkConfigRfc2782Srv
	nc.Transport = defaultNetworkConfigTransport
	nc.Rfc2782Service = defaultNetworkConfigRfc2782Service
	nc.Timeout = defaultNetworkConfigTimeout
	nc.Reconnect = defaultNetworkConfigReconnect
	nc.MaxPendingPayloads = defaultNetworkConfigMaxPendingPayloads
}

// CodecStub holds an unknown codec configuration
// After initial parsing of configuration, these CodecStubs are turned into
// real configuration blocks for the codec given by their Name field
type CodecStub struct {
	Name    string `config:"name"`
	Unused  map[string]interface{}
	Factory interface{}
}

// Stream holds the configuration for a log stream
type Stream struct {
	Fields           map[string]interface{} `config:"fields"`
	AddHostField     bool                   `config:"add host field"`
	AddOffsetField   bool                   `config:"add offset field"`
	AddPathField     bool                   `config:"add path field"`
	AddTimezoneField bool                   `config:"add timezone field"`
	Codecs           []CodecStub            `config:"codecs"`
	DeadTime         time.Duration          `config:"dead time"`
}

// InitDefaults initialises the default configuration for a log stream
func (sc *Stream) InitDefaults() {
	sc.DeadTime = defaultStreamConfigDeadTime
	sc.AddHostField = defaultStreamConfigAddHostField
	sc.AddOffsetField = defaultStreamConfigAddOffsetField
	sc.AddPathField = defaultStreamConfigAddPathField
	sc.AddTimezoneField = defaultStreamConfigAddTimezoneField
}

// File holds the configuration for a set of paths that share the same stream
// configuration
type File struct {
	Paths  []string `config:"paths"`
	Stream `config:",embed"`
}

// Config holds all the configuration for Log Courier
type Config struct {
	General  General  `config:"general"`
	Network  Network  `config:"network"`
	Files    []File   `config:"files"`
	Includes []string `config:"includes"`
	Stdin    Stream   `config:"stdin"`
}

// NewConfig creates a new, empty, configuration structure
func NewConfig() *Config {
	return &Config{}
}

// loadFile detects the extension of the given file and loads it using the
// relevant load function
func (c *Config) loadFile(filePath string, rawConfig interface{}) error {
	ext := path.Ext(filePath)

	switch ext {
	case ".json":
		return c.loadJSONFile(filePath, rawConfig)
	case ".conf":
		return c.loadJSONFile(filePath, rawConfig)
	case ".yaml":
		return c.loadYAMLFile(filePath, rawConfig)
	}

	return fmt.Errorf("File extension '%s' is not within the known extensions: conf, json, yaml", ext)
}

// Load the configuration from the given file
func (c *Config) Load(path string) (err error) {
	// Read the main config file
	rawConfig := make(map[string]interface{})
	if err = c.loadFile(path, &rawConfig); err != nil {
		return
	}

	// Populate configuration - reporting errors on spelling mistakes etc.
	if err = c.PopulateConfig(c, rawConfig, "/"); err != nil {
		return
	}

	// Iterate includes
	for _, glob := range c.Includes {
		// Glob the path
		var matches []string
		if matches, err = filepath.Glob(glob); err != nil {
			return
		}

		for _, include := range matches {
			// Read the include
			var rawInclude []interface{}
			if err = c.loadFile(include, &rawInclude); err != nil {
				return
			}

			// Append to configuration
			vRawInclude := reflect.ValueOf(rawInclude)
			if err = c.populateSlice(reflect.ValueOf(c).Elem().FieldByName("Files"), vRawInclude, fmt.Sprintf("%s/", include)); err != nil {
				return
			}
		}
	}

	if c.General.PersistDir == "" {
		err = fmt.Errorf("/general/persist directory must be specified")
		return
	}

	// Enforce maximum of 2 GB since event transmit length is uint32
	if c.General.SpoolMaxBytes > 2*1024*1024*1024 {
		err = fmt.Errorf("/general/spool max bytes can not be greater than 2 GiB")
		return
	}

	if c.General.LineBufferBytes < 1 {
		err = fmt.Errorf("/general/line buffer bytes must be greater than 1")
		return
	}

	// Max line bytes can not be larger than spool max bytes
	if c.General.MaxLineBytes > c.General.SpoolMaxBytes {
		err = fmt.Errorf("/general/max line bytes can not be greater than /general/spool max bytes")
		return
	}

	if c.General.Host == "" {
		ret, err := os.Hostname()
		if err == nil {
			c.General.Host = ret
		} else {
			c.General.Host = defaultGeneralConfigHost
			log.Warning("Failed to determine the FQDN; using '%s'.", c.General.Host)
		}
	}

	if c.Network.Method == "" {
		c.Network.Method = defaultNetworkConfigMethod
	}
	if c.Network.Method != "failover" && c.Network.Method != "loadbalance" {
		err = fmt.Errorf("The network method (/network/method) is not recognised: %s", c.Network.Method)
		return
	}

	if len(c.Network.Servers) == 0 {
		err = fmt.Errorf("No network servers were specified (/network/servers)")
		return
	}

	servers := make(map[string]bool)
	for _, server := range c.Network.Servers {
		if _, exists := servers[server]; exists {
			err = fmt.Errorf("The list of network servers (/network/servers) must be unique: %s appears multiple times", server)
			return
		}
		servers[server] = true
	}
	servers = nil

	if registrarFunc, ok := registeredTransports[c.Network.Transport]; ok {
		if c.Network.Factory, err = registrarFunc(c, "/network/", c.Network.Unused, c.Network.Transport); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("Unrecognised transport '%s'", c.Network.Transport)
		return
	}

	for k := range c.Files {
		if len(c.Files[k].Paths) == 0 {
			err = fmt.Errorf("No paths specified for /files[%d]/", k)
			return
		}

		if err = c.initStreamConfig(fmt.Sprintf("/files[%d]", k), &c.Files[k].Stream); err != nil {
			return
		}
	}

	if err = c.initStreamConfig("/stdin", &c.Stdin); err != nil {
		return
	}

	return
}

// initStreamConfig initialises a stream configuration by creating the necessary
// codec factories the harvesters will require
func (c *Config) initStreamConfig(path string, streamConfig *Stream) (err error) {
	if len(streamConfig.Codecs) == 0 {
		streamConfig.Codecs = []CodecStub{CodecStub{Name: defaultStreamConfigCodec}}
	}

	for i := 0; i < len(streamConfig.Codecs); i++ {
		codec := &streamConfig.Codecs[i]
		if registrarFunc, ok := registeredCodecs[codec.Name]; ok {
			if codec.Factory, err = registrarFunc(c, path, codec.Unused, codec.Name); err != nil {
				return
			}
		} else {
			return fmt.Errorf("Unrecognised codec '%s' for %s", codec.Name, path)
		}
	}

	// TODO: EDGE CASE: Event transmit length is uint32, if fields length is rediculous we will fail

	return nil
}
