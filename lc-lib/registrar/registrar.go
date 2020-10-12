/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package registrar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

// Context is a registrar context key
type Context string

const (
	// ContextEndOffset is the end offset that registrar saves
	// This should be present in all event contexts that registrar needs to persist
	ContextEndOffset Context = "endOffset"
)

// LoadPreviousFunc is a callback implemented by a consumer of the Registrar,
// and is called for each part of a loaded previous state when LoadPrevious is
// called
type LoadPreviousFunc func(string, *FileState) (context.Context, error)

// Registrar persists file offsets to a file that can be read again on startup
// to resume where we left off
type Registrar struct {
	shutdownChan  <-chan struct{}
	registrarChan chan []EventProcessor
	writeTimer    *time.Timer
	persistdir    string
	statefile     string
	statepath     string
	state         map[context.Context]*FileState
}

// NewRegistrar creates a new Registrar associated with a file in a directory
func NewRegistrar(persistDir string) *Registrar {
	ret := &Registrar{
		registrarChan: make(chan []EventProcessor, 16), // TODO: Make configurable?
		writeTimer:    time.NewTimer(0),
		persistdir:    persistDir,
		statefile:     ".log-courier",
		state:         make(map[context.Context]*FileState),
	}

	ret.statepath = path.Join(ret.persistdir, ret.statefile)
	<-ret.writeTimer.C

	return ret
}

// SetShutdownChan sets the shutdown channel
func (r *Registrar) SetShutdownChan(shutdownChan <-chan struct{}) {
	r.shutdownChan = shutdownChan
}

// Acknowledge is called when an acknowledgement is made by the publisher
func (r *Registrar) Acknowledge(events []*event.Event) {
	r.registrarChan <- []EventProcessor{NewAckEvent(events)}
}

// LoadPrevious loads the previous state from the file
func (r *Registrar) LoadPrevious(callbackFunc LoadPreviousFunc) (havePrevious bool, err error) {
	data := make(map[string]*FileState)

	// Load the previous state - opening RDWR ensures we can write too and fail early
	// c_filename is what we will use to test create capability
	filename := r.persistdir + string(os.PathSeparator) + ".log-courier"
	newFilename := r.persistdir + string(os.PathSeparator) + ".log-courier.new"

	var f *os.File
	f, err = os.OpenFile(filename, os.O_RDWR, 0600)
	if err != nil {
		// Fail immediately if this is not a path not found error
		if !os.IsNotExist(err) {
			return
		}

		// Try the .new file - maybe we failed mid-move
		filename = newFilename
		f, err = os.OpenFile(filename, os.O_RDWR, 0600)
	}

	if err != nil {
		// Did we fail, or did it just not exist?
		if !os.IsNotExist(err) {
			return
		}
		return false, nil
	}

	// Parse the data
	log.Notice("Loading registrar data from %s", filename)
	havePrevious = true

	decoder := json.NewDecoder(f)
	decoder.Decode(&data)
	f.Close()

	r.state = make(map[context.Context]*FileState, len(data))

	var ctx context.Context
	for file, state := range data {
		if ctx, err = callbackFunc(file, state); err != nil {
			return
		}
		r.state[ctx] = state
	}

	// Test we can successfully save new states by attempting to save now
	if err = r.writeRegistry(); err != nil {
		return false, fmt.Errorf("Registry write failed: %s", err)
	}

	return
}

func (r *Registrar) toCanonical() (canonical map[string]*FileState) {
	canonical = make(map[string]*FileState, len(r.state))
	for _, value := range r.state {
		if _, ok := canonical[*value.Source]; ok {
			// We should never allow this - report an error
			log.Errorf("BUG: Unexpected registrar conflict detected for %s", *value.Source)
		}
		canonical[*value.Source] = value
	}
	return
}

// Init does nothing, as Registrar needs no setup
func (r *Registrar) Init(cfg *config.Config) error {
	return nil
}

// Run starts the registrar - it is called by the pipeline
func (r *Registrar) Run() {
	pendingWrite := false
	// TODO: Make configurable?
	r.writeTimer.Reset(time.Second)

RegistrarLoop:
	for {
		select {
		case spool := <-r.registrarChan:
			for _, event := range spool {
				event.process(r.state)
			}

			pendingWrite = true
		case <-r.writeTimer.C:
			// TODO: Make configurable?
			r.writeTimer.Reset(time.Second)

			if !pendingWrite {
				continue
			}

			r.tryWriteRegistry()
			pendingWrite = false
		case <-r.shutdownChan:
			// Sink has completed and we're now shutting down
			break RegistrarLoop
		}
	}

	if pendingWrite {
		r.tryWriteRegistry()
	}

	log.Info("Registrar exiting")
}

// tryWriteRegistry attempts to write the state file and logs any problems
func (r *Registrar) tryWriteRegistry() {
	if err := r.writeRegistry(); err != nil {
		log.Errorf("Registry write failed: %s", err)
	}

	log.Debug("Written registry file to %s", r.statepath)
}
