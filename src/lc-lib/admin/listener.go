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
  "strings"
  "time"
)

type NetListener interface {
  Accept() (net.Conn, error)
  Close() error
  Addr() net.Addr
  SetDeadline(time.Time) error
}

type Listener struct {
  core.PipelineSegment
  core.PipelineConfigReceiver

  config          *core.GeneralConfig
  command_chan    chan string
  response_chan   chan *Response
  listener        NetListener
  client_shutdown chan interface{}
  client_started  chan interface{}
  client_ended    chan interface{}
}

func NewListener(pipeline *core.Pipeline, config *core.GeneralConfig) (*Listener, error) {
  var err error

  ret := &Listener{
    config:          config,
    command_chan:    make(chan string),
    response_chan:   make(chan *Response),
    client_shutdown: make(chan interface{}),
    // TODO: Make this limit configurable
    client_started:  make(chan interface{}, 50),
    client_ended:    make(chan interface{}, 50),
  }

  if ret.listener, err = ret.listen(config); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (l *Listener) listen(config *core.GeneralConfig) (NetListener, error) {
  bind := strings.SplitN(config.AdminBind, ":", 2)
  if len(bind) == 1 {
    bind = append(bind, bind[0])
    bind[0] = "tcp"
  }

  switch bind[0] {
  case "tcp":
    return l.listenTCP("tcp", bind[1])
  case "tcp4":
    return l.listenTCP("tcp4", bind[1])
  case "tcp6":
    return l.listenTCP("tcp6", bind[1])
  case "unix":
    return l.listenUnix(bind[1])
  }

  return nil, fmt.Errorf("Unknown transport specified for admin bind: '%s'", bind[0])
}

func (l *Listener) listenTCP(transport, addr string) (NetListener, error) {
  taddr, err := net.ResolveTCPAddr(transport, addr)
  if err != nil {
    return nil, fmt.Errorf("The admin bind address specified is not valid: %s", err)
  }

  listener, err := net.ListenTCP(transport, taddr)
  if err != nil {
    return nil, err
  }

  return listener, nil
}

func (l *Listener) listenUnix(addr string) (NetListener, error) {
  uaddr, err := net.ResolveUnixAddr("unix", addr)
  if err != nil {
    return nil, fmt.Errorf("The admin bind address specified is not valid: %s", err)
  }

  listener, err := net.ListenUnix("unix", uaddr)
  if err != nil {
    return nil, err
  }

  return listener, nil
}

func (l *Listener) OnCommand() <-chan string {
  return l.command_chan
}

func (l *Listener) Respond(response *Response) {
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
        if config.General.AdminBind != l.config.AdminBind {
          new_listener, err := l.listen(&config.General)
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

    conn, err := l.listener.Accept()
    if err != nil {
      if net_err, ok := err.(*net.OpError); ok && net_err.Timeout() {
        continue
      }
      log.Warning("Failed to accept admin connection: %s", err)
    }

    log.Debug("New admin connection from %s", conn.RemoteAddr())

    l.startServer(conn)
  }

  // Shutdown listener
  l.listener.Close()

  // Trigger shutdowns
  close(l.client_shutdown)

  // Wait for shutdowns
  for {
    if len(l.client_started) == 0 {
      break
    }

    select {
      case <-l.client_ended:
        <-l.client_started
      default:
    }
  }
}

func (l *Listener) startServer(conn net.Conn) {
  server := newServer(l, conn)

  select {
    case <-l.client_ended:
      <-l.client_started
    default:
  }

  select {
    case l.client_started <- 1:
    default:
      // TODO: Make this limit configurable
      log.Warning("Refused admin connection: Admin connection limit (50) reached")
      return
  }

  go server.Run()
}
