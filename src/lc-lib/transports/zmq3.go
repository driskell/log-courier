// +build zmq_3_x

/*
 * Copyright 2014 Jason Woods.
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

package transports

/*
#cgo pkg-config: libzmq
#include <zmq.h>

struct zmq_event_t_wrap {
  int event;
  char *addr;
  int fd;
};
*/
import "C"

import (
  "fmt"
  zmq "github.com/alecthomas/gozmq"
  "syscall"
  "unsafe"
)

func (f *TransportZmqFactory) processConfig(config_path string) (err error) {
  return nil
}

func (t *TransportZmq) configureSocket() error {
  return nil
}

// Process ZMQ 3.2.x monitor messages
// http://api.zeromq.org/3-2:zmq-socket-monitor
func (t *TransportZmq) processMonitorIn() (ok bool) {
  for {
    // Bring in the messages
  RetryRecv:
    data, err := t.monitor.Recv(zmq.DONTWAIT)
    if err != nil {
      switch err {
      case syscall.EINTR:
        // Try again
        goto RetryRecv
      case syscall.EAGAIN:
        // No more messages
        ok = true
        return
      }

      // Failure
      t.recv_chan <- fmt.Errorf("Monitor zmq.Socket.Recv failure %s", err)
      return
    }

    switch t.event.part {
    case Monitor_Part_Header:
      event := (*C.struct_zmq_event_t_wrap)(unsafe.Pointer(&data[0]))
      t.event.event = zmq.Event(event.event)
      if event.addr == nil {
        t.event.data = ""
      } else {
        // TODO: Fix this - data has been feed by zmq_msg_close!
        //t.event.data = C.GoString(event.addr)
        t.event.data = ""
      }
      t.event.val = int32(event.fd)
      t.event.Log()
    default:
      log.Debug("Extraneous data in monitor message. Silently discarding.")
      continue
    }

    more, err := t.monitor.RcvMore()
    if err != nil {
      // Failure
      t.recv_chan <- fmt.Errorf("Monitor zmq.Socket.RcvMore failure %s", err)
      return
    }

    if !more {
      t.event.part = Monitor_Part_Header
      continue
    }

    if t.event.part <= Monitor_Part_Data {
      t.event.part++
    }
  }
}
