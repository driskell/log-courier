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
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/registrar"
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

type TimeoutFunc func(*Publisher, *Endpoint)

type Publisher struct {
	core.PipelineSegment
	//core.PipelineConfigReceiver
	core.PipelineSnapshotProvider

	sync.RWMutex

	config           *core.NetworkConfig
	endpointSink     *EndpointSink

	firstPayload     *PendingPayload
	lastPayload      *PendingPayload
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

	timeoutTimer *time.Timer
	timeoutHead  *Endpoint
	ifReadyChan  <-chan *Endpoint
	readyHead    *Endpoint
	ifSpoolChan  <-chan []*core.EventDescriptor
	nextSpool    []*core.EventDescriptor
}

func NewPublisher(pipeline *core.Pipeline, config *core.NetworkConfig, registrar registrar.Registrator) *Publisher {
	ret := &Publisher{
		config: config,
		endpointSink: NewEndpointSink(config),
		spoolChan: make(chan []*core.EventDescriptor, 1),
		timeoutTimer: time.NewTimer(1 * time.Second),
	}

	ret.timeoutTimer.Stop()

	if registrar == nil {
		ret.registrarSpool = newNullEventSpool()
	} else {
		ret.registrarSpool = registrar.Connect()
	}

	// TODO: Option for round robin instead of load balanced?
	for _, server := range config.Servers {
		addressPool := NewAddressPool(server)
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

	p.waitReady()
	p.waitSpool()

PublishLoop:
	for {
		select {
		case endpoint := <-p.ifReadyChan:
		log.Debug("Endpoint ready")
			if p.nextSpool != nil {
				// We have events, send it to the endpoint and wait for more
				p.sendPayload(endpoint, p.nextSpool)
				p.nextSpool = nil
				p.waitSpool()
			} else {
				// No events, save on the ready list and start the keepalive timer if none set
				p.registerReady(endpoint)
				if endpoint.TimeoutFunc == nil {
					p.registerTimeout(endpoint, time.Now().Add(keepalive_timeout), (*Publisher).timeoutKeepalive)
				}

				p.ifReadyChan = nil
			}
		case spool := <-p.ifSpoolChan:
			log.Debug("Spool received")
			if p.readyHead != nil {
				// We have a ready endpoint, send the spool to it
				p.sendPayload(p.readyHead, spool)
				p.readyHead = p.readyHead.NextReady
				p.waitReady()
			} else {
				// No ready endpoint, wait for one
				p.nextSpool = spool
				p.ifSpoolChan = nil
			}
		case msg := <-p.endpointSink.ResponseChan:
			var err error
			switch msg.Response.(type) {
			case *AckResponse:
				err = p.processAck(msg.Endpoint(), msg.Response.(*AckResponse))
				if p.shuttingDown && p.numPayloads == 0 {
					break PublishLoop
				}
			case *PongResponse:
				err = p.processPong(msg.Endpoint(), msg.Response.(*PongResponse))
			default:
				err = fmt.Errorf("BUG: Unknown message type \"%T\"", msg)
			}
			if err != nil {
				p.failEndpoint(msg.Endpoint(), err)
			}
		case failure := <-p.endpointSink.FailChan:
			p.failEndpoint(failure.Endpoint, failure.Error)
		case <-p.timeoutTimer.C:
			// Process triggered timers
			for {
				endpoint := p.timeoutHead
				p.timeoutHead = p.timeoutHead.NextTimeout

				callback := endpoint.TimeoutFunc
				endpoint.TimeoutFunc = nil
				callback(p, endpoint)

				if p.timeoutHead == nil || p.timeoutHead.TimeoutDue.After(time.Now()) {
					continue
				}
			}
		case <-statsTimer.C:
			p.updateStatistics()
			statsTimer.Reset(time.Second)
		case <-onShutdown:
			if p.numPayloads == 0 {
				break PublishLoop
			}

			onShutdown = nil
			p.ifSpoolChan = nil
			p.shuttingDown = true
		}
		log.Debug("Looping")
	}

	p.endpointSink.Shutdown()
	p.endpointSink.Wait()
	p.registrarSpool.Close()

	log.Info("Publisher exiting")

	p.Done()
}

func (p *Publisher) waitReady() {
	p.ifReadyChan = p.endpointSink.ReadyChan
}

func (p *Publisher) waitSpool() {
	p.ifSpoolChan = p.spoolChan
}

func (p *Publisher) sendPayload(endpoint *Endpoint, events []*core.EventDescriptor) {
	// If this is the first payload, start the network timeout
	if !endpoint.HasPending() {
		p.registerTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	}

	payload, err := NewPendingPayload(events)
	if err != nil {
		return
	}

	endpoint.SendPayload(payload)

	if p.firstPayload == nil {
		p.firstPayload = payload
	} else {
		p.lastPayload.nextPayload = payload
	}
	p.lastPayload = payload

	p.Lock()
	p.numPayloads++
	p.Unlock()
}

func (p *Publisher) processAck(endpoint *Endpoint, msg *AckResponse) error {
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

			payload = payload.nextPayload
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
	if endpoint.HasPending() {
		p.registerTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)
	}

	return nil
}

