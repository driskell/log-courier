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
	"github.com/driskell/log-courier/src/lc-lib/payload"
	"github.com/driskell/log-courier/src/lc-lib/registrar"
	"github.com/driskell/log-courier/src/lc-lib/transports"
	"sync"
	"time"
)

var (
	ErrNetworkTimeout = errors.New("Server did not respond within network timeout")
	ErrNetworkPing    = errors.New("Server did not respond to keepalive")
)

const (
	// TODO(driskell): Make the idle timeout configurable like the network timeout is?
	keepalive_timeout time.Duration = 900 * time.Second
)

type TimeoutFunc func(*Publisher, *endpoint.Endpoint)

type Publisher struct {
	core.PipelineSegment
	//core.PipelineConfigReceiver
	core.PipelineSnapshotProvider

	sync.RWMutex

	config           *config.Network
	endpointSink     *endpoint.Sink

	firstPayload     *payload.Pending
	lastPayload      *payload.Pending
	numPayloads      int64
	outOfSync        int
	spoolChan        chan []*core.EventDescriptor
	registrarSpool   registrar.EventSpooler
	shuttingDown     bool

	line_count       int64
	line_speed       float64
	last_line_count  int64
	last_measurement time.Time
	seconds_no_ack   int

	ifSpoolChan  <-chan []*core.EventDescriptor
	nextSpool    []*core.EventDescriptor
}

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

	// TODO: Option for round robin instead of load balanced?
	for _, server := range config.Servers {
		addressPool := addresspool.NewPool(server)
		ret.endpointSink.AddEndpoint(server, addressPool)
	}

	pipeline.Register(ret)

	return ret
}

func (p *Publisher) Connect() chan<- []*core.EventDescriptor {
	return p.spoolChan
}

func (p *Publisher) Run() {
	statsTimer := time.NewTimer(time.Second)
	onShutdown := p.OnShutdown()

	p.ifSpoolChan = p.spoolChan

PublishLoop:
	for {
		select {
		case endpoint := <-p.endpointSink.ReadyChan():
			p.readyEndpoint(endpoint)
		case spool := <-p.ifSpoolChan:
			for p.endpointSink.HasReady() {
				// We have ready endpoints, send the spool
				endpoint := p.endpointSink.NextReady()

				log.Debug("[%s] %d new events queued, sending to endpoint", endpoint.Server(), len(spool))
				err := p.sendPayload(endpoint, spool)

				if err == nil {
					continue PublishLoop
				}
			}

			log.Debug("%d new events queued, awaiting endpoint readiness", len(spool))
			// No ready endpoint, wait for one
			p.nextSpool = spool
			p.ifSpoolChan = nil
		case msg := <-p.endpointSink.ResponseChan():
			var err error

			endpoint := msg.Endpoint().(*endpoint.Endpoint)

			switch msg := msg.(type) {
			case *transports.AckResponse:
				err = p.processAck(endpoint, msg)
				if p.shuttingDown && p.numPayloads == 0 {
					log.Debug("Final ACK received, shutting down")
					break PublishLoop
				}
			case *transports.PongResponse:
				err = p.processPong(endpoint, msg)
			default:
				err = fmt.Errorf("[%s] BUG ASSERTION: Unknown message type \"%T\"", endpoint.Server(), msg)
			}

			if err != nil {
				p.failEndpoint(endpoint, err)
			}
		case failure := <-p.endpointSink.FailChan():
			p.failEndpoint(failure.Endpoint, failure.Error)
		case <-p.endpointSink.TimeoutChan():
			// Process triggered timers
			for {
				endpoint, callback, more := p.endpointSink.NextTimeout()
				if endpoint != nil {
					break
				}

				log.Debug("[%s] Processing timeout", endpoint.Server())
				callback.(TimeoutFunc)(p, endpoint)

				if !more {
					break
				}
			}

			p.endpointSink.ResetTimeoutTimer()
		case <-statsTimer.C:
			p.updateStatistics()
			statsTimer.Reset(time.Second)
		case <-onShutdown:
			if p.numPayloads == 0 {
				log.Debug("Publisher has no outstanding payloads, shutting down")
				break PublishLoop
			}

			log.Warning("Publisher has outstanding payloads, waiting for responses before shutting down")
			onShutdown = nil
			p.ifSpoolChan = nil
			p.shuttingDown = true
		}
	}

	p.endpointSink.Shutdown()
	p.endpointSink.Wait()
	p.registrarSpool.Close()

	log.Info("Publisher exiting")

	p.Done()
}

