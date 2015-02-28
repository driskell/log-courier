/*
 * Copyright 2014 Jason Woods.
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

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/op/go-logging"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

const (
	default_GeneralConfig_AdminEnabled       bool          = false
	default_GeneralConfig_AdminBind          string        = "tcp:127.0.0.1:1234"
	default_GeneralConfig_PersistDir         string        = "."
	default_GeneralConfig_ProspectInterval   time.Duration = 10 * time.Second
	default_GeneralConfig_SpoolSize          int64         = 1024
	default_GeneralConfig_SpoolMaxBytes      int64         = 10485760
	default_GeneralConfig_SpoolTimeout       time.Duration = 5 * time.Second
	default_GeneralConfig_LineBufferBytes    int64         = 16384
	default_GeneralConfig_MaxLineBytes       int64         = 1048576
	default_GeneralConfig_LogLevel           logging.Level = logging.INFO
	default_GeneralConfig_LogStdout          bool          = true
	default_GeneralConfig_LogSyslog          bool          = false
	default_NetworkConfig_Transport          string        = "tls"
	default_NetworkConfig_Rfc2782Srv         bool          = true
	default_NetworkConfig_Rfc2782Service     string        = "courier"
	default_NetworkConfig_Timeout            time.Duration = 15 * time.Second
	default_NetworkConfig_Reconnect          time.Duration = 1 * time.Second
	default_NetworkConfig_MaxPendingPayloads int64         = 10
	default_StreamConfig_Codec               string        = "plain"
	default_StreamConfig_DeadTime            int64         = 86400
)

var (
	default_GeneralConfig_Host string = "localhost.localdomain"
)

type Config struct {
	General  GeneralConfig `config:"general"`
	Network  NetworkConfig `config:"network"`
	Files    []FileConfig  `config:"files"`
	Includes []string      `config:"includes"`
	Stdin    StreamConfig  `config:"stdin"`
}

type GeneralConfig struct {
	AdminEnabled     bool          `config:"admin enabled"`
	AdminBind        string        `config:"admin listen address"`
	PersistDir       string        `config:"persist directory"`
	ProspectInterval time.Duration `config:"prospect interval"`
	SpoolSize        int64         `config:"spool size"`
	SpoolMaxBytes    int64         `config:"spool max bytes"`
	SpoolTimeout     time.Duration `config:"spool timeout"`
	LineBufferBytes  int64         `config:"line buffer bytes"`
	MaxLineBytes     int64         `config:"max line bytes"`
	LogLevel         logging.Level `config:"log level"`
	LogStdout        bool          `config:"log stdout"`
	LogSyslog        bool          `config:"log syslog"`
	LogFile          string        `config:"log file"`
	Host             string        `config:"host"`
}

type NetworkConfig struct {
	Transport          string        `config:"transport"`
	Servers            []string      `config:"servers"`
	Rfc2782Srv         bool          `config:"rfc 2782 srv"`
	Rfc2782Service     string        `config:"rfc 2782 service"`
	Timeout            time.Duration `config:"timeout"`
	Reconnect          time.Duration `config:"reconnect"`
	MaxPendingPayloads int64         `config:"max pending payloads"`

	Unused           map[string]interface{}
	TransportFactory TransportFactory
}

type CodecConfigStub struct {
	Name string `config:"name"`

	Unused map[string]interface{}
}

type StreamConfig struct {
	Fields   map[string]interface{} `config:"fields"`
	Codec    CodecConfigStub        `config:"codec"`
	DeadTime time.Duration          `config:"dead time"`

	CodecFactory CodecFactory
}

type FileConfig struct {
	Paths []string `config:"paths"`

	StreamConfig `config:",embed"`
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) loadFile(path string) (stripped *bytes.Buffer, err error) {
	stripped = new(bytes.Buffer)

	file, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("Failed to open config file: %s", err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		err = fmt.Errorf("Stat failed for config file: %s", err)
		return
	}
	if stat.Size() > (10 << 20) {
		err = fmt.Errorf("Config file too large (%s)", stat.Size())
		return
	}

	// Strip comments and read config into stripped
	var s, p, state int
	{
		// Pull the config file into memory
		buffer := make([]byte, stat.Size())
		_, err = file.Read(buffer)
		if err != nil {
			return
		}

		for p < len(buffer) {
			b := buffer[p]
			if state == 0 {
				// Main body
				if b == '"' {
					state = 1
				} else if b == '\'' {
					state = 2
				} else if b == '#' {
					state = 3
					stripped.Write(buffer[s:p])
				} else if b == '/' {
					state = 4
				}
			} else if state == 1 {
				// Double-quoted string
				if b == '\\' {
					state = 5
				} else if b == '"' {
					state = 0
				}
			} else if state == 2 {
				// Single-quoted string
				if b == '\\' {
					state = 6
				} else if b == '\'' {
					state = 0
				}
			} else if state == 3 {
				// End of line comment (#)
				if b == '\r' || b == '\n' {
					state = 0
					s = p + 1
				}
			} else if state == 4 {
				// Potential start of multiline comment
				if b == '*' {
					state = 7
					stripped.Write(buffer[s : p-1])
				} else {
					state = 0
				}
			} else if state == 5 {
				// Escape within double quote
				state = 1
			} else if state == 6 {
				// Escape within single quote
				state = 2
			} else if state == 7 {
				// Multiline comment (/**/)
				if b == '*' {
					state = 8
				}
			} else { // state == 8
				// Potential end of multiline comment
				if b == '/' {
					state = 0
					s = p + 1
				} else {
					state = 7
				}
			}
			p++
		}
		stripped.Write(buffer[s:p])
	}

	return
}

