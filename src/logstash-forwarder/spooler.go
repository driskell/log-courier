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
