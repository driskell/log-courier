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

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/admin/api"
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

// Pool manages a list of receivers
type Pool struct {
	// Pipeline
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config

	// Internal
	ackChan           chan []*event.Event
	eventChan         chan transports.Event
	receivers         map[transports.Receiver]*poolReceiverStatus
	receiversByListen map[string]transports.Receiver
	scheduler         *scheduler.Scheduler
	connectionLock    sync.RWMutex
	connectionStatus  map[interface{}]*poolConnectionStatus

	apiConfig      *admin.Config
	apiConnections api.Array
	apiListeners   api.Array
	apiStatus      *apiStatus
	api.Node
}

// NewPool creates a new receiver pool
func NewPool(app *core.App) *Pool {
	return &Pool{
		apiConfig:        admin.FetchConfig(app.Config()),
		ackChan:          make(chan []*event.Event),
		eventChan:        make(chan transports.Event),
		scheduler:        scheduler.NewScheduler(),
		connectionStatus: make(map[interface{}]*poolConnectionStatus),
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

	if r.apiConfig.Enabled {
		r.apiStatus = &apiStatus{r: r}
		r.SetEntry("listeners", &r.apiListeners)
		r.SetEntry("connections", &r.apiConnections)
		r.SetEntry("status", r.apiStatus)
		r.apiConfig.SetEntry("receiver", r)
	}

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
		case <-r.scheduler.OnNext():
			for {
				connection := r.scheduler.Next()
				if connection == nil {
					break
				}
				r.connectionLock.Lock()
				expectedAck := r.connectionStatus[connection].progress[0]
				expectedEvent := expectedAck.event
				receiver := expectedEvent.Context().Value(transports.ContextReceiver).(transports.Receiver)
				receiver.Acknowledge(expectedEvent.Context(), expectedEvent.Nonce(), expectedAck.sequence)
				r.scheduler.Set(connection, time.Second*5)
				r.connectionLock.Unlock()
			}
			r.scheduler.Reschedule()
		case events := <-r.ackChan:
			r.connectionLock.Lock()
			currentContext := events[0].Context()
			connection := currentContext.Value(transports.ContextConnection)
			position := currentContext.Value(poolContextEventPosition).(*poolEventPosition)
			for _, item := range events[1:] {
				nextConnection := item.Context().Value(transports.ContextConnection)
				nextPosition := item.Context().Value(poolContextEventPosition).(*poolEventPosition)
				// Also check for backwards or same sequence - this effectively manages cases of duplicate nonce as sequence will remain same or reset to 0
				if nextConnection != connection || *nextPosition.nonce != *position.nonce || nextPosition.sequence <= position.sequence {
					r.ackEventsEvent(currentContext, connection, position.nonce, position.sequence)
					currentContext = item.Context()
					connection = nextConnection
				}
				position = nextPosition
			}
			r.ackEventsEvent(currentContext, connection, position.nonce, position.sequence)
			r.scheduler.Reschedule()
			r.connectionLock.Unlock()

			// If shutting down, have all acknowledgemente been handled, and all receivers closed?
			if shutdownChan == nil && len(r.receivers) == 0 && len(r.connectionStatus) == 0 {
				break ReceiverLoop
			}
		case receiverEvent := <-eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.ConnectEvent:
				// New connection - start an idle timeout
				connection := eventImpl.Context().Value(transports.ContextConnection)
				receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
				r.startIdleTimeout(eventImpl.Context(), receiver, connection)
				r.connectionLock.Lock()
				r.connectionStatus[connection] = newPoolConnectionStatus(r, r.receivers[receiver].listen, eventImpl.Remote(), eventImpl.Desc())
				r.apiConnections.AddEntry(eventImpl.Remote(), r.connectionStatus[connection])
				r.connectionLock.Unlock()
			case transports.EventsEvent:
				r.connectionLock.Lock()
				connection := eventImpl.Context().Value(transports.ContextConnection)
				receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
				connectionStatus := r.connectionStatus[connection]
				// Schedule partial ack if this is first set of events
				if len(connectionStatus.progress) == 0 {
					r.scheduler.Set(connection, 5*time.Second)
				}
				connectionStatus.progress = append(connectionStatus.progress, &poolEventProgress{event: eventImpl, sequence: 0})
				r.connectionLock.Unlock()
				// Build the events with our acknowledger and submit the bundle
				var events = make([]*event.Event, len(eventImpl.Events()))
				for idx, item := range eventImpl.Events() {
					ctx := context.WithValue(eventImpl.Context(), poolContextEventPosition, &poolEventPosition{nonce: eventImpl.Nonce(), sequence: uint32(idx + 1)})
					item := event.NewEvent(ctx, r, item)
					r.addEventSource(item, connectionStatus)
					events[idx] = item
				}
				spool = append(spool, events)
				spoolChan = r.output
				// Stop reading events if this client breached our limit
				if len(r.connectionStatus[connection].progress) > int(r.receivers[receiver].config.MaxPendingPayloads) {
					receiver.ShutdownConnectionRead(eventImpl.Context(), fmt.Errorf("max pending payloads exceeded"))
				}
			case *transports.EndEvent:
				// Connection EOF
				r.connectionLock.Lock()
				connection := eventImpl.Context().Value(transports.ContextConnection)
				if partialAcks, ok := r.connectionStatus[connection]; !ok || len(partialAcks.progress) == 0 {
					// Nothing left to ack - close send side
					receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
					receiver.ShutdownConnection(eventImpl.Context())
					// Receive side is closed, and we're just sending, so it'll close automatically once flushed, so clear all scheduled timeouts
					r.apiConnections.RemoveEntry(r.connectionStatus[connection].remote)
					delete(r.connectionStatus, connection)
					r.scheduler.Remove(connection)
				} else {
					// Add to the scheduler a nil progress to signal that when we finish ack everything - this connection can close
					r.connectionStatus[connection].progress = append(r.connectionStatus[connection].progress, nil)
				}
				r.connectionLock.Unlock()
			case *transports.StatusEvent:
				if eventImpl.StatusChange() == transports.Finished {
					// Remove the receiver from our list and exit if all receivers are finished
					receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
					status := r.receivers[receiver]
					delete(r.receivers, receiver)
					// Only a receiver that is active will exist in the receiversByListen index
					// Inactive receivers are ones removed during a config reload for listens we don't want anymore
					if status.active {
						delete(r.receiversByListen, status.listen)
					}
					// If shutting down, have all acknowledgemente been handled, and all receivers closed?
					if shutdownChan == nil && len(r.receivers) == 0 && len(r.connectionStatus) == 0 {
						break ReceiverLoop
					}
				}
			case *transports.PingEvent:
				// Immediately send a pong back - ignore failure - remote will time itself out
				connection := eventImpl.Context().Value(transports.ContextConnection)
				receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
				if status, ok := r.connectionStatus[connection]; ok && len(status.progress) != 0 {
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
	status, ok := r.connectionStatus[connection]
	if !ok || len(status.progress) == 0 {
		panic(fmt.Sprintf("Out of order acknowledgement: Nonce=%x; Sequence=%d; ExpectedNonce=<none>; ExpectedSequenceMin=<none>; ExpectedSequenceMax=<none>", *nonce, sequence))
	}

	partialAcks := status.progress
	expectedAck := partialAcks[0]
	expectedEvent := expectedAck.event
	endSequence := expectedEvent.Count()
	if *expectedEvent.Nonce() != *nonce || sequence < expectedAck.sequence || sequence > endSequence {
		panic(fmt.Sprintf("Out of order acknowledgement: Nonce=%x; Sequence=%d; ExpectedNonce=%x; ExpectedSequenceMin=%d; ExpectedSequenceMax=%d", *nonce, sequence, *expectedEvent.Nonce(), expectedAck.sequence, endSequence))
	}

	status.lines += int64(sequence - expectedAck.sequence)

	lastAck := false
	closeConnection := false
	if sequence == endSequence {
		if len(partialAcks) == 1 {
			lastAck = true
		} else if partialAcks[1] == nil {
			lastAck = true
			closeConnection = true
		}
		copy(partialAcks, partialAcks[1:])
		status.progress = partialAcks[:len(partialAcks)-1]
		if lastAck || closeConnection {
			r.scheduler.Remove(connection)
		} else {
			r.scheduler.Set(connection, time.Second*5)
		}
	} else {
		r.scheduler.Set(connection, time.Second*5)
		expectedAck.sequence = sequence
	}

	receiver := ctx.Value(transports.ContextReceiver).(transports.Receiver)
	if err := receiver.Acknowledge(ctx, nonce, sequence); err != nil {
		r.failConnection(ctx, receiver, connection, err)
	} else if lastAck && closeConnection {
		// Either it's the last ack for a connection on a shutting down receiver, or the connection read closed, so shutdown the connection
		receiver.ShutdownConnection(ctx)
		r.apiConnections.RemoveEntry(status.remote)
		delete(r.connectionStatus, connection)
	} else if lastAck {
		r.startIdleTimeout(ctx, receiver, connection)
	}
}

// startIdleTimeout scheduled an idle timeout timer
// All connections should send pings or some other data within a timeout period
func (r *Pool) startIdleTimeout(ctx context.Context, receiver transports.Receiver, connection interface{}) {
	// Set a network timeout - we should be receiving pings - close connections that do nothing
	// TODO: Make configurable - it's not configurable on courier side yet and it's 900 there
	r.scheduler.SetCallback(connection, 1000*time.Second, func() {
		r.failConnection(ctx, receiver, connection, fmt.Errorf("no data received within timeout"))
	})
}

// failConnection cleans up resources and fails the connection
// It will stop the partial acks but will continue to remember the connection to deal with incoming acknowledgements
// for events already passed through the pipeline
func (r *Pool) failConnection(ctx context.Context, receiver transports.Receiver, connection interface{}, err error) {
	r.scheduler.Remove(connection)

	// Fail connection - but only if the error wasn't InvalidState, which means it's dead anyway
	// Saves us throwing errors due to lagging acknowledgements for a dead connection
	if err != transports.ErrInvalidState {
		receiver.FailConnection(ctx, err)
	}
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
						// Receiver already exists, does it need restarting? If so let us create new, otherwise keep it
						if !existing.Factory().ShouldRestart(cfgEntry.Factory) {
							newReceivers[existing] = &poolReceiverStatus{config: cfgEntry, listen: listen, active: true}
							newReceiversByListen[listen] = existing
							continue
						}
					}
				}
				// Create new receiver
				newReceiversByListen[listen] = cfgEntry.Factory.NewReceiver(context.Background(), listen, r.eventChan)
				newReceivers[newReceiversByListen[listen]] = &poolReceiverStatus{config: cfgEntry, listen: listen, active: true}
				receiverApi := &api.KeyValue{}
				receiverApi.SetEntry("listen", api.String(listen))
				receiverApi.SetEntry("maxPendingPayloads", api.Number(cfgEntry.MaxPendingPayloads))
				r.apiListeners.AddEntry(listen, receiverApi)
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
			r.apiListeners.RemoveEntry(status.listen)
		}
	}

	r.receivers = newReceivers
	r.receiversByListen = newReceiversByListen
}

// shutdown stops all receivers
func (r *Pool) shutdown() {
	for _, receiver := range r.receiversByListen {
		receiver.Shutdown()
	}
}

func (r *Pool) addEventSource(item *event.Event, connectionStatus *poolConnectionStatus) {
	source, ok := item.MustResolve("agent[source]", nil).([]string)
	if !ok {
		// Overwrite invalid / missing map
		source = make([]string, 0, 1)
	}
	source = append(source, connectionStatus.label)
	item.MustResolve("agent[source]", source)
}
