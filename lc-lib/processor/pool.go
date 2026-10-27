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

package processor

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
)

// Pool manages routines that perform sequences of mutations on events
type Pool struct {
	input        chan []*event.Event
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config

	cfg           *config.Config
	pipelines     *Config
	debugEvents   bool
	escapeControl bool
	fanout        chan []*event.Event
	collector     chan []*event.Event
	guard         *timestampGuard
	// routines is the number of processor routines running, and processed counts the events each has processed
	routines  atomic.Int64
	processed [maxProcessorRoutines]atomic.Int64
	// speeds measures how fast each routine processes events, guarded by speedsMutex
	speeds      [maxProcessorRoutines]core.SpeedMeter
	speedsMutex sync.Mutex

	apiConfig *admin.Config
}

// NewPool creates a new processor pool
func NewPool(app *core.App) *Pool {
	return &Pool{
		input:     make(chan []*event.Event, 1),
		guard:     &timestampGuard{},
		apiConfig: admin.FetchConfig(app.Config()),
	}
}

// Input returns the channel to send events to the processors with
func (p *Pool) Input() chan<- []*event.Event {
	return p.input
}

// SetOutput sets the output channel
func (p *Pool) SetOutput(output chan<- []*event.Event) {
	p.output = output
}

// SetShutdownChan sets the shutdown channel
func (p *Pool) SetShutdownChan(shutdownChan <-chan struct{}) {
	p.shutdownChan = shutdownChan
}

// SetConfigChan sets the config channel
func (p *Pool) SetConfigChan(configChan <-chan *config.Config) {
	p.configChan = configChan
}

// Init initialises
func (p *Pool) Init(cfg *config.Config) error {
	p.applyConfig(cfg)

	if p.apiConfig.Enabled {
		processorAPI := &api.Node{}
		processorAPI.SetEntry("status", &apiStatus{p: p})
		processorAPI.SetEntry("routines", &apiRoutines{p: p})
		p.apiConfig.SetEntry("processor", processorAPI)
	}

	return nil
}

// Run starts the processing routines
func (p *Pool) Run() {
	shutdownChan := p.shutdownChan
	measureTicker := time.NewTicker(time.Second)
	defer measureTicker.Stop()

	for {
		var newConfig *config.Config
		shutdown, reloading := false, false
		routineCount := p.cfg.GeneralPart("processor").(*General).ProcessorRoutines
		inProgress := 0
		inputChan := p.input

		// Setup channels
		p.fanout = make(chan []*event.Event, routineCount)
		p.collector = make(chan []*event.Event, routineCount)

		log.Infof("Processor starting %d routines", routineCount)
		p.routines.Store(int64(routineCount))
		for i := 0; i < routineCount; i++ {
			go p.processorRoutine(i)
		}

	PipelineLoop:
		for {
			select {
			case <-shutdownChan:
				shutdown = true
				shutdownChan = nil
				log.Infof("Processor shutting down %d routines", routineCount)
			case now := <-measureTicker.C:
				p.takeMeasurements(now)
			case newConfig = <-p.configChan:
				// Request shutdown so we can restart with new configuration
				close(p.fanout)
				// Do not send any more events - as the processors are shutting down
				// they will not enter the collector - so fanout could block as we have 2xroutine limit
				// which assumes something inside the collector
				reloading = true
				inputChan = nil
			case events := <-inputChan:
				// Closed input means shutting down gracefully
				if events == nil {
					shutdown = true
					close(p.fanout)
					inputChan = nil
					continue
				}
				// Max number of calls to p.fanout must not exceed 2xroutine
				// That will account for each routine inside a call to collector
				// And then one extra on the channel waiting
				// Any subsequent send would block - yet the processor is waiting on us to collect
				inProgress++
				if inProgress >= routineCount*2 {
					inputChan = nil
				}

				select {
				case <-p.shutdownChan:
				case p.fanout <- events:
				}
			case events := <-p.collector:
				if events == nil {
					// A routine shutdown
					routineCount--
					if routineCount == 0 {
						// All routines complete
						break PipelineLoop
					}
					continue
				}

				inProgress--
				if !reloading && !shutdown && inputChan == nil {
					inputChan = p.input
				}

				// Forwarded as soon as processed, as acknowledgements no longer depend on the order events are output
				select {
				case <-p.shutdownChan:
					shutdown = true
				case p.output <- events:
				}
			}
		}

		if shutdown {
			break
		}

		log.Info("Processor applying new configuration")
		p.applyConfig(newConfig)
	}

	close(p.output)
	log.Info("Processor exiting")
}

// takeMeasurements measures how fast each routine is processing events
func (p *Pool) takeMeasurements(now time.Time) {
	p.speedsMutex.Lock()
	defer p.speedsMutex.Unlock()
	for id, routines := 0, int(p.routines.Load()); id < routines; id++ {
		p.speeds[id].Measure(p.processed[id].Load(), now)
	}
}

// speed returns how fast the routine with the given id is processing events per second
func (p *Pool) speed(id int) float64 {
	p.speedsMutex.Lock()
	defer p.speedsMutex.Unlock()
	return p.speeds[id].Speed()
}

// processorRoutine runs a single routine for processing
func (p *Pool) processorRoutine(id int) {
	defer func() {
		log.Infof("[Processor %d] Processor routine exiting", id)
		p.collector <- nil
	}()
	for {
		select {
		case <-p.shutdownChan:
			return
		case events := <-p.fanout:
			if events == nil {
				return
			}

			start := time.Now()
			for idx, event := range events {
				events[idx] = p.processEvent(event)
			}

			log.Debugf("[Processor %d] Processed %d events in %v", id, len(events), time.Since(start))
			p.processed[id].Add(int64(len(events)))

			select {
			case <-p.shutdownChan:
				return
			case p.collector <- events:
			}
		}
	}
}

// processEvent processes a single event
func (p *Pool) processEvent(evnt *event.Event) *event.Event {
	for _, entry := range p.pipelines.AST {
		evnt = entry.Process(evnt)
	}
	// Escape after the pipeline so actions can still parse data separated by control characters
	if p.escapeControl {
		evnt = escapeControlEvent(evnt)
	}
	if p.guard.Check(evnt, time.Now()) {
		log.Debugf("Dropping event with out of range timestamp: %v", evnt.MustResolve("@timestamp", nil))
	}
	evnt.ClearCache()
	if p.debugEvents {
		eventJSON, _ := json.Marshal(evnt.Data())
		log.Debugf("Final event: %s", eventJSON)
	}
	return evnt
}

// applyConfig applies the given configuration
func (p *Pool) applyConfig(cfg *config.Config) {
	p.cfg = cfg
	p.pipelines = FetchConfig(cfg)
	general := cfg.GeneralPart("processor").(*General)
	p.debugEvents = general.DebugEvents
	p.escapeControl = general.EscapeControlCharacters
	p.guard.SetLimits(general.MaximumEventAge, general.MaximumFutureEventAge)
}
