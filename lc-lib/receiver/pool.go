/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	return &Pool{}
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
	r.updateReceivers(cfg)
	return nil
}

// Run starts listening
func (r *Pool) Run() {
	shutdownChan := r.shutdownChan

ReceiverLoop:
	for {
		select {
		case <-shutdownChan:
			if len(r.receivers) == 0 {
				// Nothing to wait to shutdown, return now, don't even log
				break ReceiverLoop
			}
			log.Info("Receiver pool is shutting down receivers")
			r.shutdown()
			shutdownChan = nil
			break
		case newConfig := <-r.configChan:
			r.updateReceivers(newConfig)
			break
		case receiverEvent := <-r.eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.EventsEvent:
				r.output <- eventImpl.Events()
			case *transports.StatusEvent:
				if eventImpl.StatusChange() == transports.Finished {
					// Only remove from our list if we're shutting down and equal nil
					// Those entries that shutdown but are not nil are likely a config reload that brought it back after a previous shutdown so ignore the one that shuts down
					key := eventImpl.Context().Value(poolContextListen).(string)
					if existing, has := r.receivers[key]; has && existing == nil {
						delete(r.receivers, key)
						if len(r.receivers) == 0 && shutdownChan == nil {
							break ReceiverLoop
						}
					}
				}
			}
			break
		}
	}

	log.Info("Receiver pool exiting")
}

// updateReceivers updates the list of running receivers
func (r *Pool) updateReceivers(newConfig *config.Config) {
	receiversConfig := transports.FetchReceiversConfig(newConfig)
	newReceivers := make(map[string]transports.Receiver)
	for _, cfgEntry := range receiversConfig {
		if cfgEntry.Enabled {
			for _, listen := range cfgEntry.Listen {
				if _, has := newReceivers[listen]; has {
					log.Warning("Ignoring duplicate receiver listen address: %s", listen)
					continue
				}
				if r.receivers != nil {
					// If already exists and isn't nil (which means shutting down) then reload config
					if existing, has := r.receivers[listen]; has && existing != nil {
						// Receiver already exists, update its configuration
						existing.ReloadConfig(newConfig, cfgEntry.Factory)
						delete(r.receivers, listen)
						newReceivers[listen] = existing
						continue
					}
				}
				// Create new receiver
				pool := addresspool.NewPool(listen)
				newReceivers[listen] = cfgEntry.Factory.NewReceiver(context.WithValue(context.Background(), poolContextListen, listen), pool, r.eventChan)
			}
		}
	}

	// Anything left in existing receivers was not updated so should be shutdown
	if r.receivers != nil {
		for key, receiver := range r.receivers {
			receiver.Shutdown()
			newReceivers[key] = nil
		}
	}

	r.receivers = newReceivers
}

// shutdown stops all receivers
func (r *Pool) shutdown() {
	for key, receiver := range r.receivers {
		receiver.Shutdown()
		r.receivers[key] = nil
	}
}
