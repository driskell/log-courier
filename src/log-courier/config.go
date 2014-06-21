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
  "errors"
  "fmt"
  "math"
  "os"
  "reflect"
  "time"
)

const (
  default_GeneralConfig_PersistDir       string = "."
  default_GeneralConfig_ProspectInterval int64  = 10
  default_NetworkConfig_Timeout          int64  = 15
  default_NetworkConfig_Reconnect        int64  = 1
  default_FileConfig_DeadTime            int64  = 86400
)

type Config struct {
  General GeneralConfig `json:"general"`
  Network NetworkConfig `json:"network"`
  Files   []FileConfig  `json:"files"`
}

type GeneralConfig struct {
  PersistDir       string        `json:"persist directory"`
  ProspectInterval time.Duration `json:"prospect interval"`
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
  Paths    []string          `json:"paths"`
  Fields   map[string]string `json:"fields"`
  Codec    CodecConfigStub   `json:"codec"`
  codec    CodecFactory
  DeadTime time.Duration `json:"dead time"`
}

func LoadConfig(path string) (config *Config, err error) {
  config_file, err := os.Open(path)
  if err != nil {
    err = errors.New(fmt.Sprintf("Failed to open config file: %s", err))
    return
  }
  defer config_file.Close()

  fi, err := config_file.Stat()
  if err != nil {
    err = errors.New(fmt.Sprintf("Stat failed for config file: %s", err))
    return
  }
  if fi.Size() > (10 << 20) {
    err = errors.New(fmt.Sprintf("Config file too large (%s)", fi))
    return
  }

  // Strip comments and read config into stripped
  var s, p, state int
  var stripped bytes.Buffer
  {
    // Pull the config file into memory
    buffer := make([]byte, fi.Size())
    _, err = config_file.Read(buffer)
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

  // Pull the entire structure into config
  raw_config := make(map[string]interface{})
  err = json.Unmarshal(stripped.Bytes(), &raw_config)
  if err != nil {
    return
  }

  // Populate configuration - reporting errors on spelling mistakes etc
  config = &Config{}
  if err = PopulateConfig(config, "/", raw_config); err != nil {
    return
  }

  // Fill in defaults for GeneralConfig
  if config.General.PersistDir == "" {
    config.General.PersistDir = default_GeneralConfig_PersistDir
  }
  if config.General.ProspectInterval == time.Duration(0) {
    config.General.ProspectInterval = time.Duration(default_GeneralConfig_ProspectInterval) * time.Second
  }

  // Process through NetworkConfig
  if config.Network.Transport == "" {
    config.Network.Transport = "tls"
  }
  transport_name := config.Network.Transport

  if transport_name == "tls" {
    if config.Network.transport, err = NewTransportTlsFactory("/network/transport/", config.Network.Unused); err != nil {
      return
    }
  } else if transport_name == "zmq" {
    // TODO: Either make ZMQ compilation mandatory or use a proper factory pattern
    if NewTransportZmqFactoryIfAvailable == nil {
      err = errors.New(fmt.Sprintf("This binary was not built with 'zmq' transport support"))
      return
    }
    if config.Network.transport, err = NewTransportZmqFactoryIfAvailable("/network/transport/", config.Network.Unused); err != nil {
      return
    }
  } else {
    err = errors.New(fmt.Sprintf("Unrecognised transport '%s'", transport_name))

    return
  }

  if config.Network.Timeout == time.Duration(0) {
    config.Network.Timeout = time.Duration(default_NetworkConfig_Timeout) * time.Second
  }

  if config.Network.Reconnect == time.Duration(0) {
    config.Network.Reconnect = time.Duration(default_NetworkConfig_Reconnect) * time.Second
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
        err = errors.New(fmt.Sprintf("Unrecognised codec '%s'", codec_name))
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
          err = errors.New(fmt.Sprintf("Option %s%s must be a hash", config_path, tag))
          return
        }
      }
      value := reflect.ValueOf(raw_config[tag])
      if value.Type().AssignableTo(field.Type()) {
        field.Set(value)
      } else if value.Kind() == reflect.Slice && field.Kind() == reflect.Slice {
        elemtype := field.Type().Elem()
        if elemtype.Kind() == reflect.Struct {
          for j := 0; j < value.Len(); j++ {
            item := reflect.New(elemtype)
            if err = PopulateConfig(item.Interface(), fmt.Sprintf("%s%s[%d]/", config_path, tag, j), value.Index(j).Elem().Interface().(map[string]interface{})); err != nil {
              return
            }
            field.Set(reflect.Append(field, item.Elem()))
          }
        } else {
          for j := 0; j < value.Len(); j++ {
            field.Set(reflect.Append(field, value.Index(j).Elem()))
          }
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
            err = errors.New(fmt.Sprintf("Option %s%s[%s] must be %s or similar", config_path, tag, j, field.Type().Elem()))
            return
          }
        }
      } else if field.Type().String() == "time.Duration" {
        var duration float64
        vduration := reflect.ValueOf(duration)
        if value.Type().AssignableTo(vduration.Type()) {
          duration = value.Float()
          if duration < math.MinInt64 || duration > math.MaxInt64 {
            err = errors.New(fmt.Sprintf("Option %s%s is not a valid numeric or string duration", config_path, tag))
            return
          }
          field.Set(reflect.ValueOf(time.Duration(int64(duration)) * time.Second))
        } else if value.Kind() == reflect.String {
          var tduration time.Duration
          if tduration, err = time.ParseDuration(value.String()); err != nil {
            err = errors.New(fmt.Sprintf("Option %s%s is not a valid numeric or string duration: %s", config_path, tag, err))
            return
          }
          field.Set(reflect.ValueOf(tduration))
        } else {
          err = errors.New(fmt.Sprintf("Option %s%s must be %s or similar", config_path, tag, vduration.Type()))
          return
        }
      } else {
        err = errors.New(fmt.Sprintf("Option %s%s must be %s or similar", config_path, tag, field.Type()))
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

func ReportUnusedConfig(config_path string, raw_config map[string]interface{}) (err error) {
  for k := range raw_config {
    err = errors.New(fmt.Sprintf("Option %s%s is not available", config_path, k))
    return
  }
  return
}