// Parse a *json.SyntaxError into a pretty error message
func (c *Config) parseSyntaxError(js []byte, err error) error {
	json_err, ok := err.(*json.SyntaxError)
	if !ok {
		return err
	}

	start := bytes.LastIndex(js[:json_err.Offset], []byte("\n"))+1
	end := bytes.Index(js[start:], []byte("\n"))
	if end >= 0 {
		end += start
	} else {
		end = len(js)
	}

	line, pos := bytes.Count(js[:start], []byte("\n")), int(json_err.Offset) - start - 1

	return fmt.Errorf("%s on line %d\n%s\n%s^", err, line, js[start:end], strings.Repeat(" ", pos))
}

// TODO: Config from a TOML? Maybe a custom one
func (c *Config) Load(path string) (err error) {
	var data *bytes.Buffer

	// Read the main config file
	if data, err = c.loadFile(path); err != nil {
		return
	}

	// Pull the entire structure into raw_config
	raw_config := make(map[string]interface{})
	if err = json.Unmarshal(data.Bytes(), &raw_config); err != nil {
		return c.parseSyntaxError(data.Bytes(), err)
	}

	// Fill in defaults where the zero-value is a valid setting
	c.General.AdminEnabled = default_GeneralConfig_AdminEnabled
	c.General.LogLevel = default_GeneralConfig_LogLevel
	c.General.LogStdout = default_GeneralConfig_LogStdout
	c.General.LogSyslog = default_GeneralConfig_LogSyslog
	c.Network.Rfc2782Srv = default_NetworkConfig_Rfc2782Srv

	// Populate configuration - reporting errors on spelling mistakes etc.
	if err = c.PopulateConfig(c, "/", raw_config); err != nil {
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
			if data, err = c.loadFile(include); err != nil {
				return
			}

			// Pull the structure into raw_include
			raw_include := make([]interface{}, 0)
			if err = json.Unmarshal(data.Bytes(), &raw_include); err != nil {
				return
			}

			// Append to configuration
			if err = c.populateConfigSlice(reflect.ValueOf(c).Elem().FieldByName("Files"), fmt.Sprintf("%s/", include), raw_include); err != nil {
				return
			}
		}
	}

	if c.General.AdminBind == "" {
		c.General.AdminBind = default_GeneralConfig_AdminBind
	}

	if c.General.PersistDir == "" {
		c.General.PersistDir = default_GeneralConfig_PersistDir
	}

	if c.General.ProspectInterval == time.Duration(0) {
		c.General.ProspectInterval = default_GeneralConfig_ProspectInterval
	}

	if c.General.SpoolSize == 0 {
		c.General.SpoolSize = default_GeneralConfig_SpoolSize
	}

	// Enforce maximum of 2 GB since event transmit length is uint32
	if c.General.SpoolMaxBytes == 0 {
		c.General.SpoolMaxBytes = default_GeneralConfig_SpoolMaxBytes
	}
	if c.General.SpoolMaxBytes > 2*1024*1024*1024 {
		err = fmt.Errorf("/general/spool max bytes can not be greater than 2 GiB")
		return
	}

	if c.General.SpoolTimeout == time.Duration(0) {
		c.General.SpoolTimeout = default_GeneralConfig_SpoolTimeout
	}

	if c.General.LineBufferBytes == 0 {
		c.General.LineBufferBytes = default_GeneralConfig_LineBufferBytes
	}
	if c.General.LineBufferBytes < 1 {
		err = fmt.Errorf("/general/line buffer bytes must be greater than 1")
		return
	}

	// Max line bytes can not be larger than spool max bytes
	if c.General.MaxLineBytes == 0 {
		c.General.MaxLineBytes = default_GeneralConfig_MaxLineBytes
	}
	if c.General.MaxLineBytes > c.General.SpoolMaxBytes {
		err = fmt.Errorf("/general/max line bytes can not be greater than /general/spool max bytes")
		return
	}

	if c.General.Host == "" {
		ret, err := os.Hostname()
		if err == nil {
			c.General.Host = ret
		} else {
			c.General.Host = default_GeneralConfig_Host
			log.Warning("Failed to determine the FQDN; using '%s'.", c.General.Host)
		}
	}

	if c.Network.Transport == "" {
		c.Network.Transport = default_NetworkConfig_Transport
	}

	if registrar_func, ok := registered_Transports[c.Network.Transport]; ok {
		if c.Network.TransportFactory, err = registrar_func(c, "/network/", c.Network.Unused, c.Network.Transport); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("Unrecognised transport '%s'", c.Network.Transport)
		return
	}

	if c.Network.Rfc2782Service == "" {
		c.Network.Rfc2782Service = default_NetworkConfig_Rfc2782Service
	}
	if c.Network.Timeout == time.Duration(0) {
		c.Network.Timeout = default_NetworkConfig_Timeout
	}
	if c.Network.Reconnect == time.Duration(0) {
		c.Network.Reconnect = default_NetworkConfig_Reconnect
	}
	if c.Network.MaxPendingPayloads == 0 {
		c.Network.MaxPendingPayloads = default_NetworkConfig_MaxPendingPayloads
	}

	for k := range c.Files {
		if err = c.initStreamConfig(fmt.Sprintf("/files[%d]/codec/", k), &c.Files[k].StreamConfig); err != nil {
			return
		}
	}

	if err = c.initStreamConfig("/stdin", &c.Stdin); err != nil {
		return
	}

	return
}

