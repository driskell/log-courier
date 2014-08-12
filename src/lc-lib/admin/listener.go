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

type Listener struct {
  core.PipelineSegment
  core.PipelineConfigReceiver

  config        *core.GeneralConfig
  command_chan  chan string
  response_chan chan interface{}
  listener      *net.TCPListener
  conn_group    sync.WaitGroup
}

func NewListener(pipeline *core.Pipeline, config *core.GeneralConfig) (*Listener, error) {
  var err error

  ret := &Listener{
    config:        config,
    command_chan:  make(chan string),
    response_chan: make(chan interface{}),
  }

  if ret.listener, err = ret.startListening(config); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (l *Listener) startListening(config *core.GeneralConfig) (*net.TCPListener, error) {
  var addr net.TCPAddr

  addr.IP = net.ParseIP(config.AdminBind)

  if addr.IP == nil {
    return nil, fmt.Errorf("The admin bind address specified is not a valid IP address")
  }

  addr.Port = config.AdminPort

  listener, err := net.ListenTCP("tcp", &addr)
  if err != nil {
    return nil, err
  }

  return listener, nil
}

func (l *Listener) OnCommand() <-chan string {
  return l.command_chan
}

func (l *Listener) Respond(response interface{}) {
  l.response_chan <- response
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
    case config := <-l.OnConfig():
      // We can't yet disable admin during a reload
      if config.General.AdminEnabled {
        if config.General.AdminBind != l.config.AdminBind || config.General.AdminPort != l.config.AdminPort {
          new_listener, err := l.startListening(&config.General)
          if err != nil {
            log.Error("The new admin configuration failed to apply: %s", err)
            continue
          }

          l.listener.Close()
          l.listener = new_listener
          l.config = &config.General
        }
      }
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

    log.Debug("New admin connection from %s", conn.RemoteAddr())

    l.startServer(conn)
  }

  l.conn_group.Wait()
}

func (l *Listener) startServer(conn *net.TCPConn) {
  l.conn_group.Add(1)

  server := newServer(l, conn)
  go server.Run()
}
