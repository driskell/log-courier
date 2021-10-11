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

type poolReceiverStatus struct {
	listen string
	active bool
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
	receivers            map[transports.Receiver]*poolReceiverStatus
	receiversByListen    map[string]transports.Receiver
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
	var spool [][]*event.Event
	var spoolChan chan<- []*event.Event
	eventChan := r.eventChan
	shutdownChan := r.shutdownChan

ReceiverLoop:
	for {
		var nextSpool []*event.Event = nil
		if len(spool) != 0 {
			nextSpool = spool[0]
		}

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
		case <-r.partialAckSchedule.OnNext():
			for {
				connection := r.partialAckSchedule.Next()
				if connection == nil {
					break
				}
				expectedAck := r.partialAckConnection[connection][0]
				expectedEvent := expectedAck.event
				receiver := expectedEvent.Context().Value(transports.ContextReceiver).(transports.Receiver)
				receiver.Acknowledge(expectedEvent.Context(), expectedEvent.Nonce(), expectedAck.sequence)
				// Schedule again
				r.partialAckSchedule.Set(connection, time.Second*5)
			}
			r.partialAckSchedule.Reschedule()
		case events := <-r.ackChan:
			currentContext := events[0].Context()
			connection := currentContext.Value(transports.ContextConnection)
			position := currentContext.Value(poolContextEventPosition).(*poolEventPosition)
			for _, item := range events[1:] {
				nextPosition := item.Context().Value(poolContextEventPosition).(*poolEventPosition)
				if *nextPosition.nonce != *position.nonce {
					r.ackEventsEvent(currentContext, connection, position.nonce, position.sequence)
					currentContext = item.Context()
					connection = currentContext.Value(transports.ContextConnection)
				}
				position = nextPosition
			}
			r.ackEventsEvent(currentContext, connection, position.nonce, position.sequence)
			r.partialAckSchedule.Reschedule()
		case receiverEvent := <-eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.ConnectEvent:
				// New connection - start an idle timeout
				connection := eventImpl.Context().Value(transports.ContextConnection)
				receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
				r.startIdleTimeout(eventImpl.Context(), receiver, connection)
			case *transports.EventsEvent:
				connection := eventImpl.Context().Value(transports.ContextConnection)
				// Schedule partial ack if this is first set of events
				if _, has := r.partialAckConnection[connection]; !has {
					r.partialAckSchedule.Set(connection, 5*time.Second)
				}
				r.partialAckConnection[connection] = append(r.partialAckConnection[connection], &poolEventProgress{event: eventImpl, sequence: 0})
				// Build the events with our acknowledger and submit the bundle
				var events = make([]*event.Event, len(eventImpl.Events()))
				for idx, item := range eventImpl.Events() {
					ctx := context.WithValue(eventImpl.Context(), poolContextEventPosition, &poolEventPosition{nonce: eventImpl.Nonce(), sequence: uint32(idx + 1)})
					events[idx] = event.NewEventFromBytes(ctx, r, item)
				}
				spool = append(spool, events)
				spoolChan = r.output
			case *transports.EndEvent:
				// Connection EOF
				connection := eventImpl.Context().Value(transports.ContextConnection)
				if _, ok := r.partialAckConnection[connection]; !ok {
					// Nothing left to ack - close
					receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
					receiver.ShutdownConnection(eventImpl.Context())
					// Receive side is closed, and we're just sending, so it'll close automatically once flushed, so clear all scheduled timeouts
					r.partialAckSchedule.Remove(connection)
				} else {
					// Add to the partialAckSchedule a nil progress to signal that when we finish ack everything - this connection can close
					r.partialAckConnection[connection] = append(r.partialAckConnection[connection], nil)
				}
			case *transports.StatusEvent:
				if eventImpl.StatusChange() == transports.Finished {
					// Remove the receiver from our list and exit if all receivers are finished
					receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
					if status, has := r.receivers[receiver]; has {
						delete(r.receivers, receiver)
						// Only a receiver that is active will exist in the receiversByListen index
						// Inactive receivers are ones removed during a config reload for listens we don't want anymore
						if status.active {
							delete(r.receiversByListen, status.listen)
						}
						if len(r.receivers) == 0 && shutdownChan == nil {
							break ReceiverLoop
						}
					}
				}
			case *transports.PingEvent:
				// Immediately send a pong back - ignore failure - remote will time itself out
				connection := eventImpl.Context().Value(transports.ContextConnection)
				receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
				if _, ok := r.partialAckConnection[connection]; ok {
					// We should not be receiving PING if we haven't finished acknowleding - this violates protocol
					r.failConnection(eventImpl.Context(), receiver, connection, fmt.Errorf("received ping message on non-idle connection"))
				} else if err := receiver.Pong(eventImpl.Context()); err != nil {
					r.failConnection(eventImpl.Context(), receiver, connection, err)
				} else {
					// Reset idle timeout
					r.startIdleTimeout(eventImpl.Context(), receiver, connection)
				}
			}
		case spoolChan <- nextSpool:
			copy(spool, spool[1:])
			spool = spool[:len(spool)-1]
			if len(spool) == 0 {
				spoolChan = nil
			}
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

	lastAck := false
	closeConnection := false
	if sequence == endSequence {
		if len(partialAcks) == 1 {
			lastAck = true
		} else if partialAcks[1] == nil {
			lastAck = true
			closeConnection = true
		}
		if lastAck || closeConnection {
			delete(r.partialAckConnection, connection)
			r.partialAckSchedule.Remove(connection)
		} else {
			copy(r.partialAckConnection[connection], r.partialAckConnection[connection][1:])
			r.partialAckConnection[connection] = r.partialAckConnection[connection][:len(r.partialAckConnection[connection])-1]
			r.partialAckSchedule.Set(connection, time.Second*5)
		}
	} else {
		r.partialAckSchedule.Set(connection, time.Second*5)
	}

	expectedAck.sequence = sequence

	receiver := ctx.Value(transports.ContextReceiver).(transports.Receiver)
	if err := receiver.Acknowledge(ctx, nonce, sequence); err != nil {
		r.failConnection(ctx, receiver, connection, err)
	} else if lastAck && (!r.receivers[receiver].active || closeConnection) {
		// Either it's the last ack for a connection on a shutting down receiver, or the connection read closed, so shutdown the connection
		receiver.ShutdownConnection(ctx)
	} else if lastAck {
		r.startIdleTimeout(ctx, receiver, connection)
	}
}

