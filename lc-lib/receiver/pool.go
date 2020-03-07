/*
 * Copyright 2014-2019 Jason Woods.
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

package receiver

import (
	"context"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// PoolContext is a context key for Pool
type poolContext string

const (
	poolContextListen poolContext = "listen"
)

// Pool manages a list of receivers
type Pool struct {
	// Constructor
	cfg *config.Config

	// Pipeline
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config

	// Internal
	eventChan chan transports.Event
	receivers map[string]transports.Receiver
}

// NewPool creates a new receiver pool
func NewPool(app *core.App) *Pool {
	return &Pool{
		cfg: app.Config(),
	}
}

// SetOutput sets the output channel
func (r *Pool) SetOutput(output chan<- []*event.Event) {
	r.output = output
}

// SetShutdownChan sets the shutdown channel
func (r *Pool) SetShutdownChan(shutdownChan <-chan struct{}) {
	r.shutdownChan = shutdownChan
}

// SetConfigChan sets the config channel
func (r *Pool) SetConfigChan(configChan <-chan *config.Config) {
	r.configChan = configChan
}

// Init sets up the listener
func (r *Pool) Init(cfg *config.Config) error {
	r.eventChan = make(chan transports.Event)
	return nil
}

// Run starts listening
func (r *Pool) Run() {
	r.updateReceivers()

	shutdown := false

ReceiverLoop:
	for {
		select {
		case <-r.shutdownChan:
			if len(r.receivers) == 0 {
				// Nothing to wait to shutdown, return now, don't even log
				return
			}
			log.Info("Receiver pool is shutting down receivers")
			r.shutdown()
			shutdown = true
			break
		case newConfig := <-r.configChan:
			r.cfg = newConfig
			r.updateReceivers()
			break
		case receiverEvent := <-r.eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.EventsEvent:
				r.output <- eventImpl.Events()
			case *transports.StatusEvent:
				if eventImpl.StatusChange() == transports.Finished {
					delete(r.receivers, eventImpl.Context().Value(poolContextListen).(string))
					if len(r.receivers) == 0 && shutdown {
						break ReceiverLoop
					}
				}
			}
			break
		}
	}

	log.Info("Receiver pool exiting")
}

// updateReceivers updates the list of running receivers
func (r *Pool) updateReceivers() {
	receiversConfig := transports.FetchReceiversConfig(r.cfg)
	existingReceivers := r.receivers
	for _, cfgEntry := range receiversConfig {
		if cfgEntry.Enabled {
			for _, listen := range cfgEntry.Listen {
				if existing, has := r.receivers[listen]; has {
					// Receiver already exists, update its configuration
					existing.ReloadConfig(r.cfg, cfgEntry.Factory)
					delete(existingReceivers, listen)
					continue
				}
				// Create new receiver
				pool := addresspool.NewPool(listen)
				r.receivers[listen] = cfgEntry.Factory.NewReceiver(context.WithValue(context.Background(), poolContextListen, listen), pool, r.eventChan)
			}
		}
	}

	// Anything left in existingReceivers was not updated so should be shutdown
	for _, receiver := range existingReceivers {
		receiver.Shutdown()
	}
}

// shutdown stops all receivers
func (r *Pool) shutdown() {
	for _, receiver := range r.receivers {
		receiver.Shutdown()
	}
}
