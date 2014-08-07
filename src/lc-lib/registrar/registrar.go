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

package registrar

import (
  "encoding/json"
  "fmt"
  "lc-lib/core"
  "os"
  "sync"
)

type LoadPreviousFunc func(string, *FileState) (core.Stream, error)

type Registrar struct {
  core.PipelineSegment

  sync.Mutex

  registrar_chan chan []RegistrarEvent
  references     int
  persistdir     string
  statefile      string
  state          map[core.Stream]*FileState
}

func NewRegistrar(pipeline *core.Pipeline, persistdir string) *Registrar {
  ret := &Registrar{
    registrar_chan: make(chan []RegistrarEvent, 16), // TODO: Make configurable?
    persistdir:     persistdir,
    statefile:      ".log-courier",
    state:          make(map[core.Stream]*FileState),
  }

  pipeline.Register(ret)

  return ret
}

func (r *Registrar) LoadPrevious(callback_func LoadPreviousFunc) (have_previous bool, err error) {
  data := make(map[string]*FileState)

  // Load the previous state - opening RDWR ensures we can write too and fail early
  // c_filename is what we will use to test create capability
  filename := r.persistdir + string(os.PathSeparator) + ".log-courier"
  c_filename := r.persistdir + string(os.PathSeparator) + ".log-courier.new"

  var f *os.File
  f, err = os.OpenFile(filename, os.O_RDWR, 0600)
  if err != nil {
    // Fail immediately if this is not a path not found error
    if !os.IsNotExist(err) {
      return
    }

    // Try the .new file - maybe we failed mid-move
    filename, c_filename = c_filename, filename
    f, err = os.OpenFile(filename, os.O_RDWR, 0600)
  }

  if err != nil {
    // Did we fail, or did it just not exist?
    if !os.IsNotExist(err) {
      return
    }
    return false, nil
  }

  // Parse the data
  log.Notice("Loading registrar data from %s", filename)
  have_previous = true

  decoder := json.NewDecoder(f)
  decoder.Decode(&data)
  f.Close()

  r.state = make(map[core.Stream]*FileState, len(data))

  var stream core.Stream
  for file, state := range data {
    if stream, err = callback_func(file, state); err != nil {
      return
    }
    r.state[stream] = state
  }

  // Test we can successfully save new states by attempting to save now
  if err = r.writeRegistry(); err != nil {
    return false, fmt.Errorf("Registry write failed: %s", err)
  }

  return
}

func (r *Registrar) Connect() *RegistrarEventSpool {
  r.Lock()
  ret := newRegistrarEventSpool(r)
  r.references++
  r.Unlock()
  return ret
}

func (r *Registrar) dereferenceSpooler() {
  r.Lock()
  r.references--
  if r.references == 0 {
    // Shutdown registrar, all references are closed
    close(r.registrar_chan)
  }
  r.Unlock()
}

func (r *Registrar) toCanonical() (canonical map[string]*FileState) {
  canonical = make(map[string]*FileState, len(r.state))
  for _, value := range r.state {
    if _, ok := canonical[*value.Source]; ok {
      // We should never allow this - report an error
      log.Error("BUG: Unexpected registrar conflict detected for %s", *value.Source)
    }
    canonical[*value.Source] = value
  }
  return
}

func (r *Registrar) Run() {
  defer func() {
    r.Done()
  }()

RegistrarLoop:
  for {
    // Ignore shutdown channel - wait for registrar to close
    select {
    case spool := <-r.registrar_chan:
      if spool == nil {
        break RegistrarLoop
      }

      for _, event := range spool {
        event.Process(r.state)
      }

      if err := r.writeRegistry(); err != nil {
        log.Error("Registry write failed: %s", err)
      }
    }
  }

  log.Info("Registrar exiting")
}
