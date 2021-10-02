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

package publisher

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/publisher/endpoint"
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
	"github.com/driskell/log-courier/lc-lib/transports"
)

var (
	errNetworkTimeout = errors.New("Server did not respond within network timeout")
	errNetworkPing    = errors.New("Server did not respond to keepalive")
)

const (
	// TODO(driskell): Make the idle timeout configurable like the network timeout is?
	keepaliveTimeout time.Duration = 10 * time.Second
)

// Publisher handles payloads and is responsible for passing ordered
// acknowledgements to the acknowledgement handlers
// It makes all the load balancing and distribution decisions, leaving
// transport state management to the EndpointSink
// We have always used a Push mechanism for load balancing, in the sense that
// the Publisher will push out events to transports and potentially pull them
// back if it deems there's a problem, rather than letting transports pull the
// events from the Publisher and then the transport making decisions on whether
// there is a problem. This pattern continues that tradition but with there now
// potentially being multiple transports rather than just one
type Publisher struct {
	mutex sync.RWMutex

	netConfig    *transports.Config
	adminConfig  *admin.Config
	endpointSink *endpoint.Sink
	method       method

	payloadList  internallist.List
	numPayloads  int64
	outOfSync    int
	spoolChan    chan []*event.Event
	shuttingDown bool

	lineCount       int64
	lineSpeed       float64
	lastLineCount   int64
	lastMeasurement time.Time
	secondsNoAck    int

	measurementTimer *time.Timer
	ifSpoolChan      <-chan []*event.Event
	configChan       <-chan *config.Config
	nextSpool        []*event.Event
	resendList       internallist.List
}

// NewPublisher creates a new publisher instance on the given pipeline
func NewPublisher() *Publisher {
	return &Publisher{
		spoolChan: make(chan []*event.Event, 1),
	}
}

// Init initialises configuration
func (p *Publisher) Init(cfg *config.Config) error {
	p.netConfig = transports.FetchConfig(cfg)
	p.adminConfig = cfg.Section("admin").(*admin.Config)
	p.endpointSink = endpoint.NewSink(p.netConfig)

	p.endpointSink.OnAck = p.OnAck
	p.endpointSink.OnStarted = p.OnStarted
	p.endpointSink.OnFinish = p.OnFinish
	p.endpointSink.OnFail = p.OnFail
	p.endpointSink.OnPong = p.OnPong

	p.initAPI()
	p.initMethod()

	return nil
}

// initMethod initialises the method the Publisher uses to manage multiple
// endpoints
func (p *Publisher) initMethod() {
	// TODO: Factory registration for methods
	switch p.netConfig.Method {
	case "random":
		p.method = newMethodRandom(p.endpointSink, p.netConfig)
		return
	case "failover":
		p.method = newMethodFailover(p.endpointSink, p.netConfig)
		return
	case "loadbalance":
		p.method = newMethodLoadbalance(p.endpointSink, p.netConfig)
		return
	}

	panic(fmt.Sprintf("Internal error: Unknown publishing method: %s", p.netConfig.Method))
}

// Input returns the channel that receives events to be published
// TODO: Spooler doesn't need to know of publisher, only of events
func (p *Publisher) Input() chan<- []*event.Event {
	return p.spoolChan
}

// SetConfigChan sets the config channel
func (p *Publisher) SetConfigChan(configChan <-chan *config.Config) {
	p.configChan = configChan
}

// Run starts the publisher, it handles endpoint status changes send from the
// EndpointSink so it can make payload distribution decisions
func (p *Publisher) Run() {
	p.measurementTimer = time.NewTimer(time.Second)
	p.ifSpoolChan = p.spoolChan

	for {
		if p.runOnce() {
			break
		}
	}

	log.Info("Publisher exiting")
}

