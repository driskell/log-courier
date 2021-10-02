/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package null

import (
	"context"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// transportNull implements a transport that discards what it receives
// It is useful for testing
type transportNull struct {
	ctx       context.Context
	eventChan chan<- transports.Event
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *transportNull) ReloadConfig(netConfig *transports.Config, finishOnFail bool) bool {
	return false
}

// SendEvents sends an event message with given nonce to the transport - only valid after Started transport event received
func (t *transportNull) SendEvents(nonce string, events []*event.Event) error {
	log.Debugf("[Null] Sending immediate acknowledgement for payload %x", nonce)
	t.eventChan <- transports.NewAckEvent(t.ctx, &nonce, uint32(len(events)))
	return nil
}

// Ping the remote server - only valid after Started transport event received
func (t *transportNull) Ping() error {
	log.Debug("[Null] Sending immediate PONG response")
	t.eventChan <- transports.NewPongEvent(t.ctx)
	return nil
}

// Fail the transport / Shutdown hard
func (t *transportNull) Fail() {
	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished)
	return
}

// Shutdown the transport gracefully - only valid after Started transport event received
func (t *transportNull) Shutdown() {
	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished)
	return
}
