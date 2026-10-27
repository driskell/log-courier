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
	"sync/atomic"
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

type spoolEntry struct {
	events []*event.Event
	size   int
}

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
	spool             []*spoolEntry
	// Atomic so the admin API can read them while Run modifies the spool
	spoolPayloads atomic.Int64
	spoolSize     atomic.Int64
	// paused stops receiving while the spool is full
	paused bool
	// shuttingDown is set once receivers are asked to shut down, so Run exits when they and their connections finish
	shuttingDown bool
	// done is closed when Run exits
	done chan struct{}

	generalConfig  *config.General
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
		generalConfig:    app.Config().General(),
		ackChan:          make(chan []*event.Event),
		eventChan:        make(chan transports.Event),
		scheduler:        scheduler.NewScheduler(),
		connectionStatus: make(map[interface{}]*poolConnectionStatus),
		done:             make(chan struct{}),
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
	defer close(r.done)
	shutdownChan := r.shutdownChan

	for {
		select {
		case <-shutdownChan:
			if len(r.receivers) == 0 {
				// Nothing to wait to shutdown, return now, don't even log
				log.Info("Receiver pool exiting")
				return
			}
			log.Info("Receiver pool is shutting down receivers")
			r.shutdown()
			r.shuttingDown = true
			shutdownChan = nil
		case newConfig := <-r.configChan:
			r.updateReceivers(newConfig)
		case <-r.scheduler.OnNext():
			r.sendPartialAcks()
		case events := <-r.ackChan:
			r.handleAcks(events)
		case receiverEvent := <-r.intake():
			r.handleEvent(receiverEvent)
		case r.spoolOutput() <- r.nextSpool():
			r.dequeueSpool()
		}

		if r.finished() {
			break
		}
	}

	log.Info("Receiver pool exiting")
}

// finished returns true once shutting down with every receiver and connection closed
func (r *Pool) finished() bool {
	return r.shuttingDown && len(r.receivers) == 0 && len(r.connectionStatus) == 0
}

// intake returns the channel to receive events from, or nil while receiving is paused
func (r *Pool) intake() <-chan transports.Event {
	if r.paused {
		return nil
	}
	return r.eventChan
}

// spoolOutput returns the channel to send queued events to, or nil when there are none
func (r *Pool) spoolOutput() chan<- []*event.Event {
	if len(r.spool) == 0 {
		return nil
	}
	return r.output
}

// nextSpool returns the next queued events to send, or nil when there are none
func (r *Pool) nextSpool() []*event.Event {
	if len(r.spool) == 0 {
		return nil
	}
	return r.spool[0].events
}

// dequeueSpool removes the queued events that were sent, resuming receiving if it was paused and there is now space
func (r *Pool) dequeueSpool() {
	sent := r.spool[0]
	copy(r.spool, r.spool[1:])
	r.spool = r.spool[:len(r.spool)-1]
	r.spoolPayloads.Add(-1)
	r.spoolSize.Add(-int64(sent.size))
	// Resume when back under the limit, or once empty in case the limit is below a single payload
	if r.paused && (r.spoolSize.Load() < r.generalConfig.MaxQueueSize || len(r.spool) == 0) {
		r.paused = false
	}
}

// sendPartialAcks resends the current acknowledgement for connections whose partial acknowledgement is due
func (r *Pool) sendPartialAcks() {
	for {
		connection := r.scheduler.Next()
		if connection == nil {
			break
		}
		r.connectionLock.Lock()
		expectedAck := r.connectionStatus[connection].progress[0]
		expectedEvent := expectedAck.event
		receiver := expectedEvent.Context().Value(transports.ContextReceiver).(transports.Receiver)
		if err := receiver.Acknowledge(expectedEvent.Context(), expectedEvent.Nonce(), expectedAck.ackedThrough); err != nil {
			r.failConnection(expectedEvent.Context(), receiver, connection, err)
		}
		r.scheduler.Set(connection, time.Second*5)
		r.connectionLock.Unlock()
	}
	r.scheduler.Reschedule()
}

// handleAcks acknowledges each run of consecutive events from the same payload as one range
func (r *Pool) handleAcks(events []*event.Event) {
	r.connectionLock.Lock()
	defer r.connectionLock.Unlock()
	position := events[0].Context().Value(poolContextEventPosition).(*poolEventPosition)
	from := position.sequence
	for _, item := range events[1:] {
		next := item.Context().Value(poolContextEventPosition).(*poolEventPosition)
		if next.progress != position.progress || next.sequence != position.sequence+1 {
			r.ackRange(position.progress, from, position.sequence)
			from = next.sequence
		}
		position = next
	}
	r.ackRange(position.progress, from, position.sequence)
}

