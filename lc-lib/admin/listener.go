/*
 * Copyright 2014-2015 Jason Woods.
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
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"gopkg.in/tylerb/graceful.v1"
)

// Server provides a REST interface for administration
type Server struct {
	core.PipelineSegment
	core.PipelineConfigReceiver

	config   *Config
	listener netListener
	server   *graceful.Server
}

// NewServer creates a new admin listener on the pipeline
func NewServer(pipeline *core.Pipeline, config *config.Config) (*Server, error) {
	ret := &Server{
		config: config.Get("admin").(*Config),
	}

	listener, err := ret.listen(ret.config)
	if err != nil {
		return nil, err
	}

	ret.initServer(listener)

	pipeline.Register(ret)

	return ret, nil
}

func (l *Server) listen(config *Config) (netListener, error) {
	bind := strings.SplitN(config.Bind, ":", 2)
	if len(bind) == 1 {
		bind = append(bind, bind[0])
		bind[0] = "tcp"
	}

	if listener, ok := registeredListeners[bind[0]]; ok {
		log.Info("[admin] REST admin now listening on %s:%s", bind[0], bind[1])
		return listener(bind[0], bind[1])
	}

	return nil, fmt.Errorf("Unknown transport specified for admin bind: '%s'", bind[0])
}

func (l *Server) initServer(listener netListener) {
	l.listener = listener
	l.server = &graceful.Server{
		// We handle shutdown ourselves
		NoSignalHandling: true,
		// The HTTP server
		Server: &http.Server{
			// TODO: Make all these configurable?
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				l.handle(w, r)
			}),
		},
	}
}

func (l *Server) startServer() {
	// TODO: error checking
	go l.server.Serve(l.listener)
}

// Run is the main routine for the admin listener
func (l *Server) Run() {
	defer func() {
		l.Done()
	}()

	l.startServer()

	var closingOldServer <-chan struct{}
	var reloadingConfig *Config
	var shuttingDown, shutdownStarted bool

ListenerLoop:
	for {
		select {
		case <-l.OnShutdown():
			shuttingDown = true
			if closingOldServer == nil {
				// Start the shutdown
				closingOldServer = l.shutdownServer()
				shutdownStarted = true
			}
		case config := <-l.OnConfig():
			// We can't yet disable admin during a reload
			aconfig := config.Get("admin").(*Config)
			if aconfig.Enabled {
				if aconfig.Bind != l.config.Bind {
					// Delay reload if still waiting for old server to close
					if closingOldServer != nil {
						reloadingConfig = aconfig
						continue
					}

					closingOldServer = l.reloadServer(aconfig)
				}
			}
		case <-closingOldServer:
			log.Info("[admin] REST administration stopped")
			if shuttingDown {
				// Is shutdown in progress? Leave if so
				if shutdownStarted {
					break ListenerLoop
				}

				// Need to start the shutdown
				closingOldServer = l.shutdownServer()
				shutdownStarted = true
				continue
			}

			if reloadingConfig != nil {
				// Another reload queued - process it
				closingOldServer = l.reloadServer(reloadingConfig)
				continue
			}
		default:
		}
	}

	log.Info("[admin] REST administration exited")
}

func (l *Server) shutdownServer() <-chan struct{} {
	// TODO: Make configurable? This is the shutdown timeout
	l.server.Stop(10 * time.Second)
	return l.server.StopChan()
}

func (l *Server) reloadServer(config *Config) <-chan struct{} {
	newListener, err := l.listen(config)
	if err != nil {
		log.Error("The new admin configuration failed to apply: %s", err)
		return nil
	}

	stopChan := l.shutdownServer()

	l.initServer(newListener)
	l.startServer()

	return stopChan
}

func (l *Server) handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		panicArg := recover()
		if panicArg == nil {
			return
		}

		// Only keep normal errors
		err, ok := panicArg.(error)
		if !ok {
			panic(panicArg)
			return
		}

		// Don't keep runtime errors or we'll miss stack trace
		if _, ok := err.(runtime.Error); ok {
			panic(err)
		}

		switch err {
		case ErrNotImplemented:
			l.errorResponse(w, r, err, http.StatusNotImplemented)
			return
		}

		l.errorResponse(w, r, err, http.StatusInternalServerError)
		log.Info("[admin] Request error: %s", err.Error())
	}()

	if r.Method != "GET" && r.Method != "POST" && r.Method != "PUT" {
		l.accessLog(r, http.StatusMethodNotAllowed)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for leading forward slash
	if len(r.URL.Path) == 0 || r.URL.Path[0] != '/' {
		panic(ErrNotFound)
		return
	}

	parts := strings.Split(r.URL.Path[1:], "/")
	root := APIEntry(&l.config.APINode)

	if len(parts) == 1 && parts[0] == "" {
		parts = parts[:0]
	}
	for _, part := range parts {
		newRoot, err := root.Get(part)
		if err != nil {
			panic(err)
		}
		if newRoot == nil {
			panic(ErrNotFound)
			return
		}

		root = newRoot
	}

	// Call?
	if r.Method != "GET" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		err = root.Call(r.Form)
		if err != nil {
			panic(err)
		}

		l.accessLog(r, http.StatusOK)
		return
	}

	// Ensure up to date values
	if err := root.Update(); err != nil {
		panic(err)
	}

	var err error
	var contentType string
	var response []byte

	if r.URL.Query().Get("w") == "pretty" {
		contentType = "text/plain"
		response, err = root.HumanReadable("")
	} else {
		contentType = "application/json"
		response, err = json.Marshal(root)
	}

	if err != nil {
		panic(err)
	}

	l.accessLog(r, http.StatusOK)
	w.Header().Add("Content-Type", contentType)
	w.Write(response)
}

func (l *Server) errorResponse(w http.ResponseWriter, r *http.Request, err error, c int) {
	structErr := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}

	jsonError, encodeErr := json.Marshal(structErr)
	if encodeErr != nil {
		l.accessLog(r, http.StatusServiceUnavailable)
		http.Error(w, encodeErr.Error(), http.StatusServiceUnavailable)
	}

	l.accessLog(r, c)
	http.Error(w, string(jsonError), c)
}

func (l *Server) accessLog(r *http.Request, c int) {
	log.Debug("[admin] %s %s %d", r.Method, r.URL.Path, c)
}
