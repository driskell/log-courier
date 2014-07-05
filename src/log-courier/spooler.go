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
  "time"
)

type Spooler struct {
  control     *LogCourierControl
  max_size     uint64
  idle_timeout time.Duration
  spool        []*FileEvent
}

func NewSpooler(max_size uint64, idle_timeout time.Duration, control *LogCourierMasterControl) *Spooler {
  return &Spooler{
    control: control.Register(),
    max_size: max_size,
    idle_timeout: idle_timeout,
    spool: make([]*FileEvent, 0, max_size),
  }
}

func (s *Spooler) Spool(input <-chan *FileEvent, output chan<- []*FileEvent) {
  defer func() {
    s.control.Done()
  }()

  timer := time.NewTimer(s.idle_timeout)

SpoolerLoop:
  for {
    select {
    case event := <-input:
      s.spool = append(s.spool, event)

      // Flush if full
      if len(s.spool) == cap(s.spool) {
        if !s.sendSpool(output) {
          break SpoolerLoop
        }
        timer.Reset(s.idle_timeout)
      }
    case <-timer.C:
      // Flush what we have, if anything
      if len(s.spool) > 0 {
        if !s.sendSpool(output) {
          break SpoolerLoop
        }
      }

      timer.Reset(s.idle_timeout)
    case <-s.control.ShutdownSignal():
      break SpoolerLoop
    }
  }

  log.Info("Spooler shutdown complete")
}

func (s *Spooler) sendSpool(output chan<- []*FileEvent) bool {
  select {
  case <-s.control.ShutdownSignal():
    return false
  case output <- s.spool:
  }
  s.spool = make([]*FileEvent, 0, s.max_size)
  return true
}