// handleEvent processes an event from a receiver
func (r *Pool) handleEvent(receiverEvent transports.Event) {
	switch eventImpl := receiverEvent.(type) {
	case *transports.ConnectEvent:
		r.handleConnect(eventImpl)
	case transports.EventsEvent:
		r.handleEvents(eventImpl)
	case *transports.EndEvent:
		r.handleEnd(eventImpl)
	case *transports.DisconnectEvent:
		r.handleDisconnect(eventImpl)
	case *transports.StatusEvent:
		r.handleStatus(eventImpl)
	case *transports.PingEvent:
		r.handlePing(eventImpl)
	}
}

// handleConnect starts tracking a new connection, with an idle timeout
func (r *Pool) handleConnect(eventImpl *transports.ConnectEvent) {
	connection := eventImpl.Context().Value(transports.ContextConnection)
	receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
	r.startIdleTimeout(eventImpl.Context(), receiver, connection)
	r.connectionLock.Lock()
	connectionStatus := newPoolConnectionStatus(r, r.receivers[receiver].config.Name, r.receivers[receiver].listen, eventImpl.Remote(), eventImpl.Desc(), receiver.SupportsAck())
	r.connectionStatus[connection] = connectionStatus
	r.connectionLock.Unlock()
	// ReplaceEntry outside of connection lock, as entry updates that hold the entry lock will take the connection lock
	// We replace because a reconnect on the same port could occur before we get around to handling the disconnection, and we're keyed by port
	r.apiConnections.ReplaceEntry(eventImpl.Remote(), connectionStatus)
}

// handleEvents builds events from a received payload and queues them, pausing receiving once the queue is full
func (r *Pool) handleEvents(eventImpl transports.EventsEvent) {
	size := calcSize(eventImpl)
	r.connectionLock.Lock()
	connection := eventImpl.Context().Value(transports.ContextConnection)
	receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
	connectionStatus := r.connectionStatus[connection]
	var acker event.Acknowledger
	var progress *poolEventProgress
	if receiver.SupportsAck() {
		if len(r.connectionStatus[connection].progress)+1 > int(r.receivers[receiver].config.MaxPendingPayloads) {
			receiver.ShutdownConnectionRead(eventImpl.Context(), fmt.Errorf("max pending payloads exceeded"))
			r.connectionLock.Unlock()
			return
		}
		// Schedule partial ack if this is first set of events
		if len(connectionStatus.progress) == 0 {
			r.scheduler.Set(connection, 5*time.Second)
		}
		progress = &poolEventProgress{event: eventImpl}
		connectionStatus.progress = append(connectionStatus.progress, progress)
		acker = r
	} else {
		// Reset idle timeout
		r.startIdleTimeout(eventImpl.Context(), receiver, connection)
		// Count lines here as there is no ACK to do it
		connectionStatus.lines += int64(eventImpl.Count())
	}
	connectionStatus.bytes += eventImpl.Size()
	r.connectionLock.Unlock()
	// Build the events with our acknowledger and submit the bundle
	var events = make([]*event.Event, eventImpl.Count())
	var ctx context.Context
	// Raw events stay encoded so decoding happens later in the parallel processor routines
	var rawEvents [][]byte
	var decodedEvents []map[string]interface{}
	if rawEventsImpl, ok := eventImpl.(transports.RawEventsEvent); ok {
		rawEvents = rawEventsImpl.RawEvents()
	} else {
		decodedEvents = eventImpl.Events()
	}
	for idx := range events {
		if acker == nil {
			ctx = eventImpl.Context()
		} else {
			ctx = context.WithValue(eventImpl.Context(), poolContextEventPosition, &poolEventPosition{progress: progress, sequence: uint32(idx + 1)})
		}
		var item *event.Event
		if rawEvents != nil {
			item = event.NewEventFromBytes(ctx, acker, rawEvents[idx])
		} else {
			item = event.NewEvent(ctx, acker, decodedEvents[idx])
		}
		item.MustResolveMetadata("receiver", connectionStatus.metadataReceiver)
		events[idx] = item
	}
	spoolEntry := &spoolEntry{events, size}
	r.spool = append(r.spool, spoolEntry)
	r.spoolPayloads.Add(1)
	r.spoolSize.Add(int64(spoolEntry.size))
	// A full queue stops receiving so connections wait to send rather than being closed
	if r.spoolSize.Load() >= r.generalConfig.MaxQueueSize {
		r.paused = true
	}
}

