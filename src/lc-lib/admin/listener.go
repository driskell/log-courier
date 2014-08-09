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
  "lc-lib/core"
  "net"
  "sync"
  "time"
)

type LogCourierAdmin interface {
  FetchSnapshot() []core.Snapshot
}

type Listener struct {
  core.PipelineSegment

  host       string
  port       int
  listener   *net.TCPListener
  conn_group sync.WaitGroup
}

func NewListener(pipeline *core.Pipeline, lc LogCourierAdmin) (*Listener, error) {
  ret := &Listener{
    host: "127.0.0.1",
    port: 1234,
  }

  if err := ret.init(); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (l *Listener) init() (err error) {
  addr := &net.TCPAddr{
    IP:   net.ParseIP(l.host),
    Port: l.port,
  }

  l.listener, err = net.ListenTCP("tcp", addr)
  return
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
      log.Warning("Admin connection failure: %s", err)
    }

    log.Info("New admin connection from %s", conn.RemoteAddr())

    l.startServer(conn)
  }

  l.conn_group.Wait()
}

func (l *Listener) startServer(conn *net.TCPConn) {
  l.conn_group.Add(1)

  server := newServer(&l.conn_group, conn)
  go server.Run()
}
