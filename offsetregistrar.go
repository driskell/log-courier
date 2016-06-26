/*
 * Copyright 2014-2015 Jason Woods.
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

package main

import (
	"sync"

	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

// OffsetRegistrar is a registrar that receives event acknowledgements and does
// not allow shutdown until a given offset has been acknowledged
// Used for STDIN harvesting to ensure all data was acknowledged before shutting
// down after EOF
type OffsetRegistrar struct {
	core.PipelineSegment

	sync.Mutex

	group         sync.WaitGroup
	registrarChan chan []*event.Event
	signalChan    chan int64
	waitOffset    *int64
	lastOffset    int64
}

// newOffsetRegistrar creates a new OffsetRegistrar
func newOffsetRegistrar(app *core.App) *OffsetRegistrar {
	ret := &OffsetRegistrar{
		registrarChan: make(chan []*event.Event, 16),
		signalChan:    make(chan int64, 1),
	}

	ret.group.Add(1)

	event.RegisterForAck(harvester.EventType, ret.ackFunc)

	return ret
}

// ackFunc is called when an acknowledgement is made by the publisher
func (r *OffsetRegistrar) ackFunc(events []*event.Event) {
	r.registrarChan <- events
}

// Run starts the offset registrar routine
func (r *OffsetRegistrar) Run() {
	defer func() {
		r.Done()
		r.group.Done()
	}()

OffsetLoop:
	for {
		select {
		case signal := <-r.signalChan:
			r.waitOffset = new(int64)
			*r.waitOffset = signal

			if r.lastOffset == signal {
				break OffsetLoop
			}

			log.Debug("Offset registrar received stdin EOF offset of %d", *r.waitOffset)
		case events := <-r.registrarChan:
			context := events[len(events)-1].Context().(*harvester.EventContext)

			r.lastOffset = context.Offset

			if r.waitOffset != nil && context.Offset >= *r.waitOffset {
				log.Debug("Offset registrar has reached end of stdin")
				break OffsetLoop
			}
		case <-r.OnShutdown():
			break OffsetLoop
		}
	}

	log.Info("Offset registrar exiting")
}

// Wait does not return until the given offset was received by the
// OffsetRegistrar from the publisher
func (r *OffsetRegistrar) Wait(offset int64) {
	r.signalChan <- offset
	r.group.Wait()
}
