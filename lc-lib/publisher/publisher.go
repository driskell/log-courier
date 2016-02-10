/*
 * Copyright 2014-2015 Jason Woods.
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
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/endpoint"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/payload"
	"github.com/driskell/log-courier/lc-lib/registrar"
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

// Publisher handles payloads and is responsible for passing ordered
// acknowledgements to the Registrar
// It makes all the load balancing and distribution decisions, leaving
// transport state management to the EndpointSink
// We have always used a Push mechanism for load balancing, in the sense that
// the Publisher will push out events to transports and potentially pull them
// back if it deems there's a problem, rather than letting transports pull the
// events from the Publisher and then the transport making decisions on whether
// there is a problem. This pattern continues that tradition but with there now
// potentially being multiple transports rather than just one
// TODO: Extrapolate the load balance / failover logic to other interfaces?
//       I'm thinking not, as the difference is very little
type Publisher struct {
	core.PipelineSegment
	core.PipelineConfigReceiver
	core.PipelineSnapshotProvider

	sync.RWMutex

	config       *config.Network
	endpointSink *endpoint.Sink

	payloadList    internallist.List
	numPayloads    int64
	outOfSync      int
	spoolChan      chan []*core.EventDescriptor
	registrarSpool registrar.EventSpooler
	shuttingDown   bool

	lineCount       int64
	lineSpeed       float64
	lastLineCount   int64
	lastMeasurement time.Time
	secondsNoAck    int

	statsTimer  *time.Timer
	onShutdown  <-chan interface{}
	ifSpoolChan <-chan []*core.EventDescriptor
	nextSpool   []*core.EventDescriptor
	resendList  internallist.List
}

// NewPublisher creates a new publisher instance on the given pipeline
func NewPublisher(pipeline *core.Pipeline, config *config.Network, registrar registrar.Registrator) *Publisher {
	ret := &Publisher{
		config:       config,
		spoolChan:    make(chan []*core.EventDescriptor, 1),
		endpointSink: endpoint.NewSink(config),
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

// Run starts the publisher, it handles endpoint status changes send from the
// EndpointSink so it can make payload distribution decisions
func (p *Publisher) Run() {
	p.statsTimer = time.NewTimer(time.Second)
	p.onShutdown = p.OnShutdown()
	p.ifSpoolChan = p.spoolChan

	for {
		if p.runOnce() {
			break
		}
	}

	p.registrarSpool.Close()

	log.Info("Publisher exiting")

	p.Done()
}

// runOnce runs a single iteration of the Publisher loop
// Called continuously by Run until shutdown is completed, at which point the
// return value changed from false to true to signal completion
func (p *Publisher) runOnce() bool {
PublisherSelect:
	select {
	case event := <-p.endpointSink.EventChan():
		// Endpoint Sink processes the events, and feeds back relevant changes
		p.endpointSink.ProcessEvent(event, p)

		// TODO: Pass shutdown back through sink?
		// If all finished, we're done
		if p.shuttingDown && p.endpointSink.Count() == 0 {
			// TODO: What about out of sync ACK?
			return true
		}
	case spool := <-p.ifSpoolChan:
		for p.endpointSink.HasReady() {
			// We have ready endpoints, send the spool
			endpoint := p.endpointSink.NextReady()

			// Do we need to reduce spool size?
			// TODO: Health related splitting of spools
			/*var sendSpool []*core.EventDescriptor
			if endpoint.Health.MaxSpoolSize >= len(spool) {
				sendSpool, spool = spool, nil
			} else {
				sendSpool, spool := p.splitSpool(spool, endpoint.Health.MaxSpoolSize)
			}*/

			log.Debug("[%s] %d new events, sending to endpoint", endpoint.Server(), len(spool))

			if err := p.sendEvents(endpoint, spool); err == nil {
				break PublisherSelect
			}
		}

		log.Debug("%d new events queued awaiting endpoint readiness", len(spool))
		// No ready endpoint, wait for one
		p.nextSpool = spool
		p.ifSpoolChan = nil
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
	case <-p.statsTimer.C:
		p.updateStatistics()
		p.statsTimer.Reset(time.Second)
	case config := <-p.OnConfig():
		p.reloadConfig(config)
	case <-p.onShutdown:
		p.onShutdown = nil
		p.ifSpoolChan = nil
		p.nextSpool = nil
		p.shuttingDown = true

		p.endpointSink.Shutdown()
	}

	return false
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

// OnAck handles acknowledgements from endpoints
// It keeps track of how many out of sync acknowldgements have been made so
// shutdown can be postponed if we've received Acks for newer events before
// older events. It also serialises the Ack offsets for correct registrar
// storage to ensure the registrar offsets are always sequential
func (p *Publisher) OnAck(endpoint *endpoint.Endpoint, pendingPayload *payload.Payload, firstAck bool) {
	// Expect next ACK within network timeout if we still have pending
	if endpoint.NumPending() != 0 {
		log.Debug("[%s] Resetting pending timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	} else {
		log.Debug("[%s] Last payload acknowledged, starting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepaliveTimeout), (*Publisher).timeoutKeepalive)
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
}

// OnPong handles when endpoints receive a pong message
func (p *Publisher) OnPong(endpoint *endpoint.Endpoint) {
	// If we haven't started sending anything, return to keepalive timeout
	if endpoint.NumPending() == 0 {
		log.Debug("[%s] Resetting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutKeepalive)
	}
}

// OnFinish handles when endpoints are finished
// Should return false if the endpoint is not to be reinitialised, such as when
// shutting down
func (p *Publisher) OnFinish(endpoint *endpoint.Endpoint) bool {
	// Don't recreate anything if shutting down
	return !p.shuttingDown
}

// OnFail handles a failed endpoint
func (p *Publisher) OnFail(endpoint *endpoint.Endpoint) {
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

// OnReady handles an endpoint that is now ready for events
func (p *Publisher) OnReady(endpoint *endpoint.Endpoint) {
	// If we're in failover mode, only send if this is the priority endpoint
	// TODO: Endpoint priority to be determined by publisher, along with load balancing
	if p.config.Method == "failover" && !p.endpointSink.IsPriorityEndpoint(endpoint) {
		log.Debug("[%s] Endpoint is standing by", endpoint.Server())
		return
	}

	if p.sendIfAvailable(endpoint) {
		return
	}

	log.Debug("[%s] Send is now ready, awaiting new events", endpoint.Server())

	if !endpoint.HasTimeout() {
		log.Debug("[%s] Starting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepaliveTimeout), (*Publisher).timeoutKeepalive)
	}
}

// forceEndpointFailure is called by Publisher to force an endpoint to enter
// the failed status. It reports the error and then processes the failure.
func (p *Publisher) forceEndpointFailure(endpoint *endpoint.Endpoint, err error) {
	log.Error("[%s] Failing endpoint: %s", endpoint.Server(), err)
	p.endpointSink.ForceFailure(endpoint)
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
