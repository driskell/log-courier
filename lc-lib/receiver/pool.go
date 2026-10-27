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
	"errors"
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

// closedConnectionRetention is how long a connection that has gone stays listed in the API
const closedConnectionRetention = 5 * time.Second

// idleConnectionTimeout is how long a connection may send nothing before it is closed
// TODO: Make configurable - it's not configurable on courier side yet and it's 900 there
const idleConnectionTimeout = 1000 * time.Second

// errIdleConnection is why a connection that sent nothing within the idle timeout is closed
var errIdleConnection = errors.New("no data received within timeout")

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
	// queue holds received events until the spooler takes them, drained fairly across connections
	queue *fairQueue
	// Atomic so the admin API can read them while Run modifies the spool
	spoolSize atomic.Int64
	// admissions counts each time events were taken from a connection's receive window, atomic for the admin API
	admissions atomic.Int64
	// admission decides when the events held in each connection's receive window are taken into the queue
	admission *admissionControl
	// shuttingDown is set once receivers are asked to shut down, so Run exits when they and their connections finish
	shuttingDown bool
	// done is closed when Run exits
	done chan struct{}
	// closedRetention is how long a connection that has gone stays listed in the API
	closedRetention time.Duration
	// idleTimeout is how long a connection may send nothing before it is closed
	idleTimeout time.Duration

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
		queue:            newFairQueue(),
		admission:        newAdmissionControl(),
		done:             make(chan struct{}),
		closedRetention:  closedConnectionRetention,
		idleTimeout:      idleConnectionTimeout,
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
	measureTicker := time.NewTicker(time.Second)
	defer measureTicker.Stop()

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
		case now := <-measureTicker.C:
			r.takeMeasurements(now)
		case events := <-r.ackChan:
			r.handleAcks(events)
		case receiverEvent := <-r.eventChan:
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

// spoolOutput returns the channel to send queued events to, or nil when there are none
func (r *Pool) spoolOutput() chan<- []*event.Event {
	if r.queue.peek() == nil {
		return nil
	}
	return r.output
}

// nextSpool returns the next queued events to send, or nil when there are none
func (r *Pool) nextSpool() []*event.Event {
	if slice := r.queue.peek(); slice != nil {
		return slice.events
	}
	return nil
}

// dequeueSpool removes the queued events that were sent, admitting waiting connections now there may be room
func (r *Pool) dequeueSpool() {
	size, _ := r.queue.pop()
	r.spoolSize.Add(-int64(size))
	r.admit()
}