// startIdleTimeout scheduled an idle timeout timer
// All connections should send pings or some other data within a timeout period
func (r *Pool) startIdleTimeout(ctx context.Context, receiver transports.Receiver, connection interface{}) {
	// Set a network timeout - we should be receiving pings - close connections that do nothing
	r.partialAckSchedule.SetCallback(connection, 15*time.Second, func() {
		r.failConnection(ctx, receiver, connection, fmt.Errorf("no data received within timeout"))
	})
}

// failConnection cleans up resources and fails the connection
func (r *Pool) failConnection(ctx context.Context, receiver transports.Receiver, connection interface{}, err error) {
	r.partialAckSchedule.Remove(connection)
	delete(r.partialAckConnection, connection)
	receiver.FailConnection(ctx, err)
}

// updateReceivers updates the list of running receivers
func (r *Pool) updateReceivers(newConfig *config.Config) {
	receiversConfig := transports.FetchReceiversConfig(newConfig)
	newReceivers := make(map[transports.Receiver]*poolReceiverStatus)
	newReceiversByListen := make(map[string]transports.Receiver)
	for _, cfgEntry := range receiversConfig {
		if cfgEntry.Enabled {
			for _, listen := range cfgEntry.Listen {
				if _, has := newReceiversByListen[listen]; has {
					log.Warning("Ignoring duplicate receiver listen address: %s", listen)
					continue
				}
				if r.receivers != nil {
					// If already exists as active then reload config
					if existing, has := r.receiversByListen[listen]; has {
						// Receiver already exists, update its configuration
						existing.ReloadConfig(newConfig, cfgEntry.Factory)
						newReceivers[existing] = &poolReceiverStatus{listen: listen, active: true}
						newReceiversByListen[listen] = existing
						continue
					}
				}
				// Create new receiver
				pool := addresspool.NewPool(listen)
				newReceiversByListen[listen] = cfgEntry.Factory.NewReceiver(context.Background(), pool, r.eventChan)
				newReceivers[newReceiversByListen[listen]] = &poolReceiverStatus{listen: listen, active: true}
			}
		}
	}

	// Anything left in existing receivers was not updated so should be shutdown and copied across as inactive
	if r.receivers != nil {
		for receiver, status := range r.receivers {
			if _, has := newReceivers[receiver]; has {
				// We still kept it alive, skip it
				continue
			}
			receiver.Shutdown()
			status.active = false
			newReceivers[receiver] = status
		}
	}

	r.receivers = newReceivers
	r.receiversByListen = newReceiversByListen
}

// shutdown stops all receivers
func (r *Pool) shutdown() {
	for receiver, status := range r.receivers {
		receiver.Shutdown()
		status.active = false
	}
}