// runOnce runs a single iteration of the Publisher loop
// Called continuously by Run until shutdown is completed, at which point the
// return value changed from false to true to signal completion
func (p *Publisher) runOnce() bool {
	select {
	case event := <-p.endpointSink.EventChan():
		// Endpoint Sink processes the events, and feeds back relevant changes
		if !p.endpointSink.ProcessEvent(event) {
			endpoint := event.Context().Value(endpoint.ContextSelf).(*endpoint.Endpoint)
			p.forceEndpointFailure(endpoint, fmt.Errorf("Unexpected %T message received", event))
		}

		// If all finished, we're done
		if p.shuttingDown && p.endpointSink.Count() == 0 {
			// TODO: What about out of sync ACK?
			return true
		}
	case <-p.endpointSink.TimeoutChan():
		// Process triggered timeouts
		p.endpointSink.ProcessTimeouts()
	case spool := <-p.ifSpoolChan:
		// When inputs close, sources are all closed
		if spool == nil {
			p.ifSpoolChan = nil
			p.nextSpool = nil
			p.shuttingDown = true

			// If no payloads held, nothing to wait for
			if !p.eventsHeld() && p.numPayloads == 0 {
				// If no endpoints, no shutdown necessary
				if p.endpointSink.Count() == 0 {
					return true
				}
				p.endpointSink.Shutdown()
			} else {
				log.Info("Publisher will prevent shutdown until outsanding payloads have been acknowledged or fail")
			}
			break
		}

		if p.numPayloads >= p.netConfig.MaxPendingPayloads {
			log.Debug("Maximum pending payloads of %d reached, holding %d new events", p.netConfig.MaxPendingPayloads, len(spool))
		} else if p.resendList.Len() != 0 {
			log.Debug("Holding %d new events until the resend queue is flushed", len(spool))
		} else if p.endpointSink.CanQueue() {
			if _, ok := p.sendEvents(spool); ok {
				break
			}

			log.Debug("Holding %d new events until an endpoint is ready", len(spool))
		}

		// No ready endpoint, wait for one
		p.nextSpool = spool
		p.ifSpoolChan = nil
	case <-p.measurementTimer.C:
		p.takeMeasurements()
		p.measurementTimer.Reset(time.Second)
	case config := <-p.configChan:
		p.reloadConfig(config)
	}

	return false
}

func (p *Publisher) reloadConfig(cfg *config.Config) {
	oldMethod := p.netConfig.Method
	p.netConfig = transports.FetchConfig(cfg)

	// Give sink the new config
	p.endpointSink.ReloadConfig(p.netConfig)

	// Has method changed? Init the new method and discard the old one...
	if p.netConfig.Method != oldMethod {
		p.initMethod()
	} else {
		// ...otherwise give the existing method the new configuraton
		p.method.reloadConfig(p.netConfig)
	}

	// The sink may have changed the priority endpoint after the reload, making
	// an endpoint available
	p.tryQueueHeld()
}

// OnStarted handles an endpoint that has moved from idle to now active
func (p *Publisher) OnStarted(endpoint *endpoint.Endpoint) {
	p.method.onStarted(endpoint)

	if endpoint.NumPending() != 0 {
		return
	}

	if p.tryQueueHeld() {
		return
	}

	log.Debug("[%s] Starting keepalive timeout", endpoint.Server())
	p.endpointSink.RegisterTimeout(
		&endpoint.Timeout,
		keepaliveTimeout,
		func() {
			p.timeoutKeepalive(endpoint)
		},
	)
}

// OnFinish handles when endpoints are finished
// Should return false if the endpoint is not to be reinitialised, such as when
// shutting down
func (p *Publisher) OnFinish(endpoint *endpoint.Endpoint) bool {
	// Don't recreate anything if shutting down
	if p.shuttingDown {
		return false
	}

	if endpoint.NumPending() != 0 {
		p.pullBackPending(endpoint)
	}

	// Method defines how we handle finished endpoints
	return p.method.onFinish(endpoint)
}

// OnFail handles a failed endpoint
func (p *Publisher) OnFail(endpoint *endpoint.Endpoint) {
	if endpoint.NumPending() != 0 {
		p.pullBackPending(endpoint)
	}

	// Allow method to handle what we do due to the failed endpoint
	p.method.onFail(endpoint)
}

