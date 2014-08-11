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

package admin

import (
  "fmt"
  "lc-lib/core"
  "net"
  "sync"
  "time"
)

type LogCourierAdmin interface {
  ProcessCommand(command string) (interface{}, error)
}

type Listener struct {
  core.PipelineSegment

  lc         LogCourierAdmin
  addr       net.TCPAddr
  listener   *net.TCPListener
  conn_group sync.WaitGroup
}

func NewListener(pipeline *core.Pipeline, lc LogCourierAdmin) (*Listener, error) {
  var err error

  ret := &Listener{
    lc: lc,
  }

  ret.addr.IP = net.ParseIP("127.0.0.1")

  if ret.addr.IP == nil {
    return nil, fmt.Errorf("Invalid admin listen address")
  }

  ret.addr.Port = 1234

  if ret.listener, err = net.ListenTCP("tcp", &ret.addr); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (l *Listener) Run() {
  defer func(){
    l.Done()
  }()

ListenerLoop:
  for {
    select {
    case <-l.OnShutdown():
      break ListenerLoop
    default:
    }

    l.listener.SetDeadline(time.Now().Add(time.Second))

    conn, err := l.listener.AcceptTCP()
    if err != nil {
      if net_err, ok := err.(*net.OpError); ok && net_err.Timeout() {
        continue
      }
      log.Warning("Failed to accept admin connection: %s", err)
    }

    log.Info("New admin connection from %s", conn.RemoteAddr())

    l.startServer(conn)
  }

  l.conn_group.Wait()
}

func (l *Listener) startServer(conn *net.TCPConn) {
  l.conn_group.Add(1)

  server := newServer(l.lc, &l.conn_group, conn)
  go server.Run()
}
