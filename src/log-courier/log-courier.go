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
  "encoding/json"
  "flag"
  "fmt"
  "log"
  "os"
  "runtime/pprof"
  "sync"
  "time"
)

func main() {
  logcourier := NewLogCourier()
  logcourier.Run()
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
  return lcs.sink
}

func (lcs *LogCourierControl) Done() {
  lcs.group.Done()
}

type LogCourier struct {
  control        *LogCourierMasterControl
  config         *Config
  shutdown_chan  chan os.Signal
  reload_chan    chan os.Signal
  spool_size     uint64
  idle_timeout   time.Duration
  config_file    string
  from_beginning bool
}

func NewLogCourier() *LogCourier {
  ret := &LogCourier{
    control: NewLogCourierMasterControl(),
  }
  return ret
}

func (lc *LogCourier) Run() {
  var err error

  lc.parseFlags()

  log.Printf("Log Courier starting up\n")

  if !lc.loadConfig() {
    log.Fatalf("Startup failed. Please check the configuration.\n")
  }

  event_chan := make(chan *FileEvent, 16)
  publisher_chan := make(chan []*FileEvent, 1)

  // Load the previous log file locations now, for use in prospector
  // TODO: Should this be part of Registrar? We pass registrar into Prospector
  load_resume := make(map[string]*FileState)
  state_path := lc.config.General.PersistDir + string(os.PathSeparator) + ".log-courier"
  history, err := os.Open(state_path)
  if err == nil {
    log.Printf("Loading registrar data from %s\n", state_path)

    decoder := json.NewDecoder(history)
    decoder.Decode(&load_resume)
    history.Close()
  }

  // Generate ProspectorInfo structures for registrar and prosector to communicate with
  prospector_resume := make(map[string]*ProspectorInfo, len(load_resume))
  registrar_persist := make(map[*ProspectorInfo]*FileState, len(load_resume))
  for file, filestate := range load_resume {
    prospector_resume[file] = NewProspectorInfoFromFileState(file, filestate)
    registrar_persist[prospector_resume[file]] = filestate
  }

  // Initialise pipeline
  prospector := NewProspector(lc.config, lc.from_beginning, lc.control)

  spooler := NewSpooler(lc.spool_size, lc.idle_timeout, lc.control)

  publisher := NewPublisher(&lc.config.Network, lc.control)
  if err := publisher.Init(); err != nil {
    log.Fatalf("The publisher failed to initialise: %s\n", err)
  }

  registrar := NewRegistrar(lc.config.General.PersistDir, lc.control)

  // Start the pipeline
  go prospector.Prospect(prospector_resume, registrar, event_chan)

  go spooler.Spool(event_chan, publisher_chan)

  go publisher.Publish(publisher_chan, registrar)

  go registrar.Register(registrar_persist)

  lc.shutdown_chan = make(chan os.Signal, 1)
  lc.reload_chan = make(chan os.Signal, 1)
  lc.registerSignals()

SignalLoop:
  for {
    select {
      case <-lc.shutdown_chan:
        log.Printf("Log Courier shutting down\n")
        lc.cleanShutdown()
        break SignalLoop
      case <-lc.reload_chan:
        lc.reloadConfig()
    }
  }
}

func (lc *LogCourier) parseFlags() {
  var version bool
  var list_supported bool
  var config_test bool
  var cpu_profile string
  var syslog bool

  flag.BoolVar(&version, "version", false, "show version information")
  flag.BoolVar(&config_test, "config-test", false, "Test the configuration specified by -config and exit")
  flag.BoolVar(&list_supported, "list-supported", false, "List supported transports and codecs")
  flag.BoolVar(&syslog, "log-to-syslog", false, "Log to syslog instead of stdout")
  flag.StringVar(&cpu_profile, "cpuprofile", "", "write cpu profile to file")

  // TODO - These should be in the configuration file
  flag.Uint64Var(&lc.spool_size, "spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
  flag.DurationVar(&lc.idle_timeout, "idle-flush-time", 5*time.Second, "Maximum time to wait for a full spool before flushing anyway")
  flag.StringVar(&lc.config_file, "config", "", "The config file to load")
  flag.BoolVar(&lc.from_beginning, "from-beginning", false, "Read new files from the beginning, instead of the end")

  flag.Parse()

  if version {
    fmt.Printf("Log Courier version 0.10\n")
    os.Exit(0)
  }

  if config_test {
    if lc.loadConfig() {
      fmt.Printf("Configuration OK\n")
      os.Exit(0)
    }
    fmt.Printf("Configuration test failed!\n")
    os.Exit(1)
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

  if syslog {
    lc.configureSyslog()
  } else {
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)
  }

  if cpu_profile != "" {
    log.Printf("Starting CPU profiler\n")
    f, err := os.Create(cpu_profile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    go func() {
      time.Sleep(60 * time.Second)
      pprof.StopCPUProfile()
      panic("CPU profile completed")
    }()
  }
}

func (lc *LogCourier) loadConfig() bool {
  lc.config = NewConfig()
  if err := lc.config.Load(lc.config_file); err != nil {
    log.Printf("%s\n", err)
    return false
  }

  if len(lc.config.Files) == 0 {
    log.Printf("No file groups were found in the configuration.\n")
    return false
  }

  return true
}

func (lc *LogCourier) reloadConfig() {
  log.Printf("Reloading configuration.\n")
  if lc.loadConfig() {
    lc.control.SendConfig(lc.config)
  }
}

func (lc *LogCourier) cleanShutdown() {
  lc.control.Shutdown()
  lc.control.Wait()
}