func (p *Publisher) sendPayload(endpoint *endpoint.Endpoint, events []*core.EventDescriptor) error {
	payload := payload.NewPending(events)

	if p.firstPayload == nil {
		p.firstPayload = payload
	} else {
		p.lastPayload.NextPayload = payload
	}
	p.lastPayload = payload

	p.Lock()
	p.numPayloads++
	p.Unlock()

	// Don't queue if send fails and fail the endpoint
	if err := endpoint.SendPayload(payload); err != nil {
		p.failEndpoint(endpoint, err)
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
	payload, firstAck := endpoint.ProcessAck(msg)

	// We potentially receive out-of-order ACKs due to payloads distributed across servers
	// This is where we enforce ordering again to ensure registrar receives ACK in order
	if payload == p.firstPayload {
		// The out of sync count we have will never include the first payload, so
		// take the value +1
		outOfSync := p.outOfSync + 1

		// For each full payload we mark off, we decrease this count, the first we
		// mark off will always be the first payload - thus the +1. Subsequent
		// payloads are the out of sync ones - so if we mark them off we decrease
		// the out of sync count
		for payload.HasAck() {
			p.registrarSpool.Add(registrar.NewAckEvent(payload.Rollup()))

			if !payload.Complete() {
				break
			}

			payload = payload.NextPayload
			p.firstPayload = payload
			outOfSync--
			p.outOfSync = outOfSync

			p.Lock()
			p.numPayloads--
			p.Unlock()

			// TODO: Resume sending if we stopped due to excessive pending payload count
			//if !p.shutdown && p.can_send == nil {
			//	p.can_send = p.transport.CanSend()
			//}

			if payload == nil {
				break
			}
		}

		p.registrarSpool.Send()
	} else if firstAck {
		// If this is NOT the first payload, and this is the first acknowledgement
		// for this payload, then increase out of sync payload count
		p.outOfSync++
	}

	// Expect next ACK within network timeout if we still have pending
	if endpoint.NumPending() != 0 {
		log.Debug("[%s] Resetting pending timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	} else {
		log.Debug("[%s] Last payload acknowledged, starting keepalive timeout", endpoint.Server())
		p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepalive_timeout), (*Publisher).timeoutKeepalive)
	}

	// If we're no longer full, move to ready queue
	// TODO: Use "peer send queue" - Move logic to endpoint.EndpointSink
	if endpoint.IsFull() && endpoint.NumPending() < 4 {
		log.Debug("[%s] endpoint.Endpoint is no longer full (%d pending payloads)", endpoint.Server(), endpoint.NumPending())
		p.readyEndpoint(endpoint)
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

func (p *Publisher) failEndpoint(endpoint *endpoint.Endpoint, err error) {
	log.Error("[%s] endpoint.Endpoint failed: %s", endpoint.Server(), err)
	// TODO:
}

func (p *Publisher) readyEndpoint(endpoint *endpoint.Endpoint) {
	// TODO: Make configurable (bring back the "peer send queue" setting)
	if endpoint.NumPending() >= 4 {
		log.Debug("[%s] endpoint.Endpoint is full (%d pending payloads)", endpoint.Server(), endpoint.NumPending())
		p.endpointSink.RegisterFull(endpoint)
		return
	}

	if p.nextSpool != nil {
		log.Debug("[%s] Send is now ready, sending %d queued events", endpoint.Server(), len(p.nextSpool))
		// We have events, send it to the endpoint and wait for more
		if err := p.sendPayload(endpoint, p.nextSpool); err == nil {
			p.nextSpool = nil
			p.ifSpoolChan = p.spoolChan
		}
	} else {
		log.Debug("[%s] Send is now ready, awaiting new events", endpoint.Server())
		// No events, save on the ready list and start the keepalive timer if none set
		p.endpointSink.RegisterReady(endpoint)
		if !endpoint.HasTimeout() {
			log.Debug("[%s] Starting keepalive timeout", endpoint.Server())
			p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(keepalive_timeout), (*Publisher).timeoutKeepalive)
		}
	}
}

func (p *Publisher) timeoutPending(endpoint *endpoint.Endpoint) {
	// Trigger a failure
	if endpoint.IsPinging() {
		p.failEndpoint(endpoint, ErrNetworkPing)
	} else {
		p.failEndpoint(endpoint, ErrNetworkTimeout)
	}
}

func (p *Publisher) timeoutKeepalive(endpoint *endpoint.Endpoint) {
	// Timeout for PING
	log.Debug("[%s] Sending PING and starting pending timeout", endpoint.Server())
	p.endpointSink.RegisterTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)

	if err := endpoint.SendPing(); err != nil {
		p.failEndpoint(endpoint, err)
	}
}

func (p *Publisher) updateStatistics() {
	p.Lock()

	p.line_speed = core.CalculateSpeed(time.Since(p.last_measurement), p.line_speed, float64(p.line_count-p.last_line_count), &p.seconds_no_ack)

	p.last_line_count = p.line_count
	p.last_measurement = time.Now()

	p.Unlock()
}

func (p *Publisher) Snapshot() []*core.Snapshot {
	p.RLock()

	snapshot := core.NewSnapshot("Publisher")

	snapshot.AddEntry("Speed (Lps)", p.line_speed)
	snapshot.AddEntry("Published lines", p.last_line_count)
	snapshot.AddEntry("Pending Payloads", p.numPayloads)

	p.RUnlock()

	return []*core.Snapshot{snapshot}
}
