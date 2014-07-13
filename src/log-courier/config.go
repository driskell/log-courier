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

package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/op/go-logging"
  "math"
  "os"
  "path/filepath"
  "reflect"
  "time"
)

const (
  default_GeneralConfig_PersistDir       string        = "."
  default_GeneralConfig_ProspectInterval time.Duration = 10 * time.Second
  default_GeneralConfig_SpoolSize        int64         = 1024
  default_GeneralConfig_SpoolTimeout     time.Duration = 5 * time.Second
  default_GeneralConfig_LogLevel         logging.Level = logging.INFO
  default_NetworkConfig_Timeout          time.Duration = 15 * time.Second
  default_NetworkConfig_Reconnect        time.Duration = 1 * time.Second
  default_FileConfig_DeadTime            int64         = 86400
)

type Config struct {
  General  GeneralConfig `json:"general"`
  Network  NetworkConfig `json:"network"`
  Files    []FileConfig  `json:"files"`
  Includes []string      `json:"includes"`
}

type GeneralConfig struct {
  PersistDir       string        `json:"persist directory"`
  ProspectInterval time.Duration `json:"prospect interval"`
  SpoolSize        int64         `json:"spool size"`
  SpoolTimeout     time.Duration `json:"spool timeout"`
  LogLevel         logging.Level `json:"log level"`
}

var NewTransportZmqFactoryIfAvailable func(string, map[string]interface{}) (TransportFactory, error)

type NetworkConfig struct {
  Transport string `json:"transport"`
  transport TransportFactory
  Servers   []string      `json:"servers"`
  Timeout   time.Duration `json:"timeout"`
  Reconnect time.Duration `json:"reconnect"`
  Unused    map[string]interface{}
}

type CodecConfigStub struct {
  Name   string `json:"name"`
  Unused map[string]interface{}
}