// pullBackPending returns undelivered payloads from the endpoint back to the
// publisher for redelivery
func (p *Publisher) pullBackPending(endpoint *endpoint.Endpoint) {
	// Pull back pending payloads so we can requeue them onto other endpoints
	for _, pendingPayload := range endpoint.PullBackPending() {
		pendingPayload.Resending = true
		pendingPayload.ResetSequence()
		p.resendList.PushBack(&pendingPayload.ResendElement)
	}

	// If any ready now, requeue immediately
	p.tryQueueHeld()

	log.Debug("%d payloads held for resend", p.resendList.Len())
}

// OnAck handles acknowledgements from endpoints
// It keeps track of how many out of sync acknowldgements have been made so
// shutdown can be postponed if we've received Acks for newer events before
// older events. It also serialises the Ack offsets for correct handling
// so events are always acknowledged sequentially
func (p *Publisher) OnAck(endpoint *endpoint.Endpoint, pendingPayload *payload.Payload, firstAck bool, lineCount int) {
	// Expect next ACK within network timeout if we still have pending
	if endpoint.NumPending() > 0 {
		p.endpointSink.RegisterTimeout(
			&endpoint.Timeout,
			p.netConfig.Timeout,
			func() {
				p.timeoutPending(endpoint)
			},
		)
	} else {
		p.endpointSink.RegisterTimeout(
			&endpoint.Timeout,
			keepaliveTimeout,
			func() {
				p.timeoutKeepalive(endpoint)
			},
		)
	}

	complete := pendingPayload.Complete()

	// If we're on the resend queue and just completed, remove it
	// Handle the condition occurring where the endpoint incorrectly reports a
	// failure but then afterwards reports an acknowledgement, which means we're
	// acknowledging a payload still on the resendList
	if pendingPayload.Resending && complete {
		pendingPayload.Resending = false
		p.resendList.Remove(&pendingPayload.ResendElement)
	}

	numComplete := int64(0)

	// We potentially receive out-of-order ACKs due to payloads distributed across servers
	// This is where we enforce ordering again to ensure the handlers receive ACKs in order
	if pendingPayload == p.payloadList.Front().Value.(*payload.Payload) {
		// The out of sync count we have will never include the first payload, so
		// take the value +1
		outOfSync := p.outOfSync + 1

		// For each full payload we mark off, we decrease this count, the first we
		// mark off will always be the first payload - thus the +1. Subsequent
		// payloads are the out of sync ones - so if we mark them off we decrease
		// the out of sync count
		for pendingPayload.HasAck() {
			event.DispatchAck(pendingPayload.Rollup())

			if !pendingPayload.Complete() {
				break
			}

			p.payloadList.Remove(&pendingPayload.Element)
			outOfSync--
			p.outOfSync = outOfSync

			numComplete++

			if p.payloadList.Len() == 0 {
				break
			}

			pendingPayload = p.payloadList.Front().Value.(*payload.Payload)
		}
	} else if firstAck {
		// If this is NOT the first payload, and this is the first acknowledgement
		// for this payload, then increase out of sync payload count
		p.outOfSync++
	}

	p.mutex.Lock()
	if numComplete != 0 {
		p.numPayloads -= numComplete
	}
	p.lineCount += int64(lineCount)
	p.mutex.Unlock()

	if complete {
		// Resume sending if we stopped due to excessive pending payload count
		p.tryQueueHeld()

		// If last payload confirmed, begin shutdown
		if p.shuttingDown && !p.eventsHeld() && p.numPayloads == 0 {
			p.endpointSink.Shutdown()
		}
	}
}

// OnPong handles when endpoints receive a pong message
func (p *Publisher) OnPong(endpoint *endpoint.Endpoint) {
	// If we haven't started sending anything, return to keepalive timeout
	if endpoint.NumPending() == 0 {
		log.Debug("[%s] Resetting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(
			&endpoint.Timeout,
			keepaliveTimeout,
			func() {
				p.timeoutKeepalive(endpoint)
			},
		)
	}
}

