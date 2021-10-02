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
	"fmt"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/scheduler"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// PoolContext is a context key for Pool
type poolContext string

const (
	poolContextReceiver      poolContext = "receiver"
	poolContextEventPosition poolContext = "eventpos"
)

type poolEventPosition struct {
	nonce    *string
	sequence uint32
}

type poolEventProgress struct {
	event    *transports.EventsEvent
	sequence uint32
}

// Pool manages a list of receivers
type Pool struct {
	// Pipeline
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config

	// Internal
	ackChan              chan []*event.Event
	eventChan            chan transports.Event
	receivers            map[string]transports.Receiver
	partialAckMutex      sync.Mutex
	partialAckSchedule   *scheduler.Scheduler
	partialAckConnection map[interface{}][]*poolEventProgress
}

// NewPool creates a new receiver pool
func NewPool(app *core.App) *Pool {
	return &Pool{
		ackChan:              make(chan []*event.Event),
		eventChan:            make(chan transports.Event),
		partialAckSchedule:   scheduler.NewScheduler(),
		partialAckConnection: make(map[interface{}][]*poolEventProgress),
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
	r.updateReceivers(cfg)
	return nil
}

// Run starts listening
func (r *Pool) Run() {
	var spool []*event.Event
	var spoolChan chan<- []*event.Event
	eventChan := r.eventChan
	shutdownChan := r.shutdownChan

ReceiverLoop:
	for {
		partialAckNext := r.partialAckSchedule.OnNext()

		select {
		case <-shutdownChan:
			if len(r.receivers) == 0 {
				// Nothing to wait to shutdown, return now, don't even log
				break ReceiverLoop
			}
			log.Info("Receiver pool is shutting down receivers")
			r.shutdown()
			shutdownChan = nil
		case newConfig := <-r.configChan:
			r.updateReceivers(newConfig)
		case <-partialAckNext:
			if connection := r.partialAckSchedule.Next(); connection != nil {
				expectedAck := r.partialAckConnection[connection][0]
				expectedEvent := expectedAck.event
				receiverKey := expectedEvent.Context().Value(poolContextReceiver).(string)
				if receiver, has := r.receivers[receiverKey]; has && receiver != nil {
					receiver.Acknowledge(expectedEvent.Context(), expectedEvent.Nonce(), expectedAck.sequence)
					// Schedule again
					r.partialAckSchedule.Set(connection, time.Second*5)
				} else {
					// Receiver is shut(ting) down so stop processing it
					delete(r.partialAckConnection, connection)
				}
			}
			r.partialAckSchedule.Reschedule(true)
		case events := <-r.ackChan:
			connection := events[0].Context().Value(transports.ContextConnection)
			position := events[0].Context().Value(poolContextEventPosition).(*poolEventPosition)
			for _, item := range events[1:] {
				nextPosition := item.Context().Value(poolContextEventPosition).(*poolEventPosition)
				if *nextPosition.nonce != *position.nonce {
					r.ackEventsEvent(item.Context(), connection, position.nonce, position.sequence)
					connection = item.Context().Value(transports.ContextConnection)
				}
				position = nextPosition
			}
			r.ackEventsEvent(events[0].Context(), connection, position.nonce, position.sequence)
		case receiverEvent := <-eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.EventsEvent:
				connection := eventImpl.Context().Value(transports.ContextConnection)
				r.partialAckConnection[connection] = append(r.partialAckConnection[connection], &poolEventProgress{event: eventImpl, sequence: 0})
				r.partialAckSchedule.Set(connection, 5*time.Second)
				r.partialAckSchedule.Reschedule(false)
				// Build the events with our acknowledger and submit the bundle
				var events = make([]*event.Event, len(eventImpl.Events()))
				for idx, item := range eventImpl.Events() {
					ctx := context.WithValue(eventImpl.Context(), poolContextEventPosition, &poolEventPosition{nonce: eventImpl.Nonce(), sequence: uint32(idx + 1)})
					events[idx] = event.NewEventFromBytes(ctx, r, item)
				}
				spool = events
				spoolChan = r.output
			case *transports.StatusEvent:
				if eventImpl.StatusChange() == transports.Finished {
					// Only remove from our list if we're shutting down and equal nil
					// Those entries that shutdown but are not nil are likely a config reload that brought it back after a previous shutdown so ignore the one that shuts down
					key := eventImpl.Context().Value(poolContextReceiver).(string)
					if existing, has := r.receivers[key]; has && existing == nil {
						delete(r.receivers, key)
						if len(r.receivers) == 0 && shutdownChan == nil {
							break ReceiverLoop
						}
					}
				}
			case *transports.PingEvent:
				// Immediately send a pong back - ignore failure - remote will time itself out
				// TODO: Receiving a ping when we have outstanding events is invalid as per protocol, kill connection
				key := eventImpl.Context().Value(poolContextReceiver).(string)
				if receiver, has := r.receivers[key]; has && receiver != nil {
					receiver.Pong(eventImpl.Context())
				}
			}
		case spoolChan <- spool:
			spool = nil
			spoolChan = nil
		}
	}

	log.Info("Receiver pool exiting")
}

// Acknowledge processes event acknowledgements (implements event.Acknowledger)
func (r *Pool) Acknowledge(events []*event.Event) {
	r.ackChan <- events
}

// ackEventsEvent processes the acknowledgement, updating any pending partial acknowledgement schedules
func (r *Pool) ackEventsEvent(ctx context.Context, connection interface{}, nonce *string, sequence uint32) {
	partialAcks, ok := r.partialAckConnection[connection]
	if !ok {
		panic(fmt.Sprintf("Out of order acknowledgement: Nonce=%x; Sequence=%d; ExpectedNonce=<none>; ExpectedSequenceMin=<none>; ExpectedSequenceMax=<none>", *nonce, sequence))
	}

	expectedAck := partialAcks[0]
	expectedEvent := expectedAck.event
	endSequence := expectedEvent.Count()
	if *expectedEvent.Nonce() != *nonce || sequence < expectedAck.sequence || sequence > endSequence {
		panic(fmt.Sprintf("Out of order acknowledgement: Nonce=%x; Sequence=%d; ExpectedNonce=%x; ExpectedSequenceMin=%d; ExpectedSequenceMax=%d", *nonce, sequence, *expectedEvent.Nonce(), expectedAck.sequence, endSequence))
	}

	if sequence == endSequence {
		if len(partialAcks) == 1 {
			delete(r.partialAckConnection, connection)
			r.partialAckSchedule.Remove(connection)
		} else {
			r.partialAckConnection[connection] = r.partialAckConnection[connection][1:]
			r.partialAckSchedule.Set(connection, time.Second*5)
		}
	} else {
		r.partialAckSchedule.Set(connection, time.Second*5)
	}

	expectedAck.sequence = sequence

	key := ctx.Value(poolContextReceiver).(string)
	if receiver, has := r.receivers[key]; has && receiver != nil {
		receiver.Acknowledge(ctx, nonce, sequence)
	}
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
				newReceivers[listen] = cfgEntry.Factory.NewReceiver(context.WithValue(context.Background(), poolContextReceiver, listen), pool, r.eventChan)
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
