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
)

type PendingPayload struct {
  events     []*FileEvent
  num_events uint32
  payload    []byte
  timeout    time.Time
}

type Publisher struct {
  config           *NetworkConfig
  hostname         string
  transport        Transport
  pending_ping     bool
  pending_payloads map[string]*PendingPayload
}

func (p *Publisher) Init() error {
  var err error

  p.hostname, err = os.Hostname()
  if err != nil {
    log.Printf("Failed to determine the FQDN; using localhost.localdomain.\n")
    p.hostname = default_publisher_hostname
  }

  // Set up the selected transport (currently only TLS)
  if p.transport, err = CreateTransportTls(p.config); err != nil {
    return err
  }

  p.pending_payloads = make(map[string]*PendingPayload)

  return nil
}

func (p *Publisher) Publish(input chan []*FileEvent, registrar_chan chan []RegistrarEvent) {
  var input_toggle chan []*FileEvent
  var buffer bytes.Buffer
  var err error

  // TODO(driskell): Make the idle timeout configurable like the network timeout is?
  timer := time.NewTimer(keepalive_timeout)

  for {
    p.transport.Connect()
    input_toggle = nil

  SelectLoop:
    for {
      // TODO: implement shutdown select
      select {
        case <-p.transport.CanSend():
          // Ready to send, enable event wait
          // TODO: If pending_payloads is large, don't send anymore, leave it nil, then when we receive ack do this
          // TODO: Process pending payloads we have not received a response for yet, queueing them as priority (switch input_toggle?) and regenerating compressor as required
          input_toggle = input

          // Continue loop so we don't reset the ping timer - we've not performed any activity just yet
          continue
        case events := <-input_toggle:
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
            break SelectLoop
          }

          // Save pending payload until we receive ack, and discard buffer
          payload := &PendingPayload{events: events, num_events: uint32(len(events)), payload: buffer.Bytes(), timeout: time.Now().Add(p.config.timeout)}
          p.pending_payloads[nonce] = payload
          buffer.Reset()

          if err = p.writeJdat(payload.payload); err != nil {
            break SelectLoop
          }

          // Wait for send signal again
          input_toggle = nil
        case <-p.transport.CanRecv():
          var signature, message []byte

          // Receive message
          if signature, message, err = p.readMessage(); err != nil {
            break SelectLoop
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
          if err = p.writePing(); err != nil {
            break SelectLoop
          }
      } /* select */

      // Reset the timer
      timer.Reset(keepalive_timeout)
    } /* loop forever, break to reconnect */

    // TODO: change this logic so we fail a specific connection instead of all (specifically for ZMQ)
    log.Printf("Transport error, will reconnect: %s\n", err)
    p.transport.Disconnect()

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

func (p *Publisher) writePing() (err error) {
  if _, err = p.transport.Write([]byte("PING")); err != nil {
    return err
  }
  if err = binary.Write(p.transport, binary.BigEndian, 0); err != nil {
    return err
  }

  p.pending_ping = true

  // Flush the ping frame
  return p.transport.Flush()
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
    if err = p.bufferJdatDataEvent(compressor, event); err != nil {
      return
    }
  }

  compressor.Close()

  return nil
}

func (p *Publisher) bufferJdatDataEvent(output io.Writer, event *FileEvent) (err error) {
  var value []byte
  value, err = json.Marshal(*event.Event)
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

func (p *Publisher) writeJdat(data []byte) (err error) {
  if _, err = p.transport.Write([]byte("JDAT")); err != nil {
    return
  }
  if err = binary.Write(p.transport, binary.BigEndian, uint32(len(data))); err != nil {
    return
  }

  if _, err = p.transport.Write(data); err != nil {
    return
  }

  return p.transport.Flush()
}

func (p *Publisher) readMessage() (signature []byte, message []byte, err error) {
  var frame []byte

  // Read will return a single frame
  if frame, err = p.transport.Read(); err != nil {
    return
  }

  // Return the signature and the message (ignore length that sits between)
  return frame[0:4], frame[8:], nil
}

func (p *Publisher) processPong(message []byte) error {
  if len(message) > 8 {
    return errors.New(fmt.Sprintf("PONG message overflow (%d)", len(message)))
  }

  // Were we pending a ping?
  if !p.pending_ping {
    return errors.New("Unexpected PONG received")
  }

  p.pending_ping = false
  return nil
}

func (p *Publisher) processAck(message []byte, registrar_chan chan []RegistrarEvent) (err error) {
  if len(message) != 20 {
    err = errors.New(fmt.Sprintf("ACKN message corruption (%d)", len(message)))
    return
  }

  // Read the nonce and sequence number acked
  nonce, sequence := string(message[:16]), binary.BigEndian.Uint32(message[16:20])

  // TODO: If we ack'd OK and we refused to send due to large pending_payloads, re-enable input_toggle in here somehow

  // Grab the payload the ACK corresponds to by using nonce
  payload, found := p.pending_payloads[nonce]
  if !found {
    err = errors.New("ACK for unknown payload received")
    return
  }

  // Full ACK?
  if sequence == payload.num_events {
    // Give the registrar the remainder of the events so it can save to state the new offsets, and drop from pending payloads
    registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events}}
    delete(p.pending_payloads, nonce)
  } else {
    // Only process the ACK if something was actually processed
    if sequence != 0 {
      // Send the events to registrar so it can save to state the new offsets and update pending payload, wiping the compressed part so it is regenerated if needed
      registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events[:sequence]}}
      payload.events = payload.events[sequence:]
      payload.payload = nil
    }

    // Update the retry timeout on the payload
    payload.timeout = time.Now().Add(keepalive_timeout)
  }

  return
}
