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

package processor

import (
	"context"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/google/cel-go/common/types"
)

// Pool manages routines that perform sequences of mutations on events
type Pool struct {
	input        chan []*event.Event
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config

	wait      sync.WaitGroup
	pipelines []*PipelineConfig
}

// NewPool creates a new processor pool
func NewPool(app *core.App) *Pool {
	return &Pool{
		input: make(chan []*event.Event, 16), // TODO: Make configurable?
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

// Init does nothing as nothing to initialise
func (p *Pool) Init(cfg *config.Config) error {
	p.pipelines = FetchConfig(cfg)
	return nil
}

// Run starts the processing routines
func (p *Pool) Run() {
	for {
		ctx, shutdownFunc := context.WithCancel(context.Background())

		for i := 0; i < 4; i++ {
			p.wait.Add(1)
			go p.processorRoutine(ctx.Done(), nil)
		}

		p.wait.Add(1)
		newConfig := p.processorRoutine(nil, p.configChan)
		if newConfig == nil {
			// If primary exits without config reload data it means we're shutting down
			// All other routines will follow shortly
			shutdownFunc()
			p.wait.Wait()
			break
		}

		// Soft stop all other routines and restart
		shutdownFunc()
		p.wait.Wait()

		p.pipelines = FetchConfig(newConfig)
	}

	close(p.output)
	log.Info("Processor exiting")
}

// processorRoutine runs a single routine for processing
func (p *Pool) processorRoutine(softShutdownChan <-chan struct{}, configChan <-chan *config.Config) *config.Config {
	defer func() {
		p.wait.Done()
	}()

	for {
		select {
		case <-p.shutdownChan:
			return nil
		case <-softShutdownChan:
			return nil
		case newConfig := <-configChan:
			// Return the new configuration
			return newConfig
		case events := <-p.input:
			if events == nil {
				// Finished!
				return nil
			}

			start := time.Now()
			for idx, event := range events {
				events[idx] = p.processEvent(event)
			}
			log.Debugf("Processed %d events in %v", len(events), time.Since(start))

			select {
			case <-p.shutdownChan:
				return nil
			case p.output <- events:
			}
		}
	}
}

// processEvent processes a single event
func (p *Pool) processEvent(event *event.Event) *event.Event {
	for _, pipeline := range p.pipelines {
		val, _, err := pipeline.conditionProgram.Eval(map[string]interface{}{"event": event.Data()})
		if err != nil {
			log.Warningf("Failed to evaluate pipeline condition: [%s] -> %s", pipeline.ConditionExpr, err)
			continue
		}
		if val == types.True {
			event = p.processEventInPipeline(pipeline, event)
		}
	}
	return event
}

// processEventInPipeline processes a single event in a pipeline
func (p *Pool) processEventInPipeline(pipeline *PipelineConfig, event *event.Event) *event.Event {
	for _, action := range pipeline.Actions {
		event = action.Handler.Process(event)
	}
	event.ClearCache()
	return event
}
