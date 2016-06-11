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

	"gopkg.in/op/go-logging.v1"
)

var (
	// DefaultConfigurationFile is a path to the default configuration file to
	// load, this can be changed during init()
	DefaultConfigurationFile = ""

	// DefaultGeneralPersistDir is a path to the default directory to store
	DefaultGeneralPersistDir = ""
)

const (
	defaultGeneralHost               string        = "localhost.localdomain"
	defaultGeneralLogLevel           logging.Level = logging.INFO
	defaultGeneralLogStdout          bool          = true
	defaultGeneralLogSyslog          bool          = false
	defaultGeneralLineBufferBytes    int64         = 16384
	defaultGeneralMaxLineBytes       int64         = 1048576
	defaultGeneralProspectInterval   time.Duration = 10 * time.Second
	defaultGeneralSpoolMaxBytes      int64         = 10485760
	defaultGeneralSpoolSize          int64         = 1024
	defaultGeneralSpoolTimeout       time.Duration = 5 * time.Second
	defaultNetworkBackoff            time.Duration = 5 * time.Second
	defaultNetworkBackoffMax         time.Duration = 300 * time.Second
	defaultNetworkMaxPendingPayloads int64         = 10
	defaultNetworkMethod             string        = "random"
	defaultNetworkRfc2782Service     string        = "courier"
	defaultNetworkRfc2782Srv         bool          = true
	defaultNetworkTimeout            time.Duration = 15 * time.Second
	defaultNetworkTransport          string        = "tls"
	defaultStreamAddHostField        bool          = true
	defaultStreamAddOffsetField      bool          = true
	defaultStreamAddPathField        bool          = true
	defaultStreamAddTimezoneField    bool          = false
	defaultStreamCodec               string        = "plain"
	defaultStreamDeadTime            time.Duration = 1 * time.Hour
)

// Section is implemented by external config structures that will be
// registered with the config package
type Section interface {
	Validate() error
}

// SectionCreator creates new Section structures
type SectionCreator func() Section

// registeredSectionCreators contains a list of registered external Section
// creators that should be processed in all new Config structures
var registeredSectionCreators = make(map[string]SectionCreator)