func (p *Publisher) processPong(endpoint *Endpoint, msg *PongResponse) error {
	// TODO: Move to endpoint
	if !endpoint.PongPending {
		return fmt.Errorf("Unexpected PONG received")
	}

	endpoint.PongPending = false

	// If we haven't started sending anything, return to keepalive timeout
	if !endpoint.HasPending() {
		p.registerTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutKeepalive)
	}

	return nil
}

func (p *Publisher) failEndpoint(endpoint *Endpoint, err error) {
	// TODO:
}

func (p *Publisher) registerReady(endpoint *Endpoint) {
	// TODO: Enhance this to be fairer - maybe least pending payloads order
	if p.readyHead != nil {
		p.readyHead.NextReady = endpoint
	}
	p.readyHead = endpoint
}

func (p *Publisher) registerTimeout(endpoint *Endpoint, timeoutDue time.Time, timeoutFunc TimeoutFunc) {
	endpoint.TimeoutFunc = timeoutFunc
	endpoint.TimeoutDue = timeoutDue

	head := p.timeoutHead

	if head == nil || head.TimeoutDue.After(timeoutDue) {
		p.timeoutHead = endpoint
		endpoint.NextTimeout = head
		return
	}

	var prev *Endpoint
	for prev = head; head != nil; prev, head = head, head.NextTimeout {
		if head.TimeoutDue.After(timeoutDue) {
			prev.NextTimeout = endpoint
			endpoint.NextTimeout = head
			return
		}
	}

	prev.NextTimeout = endpoint
	endpoint.NextTimeout = nil
}

func (p *Publisher) timeoutPending(endpoint *Endpoint) {
	// Trigger a failure
	if endpoint.PongPending {
		endpoint.Remote.Fail(ErrNetworkTimeout)
	} else {
		endpoint.Remote.Fail(ErrNetworkPing)
	}
}