// takeMeasurements measures how fast each connection is receiving lines
func (r *Pool) takeMeasurements(now time.Time) {
	r.connectionLock.Lock()
	defer r.connectionLock.Unlock()
	for _, status := range r.connectionStatus {
		status.receivedSpeed.Measure(status.received, now)
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
		receiver := expectedAck.ctx.Value(transports.ContextReceiver).(transports.Receiver)
		if err := receiver.Acknowledge(expectedAck.ctx, expectedAck.nonce, expectedAck.ackedThrough); err != nil {
			r.failConnection(expectedAck.ctx, receiver, connection, err)
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
	case *transports.AdmissionEvent:
		r.handleAdmission(eventImpl)
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

// handleAdmission takes a connection's request for the events held in its receive window to be taken, taking them
// now if they may be
func (r *Pool) handleAdmission(eventImpl *transports.AdmissionEvent) {
	connection := eventImpl.Context().Value(transports.ContextConnection)
	status, ok := r.connectionStatus[connection]
	if !ok {
		return
	}
	if status.window == nil {
		r.connectionLock.Lock()
		status.window = eventImpl.Window()
		r.connectionLock.Unlock()
	}
	r.admission.request(connection, eventImpl.Window())
	r.admit()
}

// admit takes the events held by the connections that may now proceed, given what each has queued and the room left
func (r *Pool) admit() {
	room := func() bool {
		return r.spoolSize.Load() < r.generalConfig.MaxQueueSize
	}
	r.admission.admit(r.queue.queued, room, func(request *admissionRequest) {
		r.admissions.Add(1)
		size := 0
		for _, segment := range request.window.Take() {
			size += r.receiveSegment(request.window.Context(), segment)
		}
		r.admission.charge(request.connection, size)
	})
}

// receiveSegment builds events from a segment of a received payload and queues them, returning its size
func (r *Pool) receiveSegment(ctx context.Context, segment *transports.EventsSegment) int {
	r.connectionLock.Lock()
	connection := ctx.Value(transports.ContextConnection)
	receiver := ctx.Value(transports.ContextReceiver).(transports.Receiver)
	connectionStatus, ok := r.connectionStatus[connection]
	if !ok {
		r.connectionLock.Unlock()
		return 0
	}
	var acker event.Acknowledger
	var progress *poolEventProgress
	if receiver.SupportsAck() {
		if segment.Offset == 0 {
			// Schedule partial ack if this is first set of events
			if len(connectionStatus.progress) == 0 {
				r.scheduler.Set(connection, 5*time.Second)
			}
			progress = &poolEventProgress{ctx: ctx, nonce: segment.Nonce}
			connectionStatus.progress = append(connectionStatus.progress, progress)
		} else {
			progress = connectionStatus.receiving()
		}
		progress.count += segment.Skipped + segment.Count()
		progress.final = segment.Final
		if segment.Skipped != 0 {
			// Skipped events are never queued, so they are acknowledged as soon as they are known
			r.ackRange(progress, segment.Offset+1, segment.Offset+segment.Skipped)
		} else if progress.complete() {
			r.sendAck(connectionStatus, progress)
		}
		acker = r
	} else {
		// Reset idle timeout
		r.startIdleTimeout(ctx, receiver, connection)
		// Count lines here as there is no ACK to do it
		connectionStatus.lines += int64(segment.Count())
	}
	connectionStatus.bytes += segment.Size
	connectionStatus.received += int64(segment.Skipped + segment.Count())
	r.connectionLock.Unlock()
	if segment.Count() == 0 {
		return 0
	}
	// Build the events with our acknowledger and submit the bundle
	var events = make([]*event.Event, segment.Count())
	// Raw events stay encoded so decoding happens later in the parallel processor routines
	first := segment.Offset + segment.Skipped + 1
	for idx := range events {
		eventCtx := ctx
		if acker != nil {
			eventCtx = context.WithValue(ctx, poolContextEventPosition, &poolEventPosition{progress: progress, sequence: first + uint32(idx)})
		}
		var item *event.Event
		if segment.Raw != nil {
			item = event.NewEventFromBytes(eventCtx, acker, segment.Raw[idx])
		} else {
			item = event.NewEvent(eventCtx, acker, segment.Decoded[idx])
		}
		item.MustResolveMetadata("receiver", connectionStatus.metadataReceiver)
		events[idx] = item
	}
	r.queue.push(connection, events, segment.Size)
	r.spoolSize.Add(int64(segment.Size))
	return segment.Size
}

// handleEnd closes a connection that finished sending, once everything it sent is acknowledged
func (r *Pool) handleEnd(eventImpl *transports.EndEvent) {
	r.connectionLock.Lock()
	defer r.connectionLock.Unlock()
	connection := eventImpl.Context().Value(transports.ContextConnection)
	partialAcks, ok := r.connectionStatus[connection]
	if !ok || len(partialAcks.progress) == 0 {
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

// handleDisconnect stops tracking a connection that has gone, leaving it listed in the API for a while as closed
func (r *Pool) handleDisconnect(eventImpl *transports.DisconnectEvent) {
	r.connectionLock.Lock()
	connection := eventImpl.Context().Value(transports.ContextConnection)
	connectionStatus := r.connectionStatus[connection]
	delete(r.connectionStatus, connection)
	r.scheduler.Remove(connection)
	connectionStatus.closed = true
	connectionStatus.failure = eventImpl.Err()
	r.connectionLock.Unlock()
	r.admission.forget(connection)
	r.admit()
	// Removed from its own routine, outside of the connection lock, as entry updates that hold the entry lock will take the connection lock
	// We only remove if value matches, in case a reconnection occurred on same port and replaced it
	time.AfterFunc(r.closedRetention, func() {
		r.apiConnections.RemoveEntryIfValue(connectionStatus.remote, connectionStatus)
	})
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
	status, ok := r.connectionStatus[connection]
	if ok && len(status.progress) != 0 {
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
	status, ok := r.connectionStatus[progress.ctx.Value(transports.ContextConnection)]
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
	r.sendAck(status, progress)
}

// sendAck reports how far a payload is acknowledged, and stops tracking it once it is complete
func (r *Pool) sendAck(status *poolConnectionStatus, progress *poolEventProgress) {
	ctx := progress.ctx
	connection := ctx.Value(transports.ContextConnection)
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
	if err := receiver.Acknowledge(ctx, progress.nonce, progress.ackedThrough); err != nil {
		r.failConnection(ctx, receiver, connection, err)
	} else if lastAck && status.closing {
		// The connection finished sending and everything it sent is now acknowledged, so shut it down
		receiver.ShutdownConnection(ctx)
	} else if lastAck {
		r.startIdleTimeout(ctx, receiver, connection)
	}
}

// startIdleTimeout schedules closing a connection that sends nothing within the idle timeout, which fails one that
// should be sending pings, and gracefully closes one without acknowledgements as it has no pings to keep it open
func (r *Pool) startIdleTimeout(ctx context.Context, receiver transports.Receiver, connection interface{}) {
	r.scheduler.SetCallback(connection, r.idleTimeout, func() {
		if receiver.SupportsAck() {
			r.failConnection(ctx, receiver, connection, errIdleConnection)
			return
		}
		receiver.ShutdownConnectionRead(ctx, errIdleConnection)
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
