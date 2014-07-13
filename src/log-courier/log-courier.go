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
  "flag"
  "fmt"
  "github.com/op/go-logging"
  stdlog "log"
  "os"
  "runtime/pprof"
  "sync"
  "time"
)

const Log_Courier_Version string = "0.11"

var log *logging.Logger

func init() {
  log = logging.MustGetLogger("")
}

func main() {
  logcourier := NewLogCourier()
  logcourier.Run()
}

type LogCourierPlatform interface {
  Init()
  ConfigureLogging([]logging.Backend)
}

type LogCourierMasterControl struct {
  signal chan interface{}
  sinks  map[*LogCourierControl]chan *Config
  group  sync.WaitGroup
}

func NewLogCourierMasterControl() *LogCourierMasterControl {
  return &LogCourierMasterControl{
    signal: make(chan interface{}),
    sinks:  make(map[*LogCourierControl]chan *Config),
  }
}

func (lcs *LogCourierMasterControl) Shutdown() {
  close(lcs.signal)
}

func (lcs *LogCourierMasterControl) SendConfig(config *Config) {
  for _, sink := range lcs.sinks {
    sink <- config
  }
}

func (lcs *LogCourierMasterControl) Register() *LogCourierControl {
  return lcs.register()
}

func (lcs *LogCourierMasterControl) RegisterWithRecvConfig() *LogCourierControl {
  ret := lcs.register()

  config_chan := make(chan *Config)
  lcs.sinks[ret] = config_chan
  ret.sink = config_chan

  return ret
}

func (lcs *LogCourierMasterControl) register() *LogCourierControl {
  lcs.group.Add(1)

  return &LogCourierControl{
    signal: lcs.signal,
    group:  &lcs.group,
  }
}

func (lcs *LogCourierMasterControl) Wait() {
  lcs.group.Wait()
}

type LogCourierControl struct {
  signal <-chan interface{}
  sink   <-chan *Config
  group  *sync.WaitGroup
}

func (lcs *LogCourierControl) ShutdownSignal() <-chan interface{} {
  return lcs.signal
}

func (lcs *LogCourierControl) RecvConfig() <-chan *Config {
  if lcs.sink == nil {
    panic("RecvConfig invalid: LogCourierControl was not registered with RegisterWithRecvConfig")
  }
  return lcs.sink
}

func (lcs *LogCourierControl) Done() {
  lcs.group.Done()
}

type LogCourier struct {
  control        *LogCourierMasterControl
  platform       LogCourierPlatform
  config         *Config
  shutdown_chan  chan os.Signal
  reload_chan    chan os.Signal
  config_file    string
  from_beginning bool
}

func NewLogCourier() *LogCourier {
  ret := &LogCourier{
    control:  NewLogCourierMasterControl(),
    platform: NewLogCourierPlatform(),
  }
  return ret
}

func (lc *LogCourier) Run() {
  lc.startUp()

  event_chan := make(chan *FileEvent, 16)
  publisher_chan := make(chan []*FileEvent, 1)

  log.Info("Starting pipeline")

  registrar := NewRegistrar(lc.config.General.PersistDir, lc.control)

  publisher, err := NewPublisher(&lc.config.Network, registrar, lc.control)
  if err != nil {
    log.Fatalf("Failed to initialise: %s", err)
  }

  spooler := NewSpooler(&lc.config.General, lc.control)

  prospector, err := NewProspector(lc.config, lc.from_beginning, registrar, lc.control)
  if err != nil {
    log.Fatalf("Failed to initialise: %s", err)
  }

  // Start the pipeline
  go prospector.Prospect(event_chan)

  go spooler.Spool(event_chan, publisher_chan)

  go publisher.Publish(publisher_chan)

  go registrar.Register()

  log.Notice("Pipeline ready")

  lc.shutdown_chan = make(chan os.Signal, 1)
  lc.reload_chan = make(chan os.Signal, 1)
  lc.registerSignals()

SignalLoop:
  for {
    select {
      case <-lc.shutdown_chan:
        lc.cleanShutdown()
        break SignalLoop
      case <-lc.reload_chan:
        lc.reloadConfig()
    }
  }

  log.Notice("Exiting")
}

func (lc *LogCourier) startUp() {
  var version bool
  var config_test bool
  var list_supported bool
  var cpu_profile string

  flag.BoolVar(&version, "version", false, "show version information")
  flag.BoolVar(&config_test, "config-test", false, "Test the configuration specified by -config and exit")
  flag.BoolVar(&list_supported, "list-supported", false, "List supported transports and codecs")
  flag.StringVar(&cpu_profile, "cpuprofile", "", "write cpu profile to file")

  // This MAY add some flags
  lc.platform.Init()

  flag.StringVar(&lc.config_file, "config", "", "The config file to load")
  flag.BoolVar(&lc.from_beginning, "from-beginning", false, "On first run, read new files from the beginning instead of the end")

  flag.Parse()

  if version {
    fmt.Printf("Log Courier version %s\n", Log_Courier_Version)
    os.Exit(0)
  }

  if list_supported {
    fmt.Printf("Available transports:\n")
    for _, transport := range AvailableTransports() {
      fmt.Printf("  %s\n", transport)
    }

    fmt.Printf("Available codecs:\n")
    for _, codec := range AvailableCodecs() {
      fmt.Printf("  %s\n", codec)
    }
    os.Exit(0)
  }

  if lc.config_file == "" {
    fmt.Fprintf(os.Stderr, "Please specify a configuration file with -config.\n\n")
    flag.PrintDefaults()
    os.Exit(1)
  }

  err := lc.loadConfig()

  if config_test {
    if err == nil {
      fmt.Printf("Configuration OK\n")
      os.Exit(0)
    }
    fmt.Printf("Configuration test failed: %s\n", err)
    os.Exit(1)
  }

  if err != nil {
    fmt.Printf("Configuration error: %s\n", err)
    os.Exit(1)
  }

  lc.configureLogging()

  if cpu_profile != "" {
    log.Notice("Starting CPU profiler")
    f, err := os.Create(cpu_profile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    go func() {
      time.Sleep(60 * time.Second)
      pprof.StopCPUProfile()
      log.Panic("CPU profile completed")
    }()
  }
}

func (lc *LogCourier) configureLogging() {
  // First, the stdout backend
  backends := make([]logging.Backend, 1)
  stderr_backend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lmicroseconds)
  backends[0] = stderr_backend

  // Set backends BEFORE log level (or we reset log level)
  logging.SetBackend(backends...)

  // Set the logging level
  logging.SetLevel(lc.config.General.LogLevel, "")

  lc.platform.ConfigureLogging(backends)
}

func (lc *LogCourier) loadConfig() error {
  lc.config = NewConfig()
  if err := lc.config.Load(lc.config_file); err != nil {
    return err
  }

  if len(lc.config.Files) == 0 {
    return fmt.Errorf("No file groups were found in the configuration.")
  }

  return nil
}

func (lc *LogCourier) reloadConfig() {
  if err := lc.loadConfig(); err != nil {
    log.Warning("Configuration error, reload unsuccessful: %s", err)
    return
  }

  log.Notice("Configuration reload successful")

  // Update the log level
  logging.SetLevel(lc.config.General.LogLevel, "")

  // Pass the new config to the pipeline workers
  lc.control.SendConfig(lc.config)
}

func (lc *LogCourier) cleanShutdown() {
  log.Notice("Initiating shutdown")
  lc.control.Shutdown()
  lc.control.Wait()
}