// General holds the general configuration
type General struct {
	GlobalFields     map[string]interface{} `config:"global fields"`
	Host             string                 `config:"host"`
	LineBufferBytes  int64                  `config:"line buffer bytes"`
	LogFile          string                 `config:"log file"`
	LogLevel         logging.Level          `config:"log level"`
	LogStdout        bool                   `config:"log stdout"`
	LogSyslog        bool                   `config:"log syslog"`
	MaxLineBytes     int64                  `config:"max line bytes"`
	PersistDir       string                 `config:"persist directory"`
	ProspectInterval time.Duration          `config:"prospect interval"`
	SpoolSize        int64                  `config:"spool size"`
	SpoolMaxBytes    int64                  `config:"spool max bytes"`
	SpoolTimeout     time.Duration          `config:"spool timeout"`
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

// Network holds network related configuration
type Network struct {
	Factory interface{}

	Backoff            time.Duration `config:"failure backoff"`
	BackoffMax         time.Duration `config:"failure backoff max"`
	MaxPendingPayloads int64         `config:"max pending payloads"`
	Method             string        `config:"method"`
	Rfc2782Service     string        `config:"rfc 2782 service"`
	Rfc2782Srv         bool          `config:"rfc 2782 srv"`
	Servers            []string      `config:"servers"`
	Timeout            time.Duration `config:"timeout"`
	Transport          string        `config:"transport"`

	Unused map[string]interface{}
}

// InitDefaults initiases default values for the network configuration
func (nc *Network) InitDefaults() {
	nc.Backoff = defaultNetworkBackoff
	nc.BackoffMax = defaultNetworkBackoffMax
	nc.MaxPendingPayloads = defaultNetworkMaxPendingPayloads
	nc.Method = defaultNetworkMethod
	nc.Rfc2782Service = defaultNetworkRfc2782Service
	nc.Rfc2782Srv = defaultNetworkRfc2782Srv
	nc.Timeout = defaultNetworkTimeout
	nc.Transport = defaultNetworkTransport
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
	AddHostField     bool                   `config:"add host field"`
	AddOffsetField   bool                   `config:"add offset field"`
	AddPathField     bool                   `config:"add path field"`
	AddTimezoneField bool                   `config:"add timezone field"`
	Codecs           []CodecStub            `config:"codecs"`
	DeadTime         time.Duration          `config:"dead time"`
	Fields           map[string]interface{} `config:"fields"`
}

// InitDefaults initialises the default configuration for a log stream
func (sc *Stream) InitDefaults() {
	sc.AddHostField = defaultStreamAddHostField
	sc.AddOffsetField = defaultStreamAddOffsetField
	sc.AddPathField = defaultStreamAddPathField
	sc.AddTimezoneField = defaultStreamAddTimezoneField
	sc.DeadTime = defaultStreamDeadTime
}

// File holds the configuration for a set of paths that share the same stream
// configuration
type File struct {
	Paths  []string `config:"paths"`
	Stream `config:",embed"`
}

// Config holds all the configuration for Log Courier
type Config struct {
	Files    []File   `config:"files"`
	General  General  `config:"general"`
	Includes []string `config:"includes"`
	Network  Network  `config:"network"`
	Stdin    Stream   `config:"stdin"`
	// Dynamic sections
	// TODO: All top level sections to use this
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
// If initFactories is false, factories (such as codec names or transport
// names) are not initialised so they do not need to be built in
func (c *Config) Load(path string, initFactories bool) (err error) {
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
		ret, hostErr := os.Hostname()
		if hostErr == nil {
			c.General.Host = ret
		} else {
			c.General.Host = defaultGeneralHost
			log.Warning("Failed to determine the FQDN: %s", hostErr)
			log.Warning("Falling back to using default hostname: %s", c.General.Host)
		}
	}

	// Ensure all GlobalFields are map[string]interface{}
	if err = c.fixMapKeys("/general/global fields", c.General.GlobalFields); err != nil {
		return
	}

	// TODO: Network method factory in publisher
	if c.Network.Method == "" {
		c.Network.Method = defaultNetworkMethod
	}
	if c.Network.Method != "random" && c.Network.Method != "failover" && c.Network.Method != "loadbalance" {
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

	if initFactories {
		if registrarFunc, ok := registeredTransports[c.Network.Transport]; ok {
			if c.Network.Factory, err = registrarFunc(c, "/network/", c.Network.Unused, c.Network.Transport); err != nil {
				return
			}
		} else {
			err = fmt.Errorf("Unrecognised transport '%s'", c.Network.Transport)
			return
		}
	}

	for k := range c.Files {
		if len(c.Files[k].Paths) == 0 {
			err = fmt.Errorf("No paths specified for /files[%d]/", k)
			return
		}

		if err = c.initStreamConfig(fmt.Sprintf("/files[%d]", k), &c.Files[k].Stream, initFactories); err != nil {
			return
		}
	}

	if err = c.initStreamConfig("/stdin", &c.Stdin, initFactories); err != nil {
		return
	}

	// Validate the registered configurables
	for _, section := range c.Sections {
		if err = section.Validate(); err != nil {
			return
		}
	}

	return
}

// initStreamConfig initialises a stream configuration by creating the necessary
// codec factories the harvesters will require
func (c *Config) initStreamConfig(path string, streamConfig *Stream, initFactories bool) (err error) {
	if !initFactories {
		// Currently only codec factory is initialised, so skip if we're not doing that
		return nil
	}

	if len(streamConfig.Codecs) == 0 {
		streamConfig.Codecs = []CodecStub{CodecStub{Name: defaultStreamCodec}}
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

	// Ensure all Fields are map[string]interface{}
	if err = c.fixMapKeys(path+"/fields", streamConfig.Fields); err != nil {
		return
	}

	// TODO: EDGE CASE: Event transmit length is uint32, if fields length is rediculous we will fail

	return nil
}

// Get returns the requested dynamic configuration entry
func (c *Config) Get(name string) interface{} {
	ret, ok := c.Sections[name]
	if !ok {
		return nil
	}

	return ret
}

// fixMapKeys converts any map entries where the keys are interface{} values
// into map entries where the key is a string. It returns an error if any key is
// found that is not a string.
// This is important as json.Encode will not encode a map where the keys are not
// concrete strings.
func (c *Config) fixMapKeys(path string, value map[string]interface{}) error {
	for k, v := range value {
		switch vt := v.(type) {
		case map[string]interface{}:
			if err := c.fixMapKeys(path+"/"+k, vt); err != nil {
				return err
			}
		case map[interface{}]interface{}:
			fixedValue, err := c.fixMapInterfaceKeys(path+"/"+k, vt)
			if err != nil {
				return err
			}

			value[k] = fixedValue
		}
	}

	return nil
}

func (c *Config) fixMapInterfaceKeys(path string, value map[interface{}]interface{}) (map[string]interface{}, error) {
	fixedMap := make(map[string]interface{})

	for k, v := range value {
		ks, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("Invalid non-string key at %s", path)
		}

		switch vt := v.(type) {
		case map[string]interface{}:
			if err := c.fixMapKeys(path+"/"+ks, vt); err != nil {
				return nil, err
			}

			fixedMap[ks] = vt
		case map[interface{}]interface{}:
			fixedValue, err := c.fixMapInterfaceKeys(path+"/"+ks, vt)
			if err != nil {
				return nil, err
			}

			fixedMap[ks] = fixedValue
		default:
			fixedMap[ks] = vt
		}
	}

	return fixedMap, nil
}

// RegisterConfigSection registers a new Section creator which will be used to
// create new sections that will be available via Get() in all created Config
// structures
func RegisterConfigSection(name string, creator SectionCreator) {
	registeredSectionCreators[name] = creator
}