func (p *Publisher) timeoutKeepalive(endpoint *Endpoint) {
	// Timeout for PING
	p.registerTimeout(endpoint, time.Now().Add(p.config.Timeout), (*Publisher).timeoutPending)

	endpoint.SendPing()
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







/*
func (p *Publisher) RunOld() {
	defer func() {
		p.Done()
	}()

	var input_toggle <-chan []*core.EventDescriptor
	var retry_payload *pendingPayload
	var err error
	var reload int
	var hold bool

	timer := time.NewTimer(keepalive_timeout)
	stats_timer := time.NewTimer(time.Second)

	control_signal := p.OnShutdown()
	delay_shutdown := func() {
		// Flag shutdown for when we finish pending payloads
		// TODO: Persist pending payloads and resume? Quicker shutdown
		log.Warning("Delaying shutdown to wait for pending responses from the server")
		control_signal = nil
		p.shutdown = true
		p.can_send = nil
		input_toggle = nil
	}

PublishLoop:
	for {
		// Do we need to reload transport?
		if reload == core.Reload_Transport {
			// Shutdown and reload transport
			p.transport.Shutdown()

			if err = p.loadTransport(); err != nil {
				log.Error("The new transport configuration failed to apply: %s", err)
			}

			reload = core.Reload_None
		} else if reload != core.Reload_None {
			reload = core.Reload_None
		}

		if err = p.transport.Init(); err != nil {
			log.Error("Transport init failed: %s", err)

			now := time.Now()
			reconnect_due := now.Add(p.config.Reconnect)

		ReconnectTimeLoop:
			for {

				select {
				case <-time.After(reconnect_due.Sub(now)):
					break ReconnectTimeLoop
				case <-control_signal:
					// TODO: Persist pending payloads and resume? Quicker shutdown
					if p.num_payloads == 0 {
						break PublishLoop
					}

					delay_shutdown()
				case config := <-p.OnConfig():
					// Apply and check for changes
					reload = p.reloadConfig(&config.Network)

					// If a change and no pending payloads, process immediately
					if reload != core.Reload_None && p.num_payloads == 0 {
						break ReconnectTimeLoop
					}
				}

				now = time.Now()
				if now.After(reconnect_due) {
					break
				}
			}

			continue
		}

		p.Lock()
		p.status = Status_Connected
		p.Unlock()

		timer.Reset(keepalive_timeout)
		stats_timer.Reset(time.Second)

		p.pending_ping = false
		input_toggle = nil
		hold = false
		p.can_send = p.transport.CanSend()

	SelectLoop:
		for {
			select {
			case <-p.can_send:
				// Resend payloads from full retry first
				if retry_payload != nil {
					// Do we need to regenerate the payload?
					if retry_payload.payload == nil {
						if err = retry_payload.Generate(); err != nil {
							break SelectLoop
						}
					}

					// Reset timeout
					retry_payload.timeout = time.Now().Add(p.config.Timeout)

					log.Debug("Send now open: Retrying next payload")

					// Send the payload again
					if err = p.transport.Write("JDAT", retry_payload.payload); err != nil {
						break SelectLoop
					}

					// Expect an ACK within network timeout if this is the first of the retries
					if p.first_payload == retry_payload {
						timer.Reset(p.config.Timeout)
					}

					// Move to next non-empty payload
					for {
						retry_payload = retry_payload.next
						if retry_payload == nil || retry_payload.ack_events != len(retry_payload.events) {
							break
						}
					}

					break
				} else if p.out_of_sync != 0 {
					var resent bool
					if resent, err = p.checkResend(); err != nil {
						break SelectLoop
					} else if resent {
						log.Debug("Send now open: Resent a timed out payload")
						// Expect an ACK within network timeout
						timer.Reset(p.config.Timeout)
						break
					}
				}

				// No pending payloads, are we shutting down? Skip if so
				if p.shutdown {
					break
				}

				log.Debug("Send now open: Awaiting events for new payload")

				// Too many pending payloads, hold sending more until some are ACK
				if p.num_payloads >= p.config.MaxPendingPayloads {
					hold = true
				} else {
					input_toggle = p.input
				}
			case events := <-input_toggle:
				log.Debug("Sending new payload of %d events", len(events))

				// Send
				if err = p.sendNewPayload(events); err != nil {
					break SelectLoop
				}

				// Wait for send signal again
				input_toggle = nil

				if p.num_payloads >= p.config.MaxPendingPayloads {
					log.Debug("Pending payload limit of %d reached", p.config.MaxPendingPayloads)
				} else {
					log.Debug("%d/%d pending payloads now in transit", p.num_payloads, p.config.MaxPendingPayloads)
				}

				// Expect an ACK within network timeout if this is first payload after idle
				// Otherwise leave the previous timer
				if p.num_payloads == 1 {
					timer.Reset(p.config.Timeout)
				}
			case data := <-p.transport.Read():
				var signature, message []byte

				// Error? Or data?
				switch data.(type) {
				case error:
					err = data.(error)
					break SelectLoop
				default:
					signature = data.([][]byte)[0]
					message = data.([][]byte)[1]
				}

				switch {
				case bytes.Compare(signature, []byte("PONG")) == 0:
					if err = p.processPong(message); err != nil {
						break SelectLoop
					}
				case bytes.Compare(signature, []byte("ACKN")) == 0:
					if err = p.processAck(message); err != nil {
						break SelectLoop
					}
				default:
					err = fmt.Errorf("Unknown message received: % X", signature)
					break SelectLoop
				}

				// If no more pending payloads, set keepalive, otherwise reset to network timeout
				if p.num_payloads == 0 {
					// Handle shutdown
					if p.shutdown {
						break PublishLoop
					} else if reload != core.Reload_None {
						break SelectLoop
					}
					log.Debug("No more pending payloads, entering idle")
					timer.Reset(keepalive_timeout)
				} else {
					log.Debug("%d payloads still pending, resetting timeout", p.num_payloads)
					timer.Reset(p.config.Timeout)

					// Release any send hold if we're no longer at the max pending payloads
					if hold && p.num_payloads < p.config.MaxPendingPayloads {
						input_toggle = p.input
					}
				}
			case <-timer.C:
				// If we have pending payloads, we should've received something by now
				if p.num_payloads != 0 {
					err = ErrNetworkTimeout
					break SelectLoop
				}

				// If we haven't received a PONG yet this is a timeout
				if p.pending_ping {
					err = ErrNetworkPing
					break SelectLoop
				}

				log.Debug("Idle timeout: sending PING")

				// Send a ping and expect a pong back (eventually)
				// If we receive an ACK first, that's fine we'll reset timer
				// But after those ACKs we should get a PONG
				if err = p.transport.Write("PING", nil); err != nil {
					break SelectLoop
				}

				p.pending_ping = true

				// We may have just filled the send buffer
				input_toggle = nil

				// Allow network timeout to receive something
				timer.Reset(p.config.Timeout)
			case <-control_signal:
				// If no pending payloads, simply end
				if p.num_payloads == 0 {
					break PublishLoop
				}

				delay_shutdown()
			case config := <-p.OnConfig():
				// Apply and check for changes
				reload = p.reloadConfig(&config.Network)

				// If a change and no pending payloads, process immediately
				if reload != core.Reload_None && p.num_payloads == 0 {
					break SelectLoop
				}

				p.can_send = nil
			case <-stats_timer.C:
				p.updateStatistics(Status_Connected, nil)
				stats_timer.Reset(time.Second)
			}
		}

		if err != nil {
			// If we're shutting down and we hit a timeout and aren't out of sync
			// We can then quit - as we'd be resending payloads anyway
			if p.shutdown && p.out_of_sync == 0 {
				log.Error("Transport error: %s", err)
				break PublishLoop
			}

			p.updateStatistics(Status_Reconnecting, err)

			// An error occurred, reconnect after timeout
			log.Error("Transport error, will try again: %s", err)
			time.Sleep(p.config.Reconnect)
		} else {
			log.Info("Reconnecting transport")

			p.updateStatistics(Status_Reconnecting, nil)
		}

		retry_payload = p.first_payload
	}

	p.transport.Shutdown()

	// Disconnect from registrar
	p.registrar_spool.Close()

	log.Info("Publisher exiting")
}

func (p *Publisher) reloadConfig(new_config *core.NetworkConfig) int {
	old_config := p.config
	p.config = new_config

	// Transport reload will return whether we need a full reload or not
	reload := p.transport.ReloadConfig(new_config)
	if reload == core.Reload_Transport {
		return core.Reload_Transport
	}

	// Same servers?
	if len(new_config.Servers) != len(old_config.Servers) {
		return core.Reload_Servers
	}

	for i := range new_config.Servers {
		if new_config.Servers[i] != old_config.Servers[i] {
			return core.Reload_Servers
		}
	}

	return reload
}

func (p *Publisher) updateStatistics(status int, err error) {
	p.Lock()

	p.status = status

	p.line_speed = core.CalculateSpeed(time.Since(p.last_measurement), p.line_speed, float64(p.line_count-p.last_line_count), &p.seconds_no_ack)

	p.last_line_count = p.line_count
	p.last_retry_count = p.retry_count
	p.last_measurement = time.Now()

	if err == ErrNetworkTimeout || err == ErrNetworkPing {
		p.timeout_count++
	}

	p.Unlock()
}

func (p *Publisher) checkResend() (bool, error) {
	// We're out of sync (received ACKs for later payloads but not earlier ones)
	// Check timeouts of earlier payloads and resend if necessary
	if payload := p.first_payload; payload.timeout.Before(time.Now()) {
		p.retry_count++

		// Do we need to regenerate the payload?
		if payload.payload == nil {
			if err := payload.Generate(); err != nil {
				return false, err
			}
		}

		// Update timeout
		payload.timeout = time.Now().Add(p.config.Timeout)

		// Requeue the payload
		p.first_payload = payload.next
		payload.next = nil
		p.last_payload.next = payload
		p.last_payload = payload

		// Send the payload again
		if err := p.transport.Write("JDAT", payload.payload); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (p *Publisher) sendNewPayload(events []*core.EventDescriptor) (err error) {
	// Calculate a nonce
	nonce := p.generateNonce()
	for {
		if _, found := p.pending_payloads[nonce]; !found {
			break
		}
		// Collision - generate again - should be extremely rare
		nonce = p.generateNonce()
	}

	var payload *pendingPayload
	if payload, err = newPendingPayload(events, nonce, p.config.Timeout); err != nil {
		return
	}

	// Save pending payload until we receive ack, and discard buffer
	p.pending_payloads[nonce] = payload
	if p.first_payload == nil {
		p.first_payload = payload
	} else {
		p.last_payload.next = payload
	}
	p.last_payload = payload

	p.Lock()
	p.num_payloads++
	p.Unlock()

	return p.transport.Write("JDAT", payload.payload)
}

func (p *Publisher) processPong(message []byte) error {
	if len(message) != 0 {
		return fmt.Errorf("PONG message overflow (%d)", len(message))
	}

	// Were we pending a ping?
	if !p.pending_ping {
		return errors.New("Unexpected PONG received")
	}

	log.Debug("PONG message received")

	p.pending_ping = false
	return nil
}

func (p *Publisher) processAck(payload *pendingPayload, ackEvents int) {
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

			payload = payload.next
			p.firstPayload = payload
			outOfSync--
			p.outOfSync = outOfSync

			p.Lock()
			p.numPayloads--
			p.Unlock()

			// Resume sending if we stopped due to excessive pending payload count
			if !p.shutdown && p.can_send == nil {
				p.can_send = p.transport.CanSend()
			}

			if payload == nil {
				break
			}
		}

		p.registrarSpool.Send()
	} else if ackEvents == 0 {
		// If this is NOT the first payload, and this is the first acknowledgement
		// for this payload, then increase out of sync payload count
		p.outOfSync++
	}
}*/
