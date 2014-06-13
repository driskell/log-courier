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

func Spool(input <-chan *FileEvent, output chan<- []*FileEvent, max_size uint64, idle_timeout time.Duration) {
  // Slice for spooling into
  spool := make([]*FileEvent, 0, max_size)

  timer := time.NewTimer(idle_timeout)

  for {
    select {
    case event := <-input:
      spool = append(spool, event)

      // Flush if full
      if len(spool) == cap(spool) {
        output <- spool
        spool = make([]*FileEvent, 0, max_size)
        timer.Reset(idle_timeout)
      }
    case <-timer.C:
      // Flush what we have, if anything
      if len(spool) > 0 {
        output <- spool
        spool = make([]*FileEvent, 0, max_size)
      }

      timer.Reset(idle_timeout)
    }
  }
}
