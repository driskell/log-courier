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
  control *LogCourierControl
  config  *GeneralConfig
  spool   []*FileEvent
}

func NewSpooler(config *GeneralConfig, control *LogCourierMasterControl) *Spooler {
  return &Spooler{
    control: control.RegisterWithRecvConfig(),
    config:  config,
    spool:   make([]*FileEvent, 0, config.SpoolSize),
  }
}

func (s *Spooler) Spool(input <-chan *FileEvent, output chan<- []*FileEvent) {
  defer func() {
    s.control.Done()
  }()

  timer_start := time.Now()
  timer := time.NewTimer(s.config.SpoolTimeout)

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
        timer_start = time.Now()
        timer.Reset(s.config.SpoolTimeout)
      }
    case <-timer.C:
      // Flush what we have, if anything
      if len(s.spool) > 0 {
        if !s.sendSpool(output) {
          break SpoolerLoop
        }
      }

      timer_start = time.Now()
      timer.Reset(s.config.SpoolTimeout)
    case signal := <-s.control.Signal():
      if signal == nil {
        // Shutdown
        break SpoolerLoop
      }

      s.control.SendSnapshot()
    case config := <-s.control.RecvConfig():
      s.config = &config.General

      // Immediate flush?
      passed := time.Now().Sub(timer_start)
      if passed >= s.config.SpoolTimeout || len(s.spool) >= int(s.config.SpoolSize) {
        if !s.sendSpool(output) {
          break SpoolerLoop
        }
        timer_start = time.Now()
        timer.Reset(s.config.SpoolTimeout)
      } else {
        timer.Reset(passed - s.config.SpoolTimeout)
      }
    }
  }

  log.Info("Spooler exiting")
}

func (s *Spooler) sendSpool(output chan<- []*FileEvent) bool {
  select {
  case <-s.control.ShutdownSignal():
    return false
  case output <- s.spool:
  }
  s.spool = make([]*FileEvent, 0, s.config.SpoolSize)
  return true
}
