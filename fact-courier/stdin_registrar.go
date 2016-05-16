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
	"github.com/driskell/log-courier/lc-lib/registrar"
)

// TODO: This is copied verbatim from Log Courier which uses this as simple
// reporting for Stdin processing and for allowing us to shutdown only once
// the event buffers have emptied
// It needs a name change to reflect better what it does in both Fact Courier
// and Log Courier
// Additionally - the real registrar need a name change too and should be tied
// into the prospector package as the file-based registrar is essentially a
// backend for prospector state

type StdinRegistrar struct {
	core.PipelineSegment

	sync.Mutex

	group          sync.WaitGroup
	registrar_chan chan []registrar.EventProcessor
	signal_chan    chan int64
	references     int
	wait_offset    *int64
	last_offset    int64
}

func newStdinRegistrar(app *core.App) *StdinRegistrar {
	ret := &StdinRegistrar{
		registrar_chan: make(chan []registrar.EventProcessor, 16),
		signal_chan:    make(chan int64, 1),
	}

	ret.group.Add(1)

	return ret
}

func (r *StdinRegistrar) Run() {
	defer func() {
		r.Done()
		r.group.Done()
	}()

	state := make(map[core.Stream]*registrar.FileState)
	state[nil] = &registrar.FileState{}

RegistrarLoop:
	for {
		select {
		case signal := <-r.signal_chan:
			r.wait_offset = new(int64)
			*r.wait_offset = signal

			if r.last_offset == signal {
				break RegistrarLoop
			}

			log.Debug("Stdin registrar received stdin EOF offset of %d", *r.wait_offset)
		case events := <-r.registrar_chan:
			for _, event := range events {
				event.Process(state)
			}

			r.last_offset = state[nil].Offset

			if r.wait_offset != nil && state[nil].Offset >= *r.wait_offset {
				log.Debug("Stdin registrar has reached end of stdin")
				break RegistrarLoop
			}
		case <-r.OnShutdown():
			break RegistrarLoop
		}
	}

	log.Info("Stdin registrar exiting")
}

func (r *StdinRegistrar) Connect() registrar.EventSpooler {
	r.Lock()
	defer r.Unlock()
	r.references++
	return newStdinEventSpool(r)
}

func (r *StdinRegistrar) Wait(offset int64) {
	r.signal_chan <- offset
	r.group.Wait()
}

func (r *StdinRegistrar) LoadPrevious(registrar.LoadPreviousFunc) (bool, error) {
	return false, nil
}

func (r *StdinRegistrar) dereferenceSpooler() {
	r.Lock()
	defer r.Unlock()
	r.references--
	if r.references == 0 {
		close(r.registrar_chan)
	}
}

type StdinEventSpool struct {
	registrar *StdinRegistrar
	events    []registrar.EventProcessor
}

func newStdinEventSpool(r *StdinRegistrar) *StdinEventSpool {
	ret := &StdinEventSpool{
		registrar: r,
	}
	ret.reset()
	return ret
}

func (r *StdinEventSpool) Close() {
	r.registrar.dereferenceSpooler()
	r.registrar = nil
}

func (r *StdinEventSpool) Add(event registrar.EventProcessor) {
	// StdinEventSpool is only interested in AckEvents
	if _, ok := event.(*registrar.AckEvent); !ok {
		return
	}

	r.events = append(r.events, event)
}

func (r *StdinEventSpool) Send() {
	if len(r.events) != 0 {
		r.registrar.registrar_chan <- r.events
		r.reset()
	}
}

func (r *StdinEventSpool) reset() {
	r.events = make([]registrar.EventProcessor, 0, 0)
}
