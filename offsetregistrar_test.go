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
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

func newTestOffsetRegistrar() (*core.Pipeline, *OffsetRegistrar) {
	pipeline := core.NewPipeline()
	// TODO: Exposing pipeline from a testing app or implementing stop/wait
	registrarImpl := newOffsetRegistrar(nil)
	pipeline.Add(registrarImpl)
	return pipeline, registrarImpl
}

func newEventSpool(offset int64) []*event.Event {
	// Prepare an event spool with single event of specified offset
	return []*event.Event{
		event.NewEvent(
			"stdin",
			map[string]interface{}{},
			&harvester.EventContext{
				Offset: offset,
			},
		),
	}
}

func TestOffsetRegistrarWait(t *testing.T) {
	p, r := newTestOffsetRegistrar()

	// Start the stdin registrar
	go func() {
		r.Run()
	}()

	r.ackFunc(newEventSpool(13))
	r.Wait(13)

	wait := make(chan int)
	go func() {
		p.Wait()
		wait <- 1
	}()

	select {
	case <-wait:
		break
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for stdin registrar shutdown")
		return
	}

	if r.lastOffset != 13 {
		t.Error("Last offset was incorrect: ", r.lastOffset)
	} else if r.waitOffset == nil || *r.waitOffset != 13 {
		t.Error("Wait offset was incorrect: ", r.waitOffset)
	}
}
