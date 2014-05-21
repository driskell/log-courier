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
  "os"
  "strconv"
  "time"
)

const (
  default_publisher_hostname string = "localhost.localdomain"
  keepalive_timeout          time.Duration = 900 * time.Second
)

type PendingPayload struct {
  events     []*FileEvent
  num_events int
  payload    []byte
  timeout    time.Time
}

type Publisher struct {
  config    *NetworkConfig
  hostname  string
  transport Transport
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

  return nil
}

func (p *Publisher) Publish(input chan []*FileEvent, registrar_chan chan []RegistrarEvent) {
  var input_toggle chan []*FileEvent
  var sequence uint32
  var buffer bytes.Buffer
  var err error

  pending_payloads := make(map[uint32]*PendingPayload)

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
          // Create the event message
          start_sequence := sequence
          // TODO: check error return
          compressor, _ := zlib.NewWriterLevel(&buffer, 3)

          for _, event := range events {
            sequence += 1
            // TODO: check error return when we implement it
            p.writeDataFrame(event, sequence, compressor)
          }

          compressor.Close()

          // Save pending payload until we receive ack, and discard buffer
          payload := &PendingPayload{events: events, num_events: len(events), payload: buffer.Bytes(), timeout: time.Now().Add(p.config.timeout)}
          pending_payloads[start_sequence] = payload
          buffer.Reset()

          if err = p.writePayload(payload); err != nil {
            break SelectLoop
          }

          // Wait for send signal again
          input_toggle = nil
        case <-p.transport.CanRecv():
          // Receive an ACK
          var ack_sequence uint32
          var payload_sequence uint32 = 0xFFFFFFFF  // TODO: Temporary until we rework protocol
          var payload *PendingPayload

          if ack_sequence, err = p.readAck(); err != nil {
            break SelectLoop
          }

          // TODO: Validate authenticity of the ack
          //       Specifically, are we sure this ack came from the same peer we sent the payload to?
          //       We will need to send a nonce with the payload and expect it back to confirm this, otherwise any peer could respond with an ACK
          //       In this case we can discard the sequence counter and simply have it reset to 1 for each payload since we can distinguish ACK with nonce

          // TODO: If we ack'd OK and we refused to send due to large pending_payloads, re-enable input_toggle in here

          // Find the pending payload - ACK will be one before (which means nothing processed yet) up to the number of the last event
          for start_sequence, this_payload := range pending_payloads {
            if payload_sequence >= start_sequence && ack_sequence >= start_sequence && ack_sequence <= start_sequence + uint32(this_payload.num_events) {
              payload_sequence = start_sequence
              payload = this_payload
            }
          }

          // Fail the connection if we get back an ACK that is out of bounds
          if payload == nil {
            err = errors.New("Out of bounds, repeated or stale ACK")
            break SelectLoop
          }

          // Full ACK?
          if ack_sequence == payload_sequence + uint32(len(payload.events)) {
            // Give the registrar the remainder of the events so it can save to state the new offsets, and drop from pending payloads
            registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events}}
            delete(pending_payloads, payload_sequence)
          } else {
            // Only process the ACK if something was actually processed, i.e. ack_sequence is not one less than the first sequence in the payload
            if ack_sequence != payload_sequence {
              // Send the events to registrar so it can save to state the new offsets and update pending payload, wiping the compressed part so it is regenerated if needed
              registrar_chan <- []RegistrarEvent{&EventsEvent{Events: payload.events[:ack_sequence - payload_sequence - 1]}}
              payload.events = payload.events[ack_sequence - payload_sequence:]
              payload.payload = nil

              // Move to the new position
              delete(pending_payloads, payload_sequence)
              pending_payloads[ack_sequence] = payload
            }

            // Update the retry timeout on the payload
            payload.timeout = time.Now().Add(keepalive_timeout)
          }
        case <-timer.C:
          log.Printf("<-timer.C")
          // We've no events to send - throw a ping (well... window frame) so our connection doesn't idle and die
          // Protocol needs changing eventually to allow for a pong - same time as JSON change I guess
          if err = p.ping(); err != nil {
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

func (p *Publisher) ping() error {
  // This just keeps connection open through firewalls
  // We don't await for a response so its not a real ping, the protocol does not provide for a real ping
  // And with a complete replacement of protocol happening soon, makes no sense to add new frames and such
  if _, err := p.transport.Write([]byte("1W")); err != nil {
    return err
  }
  if err := binary.Write(p.transport, binary.BigEndian, uint32(*spool_size)); err != nil {
    return err
  }

  // Flush the ping frame
  return p.transport.Flush()
}

func (p *Publisher) writePayload(payload *PendingPayload) (err error) {
  // Set the window size to the length of this payload in events.
  if _, err = p.transport.Write([]byte("1W")); err != nil {
    return
  }
  if err = binary.Write(p.transport, binary.BigEndian, uint32(payload.num_events)); err != nil {
    return
  }

  // Write compressed frame
  if _, err = p.transport.Write([]byte("1C")); err != nil {
    return
  }
  if err = binary.Write(p.transport, binary.BigEndian, uint32(len(payload.payload))); err != nil {
    return
  }
  if _, err = p.transport.Write(payload.payload); err != nil {
    return
  }

  // Flush the frame
  return p.transport.Flush()
}

func (p *Publisher) readAck() (sequence uint32, err error) {
  var frame []byte
  // Read will return a single frame
  if frame, err = p.transport.Read(); err != nil {
    return
  }

  // Validate its an ACK frame
  if bytes.Compare(frame[0:2], []byte{'1', 'A'}) != 0 {
    err = errors.New(fmt.Sprintf("Unknown frame received: % X", frame))
    return
  } else if len(frame) > 6 {
    err = errors.New(fmt.Sprintf("Frame overflow: % X", frame))
    return
  }

  // Read the sequence number acked
  sequence = binary.BigEndian.Uint32(frame[2:6])
  return
}

func (p *Publisher) writeJSONFrame(event *FileEvent, sequence uint32, output io.Writer) {
  // TODO: check error returns
  // Header, "1J"
  output.Write([]byte("1J"))
  // Sequence number
  binary.Write(output, binary.BigEndian, uint32(sequence))

  value, err := json.Marshal(*event.Event)
  if err != nil {
    log.Printf("JSON event encoding error: %s\n", err)
    binary.Write(output, binary.BigEndian, 2)
    output.Write([]byte("{}"))
    return
  }

  binary.Write(output, binary.BigEndian, uint32(len(value)))
  output.Write(value)
  log.Printf("JSON: %s\n", value)
}

func (p *Publisher) writeDataFrame(event *FileEvent, sequence uint32, output io.Writer) {
  // TODO: check error returns
  // Header, "2D"
  // Why version 2 data frame? Because server.rb will correctly start returning partial ACKs if we specify version 2
  // This keeps the old logstash forwarders, which broke on partial ACK, working with even the newer server.rb
  // If the newer server.rb receives a 1D it will refuse to send partial ACK, just like before
  output.Write([]byte("2D"))
  // Sequence number
  binary.Write(output, binary.BigEndian, uint32(sequence))
  // Key-value pair count
  binary.Write(output, binary.BigEndian, uint32(len(*event.Event)+1))

  p.writeKeyValue("file", *(*event.Event)["file"].(*string), output)
  p.writeKeyValue("host", p.hostname, output)
  p.writeKeyValue("offset", strconv.FormatInt((*event.Event)["offset"].(int64), 10), output)
  p.writeKeyValue("line", *(*event.Event)["message"].(*string), output)
  for k, v := range *event.Event {
    if k == "file" || k == "offset" || k == "message" {
      continue
    }
    p.writeKeyValue(k, *v.(*string), output)
  }
}

func (p *Publisher) writeKeyValue(key string, value string, output io.Writer) {
  // TODO: check error returns
  binary.Write(output, binary.BigEndian, uint32(len(key)))
  output.Write([]byte(key))
  binary.Write(output, binary.BigEndian, uint32(len(value)))
  output.Write([]byte(value))
}