func (c *Config) initStreamConfig(path string, stream_config *StreamConfig) (err error) {
	if stream_config.Codec.Name == "" {
		stream_config.Codec.Name = default_StreamConfig_Codec
	}

	if registrar_func, ok := registered_Codecs[stream_config.Codec.Name]; ok {
		if stream_config.CodecFactory, err = registrar_func(c, path, stream_config.Codec.Unused, stream_config.Codec.Name); err != nil {
			return
		}
	} else {
		return fmt.Errorf("Unrecognised codec '%s' for %s", stream_config.Codec.Name, path)
	}

	if stream_config.DeadTime == time.Duration(0) {
		stream_config.DeadTime = time.Duration(default_StreamConfig_DeadTime) * time.Second
	}

	// TODO: EDGE CASE: Event transmit length is uint32, if fields length is rediculous we will fail

	return nil
}

// TODO: This should be pushed to a wrapper or module
//       It populated dynamic configuration, automatically converting time.Duration etc.
//       Any config entries not found in the structure are moved to an "Unused" field if it exists
//       or an error is reported if "Unused" is not available
//       We can then take the unused configuration dynamically at runtime based on another value
func (c *Config) PopulateConfig(config interface{}, config_path string, raw_config map[string]interface{}) (err error) {
	vconfig := reflect.ValueOf(config).Elem()
FieldLoop:
	for i := 0; i < vconfig.NumField(); i++ {
		field := vconfig.Field(i)
		if !field.CanSet() {
			continue
		}
		fieldtype := vconfig.Type().Field(i)
		tag := fieldtype.Tag.Get("config")
		mods := strings.Split(tag, ",")
		tag = mods[0]
		mods = mods[1:]
		for _, mod := range mods {
			if mod == "embed" && field.Kind() == reflect.Struct {
				if err = c.PopulateConfig(field.Addr().Interface(), config_path, raw_config); err != nil {
					return
				}
				continue FieldLoop
			}
		}
		if tag == "" {
			continue
		}
		if _, ok := raw_config[tag]; ok {
			if field.Kind() == reflect.Struct {
				if reflect.TypeOf(raw_config[tag]).Kind() == reflect.Map {
					if err = c.PopulateConfig(field.Addr().Interface(), fmt.Sprintf("%s%s/", config_path, tag), raw_config[tag].(map[string]interface{})); err != nil {
						return
					}
					delete(raw_config, tag)
					continue
				} else {
					err = fmt.Errorf("Option %s%s must be a hash", config_path, tag)
					return
				}
			}
			value := reflect.ValueOf(raw_config[tag])
			if value.Type().AssignableTo(field.Type()) {
				field.Set(value)
			} else if value.Kind() == reflect.Slice && field.Kind() == reflect.Slice {
				if err = c.populateConfigSlice(field, fmt.Sprintf("%s%s/", config_path, tag), raw_config[tag].([]interface{})); err != nil {
					return
				}
			} else if value.Kind() == reflect.Map && field.Kind() == reflect.Map {
				if field.IsNil() {
					field.Set(reflect.MakeMap(field.Type()))
				}
				for _, j := range value.MapKeys() {
					item := value.MapIndex(j)
					if item.Elem().Type().AssignableTo(field.Type().Elem()) {
						field.SetMapIndex(j, item.Elem())
					} else {
						err = fmt.Errorf("Option %s%s[%s] must be %s or similar", config_path, tag, j, field.Type().Elem())
						return
					}
				}
			} else if field.Type().String() == "time.Duration" {
				var duration float64
				vduration := reflect.ValueOf(duration)
				fail := true
				if value.Type().AssignableTo(vduration.Type()) {
					duration = value.Float()
					if duration >= math.MinInt64 && duration <= math.MaxInt64 {
						field.Set(reflect.ValueOf(time.Duration(int64(duration)) * time.Second))
						fail = false
					}
				} else if value.Kind() == reflect.String {
					var tduration time.Duration
					if tduration, err = time.ParseDuration(value.String()); err == nil {
						field.Set(reflect.ValueOf(tduration))
						fail = false
					}
				}
				if fail {
					err = fmt.Errorf("Option %s%s must be a valid numeric or string duration", config_path, tag)
					return
				}
			} else if field.Type().String() == "logging.Level" {
				fail := true
				if value.Kind() == reflect.String {
					var llevel logging.Level
					if llevel, err = logging.LogLevel(value.String()); err == nil {
						fail = false
						field.Set(reflect.ValueOf(llevel))
					}
				}
				if fail {
					err = fmt.Errorf("Option %s%s is not a valid log level (critical, error, warning, notice, info, debug)", config_path, tag)
					return
				}
			} else if field.Kind() == reflect.Int64 {
				fail := true
				if value.Kind() == reflect.Float64 {
					number := value.Float()
					if math.Floor(number) == number {
						fail = false
						field.Set(reflect.ValueOf(int64(number)))
					}
				}
				if fail {
					err = fmt.Errorf("Option %s%s is not a valid integer", config_path, tag, field.Type())
					return
				}
			} else if field.Kind() == reflect.Int {
				fail := true
				if value.Kind() == reflect.Float64 {
					number := value.Float()
					if math.Floor(number) == number {
						fail = false
						field.Set(reflect.ValueOf(int(number)))
					}
				}
				if fail {
					err = fmt.Errorf("Option %s%s is not a valid integer", config_path, tag, field.Type())
					return
				}
			} else {
				err = fmt.Errorf("Option %s%s must be %s or similar (%s provided)", config_path, tag, field.Type(), value.Type())
				return
			}
			delete(raw_config, tag)
		}
	}
	if unused := vconfig.FieldByName("Unused"); unused.IsValid() {
		if unused.IsNil() {
			unused.Set(reflect.MakeMap(unused.Type()))
		}
		for k, v := range raw_config {
			unused.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
		return
	}
	return c.ReportUnusedConfig(config_path, raw_config)
}

func (c *Config) populateConfigSlice(field reflect.Value, config_path string, raw_config []interface{}) (err error) {
	elemtype := field.Type().Elem()
	if elemtype.Kind() == reflect.Struct {
		for j := 0; j < len(raw_config); j++ {
			item := reflect.New(elemtype)
			if err = c.PopulateConfig(item.Interface(), fmt.Sprintf("%s[%d]/", config_path, j), raw_config[j].(map[string]interface{})); err != nil {
				return
			}
			field.Set(reflect.Append(field, item.Elem()))
		}
	} else {
		for j := 0; j < len(raw_config); j++ {
			field.Set(reflect.Append(field, reflect.ValueOf(raw_config[j])))
		}
	}
	return
}

func (c *Config) ReportUnusedConfig(config_path string, raw_config map[string]interface{}) (err error) {
	for k := range raw_config {
		err = fmt.Errorf("Option %s%s is not available", config_path, k)
		return
	}
	return
}
