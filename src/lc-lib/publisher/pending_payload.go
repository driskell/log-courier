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
  "encoding/json"
  "io"
  "lc-lib/core"
  "time"
)

type pendingPayload struct {
  next          *pendingPayload
  nonce         string
  events        []*core.EventDescriptor
  num_events    int
  ack_events    int
  payload_start int
  payload       []byte
  timeout       time.Time
}

func newPendingPayload(events []*core.EventDescriptor, nonce string, hostname string, timeout time.Duration) (*pendingPayload, error) {
  payload := &pendingPayload{
    events:     events,
    nonce:      nonce,
    num_events: len(events),
    timeout:    time.Now().Add(timeout),
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
