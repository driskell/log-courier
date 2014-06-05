package main

import (
  "bytes"
  "compress/zlib"
  "encoding/binary"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "log"
  "math/rand"
  "os"
  "time"
)

const (
  default_publisher_hostname string = "localhost.localdomain"
  keepalive_timeout          time.Duration = 900 * time.Second
  max_pending_payloads       int = 100
)

type PendingPayload struct {
  next       *PendingPayload
  nonce      string
  events     []*FileEvent
  num_events int
  ack_events int
  payload_start int
  payload    []byte
}

type Publisher struct {
  config           *NetworkConfig
  hostname         string
  transport        Transport
  can_send         <-chan int
  pending_ping     bool
  pending_payloads map[string]*PendingPayload
  first_payload    *PendingPayload
  last_payload     *PendingPayload
  num_payloads     int
}

func (p *Publisher) Init() error {
  var err error

  p.hostname, err = os.Hostname()
  if err != nil {
    log.Printf("Failed to determine the FQDN; using localhost.localdomain.\n")
    p.hostname = default_publisher_hostname
  }

  // Set up the selected transport (currently only TLS)
  if p.transport, err = CreateTransportZmq(p.config); err != nil {
    return err
  }

  p.pending_payloads = make(map[string]*PendingPayload)

  return nil
}

func (p *Publisher) Publish(input <-chan []*FileEvent, registrar_chan chan<- []RegistrarEvent) {
  var input_toggle <-chan []*FileEvent
  var retry_payload *PendingPayload
  var err error

  // TODO(driskell): Make the idle timeout configurable like the network timeout is?
  timer := time.NewTimer(keepalive_timeout)

  // TODO: We should still obey network timeout if we've sent events and not yet received response
  //       as its the quickest way to detect a connection problem after idle

  for {
    if err = p.transport.Connect(); err != nil {
      log.Printf("Connect attempt failed: %s\n", err)
      // TODO: implement shutdown select
      time.Sleep(p.config.reconnect)
      continue
    }
    p.can_send = p.transport.CanSend()
    input_toggle = nil

  SelectLoop:
    for {
      // TODO: implement shutdown select
      select {
      case <-p.can_send:
        if retry_payload != nil {
          var buffer bytes.Buffer

          // Do we need to regenerate the payload? Remember to account for ACK we have but not yet sent to registrar due to out-of-order receive
          if retry_payload.payload == nil {
            if err = p.bufferJdatData(&buffer, retry_payload.events[retry_payload.ack_events:], retry_payload.nonce); err != nil {
              break SelectLoop
            }

            retry_payload.payload = buffer.Bytes()
            retry_payload.payload_start = retry_payload.ack_events
          }

          // Send the payload again
          if err = p.transport.Write("JDAT", retry_payload.payload); err != nil {
            break SelectLoop
          }

          retry_payload = retry_payload.next
          break
        }

        // No pending payloads, enable event wait
        input_toggle = input

        // Continue loop so we don't reset the ping timer - we've not performed any activity just yet
        continue
      case events := <-input_toggle:
        // Send JDAT
        if err = p.sendJdat(events); err != nil {
          break SelectLoop
        }

        // Wait for send signal again
        input_toggle = nil

        if p.num_payloads >= max_pending_payloads {
          // Too many pending payloads, disable send temporarily
          p.can_send = nil
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
          if err = p.processAck(message, registrar_chan); err != nil {
            break SelectLoop
          }
        default:
          err = errors.New(fmt.Sprintf("Unknown message received: % X", signature))
          break SelectLoop
        }
      case <-timer.C:
        log.Printf("<-timer.C")
        // If we haven't received a PONG yet this is a timeout
        if p.pending_ping {
          err = errors.New("Server did not respond to PING")
          break SelectLoop
        }

        // If the send buffer is full, we should have been receiving ACK by now...
        if input_toggle == nil {
          err = errors.New("Server stopped responding")
          break SelectLoop
        }

        // Send a ping and expect a pong back (eventually)
        // If we receive an ACK first, that's fine we'll reset timer
        // But after those ACKs we should get a PONG
        if err = p.transport.Write("PING", nil); err != nil {
          break SelectLoop
        }

        // We may have just filled the send buffer
        input_toggle = nil
      } /* select */

      // Reset the timer
      timer.Reset(keepalive_timeout)
    } /* loop forever, break to reconnect */

    // Disconnect and retry payloads
    log.Printf("Transport error, will reconnect: %s\n", err)
    p.transport.Disconnect()
    retry_payload = p.first_payload

    time.Sleep(p.config.reconnect)
  } /* Publish: for loop, break to shutdown */
} // Publish

