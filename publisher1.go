package main

import (
  "bytes"
  "compress/zlib"
  "encoding/binary"
  "io"
  "log"
  "os"
  "strconv"
  "time"
)

const default_Publisher_Hostname string = "localhost.localdomain"

type Publisher struct {
  config    *NetworkConfig
  hostname  string
  transport Transport
}

func (p *Publisher) Publish(input chan []*FileEvent, registrar_chan chan []RegistrarEvent) {
  var buffer bytes.Buffer
  var compressed_payload []byte
  var last_ack_sequence uint32
  var sequence uint32
  var err error

  p.hostname, err = os.Hostname()
  if err != nil {
    log.Printf("Failed to determine the FQDN; using localhost.localdomain.\n")
    p.hostname = default_Publisher_Hostname
  }

  // Set up the selected transport (currently only TLS)
  p.transport = CreateTransportTls(p.config)

  p.transport.Connect()
  defer p.transport.Disconnect()

  // TODO(driskell): Make the idle timeout configurable like the network timeout is?
  timer := time.NewTimer(900 * time.Second)

  for {
    select {
    case events := <-input:
      for {
        // Do we need to populate the buffer again? Or do we already have it done?
        if buffer.Len() == 0 {
          sequence = last_ack_sequence
          compressor, _ := zlib.NewWriterLevel(&buffer, 3)

          for _, event := range events {
            sequence += 1
            p.writeDataFrame(event, sequence, compressor)
          }
          compressor.Flush()
          compressor.Close()

          compressed_payload = buffer.Bytes()
        }

        // Set the window size to the length of this payload in events.
        _, err = p.transport.Write([]byte("1W"))
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        err = binary.Write(p.transport, binary.BigEndian, uint32(len(events)))
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        _, err = p.transport.Flush()
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }

        // Write compressed frame
        _, err = p.transport.Write([]byte("1C"))
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        err = binary.Write(p.transport, binary.BigEndian, uint32(len(compressed_payload)))
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        _, err = p.transport.Write(compressed_payload)
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }
        _, err = p.transport.Flush()
        if err != nil {
          log.Printf("Transport error, will reconnect: %s\n", err)
          goto RetryPayload
        }

        // Read ack
        for {
          var frame [2]byte

          err = binary.Read(p.transport, binary.BigEndian, &frame)
          if err != nil {
            log.Printf("Transport error, will reconnect: %s\n", err)
            goto RetryPayload
          }

          if frame == [2]byte{'1', 'A'} {
            var ack_sequence uint32

            // Read the sequence number acked
            err = binary.Read(p.transport, binary.BigEndian, &ack_sequence)
            if err != nil {
              log.Printf("Transport error, will reconnect: %s\n", err)
              goto RetryPayload
            }

            if sequence == ack_sequence {
              // Give the registrar the remainder of the events so it can save to state the new offsets
              registrar_chan <- []RegistrarEvent{&EventsEvent{Events: events}}
              last_ack_sequence = ack_sequence
              // All acknowledged! Stop reading acks
              break
            }

            // NOTE(driskell): If the server is busy and not yet processed anything, we MAY
            // end up receiving an ack for the last sequence in the previous payload, or 0
            if ack_sequence == last_ack_sequence {
              // Just keep waiting
              continue
            } else if ack_sequence-last_ack_sequence > uint32(len(events)) {
              // This is wrong - we've already had an ack for these
              log.Printf("Socket error, will reconnect: Repeated ACK\n")
              goto RetryPayload
            }

            // Send the events to registrar so it can save to state the new offsets
            registrar_chan <- []RegistrarEvent{&EventsEvent{Events: events[:ack_sequence-last_ack_sequence]}}
            events = events[ack_sequence-last_ack_sequence:]
            last_ack_sequence = ack_sequence

            // Reset the events buffer so it gets regenerated if we need to retry the payload
            buffer.Truncate(0)
            continue
          }

          // Unknown frame!
          log.Printf("Socket error, will reconnect: Unknown frame received: %s\n", frame)
          goto RetryPayload
        }

        // Success, stop trying to send the payload.
        break

      RetryPayload:
        // TODO(sissel): Track how frequently we timeout and reconnect. If we're
        // timing out too frequently, there's really no point in timing out since
        // basically everything is slow or down. We'll want to ratchet up the
        // timeout value slowly until things improve, then ratchet it down once
        // things seem healthy.
        p.transport.Disconnect()
        time.Sleep(p.config.reconnect)
        p.transport.Connect()
      }

      // Reset the events buffer
      buffer.Truncate(0)

      // Reset the timer
      timer.Reset(900 * time.Second)
    case <-timer.C:
      // We've no events to send - throw a ping (well... window frame) so our connection doesn't idle and die
      err = p.ping()
      if err != nil {
        log.Printf("Transport error during ping, will reconnect: %s\n", err)
        p.transport.Disconnect()
        time.Sleep(p.config.reconnect)
        p.transport.Connect()
      }

      // Reset the timer
      timer.Reset(900 * time.Second)
    } /* select */
  } /* for */
} // Publish

func (p *Publisher) ping() error {
  // This just keeps connection open through firewalls
  // We don't await for a response so its not a real ping, the protocol does not provide for a real ping
  // And with a complete replacement of protocol happening soon, makes no sense to add new frames and such
  _, err := p.transport.Write([]byte("1W"))
  if err != nil {
    return err
  }
  err = binary.Write(p.transport, binary.BigEndian, uint32(*spool_size))
  if err != nil {
    return err
  }
  _, err = p.transport.Flush()
  if err != nil {
    return err
  }

  return nil
}

func (p *Publisher) writeDataFrame(event *FileEvent, sequence uint32, output io.Writer) {
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
  p.writeKeyValue("line", *(*event.Event)["line"].(*string), output)
  for k, v := range *event.Event {
    if k == "file" || k == "offset" || k == "line" {
      continue
    }
    p.writeKeyValue(k, *v.(*string), output)
  }
}

func (p *Publisher) writeKeyValue(key string, value string, output io.Writer) {
  binary.Write(output, binary.BigEndian, uint32(len(key)))
  output.Write([]byte(key))
  binary.Write(output, binary.BigEndian, uint32(len(value)))
  output.Write([]byte(value))
}