// handleEnd closes a connection that finished sending, once everything it sent is acknowledged
func (r *Pool) handleEnd(eventImpl *transports.EndEvent) {
	r.connectionLock.Lock()
	defer r.connectionLock.Unlock()
	connection := eventImpl.Context().Value(transports.ContextConnection)
	if partialAcks, ok := r.connectionStatus[connection]; !ok || len(partialAcks.progress) == 0 {
		// Nothing left to ack - close send side
		receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
		receiver.ShutdownConnection(eventImpl.Context())
		// Receive side is closed, and we're just sending, so it'll close automatically once flushed, so clear all scheduled timeouts
		r.scheduler.Remove(connection)
	} else {
		// Close once the remaining payloads are acknowledged
		partialAcks.closing = true
	}
}

// handleDisconnect stops tracking a connection that has gone
func (r *Pool) handleDisconnect(eventImpl *transports.DisconnectEvent) {
	r.connectionLock.Lock()
	connection := eventImpl.Context().Value(transports.ContextConnection)
	connectionStatus := r.connectionStatus[connection]
	delete(r.connectionStatus, connection)
	r.scheduler.Remove(connection)
	r.connectionLock.Unlock()
	// RemoveEntryIfValue outside of connection lock, as entry updates that hold the entry lock will take the connection lock
	// We only remove if value matches, in case a reconnection occurred on same port before we handled disconnect that replaced it
	r.apiConnections.RemoveEntryIfValue(connectionStatus.remote, connectionStatus)
}

// handleStatus stops tracking a receiver that has finished
func (r *Pool) handleStatus(eventImpl *transports.StatusEvent) {
	if eventImpl.StatusChange() != transports.Finished {
		return
	}
	receiver := eventImpl.Context().Value(transports.ContextReceiver).(transports.Receiver)
	status := r.receivers[receiver]
	delete(r.receivers, receiver)
	// Only a receiver that is active will exist in the receiversByListen index
	// Inactive receivers are ones removed during a config reload for listens we don't want anymore
	if status.active {
		delete(r.receiversByListen, status.listen)
	}
	if !r.finished() {
		log.Info("Receiver pool waiting on %d connections to close", len(r.connectionStatus))
	}
}

// handlePing replies to a ping from an idle connection
func (r *Pool) handlePing(eventImpl *transports.PingEvent) {
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

// Acknowledge processes event acknowledgements (implements event.Acknowledger)
func (r *Pool) Acknowledge(events []*event.Event) {
	select {
	case r.ackChan <- events:
	case <-r.done:
		// The pool exits only once every connection has gone, so there is no client left to acknowledge
	}
}

// ackRange processes the acknowledgement of a range of events from a payload, which may arrive in any order
func (r *Pool) ackRange(progress *poolEventProgress, from uint32, to uint32) {
	ctx := progress.event.Context()
	connection := ctx.Value(transports.ContextConnection)
	status, ok := r.connectionStatus[connection]
	if !ok {
		// Connection was lost and this was a late acknowledgement, abandon
		return
	}

	previous := progress.ackedThrough
	if !progress.acknowledge(from, to) {
		// A repeated acknowledgement, or one held until the events before it are acknowledged
		return
	}
	status.lines += int64(progress.ackedThrough - previous)

	if progress.complete() {
		for idx, entry := range status.progress {
			if entry == progress {
				status.progress = append(status.progress[:idx], status.progress[idx+1:]...)
				break
			}
		}
	}

	lastAck := len(status.progress) == 0
	if lastAck {
		r.scheduler.Remove(connection)
	} else {
		r.scheduler.Set(connection, time.Second*5)
	}

	receiver := ctx.Value(transports.ContextReceiver).(transports.Receiver)
	if err := receiver.Acknowledge(ctx, progress.event.Nonce(), progress.ackedThrough); err != nil {
		r.failConnection(ctx, receiver, connection, err)
	} else if lastAck && status.closing {
		// The connection finished sending and everything it sent is now acknowledged, so shut it down
		receiver.ShutdownConnection(ctx)
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
				receiverApi.SetEntry("transport", api.String(cfgEntry.Transport))
				receiverApi.SetEntry("name", api.String(cfgEntry.Name))
				if newReceiversByListen[listen].SupportsAck() {
					// Only applies if supports ack
					receiverApi.SetEntry("maxPendingPayloads", api.Number(cfgEntry.MaxPendingPayloads))
				}
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

func calcSize(eventImpl transports.EventsEvent) int {
	return eventImpl.Size()
}
