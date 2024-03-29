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

package test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// transportTest implements a transport that discards what it receives
// It is useful for testing
type transportTest struct {
	ctx       context.Context
	config    *TransportTestFactory
	eventChan chan<- transports.Event
	server    string
	finished  bool
}

// Factory returns the associated factory
func (t *transportTest) Factory() transports.TransportFactory {
	return t.config
}

// SendEvents sends an event message with given nonce to the transport - only valid after Started transport event received
func (t *transportTest) SendEvents(nonce string, events []*event.Event) error {
	endSequence := uint32(len(events))
	sequence := endSequence
	if sequence > 1 && rand.Intn(2) == 0 {
		// Generate a split acknowledgement
		sequence = uint32(rand.Intn(int(sequence)-1) + 1)
	}

	t.delayAction(func() {
		t.sendAck(nonce, sequence)
		if sequence != endSequence {
			t.delayAction(func() { t.sendAck(nonce, endSequence) }, fmt.Sprintf("[T %s] Acknowledging %d events for payload %x", t.server, endSequence, nonce))
		}
	}, fmt.Sprintf("[T %s] Acknowledging %d events for payload %x", t.server, sequence, nonce))
	return nil
}

func (t *transportTest) sendAck(nonce string, sequence uint32) {
	if t.finished {
		return
	}
	t.eventChan <- transports.NewAckEvent(t.ctx, &nonce, sequence)
}

func (t *transportTest) delayAction(action func(), message string) {
	delay := t.config.MinDelay
	if t.config.MinDelay != t.config.MaxDelay {
		delay = delay + rand.Int63n(t.config.MaxDelay-t.config.MinDelay)
	}
	// Always launch in another routine - must not block the publisher
	// which can happen if we attempt to return to the eventChan on same channel we received
	if delay == 0 {
		go func() {
			log.Debugf("%s", message)
			action()
		}()
	} else {
		log.Debugf("%s (delaying %d seconds)", message, delay)
		go func() {
			<-time.After(time.Second * time.Duration(delay))
			log.Debugf("%s", message)
			action()
		}()
	}
}

// Ping the remote server - only valid after Started transport event received
func (t *transportTest) Ping() error {
	t.delayAction(func() {
		if t.finished {
			return
		}
		t.eventChan <- transports.NewPongEvent(t.ctx)
	}, fmt.Sprintf("[T %s] Sending pong response", t.server))
	return nil
}

// Fail the transport / Shutdown hard
func (t *transportTest) Fail() {
	t.finished = true
	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
}

// Shutdown the transport gracefully - only valid after Started transport event received
func (t *transportTest) Shutdown() {
	t.finished = true
	t.eventChan <- transports.NewStatusEvent(t.ctx, transports.Finished, nil)
}
