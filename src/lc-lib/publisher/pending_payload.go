/*
* Copyright 2014 Jason Woods.
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
  "errors"
  "lc-lib/core"
  "time"
)

var (
  ErrPayloadCorrupt = errors.New("Payload is corrupt")
)

type pendingPayload struct {
  next          *pendingPayload
  nonce         string
  events        []*core.EventDescriptor
  last_sequence int
  sequence_len  int
  ack_events    int
  processed     int
  payload       []byte
  timeout       time.Time
}

func newPendingPayload(events []*core.EventDescriptor, nonce string, timeout time.Duration) (*pendingPayload, error) {
  payload := &pendingPayload{
    events:     events,
    nonce:      nonce,
    timeout:    time.Now().Add(timeout),
  }

  if err := payload.Generate(); err != nil {
    return nil, err
  }

  return payload, nil
}

func (pp *pendingPayload) Generate() (err error) {
  var buffer bytes.Buffer

  // Assertion
  if len(pp.events) == 0 {
    return ErrPayloadCorrupt
  }

  // Begin with the nonce
  if _, err = buffer.Write([]byte(pp.nonce)[0:16]); err != nil {
    return
  }

  var compressor *zlib.Writer
  if compressor, err = zlib.NewWriterLevel(&buffer, 3); err != nil {
    return
  }

  // Append all the events
  for _, event := range pp.events[pp.ack_events:] {
    if err = binary.Write(compressor, binary.BigEndian, uint32(len(event.Event))); err != nil {
      return
    }
    if _, err = compressor.Write(event.Event); err != nil {
      return
    }
  }

  compressor.Close()

  pp.payload = buffer.Bytes()
  pp.last_sequence = 0
  pp.sequence_len = len(pp.events) - pp.ack_events

  return
}

func (pp *pendingPayload) Ack(sequence int) (int, bool) {
  if sequence <= pp.last_sequence {
    // No change
    return 0, false
  } else if sequence >= pp.sequence_len {
    // Full ACK
    lines := pp.sequence_len - pp.last_sequence
    pp.ack_events = len(pp.events)
    pp.last_sequence = sequence
    pp.payload = nil
    return lines, true
  }

  lines := sequence - pp.last_sequence
  pp.ack_events += lines
  pp.last_sequence = sequence
  pp.payload = nil
  return lines, false
}

func (pp *pendingPayload) HasAck() bool {
  return pp.ack_events != 0
}

func (pp *pendingPayload) Complete() bool {
  return len(pp.events) == 0
}

func (pp *pendingPayload) Rollup() []*core.EventDescriptor {
  pp.processed += pp.ack_events
  rollup := pp.events[:pp.ack_events]
  pp.events = pp.events[pp.ack_events:]
  pp.ack_events = 0
  return rollup
}
