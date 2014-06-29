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
  "log"
  "os"
  "os/signal"
  "runtime/pprof"
  "sync"
  "time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var spool_size = flag.Uint64("spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
var idle_timeout = flag.Duration("idle-flush-time", 5*time.Second, "Maximum time to wait for a full spool before flushing anyway")
var config_file = flag.String("config", "", "The config file to load")
var use_syslog = flag.Bool("log-to-syslog", false, "Log to syslog instead of stdout")
var from_beginning = flag.Bool("from-beginning", false, "Read new files from the beginning, instead of the end")

var shutdown_signals []os.Signal

func init() {
  // All systems support os.Interrupt, so add to shutdown signals
  RegisterShutdownSignal(os.Interrupt)
}

func RegisterShutdownSignal(signal os.Signal) {
  shutdown_signals = append(shutdown_signals, signal)
}

func main() {
  var flag_version bool

  flag.BoolVar(&flag_version, "version", false, "show version information")

  flag.Parse()

  if flag_version {
    log.Printf("Log Courier version 0.10\n")
    return
  }

  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    go func() {
      time.Sleep(60 * time.Second)
      pprof.StopCPUProfile()
      panic("done")
    }()
  }

  if *use_syslog {
    configureSyslog()
  } else {
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)
  }

  log.Printf("Log Courier starting up\n")

  shutdown := make(chan os.Signal, 1)
  signal.Notify(shutdown, shutdown_signals...)

  logcourier := NewLogCourier()
  logcourier.StartCourier(*config_file)

  select {
    case <-shutdown:
      log.Printf("Log Courier shutting down\n")
      logcourier.Shutdown()
  }
}

type LogCourierShutdown struct {
  signal chan interface{}
  group sync.WaitGroup
}

func NewLogCourierShutdown() *LogCourierShutdown {
  return &LogCourierShutdown{
    signal: make(chan interface{}),
  }
}

func (lcs *LogCourierShutdown) Signal() <-chan interface{} {
  return lcs.signal
}

func (lcs *LogCourierShutdown) Shutdown() {
  close(lcs.signal)
}

func (lcs *LogCourierShutdown) Done() {
  lcs.group.Done()
}

func (lcs *LogCourierShutdown) Add() *LogCourierShutdown {
  lcs.group.Add(1)
  return lcs
}

func (lcs *LogCourierShutdown) Wait() {
  lcs.group.Wait()
}

type LogCourier struct {
  shutdown *LogCourierShutdown
  config   *Config
}

func NewLogCourier() *LogCourier {
  ret := &LogCourier{
    shutdown: NewLogCourierShutdown(),
  }
  return ret
}

func (lc *LogCourier) StartCourier(config_file string) {
  var err error

  lc.config = NewConfig()
  if err = lc.config.Load(config_file); err != nil {
    log.Fatalf("%s. Please check your configuration", err)
  }

  event_chan := make(chan *FileEvent, 16)
  publisher_chan := make(chan []*FileEvent, 1)

  if len(lc.config.Files) == 0 {
    log.Fatalf("No paths given. What files do you want to watch?")
  }

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
  prospector := NewProspector(lc.config, lc.shutdown.Add())

  spooler := NewSpooler(*spool_size, *idle_timeout, lc.shutdown.Add())

  publisher := NewPublisher(&lc.config.Network, lc.shutdown.Add())
  if err := publisher.Init(); err != nil {
    log.Fatalf("The publisher failed to initialise: %s\n", err)
  }

  registrar := NewRegistrar(lc.config.General.PersistDir, lc.shutdown.Add())

  // Start the pipeline
  go prospector.Prospect(prospector_resume, registrar, event_chan)

  go spooler.Spool(event_chan, publisher_chan)

  go publisher.Publish(publisher_chan, registrar)

  go registrar.Register(registrar_persist)
}

func (lc *LogCourier) Shutdown() {
  lc.shutdown.Shutdown()
  lc.shutdown.Wait()
}