func (p *Publisher) generateNonce() string {
  // This could maybe be made a bit more efficient
  nonce := make([]byte, 16)
  for i := 0; i < 16; i++ {
    nonce[i] = byte(rand.Intn(255))
  }
  return string(nonce)
}

func (p *Publisher) sendJdat(events []*FileEvent) (err error) {
  var buffer bytes.Buffer

  // Calculate a nonce
  nonce := p.generateNonce()
  for {
    if _, found := p.pending_payloads[nonce]; !found {
      break
    }
    // Collision - generate again - should be extremely rare
    nonce = p.generateNonce()
  }

  // Generate the data first
  if err = p.bufferJdatData(&buffer, events, nonce); err != nil {
    return
  }

  // Save pending payload until we receive ack, and discard buffer
  payload := &PendingPayload{events: events, nonce: nonce, num_events: len(events), payload_start: 0, payload: buffer.Bytes()}
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

func (p *Publisher) bufferJdatData(output io.Writer, events []*FileEvent, nonce string) (err error) {
  // Begin with the nonce
  if _, err = output.Write([]byte(nonce)); err != nil {
    return
  }

  var compressor *zlib.Writer
  if compressor, err = zlib.NewWriterLevel(output, 3); err != nil {
    return
  }

  // Append all the events
  for _, event := range events {
    // Add host field
    event.Event["host"] = p.hostname
    if err = p.bufferJdatDataEvent(compressor, event); err != nil {
      return
    }
  }

  compressor.Close()

  return nil
}

func (p *Publisher) bufferJdatDataEvent(output io.Writer, event *FileEvent) (err error) {
  var value []byte
  value, err = json.Marshal(event.Event)
  if err != nil {
    log.Printf("JSON event encoding error: %s\n", err)

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

func (p *Publisher) processPong(message []byte) error {
  if len(message) != 0 {
    return errors.New(fmt.Sprintf("PONG message overflow (%d)", len(message)))
  }

  // Were we pending a ping?
  if !p.pending_ping {
    return errors.New("Unexpected PONG received")
  }

  p.pending_ping = false
  return nil
}

func (p *Publisher) processAck(message []byte, registrar_chan chan<- []RegistrarEvent) (err error) {
  if len(message) != 20 {
    err = errors.New(fmt.Sprintf("ACKN message corruption (%d)", len(message)))
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

  // Full ACK?
  // TODO: Protocol error if sequence is too large?
  if int(sequence) >= payload.num_events - payload.payload_start {
    // No more events left for this payload, free the payload memory
    payload.ack_events = len(payload.events)
    payload.payload = nil
    delete(p.pending_payloads, nonce)
  } else {
    // Only process the ACK if something was actually processed
    if int(sequence) > payload.num_events - payload.ack_events {
      payload.ack_events = int(sequence) + payload.payload_start
      // If we need to resend, we'll need to regenerate payload, so free that memory early
      payload.payload = nil
    }
  }

  // We potentially receive out-of-order ACKs due to payloads distributed across servers
  // This is where we enforce ordering again to ensure registrar receives ACK in order
  if payload == p.first_payload {
    for payload.ack_events != 0 {
      if payload.ack_events == len(payload.events) {
        registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events}}
        payload = payload.next
        p.first_payload = payload
        p.num_payloads--

        // Resume sending if we stopped due to excessive pending payload count
        if p.can_send == nil {
          p.can_send = p.transport.CanSend()
        }
      } else {
        registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events[:payload.ack_events]}}
        payload.events = payload.events[payload.ack_events:]
        payload.num_events = len(payload.events)
        payload.ack_events = 0
        payload.payload_start = 0
      }

      if payload == nil {
        break
      }
    }
  }

  return
}
