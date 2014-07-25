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

type LogCourierMasterControl struct {
  signal         chan interface{}
  config_sinks   map[*LogCourierControl]chan *Config
  snapshot_chan  chan interface{}
  shutdown_group sync.WaitGroup
  snapshot_group sync.WaitGroup
  group_count    int
}

func NewLogCourierMasterControl() *LogCourierMasterControl {
  return &LogCourierMasterControl{
    signal:        make(chan interface{}),
    snapshot_chan: make(chan interface{}),
    config_sinks:  make(map[*LogCourierControl]chan *Config),
  }
}

func (lcs *LogCourierMasterControl) Shutdown() {
  close(lcs.signal)
}

func (lcs *LogCourierMasterControl) SendConfig(config *Config) {
  for _, sink := range lcs.config_sinks {
    sink <- config
  }
}

func (lcs *LogCourierMasterControl) Snapshot() {
  lcs.snapshot_group.Add(lcs.group_count)
  sent, received := lcs.group_count, lcs.group_count

  // Send and receive snapshot information
  for {
    select {
    case snapshot := <-lcs.snapshot_chan:
      // TODO
      log.Notice("Received: %v", snapshot)

      // Finished receiving?
      received--
      if received == 0 {
        break
      }
    case func() {
      
    }() <- 1:
      // Finished sending?
      sent--
      if sent == 0 {
        signal = nil
      }
    }
  }

  log.Notice("Snapshot complete")
}

func (lcs *LogCourierMasterControl) Register() *LogCourierControl {
  return lcs.register()
}

func (lcs *LogCourierMasterControl) RegisterWithRecvConfig() *LogCourierControl {
  ret := lcs.register()

  config_chan := make(chan *Config)
  lcs.config_sinks[ret] = config_chan
  ret.config_sink = config_chan

  return ret
}

func (lcs *LogCourierMasterControl) register() *LogCourierControl {
  lcs.shutdown_group.Add(1)
  lcs.group_count++

  return &LogCourierControl{
    signal:         lcs.signal,
    snapshot_chan:  lcs.snapshot_chan,
    shutdown_group: &lcs.shutdown_group,
    snapshot_group: &lcs.snapshot_group,
  }
}

func (lcs *LogCourierMasterControl) Wait() {
  lcs.group.Wait()
}

type LogCourierControl struct {
  signal         <-chan interface{}
  config_sink    <-chan *Config
  snapshot_chan  ->chan interface{}
  shutdown_group *sync.WaitGroup
}

func (lcs *LogCourierControl) Signal() <-chan interface{} {
  return lcs.signal
}

func (lcs *LogCourierControl) SendSnapshot() {
  lcs.snapshot_chan <- "SNAPSHOT"
}

func (lcs *LogCourierControl) RecvConfig() <-chan *Config {
  if lcs.config_sink == nil {
    panic("RecvConfig invalid: LogCourierControl was not registered with RegisterWithRecvConfig")
  }
  return lcs.config_sink
}

func (lcs *LogCourierControl) Done() {
  lcs.shutdown_group.Done()
}

type LogCourier struct {
  control        *LogCourierMasterControl
  config         *Config
  shutdown_chan  chan os.Signal
  reload_chan    chan os.Signal
  config_file    string
  from_beginning bool
  log_file       *os.File
}

func NewLogCourier() *LogCourier {
  ret := &LogCourier{
    control: NewLogCourierMasterControl(),
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
      // TODO: make part of a comm channel of some sort
    case <-time.After(5 * time.Second):
      lc.fetchSnapshot()
    }
  }

  log.Notice("Exiting")

  if lc.log_file != nil {
    lc.log_file.Close()
  }
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

  if err = lc.configureLogging(); err != nil {
    fmt.Printf("Failed to initialise logging: %s", err)
    os.Exit(1)
  }

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

func (lc *LogCourier) configureLogging() (err error) {
  backends := make([]logging.Backend, 0, 1)

  // First, the stdout backend
  if lc.config.General.LogStdout {
    backends = append(backends, logging.NewLogBackend(os.Stdout, "", stdlog.LstdFlags|stdlog.Lmicroseconds))
  }

  // Log file?
  if lc.config.General.LogFile != "" {
    lc.log_file, err = os.OpenFile(lc.config.General.LogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0640)
    if err != nil {
      return
    }

    backends = append(backends, logging.NewLogBackend(lc.log_file, "", stdlog.LstdFlags|stdlog.Lmicroseconds))
  }

  if err = lc.configureLoggingPlatform(&backends); err != nil {
    return
  }

  // Set backends BEFORE log level (or we reset log level)
  logging.SetBackend(backends...)

  // Set the logging level
  logging.SetLevel(lc.config.General.LogLevel, "")

  return nil
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

func (lc *LogCourier) fetchSnapshot() {
  lc.control.Snapshot()
  // TODO
}

func (lc *LogCourier) cleanShutdown() {
  log.Notice("Initiating shutdown")
  lc.control.Shutdown()
  lc.control.Wait()
}
