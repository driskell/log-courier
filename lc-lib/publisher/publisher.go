/*
 * Copyright 2014 Jason Woods.
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

package publisher

import (
	"errors"
	"fmt"
	"github.com/driskell/log-courier/src/lc-lib/addresspool"
	"github.com/driskell/log-courier/src/lc-lib/config"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/endpoint"
	"github.com/driskell/log-courier/src/lc-lib/internallist"
	"github.com/driskell/log-courier/src/lc-lib/payload"
	"github.com/driskell/log-courier/src/lc-lib/registrar"
	"github.com/driskell/log-courier/src/lc-lib/transports"
	"sync"
	"time"
)

var (
	errNetworkTimeout = errors.New("Server did not respond within network timeout")
	errNetworkPing    = errors.New("Server did not respond to keepalive")
)

const (
	// TODO(driskell): Make the idle timeout configurable like the network timeout is?
	keepaliveTimeout time.Duration = 900 * time.Second
)

type timeoutFunc func(*Publisher, *endpoint.Endpoint)

// Publisher handles payload and is responsible for passing ordered
// acknowledgements to the Registrar
type Publisher struct {
	core.PipelineSegment
	core.PipelineConfigReceiver
	core.PipelineSnapshotProvider

	sync.RWMutex

	config           *config.Network
	endpointSink     *endpoint.Sink

	payloadList      internallist.List
	numPayloads      int64
	outOfSync        int
	spoolChan        chan []*core.EventDescriptor
	registrarSpool   registrar.EventSpooler
	shuttingDown     bool

	lineCount       int64
	lineSpeed       float64
	lastLineCount  int64
	lastMeasurement time.Time
	secondsNoAck   int

	ifSpoolChan  <-chan []*core.EventDescriptor
	nextSpool    []*core.EventDescriptor
	resendList   internallist.List
}

// NewPublisher creates a new publisher instance on the given pipeline
func NewPublisher(pipeline *core.Pipeline, config *config.Network, registrar registrar.Registrator) *Publisher {
	ret := &Publisher{
		config: config,
		endpointSink: endpoint.NewSink(config),
		spoolChan: make(chan []*core.EventDescriptor, 1),
	}

	if registrar == nil {
		ret.registrarSpool = newNullEventSpool()
	} else {
		ret.registrarSpool = registrar.Connect()
	}

	pipeline.Register(ret)

	return ret
}

// Connect is used by Spooler
// TODO: Spooler doesn't need to know of publisher, only of events
func (p *Publisher) Connect() chan<- []*core.EventDescriptor {
	return p.spoolChan
}

// Run starts the publisher, this is usually started by the pipeline in its own
// routine
func (p *Publisher) Run() {
	statsTimer := time.NewTimer(time.Second)
	onShutdown := p.OnShutdown()

	p.ifSpoolChan = p.spoolChan

PublishLoop:
	for {
		select {
		case status := <-p.endpointSink.StatusChan():
			switch status.Status {
			case endpoint.Ready:
				// Ignore late messages from a closing endpoint
				if !status.Endpoint.IsClosing() {
					p.readyEndpoint(status.Endpoint)
				}
			case endpoint.Recovered:
				if !status.Endpoint.IsClosing() {
					p.recoverEndpoint(status.Endpoint)
					p.readyEndpoint(status.Endpoint)
				}
			case endpoint.Finished:
				if p.finishEndpoint(status.Endpoint) {
					break PublishLoop
				}
			default:
				p.failEndpoint(status.Endpoint)
			}
		case spool := <-p.ifSpoolChan:
			for p.endpointSink.HasReady() {
				// We have ready endpoints, send the spool
				endpoint := p.endpointSink.NextReady()

				log.Debug("[%s] %d new events, sending to endpoint", endpoint.Server(), len(spool))
				err := p.sendEvents(endpoint, spool)

				if err == nil {
					continue PublishLoop
				}
			}

			log.Debug("%d new events queued awaiting endpoint readiness", len(spool))
			// No ready endpoint, wait for one
			p.nextSpool = spool
			p.ifSpoolChan = nil
		case msg := <-p.endpointSink.ResponseChan():
			var err error

			endpoint := msg.Endpoint().(*endpoint.Endpoint)

			switch msg := msg.(type) {
			case *transports.AckResponse:
				err = p.processAck(endpoint, msg)
				if err == nil && p.shuttingDown && p.numPayloads == 0 {
					log.Debug("Final ACK received, shutting down")
					p.endpointSink.Shutdown()
				}
			case *transports.PongResponse:
				err = p.processPong(endpoint, msg)
			default:
				err = fmt.Errorf("[%s] BUG ASSERTION: Unknown message type \"%T\"", endpoint.Server(), msg)
			}

			if err != nil {
				p.forceEndpointFailure(endpoint, err)
			}
		case <-p.endpointSink.TimeoutChan():
			// Process triggered timers
			for {
				endpoint, callback, more := p.endpointSink.NextTimeout()
				if endpoint != nil {
					break
				}

				log.Debug("[%s] Processing timeout", endpoint.Server())
				callback.(timeoutFunc)(p, endpoint)

				if !more {
					break
				}
			}

			p.endpointSink.ResetTimeoutTimer()
		case <-statsTimer.C:
			p.updateStatistics()
			statsTimer.Reset(time.Second)
		case config := <-p.OnConfig():
			p.reloadConfig(config)
		case <-onShutdown:
			onShutdown = nil
			p.ifSpoolChan = nil
			p.nextSpool = nil
			p.shuttingDown = true

			p.endpointSink.Shutdown()
		}
	}

	p.registrarSpool.Close()

	log.Info("Publisher exiting")

	p.Done()
}

func (p *Publisher) reloadConfig(config *config.Config) {
	p.config = &config.Network

	// Give sink the new config, and allow it to shutdown removed servers
	p.endpointSink.ReloadConfig(&config.Network)

	// The sink may have changed the priority endpoint after the reload, making
	// an endpoint available for send
	if p.eventsAvailable() && p.endpointSink.HasReady() {
		p.sendIfAvailable(p.endpointSink.NextReady())
	}

	// TODO: If MaxPendingPayloads is changed, update which endpoints should
	//       be marked as full
}

func (p *Publisher) sendEvents(endpoint *endpoint.Endpoint, events []*core.EventDescriptor) error {
	pendingPayload := payload.NewPayload(events)

	p.payloadList.PushBack(&pendingPayload.Element)

	p.Lock()
	p.numPayloads++
	p.Unlock()

	return p.sendPayload(endpoint, pendingPayload)
}

func (p *Publisher) sendPayload(endpoint *endpoint.Endpoint, pendingPayload *payload.Payload) error {
	// Don't queue if send fails and fail the endpoint
	if err := endpoint.SendPayload(pendingPayload); err != nil {
		p.forceEndpointFailure(endpoint, err)
		return err
	}

	// If this is the first payload, start the network timeout
	if endpoint.NumPending() == 1 {
		log.Debug("[%s] First payload, starting pending timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	}

	return nil
}

func (p *Publisher) processAck(endpoint *endpoint.Endpoint, msg *transports.AckResponse) error {
	pendingPayload, firstAck := endpoint.ProcessAck(msg)

	// Expect next ACK within network timeout if we still have pending
	if endpoint.NumPending() != 0 {
		log.Debug("[%s] Resetting pending timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	} else {
		log.Debug("[%s] Last payload acknowledged, starting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepaliveTimeout), (*Publisher).timeoutKeepalive)
	}

	// If we're no longer full, move to ready queue
	if endpoint.IsFull() && endpoint.NumPending() < int(p.config.MaxPendingPayloads) {
		log.Debug("[%s] Endpoint is no longer full (%d pending payloads)", endpoint.Server(), endpoint.NumPending())
		p.readyEndpoint(endpoint)
	}

	// Did the endpoint actually process the ACK?
	if pendingPayload == nil {
		return nil
	}

	// If we're on the resend queue and just completed, remove it
	// This prevents a condition occurring where the endpoint incorrectly reports
	// a failure but then afterwards reports an acknowledgement
	if pendingPayload.Resending && pendingPayload.Complete() {
		log.Debug("[%s] Retransmission was successful", endpoint.Server())
		p.resendList.Remove(&pendingPayload.ResendElement)
	}

	// We potentially receive out-of-order ACKs due to payloads distributed across servers
	// This is where we enforce ordering again to ensure registrar receives ACK in order
	if pendingPayload == p.payloadList.Front().Value.(*payload.Payload) {
		// The out of sync count we have will never include the first payload, so
		// take the value +1
		outOfSync := p.outOfSync + 1

		// For each full payload we mark off, we decrease this count, the first we
		// mark off will always be the first payload - thus the +1. Subsequent
		// payloads are the out of sync ones - so if we mark them off we decrease
		// the out of sync count
		for pendingPayload.HasAck() {
			p.registrarSpool.Add(registrar.NewAckEvent(pendingPayload.Rollup()))

			if !pendingPayload.Complete() {
				break
			}

			p.payloadList.Remove(&pendingPayload.Element)
			outOfSync--
			p.outOfSync = outOfSync

			p.Lock()
			p.numPayloads--
			p.Unlock()

			// TODO: Resume sending if we stopped due to excessive pending payload count
			//if !p.shutdown && p.can_send == nil {
			//	p.can_send = p.transport.CanSend()
			//}

			if p.payloadList.Len() == 0 {
				break
			}

			pendingPayload = p.payloadList.Front().Value.(*payload.Payload)
		}

		p.registrarSpool.Send()
	} else if firstAck {
		// If this is NOT the first payload, and this is the first acknowledgement
		// for this payload, then increase out of sync payload count
		p.outOfSync++
	}

	return nil
}

func (p *Publisher) processPong(endpoint *endpoint.Endpoint, msg *transports.PongResponse) error {
	if err := endpoint.ProcessPong(); err != nil {
		return err
	}

	// If we haven't started sending anything, return to keepalive timeout
	if endpoint.NumPending() == 0 {
		log.Debug("[%s] Resetting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutKeepalive)
	}

	return nil
}

// recoverEndpoint moves an endpoint back into the idle status
func (p *Publisher) recoverEndpoint(endpoint *endpoint.Endpoint) {
	if endpoint.IsFailed() {
		log.Info("[%s] Endpoint recovered", endpoint.Server())
		p.endpointSink.RecoverFailed(endpoint)
	}
}

// finishEndpoint removes a finished endpoint, recreating it if it is still in
// in the active configuration. Returns true if shutdown is completed
func (p *Publisher) finishEndpoint(endpoint *endpoint.Endpoint) bool {
	log.Debug("[%s] Endpoint has finished", endpoint.Server())
	server := endpoint.Server()
	p.endpointSink.RemoveEndpoint(server)

	// Don't recreate anything if shutting down
	if p.shuttingDown {
		if p.endpointSink.Count() == 0 {
			// Finished shutting down!
			return true
		}
		return false
	}

	// Recreate the endpoint if it's still in the config
	for _, item := range p.config.Servers {
		if item == server {
			p.endpointSink.AddEndpoint(server, addresspool.NewPool(server))
			break
		}
	}

	return false
}

// failEndpoint marks an endpoint as failed
func (p *Publisher) failEndpoint(endpoint *endpoint.Endpoint) {
	log.Info("[%s] Marking endpoint as failed", endpoint.Server())
	p.endpointSink.RegisterFailed(endpoint)

	// Pull back pending payloads so we can requeue them onto other endpoints
	for _, pendingPayload := range endpoint.PullBackPending() {
		pendingPayload.Resending = true
		log.Debug("PushBack %v - %v", pendingPayload.ResendElement, p.resendList)
		p.resendList.PushBack(&pendingPayload.ResendElement)
		log.Debug("PushBack done")
	}

	// If any ready now, requeue immediately
	for p.resendList.Len() != 0 && p.endpointSink.HasReady() {
		// We have ready endpoints, send the spool
		endpoint := p.endpointSink.NextReady()
		pendingPayload := p.resendList.Front().Value.(*payload.Payload)

		log.Debug("[%s] %d events require resending, sending to endpoint", endpoint.Server(), len(pendingPayload.Events()))
		err := p.sendPayload(endpoint, pendingPayload)

		if err == nil {
			pendingPayload.Resending = false
			p.resendList.Remove(&pendingPayload.ResendElement)
		}
	}

	log.Debug("%d payloads held for resend", p.resendList.Len())
}

// forceEndpointFailure is called by Publisher to force an endpoint to enter
// the failed status. It reports the error and then processes the failure.
func (p *Publisher) forceEndpointFailure(endpoint *endpoint.Endpoint, err error) {
	log.Error("[%s] An error occurred: %s", endpoint.Server(), err)
	p.failEndpoint(endpoint)
}

func (p *Publisher) readyEndpoint(endpoint *endpoint.Endpoint) {
	if endpoint.NumPending() >= int(p.config.MaxPendingPayloads) {
		log.Debug("[%s] Endpoint is full (%d pending payloads)", endpoint.Server(), endpoint.NumPending())
		p.endpointSink.RegisterFull(endpoint)
		return
	}

	// If we're in failover mode, only send if this is the priority endpoint
	if p.config.Method == "failover" && !p.endpointSink.IsPriorityEndpoint(endpoint) {
		return
	}

	if p.sendIfAvailable(endpoint) {
		return
	}

	log.Debug("[%s] Send is now ready, awaiting new events", endpoint.Server())

	// No events, save on the ready list and start the keepalive timer if none set
	p.endpointSink.RegisterReady(endpoint)

	if !endpoint.HasTimeout() {
		log.Debug("[%s] Starting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepaliveTimeout), (*Publisher).timeoutKeepalive)
	}
}

func (p *Publisher) eventsAvailable() bool {
	return p.resendList.Len() != 0 || p.nextSpool != nil
}

func (p *Publisher) sendIfAvailable(endpoint *endpoint.Endpoint) bool {
	if p.resendList.Len() != 0 {
		pendingPayload := p.resendList.Front().Value.(*payload.Payload)

		log.Debug("[%s] Send is now ready, resending %d events", endpoint.Server(), len(pendingPayload.Events()))

		// We have a payload to resend, send it now
		if err := p.sendPayload(endpoint, pendingPayload); err == nil {
			pendingPayload.Resending = false
			p.resendList.Remove(&pendingPayload.ResendElement)
			log.Debug("%d payloads remain held for resend", p.resendList.Len())
		}

		return true
	}

	if p.nextSpool != nil {
		log.Debug("[%s] Send is now ready, sending %d queued events", endpoint.Server(), len(p.nextSpool))

		// We have events, send it to the endpoint and wait for more
		if err := p.sendEvents(endpoint, p.nextSpool); err == nil {
			p.nextSpool = nil
			p.ifSpoolChan = p.spoolChan
		}

		return true
	}

	return false
}

func (p *Publisher) timeoutPending(endpoint *endpoint.Endpoint) {
	// Trigger a failure
	if endpoint.IsPinging() {
		p.forceEndpointFailure(endpoint, errNetworkPing)
	} else {
		p.forceEndpointFailure(endpoint, errNetworkTimeout)
	}
}

func (p *Publisher) timeoutKeepalive(endpoint *endpoint.Endpoint) {
	// Timeout for PING
	log.Debug("[%s] Sending PING and starting pending timeout", endpoint.Server())
	p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)

	if err := endpoint.SendPing(); err != nil {
		p.forceEndpointFailure(endpoint, err)
	}
}

func (p *Publisher) updateStatistics() {
	p.Lock()

	p.lineSpeed = core.CalculateSpeed(time.Since(p.lastMeasurement), p.lineSpeed, float64(p.lineCount-p.lastLineCount), &p.secondsNoAck)

	p.lastLineCount = p.lineCount
	p.lastMeasurement = time.Now()

	p.Unlock()
}

// Snapshot returns a snapshot of the current publisher status. This is normally
// called by the pipeline when a pipeline snapshot is requested
func (p *Publisher) Snapshot() []*core.Snapshot {
	p.RLock()

	snapshot := core.NewSnapshot("Publisher")

	snapshot.AddEntry("Speed (Lps)", p.lineSpeed)
	snapshot.AddEntry("Published lines", p.lastLineCount)
	snapshot.AddEntry("Pending Payloads", p.numPayloads)

	p.RUnlock()

	return []*core.Snapshot{snapshot}
}
