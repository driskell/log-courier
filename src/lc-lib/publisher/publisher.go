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
  "bytes"
  "compress/zlib"
  "encoding/binary"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "lc-lib/core"
  "lc-lib/registrar"
  "math/rand"
  "os"
  "time"
)

const (
  default_publisher_hostname string        = "localhost.localdomain"
  keepalive_timeout          time.Duration = 900 * time.Second
  max_pending_payloads       int           = 100
)

type pendingPayload struct {
  next          *pendingPayload
  nonce         string
  events        []*core.EventDescriptor
  num_events    int
  ack_events    int
  payload_start int
  payload       []byte
  timeout       *time.Time
}

func newPendingPayload(events []*core.EventDescriptor, nonce string, hostname string) (*pendingPayload, error) {
  payload := &pendingPayload{
    events:     events,
    nonce:      nonce,
    num_events: len(events),
  }

  if err := payload.Generate(hostname); err != nil {
    return nil, err
  }

  return payload, nil
}

func (pp *pendingPayload) Generate(hostname string) (err error) {
  var buffer bytes.Buffer

  // Begin with the nonce
  if _, err = buffer.Write([]byte(pp.nonce)); err != nil {
    return
  }

  var compressor *zlib.Writer
  if compressor, err = zlib.NewWriterLevel(&buffer, 3); err != nil {
    return
  }

  // Append all the events
  for _, event := range pp.events[pp.ack_events:] {
    // Add host field
    event.Event["host"] = hostname
    if err = pp.bufferJdatDataEvent(compressor, event); err != nil {
      return
    }
  }

  compressor.Close()

  pp.payload = buffer.Bytes()
  pp.payload_start = pp.ack_events

  return
}

func (pp *pendingPayload) bufferJdatDataEvent(output io.Writer, event *core.EventDescriptor) (err error) {
  var value []byte
  value, err = json.Marshal(event.Event)
  if err != nil {
    log.Error("JSON event encoding error: %s", err)

    if err = binary.Write(output, binary.BigEndian, 2); err != nil {
      return
    }
    if _, err = output.Write([]byte("{}")); err != nil {
      return
    }

    return
  }

  if err = binary.Write(output, binary.BigEndian, uint32(len(value))); err != nil {
    return
  }
  if _, err = output.Write(value); err != nil {
    return
  }

  return nil
}

type Publisher struct {
  core.PipelineSegment
  core.PipelineConfigReceiver

  config           *core.NetworkConfig
  transport        core.Transport
  hostname         string
  can_send         <-chan int
  pending_ping     bool
  pending_payloads map[string]*pendingPayload
  first_payload    *pendingPayload
  last_payload     *pendingPayload
  num_payloads     int
  out_of_sync      int
  input            chan []*core.EventDescriptor
  registrar        *registrar.Registrar
  registrar_chan   chan<- []registrar.RegistrarEvent
  shutdown         bool
}

