package main

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "log"
  "os"
  "time"
)

const default_NetworkConfig_Timeout int64 = 15
const default_NetworkConfig_Reconnect int64 = 1

const default_FileConfig_DeadTime string = "24h"

type Config struct {
  Network NetworkConfig `json:network`
  Files   []FileConfig  `json:files`
}

type NetworkConfig struct {
  Servers        []string `json:servers`
  SSLCertificate string   `json:"ssl certificate"`
  SSLKey         string   `json:"ssl key"`
  SSLCA          string   `json:"ssl ca"`
  Timeout        int64    `json:timeout`
  timeout        time.Duration
  Reconnect      int64 `json:reconnect`
  reconnect      time.Duration
}

type FileConfig struct {
  Paths    []string               `json:paths`
  Fields   map[string]string      `json:fields`
  Codec    map[string]interface{} `json:codec`
  codec    CodecFactory
  DeadTime string `json:"dead time"`
  deadtime time.Duration
}

func LoadConfig(path string) (config *Config, err error) {
  config_file, err := os.Open(path)
  if err != nil {
    log.Printf("Failed to open config file '%s': %s\n", path, err)
    return
  }
  defer func() { config_file.Close() }()

  fi, err := config_file.Stat()
  if err != nil {
    log.Printf("Stat failed for config file: %s\n", err)
    return
  }
  if fi.Size() > (10 << 20) {
    err = errors.New("Config file too large?")
    log.Printf("Config file too large? Aborting, just in case. '%s' is %d bytes\n", path, fi)
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
      log.Printf("Failed reading config file: %s\n", err)
      return nil, err
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
          s = p
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
    log.Printf("%s\n", stripped.String())
  }

  config = &Config{}
  err = json.Unmarshal(stripped.Bytes(), config)
  if err != nil {
    log.Printf("Failed unmarshalling json: %s\n", err)
    return
  }

  if config.Network.Timeout == 0 {
    config.Network.Timeout = default_NetworkConfig_Timeout
  }

  config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second

  if config.Network.Reconnect == 0 {
    config.Network.Reconnect = default_NetworkConfig_Reconnect
  }

  config.Network.reconnect = time.Duration(config.Network.Reconnect) * time.Second

  for k := range config.Files {
    for _, path := range config.Files[k].Paths {
      log.Printf("%d: %s", k, path)
    }
    if config.Files[k].Codec != nil {
      var ok bool
      config.Files[k].Codec["name"], ok = config.Files[k].Codec["name"].(string)
      if !ok {
        err = errors.New("The name of the codec must be specified.")
        log.Printf(fmt.Sprint(err))
        return
      }
    } else {
      config.Files[k].Codec = make(map[string]interface{}, 1)
      config.Files[k].Codec["name"] = "plain"
    }

    if config.Files[k].Codec["name"] == "" || config.Files[k].Codec["name"] == "plain" {
      config.Files[k].codec, err = CreateCodecPlainFactory(config.Files[k].Codec)
    } else if config.Files[k].Codec["name"] == "multiline" {
      config.Files[k].codec, err = CreateCodecMultilineFactory(config.Files[k].Codec)
    } else {
      err = errors.New(fmt.Sprintf("Unrecognised codec '%s'. Please check your configuration.\n", config.Files[k].Codec["name"]))
      log.Printf(fmt.Sprint(err))
      return
    }
    if err != nil {
      log.Printf(fmt.Sprint(err))
      return
    }

    if config.Files[k].DeadTime == "" {
      config.Files[k].DeadTime = default_FileConfig_DeadTime
    }

    config.Files[k].deadtime, err = time.ParseDuration(config.Files[k].DeadTime)
    if err != nil {
      log.Printf("Failed to parse dead time duration '%s'. Error was: %s\n", config.Files[k].DeadTime, err)
      return
    }
  }

  return
}
