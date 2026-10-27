/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package stream

import (
	"context"
	"io"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocol struct {
	conn       tcp.Connection
	lineReader *harvester.LineReader
	// err is the error that ended the stream
	err error
}

// Negotiation does not happen for a plain event stream
func (p *protocol) Negotiation() (transports.Event, error) {
	return nil, nil
}

// SendEvents is not implemented as this is not a transport
func (p *protocol) SendEvents(nonce string, events []*event.Event) error {
	panic("Not implemented")
}

// Acknowledge is not used by a plain event stream
func (p *protocol) Acknowledge(nonce *string, sequence uint32) error {
	return nil
}

// Ping is not implemented as this is not a transport
func (p *protocol) Ping() error {
	panic("Not implemented")
}

// Pong is not implemented as we will never receiver a Ping
func (p *protocol) Pong() error {
	panic("Not implemented")
}

// Read returns the lines arriving on the stream as a message of events
func (p *protocol) Read() (transports.Event, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &streamMessage{protocol: p}, nil
}

// NonBlocking returns trues because a stream does not have well defined event boundaries so we have to check periodically
func (p *protocol) NonBlocking() bool {
	return true
}

// streamMessage reads the lines of the stream available without waiting as events, ending once more must be read
type streamMessage struct {
	protocol *protocol
	read     bool
}

var _ tcp.StreamMessage = (*streamMessage)(nil)

// Context returns the connection context
func (m *streamMessage) Context() context.Context {
	return m.protocol.conn.Context()
}

// Nonce returns the nonce used for sources without nonces
func (m *streamMessage) Nonce() *string {
	return &transports.NilNonce
}

// Next reads the next line, splitting a line longer than the receive window into several tagged events
func (m *streamMessage) Next(func(int) error) ([]byte, map[string]interface{}, int, bool, error) {
	// Ending once no further line is ready lets the receiver take the lines read so far before waiting for more
	if m.read && !m.protocol.lineReader.LineReady() {
		return nil, nil, 0, false, io.EOF
	}
	m.read = true

	data, length, err := m.protocol.lineReader.ReadItem()
	if err == harvester.ErrMaxDataSizeTruncation {
		data["tags"] = event.Tags{"splitline"}
	} else if err != nil {
		// The message ends here and Read then reports why
		m.protocol.err = err
		return nil, nil, 0, false, io.EOF
	}

	log.Debugf("Received event: %v", data)
	return nil, data, length, false, nil
}