func NewPublisher(pipeline *core.Pipeline, config *core.NetworkConfig, registrar_imp *registrar.Registrar) (*Publisher, error) {
  ret := &Publisher{
    config:         config,
    input:          make(chan []*core.EventDescriptor, 1),
    registrar:      registrar_imp,
    registrar_chan: registrar_imp.Connect(),
  }

  if err := ret.init(); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (p *Publisher) init() error {
  var err error

  p.hostname, err = os.Hostname()
  if err != nil {
    log.Warning("Failed to determine the FQDN; using localhost.localdomain.")
    p.hostname = default_publisher_hostname
  }

  p.pending_payloads = make(map[string]*pendingPayload)

  // Set up the selected transport
  if err = p.loadTransport(); err != nil {
    return err
  }

  return nil
}

func (p *Publisher) loadTransport() error {
  transport, err := p.config.TransportFactory.NewTransport(p.config)
  if err != nil {
    return err
  }

  p.transport = transport

  return nil
}

func (p *Publisher) Connect() chan<- []*core.EventDescriptor {
  return p.input
}

func (p *Publisher) Run() {
  defer func() {
    p.Done()
  }()

  var input_toggle <-chan []*core.EventDescriptor
  var retry_payload *pendingPayload
  var err error
  var reload int

  // TODO(driskell): Make the idle timeout configurable like the network timeout is?
  timer := time.NewTimer(keepalive_timeout)

  shutdown_signal := p.ShutdownSignal()

PublishLoop:
  for {
    if err = p.transport.Init(); err != nil {
      log.Error("Transport init failed: %s", err)
      // TODO: implement shutdown select
      select {
      case <-time.After(p.config.Reconnect):
        continue
      case <-p.ShutdownSignal():
        // TODO: Persist pending payloads and resume? Quicker shutdown
        if p.num_payloads == 0 {
          break PublishLoop
        }

        p.shutdown = true
      }
    }

    p.pending_ping = false
    input_toggle = nil

    if p.shutdown || p.num_payloads >= max_pending_payloads {
      p.can_send = nil
    } else {
      p.can_send = p.transport.CanSend()
    }

  SelectLoop:
    for {
      select {
      case <-p.can_send:
        // Resend payloads from full retry first
        if retry_payload != nil {
          // Do we need to regenerate the payload?
          if retry_payload.payload == nil {
            if err = retry_payload.Generate(p.hostname); err != nil {
              break SelectLoop
            }
          }

          // Reset timeout
          retry_payload.timeout = nil

          // Send the payload again
          if err = p.transport.Write("JDAT", retry_payload.payload); err != nil {
            break SelectLoop
          }

          // Move to next non-empty payload
          for {
            retry_payload = retry_payload.next
            if retry_payload == nil || retry_payload.ack_events != len(retry_payload.events) {
              break
            }
          }

          // Expect an ACK within network timeout
          if p.first_payload.timeout != nil {
            timer.Reset(p.first_payload.timeout.Sub(time.Now()))
          } else {
            timer.Reset(p.config.Timeout)
          }
          break
        } else if p.out_of_sync != 0 {
          var resent bool
          if resent, err = p.checkResend(); err != nil {
            break SelectLoop
          } else if resent {
            // Expect an ACK within network timeout
            timer.Reset(p.config.Timeout)
            break
          }
        }

        // No pending payloads, are we shutting down? Skip if so
        if p.shutdown {
          break
        }

        // Enable event wait
        input_toggle = p.input
      case events := <-input_toggle:
        // Send
        if err = p.sendNewPayload(events); err != nil {
          break SelectLoop
        }

        // Wait for send signal again
        input_toggle = nil

        if p.num_payloads >= max_pending_payloads {
          // Too many pending payloads, disable send temporarily
          p.can_send = nil
        }

        // Expect an ACK within network timeout
        if p.first_payload.timeout != nil {
          timer.Reset(p.first_payload.timeout.Sub(time.Now()))
        } else {
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
          if err = p.processAck(message, p.registrar_chan); err != nil {
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
          }
          timer.Reset(keepalive_timeout)
        } else if p.first_payload.timeout != nil {
          timer.Reset(p.first_payload.timeout.Sub(time.Now()))
        } else {
          timer.Reset(p.config.Timeout)
        }
      case <-timer.C:
        // Do we need to resend first payload?
        if p.out_of_sync != 0 {
          var resent bool
          if resent, err = p.checkResend(); err != nil {
            break SelectLoop
          } else if resent {
            // Expect an ACK within network timeout
            timer.Reset(p.config.Timeout)
            break
          }
        }

        // If we have pending payloads, we should've received something by now
        if p.num_payloads != 0 {
          err = errors.New("Server did not respond within network timeout")
          break SelectLoop
        }

        // If we haven't received a PONG yet this is a timeout
        if p.pending_ping {
          err = errors.New("Server did not respond to PING")
          break SelectLoop
        }

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
      case <-shutdown_signal:
        // If no pending payloads, simply end
        if p.num_payloads == 0 {
          break PublishLoop
        }

        // Flag shutdown for when we finish pending payloads
        // TODO: Persist pending payloads and resume? Quicker shutdown
        log.Warning("Delaying shutdown to wait for pending responses from the server")
        shutdown_signal = nil
        p.shutdown = true
        p.can_send = nil
        input_toggle = nil
      case config := <-p.RecvConfig():
        // Apply and check for changes
        reload = p.reloadConfig(&config.Network)

        // If a change and no pending payloads, process immediately
        if reload != core.Reload_None && p.num_payloads == 0 {
          break SelectLoop
        }
      }
    }

    if err != nil {
      // If we're shutting down and we hit a timeout and aren't out of sync
      // We can then quit - as we'd be resending payloads anyway
      if p.shutdown && p.out_of_sync == 0 {
        log.Error("Transport error: %s", err)
        break PublishLoop
      }

      // An error occurred, reconnect after timeout
      log.Error("Transport error, will try again: %s", err)
      time.Sleep(p.config.Reconnect)
    } else {
      // Do we need to reload transport?
      if reload == core.Reload_Transport {
        // Shutdown and reload transport
        p.transport.Shutdown()

        if err = p.loadTransport(); err != nil {
          log.Error("The new transport configuration failed to apply: %s", err)
        }
      }

      reload = core.Reload_None
    }

    retry_payload = p.first_payload
  }

  p.transport.Shutdown()

  // Disconnect from registrar
  p.registrar.Disconnect()

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

func (p *Publisher) checkResend() (bool, error) {
  // We're out of sync (received ACKs for later payloads but not earlier ones)
  // Check timeouts of earlier payloads and resend if necessary
  if payload := p.first_payload; payload.timeout.Before(time.Now()) {
    // Do we need to regenerate the payload?
    if payload.payload == nil {
      if err := payload.Generate(p.hostname); err != nil {
        return false, err
      }
    }

    // Update timeout
    timeout := time.Now().Add(p.config.Timeout)
    payload.timeout = &timeout

    // Send the payload again
    if err := p.transport.Write("JDAT", payload.payload); err != nil {
      return false, err
    }

    return true, nil
  }

  return false, nil
}

func (p *Publisher) generateNonce() string {
  // This could maybe be made a bit more efficient
  nonce := make([]byte, 16)
  for i := 0; i < 16; i++ {
    nonce[i] = byte(rand.Intn(255))
  }
  return string(nonce)
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
  if payload, err = newPendingPayload(events, nonce, p.hostname); err != nil {
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
  p.num_payloads++

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

  p.pending_ping = false
  return nil
}

func (p *Publisher) processAck(message []byte, registrar_chan chan<- []registrar.RegistrarEvent) (err error) {
  if len(message) != 20 {
    err = fmt.Errorf("ACKN message corruption (%d)", len(message))
    return
  }

  // Read the nonce and sequence number acked
  nonce, sequence := string(message[:16]), binary.BigEndian.Uint32(message[16:20])

  // Grab the payload the ACK corresponds to by using nonce
  payload, found := p.pending_payloads[nonce]
  if !found {
    // Don't fail here in case we had temporary issues and resend a payload, only for us to receive duplicate ACKN
    return
  }

  ack_events := payload.ack_events

  // Full ACK?
  // TODO: Protocol error if sequence is too large?
  if int(sequence) >= payload.num_events-payload.payload_start {
    // No more events left for this payload, free the payload memory
    payload.ack_events = len(payload.events)
    payload.payload = nil
    delete(p.pending_payloads, nonce)
  } else {
    // Only process the ACK if something was actually processed
    if int(sequence) > payload.num_events-payload.ack_events {
      payload.ack_events = int(sequence) + payload.payload_start
      // If we need to resend, we'll need to regenerate payload, so free that memory early
      payload.payload = nil
    }
  }

  // We potentially receive out-of-order ACKs due to payloads distributed across servers
  // This is where we enforce ordering again to ensure registrar receives ACK in order
  if payload == p.first_payload {
    out_of_sync := p.out_of_sync + 1
    for payload.ack_events != 0 {
      if payload.ack_events != len(payload.events) {
        registrar_chan <- []registrar.RegistrarEvent{registrar.NewEventsEvent(payload.events[:payload.ack_events])}
        payload.events = payload.events[payload.ack_events:]
        payload.num_events = len(payload.events)
        payload.ack_events = 0
        payload.payload_start = 0
        break
      }

      registrar_chan <- []registrar.RegistrarEvent{registrar.NewEventsEvent(payload.events)}
      payload = payload.next
      p.first_payload = payload
      p.num_payloads--
      out_of_sync--
      p.out_of_sync = out_of_sync

      // Resume sending if we stopped due to excessive pending payload count
      if !p.shutdown && p.can_send == nil {
        p.can_send = p.transport.CanSend()
      }

      if payload == nil {
        break
      }
    }
  } else if ack_events == 0 {
    // Mark out of sync so we resend earlier packets in case they were lost
    p.out_of_sync++
  }

  // Set a timeout of the first payload if out of sync as we should be expecting it any time
  if p.out_of_sync != 0 && p.first_payload.timeout == nil {
    timeout := time.Now().Add(p.config.Timeout)
    p.first_payload.timeout = &timeout
  }

  return
}
