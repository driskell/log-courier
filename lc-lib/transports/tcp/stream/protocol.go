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
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocol struct {
	conn       tcp.Connection
	lineReader *harvester.LineReader
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

// Read reads a line from the connection and calculates an event
// Returns nil event if shutdown, with an optional error
func (p *protocol) Read() (transports.Event, error) {
	var event map[string]interface{}
	var length int
	var err error

	for {
		event, length, err = p.lineReader.ReadItem()
		if err == nil || err == harvester.ErrMaxDataSizeTruncation {
			break
		}

		return nil, err
	}

	log.Debugf("Received event: %v", event)

	// TODO: Read multiple before generating event
	return transports.NewEventsEvent(p.conn.Context(), &transports.NilNonce, []map[string]interface{}{event}, length), nil
}

// NonBlocking returns trues because a stream does not have well defined event boundaries so we have to check periodically
func (p *protocol) NonBlocking() bool {
	return true
}
