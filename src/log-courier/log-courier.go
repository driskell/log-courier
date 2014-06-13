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
  "runtime/pprof"
  "time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var spool_size = flag.Uint64("spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
var idle_timeout = flag.Duration("idle-flush-time", 5*time.Second, "Maximum time to wait for a full spool before flushing anyway")
var config_file = flag.String("config", "", "The config file to load")
var use_syslog = flag.Bool("log-to-syslog", false, "Log to syslog instead of stdout")
var from_beginning = flag.Bool("from-beginning", false, "Read new files from the beginning, instead of the end")

func main() {
  flag.Parse()

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

  log.Printf("Log Courier starting up")

  config, err := LoadConfig(*config_file)
  if err != nil {
    log.Fatalf("%s. Please check your configuration", err)
  }

  event_chan := make(chan *FileEvent, 16)
  publisher_chan := make(chan []*FileEvent, 1)
  registrar_chan := make(chan []RegistrarEvent, 16)

  if len(config.Files) == 0 {
    log.Fatalf("No paths given. What files do you want to watch?")
  }

  // The basic model of execution:
  // - prospector: finds files in paths/globs to harvest, starts harvesters
  // - harvester: reads a file, sends events to the spooler
  // - spooler: buffers events until ready to flush to the publisher
  // - publisher: writes to the network, notifies registrar
  // - registrar: records positions of files read
  // Finally, prospector uses the registrar information, on restart, to
  // determine where in each file to resume a harvester.

  // Load the previous log file locations now, for use in prospector
  load_resume := make(map[string]*FileState)
  history, err := os.Open(".log-courier")
  if err == nil {
    wd, err := os.Getwd()
    if err != nil {
      wd = ""
    }
    log.Printf("Loading registrar data from %s/.log-courier\n", wd)

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

  // Initialise structures
  prospector := &Prospector{FileConfigs: config.Files}

  publisher := &Publisher{config: &config.Network}
  if err := publisher.Init(); err != nil {
    log.Fatalf("The publisher failed to initialise: %s\n", err)
  }

  // Start the pipeline
  go prospector.Prospect(prospector_resume, registrar_chan, event_chan)

  go Spool(event_chan, publisher_chan, *spool_size, *idle_timeout)

  go publisher.Publish(publisher_chan, registrar_chan)

  Registrar(registrar_persist, registrar_chan)
} /* main */
