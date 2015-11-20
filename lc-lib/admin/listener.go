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
	"github.com/driskell/log-courier/src/lc-lib/config"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"net"
	"strings"
	"time"
)

type Listener struct {
	core.PipelineSegment
	core.PipelineConfigReceiver

	config          *config.General
	command_chan    chan string
	response_chan   chan *Response
	listener        NetListener
	client_shutdown chan interface{}
	client_started  chan interface{}
	client_ended    chan interface{}
}

func NewListener(pipeline *core.Pipeline, config *config.General) (*Listener, error) {
	var err error

	ret := &Listener{
		config:          config,
		command_chan:    make(chan string),
		response_chan:   make(chan *Response),
		client_shutdown: make(chan interface{}),
		// TODO: Make this limit configurable
		client_started: make(chan interface{}, 50),
		client_ended:   make(chan interface{}, 50),
	}

	if ret.listener, err = ret.listen(config); err != nil {
		return nil, err
	}

	pipeline.Register(ret)

	return ret, nil
}

func (l *Listener) listen(config *config.General) (NetListener, error) {
	bind := strings.SplitN(config.AdminBind, ":", 2)
	if len(bind) == 1 {
		bind = append(bind, bind[0])
		bind[0] = "tcp"
	}

	if listener, ok := registeredListeners[bind[0]]; ok {
		return listener(bind[0], bind[1])
	}

	return nil, fmt.Errorf("Unknown transport specified for admin bind: '%s'", bind[0])
}

func (l *Listener) OnCommand() <-chan string {
	return l.command_chan
}

func (l *Listener) Respond(response *Response) {
	l.response_chan <- response
}

func (l *Listener) Run() {
	defer func() {
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