// forceEndpointFailure is called to force an endpoint to enter
// the failed status. It reports the error and then processes the failure.
func (p *Publisher) forceEndpointFailure(endpoint *endpoint.Endpoint, err error) {
	p.endpointSink.ForceFailure(endpoint)
}

// eventsHeld returns true if there are events held waiting to be queued
func (p *Publisher) eventsHeld() bool {
	return p.resendList.Len() > 0 || p.nextSpool != nil
}

// tryQueueHeld attempts to queue held payloads
func (p *Publisher) tryQueueHeld() bool {
	if !p.eventsHeld() || !p.endpointSink.CanQueue() {
		return false
	}

	if p.resendList.Len() > 0 {
		didSend := false

		for p.resendList.Len() > 0 {
			pendingPayload := p.resendList.Front().Value.(*payload.Payload)

			// We have a payload to resend, send it now
			if _, ok := p.sendPayload(pendingPayload); ok {
				pendingPayload.Resending = false
				pendingPayload.ResetSequence()
				p.resendList.Remove(&pendingPayload.ResendElement)
				log.Debug("%d payloads remain held for resend", p.resendList.Len())
				didSend = true
			}
		}

		return didSend
	}

	// Only take from nextSpool if we have space below the limit
	if p.numPayloads < p.netConfig.MaxPendingPayloads && p.nextSpool != nil {
		// We have events, send it to the endpoint and wait for more
		if _, ok := p.sendEvents(p.nextSpool); ok {
			p.nextSpool = nil
			p.ifSpoolChan = p.spoolChan
			return true
		}
	}

	return false
}

func (p *Publisher) sendEvents(events []*event.Event) (*endpoint.Endpoint, bool) {
	pendingPayload := payload.NewPayload(events)

	p.payloadList.PushBack(&pendingPayload.Element)

	p.mutex.Lock()
	p.numPayloads++
	p.mutex.Unlock()

	return p.sendPayload(pendingPayload)
}

func (p *Publisher) sendPayload(pendingPayload *payload.Payload) (*endpoint.Endpoint, bool) {
	// Attempt to queue the payload with the best endpoint
	endpoint, err := p.endpointSink.QueuePayload(pendingPayload)
	if err != nil {
		if err == transports.ErrCongestion {
			// No need to force failure - just wait for another endpoint
			return nil, false
		}
		p.forceEndpointFailure(endpoint, err)
		return nil, false
	}

	// If this is the first payload, start the network timeout
	if endpoint.NumPending() == 1 {
		p.endpointSink.RegisterTimeout(
			&endpoint.Timeout,
			p.netConfig.Timeout,
			func() {
				p.timeoutPending(endpoint)
			},
		)
	}

	return endpoint, true
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
	p.endpointSink.RegisterTimeout(
		&endpoint.Timeout,
		p.netConfig.Timeout,
		func() {
			p.timeoutPending(endpoint)
		},
	)

	if err := endpoint.SendPing(); err != nil {
		p.forceEndpointFailure(endpoint, err)
	}
}

func (p *Publisher) takeMeasurements() {
	p.mutex.Lock()
	p.lineSpeed = core.CalculateSpeed(time.Since(p.lastMeasurement), p.lineSpeed, float64(p.lineCount-p.lastLineCount), &p.secondsNoAck)
	p.lastLineCount = p.lineCount
	p.lastMeasurement = time.Now()
	p.mutex.Unlock()
}

// initAPI initialises the publisher API entries
func (p *Publisher) initAPI() {
	// Is admin loaded into the pipeline?
	if !p.adminConfig.Enabled {
		return
	}

	publisherAPI := &api.Node{}
	publisherAPI.SetEntry("endpoints", p.endpointSink.APINavigatable())
	publisherAPI.SetEntry("status", &apiStatus{p: p})

	p.adminConfig.SetEntry("publisher", publisherAPI)
}