type FileConfig struct {
  Paths    []string               `json:"paths"`
  Fields   map[string]interface{} `json:"fields"`
  Codec    CodecConfigStub        `json:"codec"`
  codec    CodecFactory
  DeadTime time.Duration `json:"dead time"`
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

// TODO: Change (config *Config) to (c *Config) - not done yet to prevent
//       feature merge conflicts
func (config *Config) Load(path string) (err error) {
  var data *bytes.Buffer

  // Read the main config file
  if data, err = config.loadFile(path); err != nil {
    return
  }

  // Pull the entire structure into raw_config
  raw_config := make(map[string]interface{})
  if err = json.Unmarshal(data.Bytes(), &raw_config); err != nil {
    return
  }

  // Fill in defaults where the zero-value is a valid setting
  config.General.LogLevel = default_GeneralConfig_LogLevel

  // Populate configuration - reporting errors on spelling mistakes etc.
  if err = PopulateConfig(config, "/", raw_config); err != nil {
    return
  }

  // Iterate includes
  for _, glob := range config.Includes {
    // Glob the path
    var matches []string
    if matches, err = filepath.Glob(glob); err != nil {
      return
    }

    for _, include := range matches {
      // Read the include
      if data, err = config.loadFile(include); err != nil {
        return
      }

      // Pull the structure into raw_include
      raw_include := make([]interface{}, 0)
      if err = json.Unmarshal(data.Bytes(), &raw_include); err != nil {
        return
      }

      // Append to configuration
      if err = PopulateConfigSlice(reflect.ValueOf(config).Elem().FieldByName("Files"), fmt.Sprintf("%s/", include), raw_include); err != nil {
        return
      }
    }
  }

  // Fill in defaults for GeneralConfig
  if config.General.PersistDir == "" {
    config.General.PersistDir = default_GeneralConfig_PersistDir
  }
  if config.General.ProspectInterval == time.Duration(0) {
    config.General.ProspectInterval = default_GeneralConfig_ProspectInterval
  }
  if config.General.SpoolSize == 0 {
    config.General.SpoolSize = default_GeneralConfig_SpoolSize
  }
  if config.General.SpoolTimeout == time.Duration(0) {
    config.General.SpoolTimeout = default_GeneralConfig_SpoolTimeout
  }

  // Process through NetworkConfig
  if config.Network.Transport == "" {
    config.Network.Transport = "tls"
  }
  transport_name := config.Network.Transport

  var factory TransportFactory
  if factory, err = NewTransportFactory("/network/", transport_name, config.Network.Unused); err == nil {
    if factory != nil {
      config.Network.transport = factory
    } else {
      err = fmt.Errorf("Unrecognised transport '%s'", transport_name)
      return
    }
  } else {
    return
  }

  if config.Network.Timeout == time.Duration(0) {
    config.Network.Timeout = default_NetworkConfig_Timeout
  }
  if config.Network.Reconnect == time.Duration(0) {
    config.Network.Reconnect = default_NetworkConfig_Reconnect
  }

  for k := range config.Files {
    if config.Files[k].Codec.Name == "" {
      config.Files[k].Codec.Name = "plain"
    }
    codec_name := config.Files[k].Codec.Name

    var factory CodecFactory
    if factory, err = NewCodecFactory(fmt.Sprintf("/files[%d]/codec/", k), codec_name, config.Files[k].Codec.Unused); err == nil {
      if factory != nil {
        config.Files[k].codec = factory
      } else {
        err = fmt.Errorf("Unrecognised codec '%s'", codec_name)
        return
      }
    } else {
      return
    }

    if config.Files[k].DeadTime == time.Duration(0) {
      config.Files[k].DeadTime = time.Duration(default_FileConfig_DeadTime) * time.Second
    }
  }

  return
}

// TODO: The below to be combined into (c *Config) - not done yet to prevent
//       conflicts during pending feature merges

// TODO: This should be pushed to a wrapper or module
//       It populated dynamic configuration, automatically converting time.Duration etc.
//       Any config entries not found in the structure are moved to an "Unused" field if it exists
//       or an error is reported if "Unused" is not available
//       We can then take the unused configuration dynamically at runtime based on another value
func PopulateConfig(config interface{}, config_path string, raw_config map[string]interface{}) (err error) {
  vconfig := reflect.ValueOf(config).Elem()
  for i := 0; i < vconfig.NumField(); i++ {
    field := vconfig.Field(i)
    if !field.CanSet() {
      continue
    }
    fieldtype := vconfig.Type().Field(i)
    name := fieldtype.Name
    if name == "Unused" {
      continue
    }
    tag := fieldtype.Tag.Get("json")
    if tag == "" {
      tag = name
    }
    if _, ok := raw_config[tag]; ok {
      if field.Kind() == reflect.Struct {
        if reflect.TypeOf(raw_config[tag]).Kind() == reflect.Map {
          if err = PopulateConfig(field.Addr().Interface(), fmt.Sprintf("%s%s/", config_path, tag), raw_config[tag].(map[string]interface{})); err != nil {
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
        if err = PopulateConfigSlice(field, fmt.Sprintf("%s%s/", config_path, tag), raw_config[tag].([]interface{})); err != nil {
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
        if value.Type().AssignableTo(vduration.Type()) {
          duration = value.Float()
          if duration < math.MinInt64 || duration > math.MaxInt64 {
            err = fmt.Errorf("Option %s%s is not a valid numeric or string duration", config_path, tag)
            return
          }
          field.Set(reflect.ValueOf(time.Duration(int64(duration)) * time.Second))
        } else if value.Kind() == reflect.String {
          var tduration time.Duration
          if tduration, err = time.ParseDuration(value.String()); err != nil {
            err = fmt.Errorf("Option %s%s is not a valid numeric or string duration: %s", config_path, tag, err)
            return
          }
          field.Set(reflect.ValueOf(tduration))
        } else {
          err = fmt.Errorf("Option %s%s must be a valid numeric or string duration", config_path, tag)
          return
        }
      } else if field.Type().String() == "logging.Level" {
        if value.Kind() == reflect.String {
          var llevel logging.Level
          if llevel, err = logging.LogLevel(value.String()); err != nil {
            err = fmt.Errorf("Option %s%s is not a valid log level (critical, error, warning, notice, info, debug)", config_path, tag)
            return
          }
          field.Set(reflect.ValueOf(llevel))
        } else {
          err = fmt.Errorf("Option %s%s must be a valid log level (critical, error, warning, notice, info, debug)", config_path, tag)
          return
        }
      } else {
        err = fmt.Errorf("Option %s%s must be %s or similar", config_path, tag, field.Type())
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
  return ReportUnusedConfig(config_path, raw_config)
}

func PopulateConfigSlice(field reflect.Value, config_path string, raw_config []interface{}) (err error) {
  elemtype := field.Type().Elem()
  if elemtype.Kind() == reflect.Struct {
    for j := 0; j < len(raw_config); j++ {
      item := reflect.New(elemtype)
      if err = PopulateConfig(item.Interface(), fmt.Sprintf("%s[%d]/", config_path, j), raw_config[j].(map[string]interface{})); err != nil {
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

func ReportUnusedConfig(config_path string, raw_config map[string]interface{}) (err error) {
  for k := range raw_config {
    err = fmt.Errorf("Option %s%s is not available", config_path, k)
    return
  }
  return
}
