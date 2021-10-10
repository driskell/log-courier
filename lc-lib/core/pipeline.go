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

package core

import (
	"math/rand"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Pipeline represents a shipper instance
// It holds references to each "pipeline worker" and provides start/stop and
// config reload and other features to them
type Pipeline struct {
	sources          []pipelineSourceSegment
	sink             pipelineSinkSegment
	processors       []pipelineProcessorSegment
	services         []pipelineServiceSegment
	signal           chan struct{}
	signalSources    chan struct{}
	group            sync.WaitGroup
	groupSources     sync.WaitGroup
	groupServices    sync.WaitGroup
	closeSinkOnce    sync.Once
	closeServiceOnce sync.Once
	configSinks      map[pipelineConfigSegment]chan *config.Config
}

// NewPipeline creates a new pipeline
func NewPipeline() *Pipeline {
	p := &Pipeline{
		signal:        make(chan struct{}),
		signalSources: make(chan struct{}),
		configSinks:   make(map[pipelineConfigSegment]chan *config.Config),
	}

	return p
}

// AddSource adds a source segment
func (p *Pipeline) AddSource(source pipelineSourceSegment) {
	p.registerSegment(source)
	p.sources = append(p.sources, source)
}

// AddProcessor adds a processor segment
func (p *Pipeline) AddProcessor(processor pipelineProcessorSegment) {
	p.registerSegment(processor)
	p.processors = append(p.processors, processor)
}

// SetSink sets the sink segment
func (p *Pipeline) SetSink(sink pipelineSinkSegment) {
	if p.sink != nil {
		panic("Can only set one sink segment")
	}
	p.registerSegment(sink)
	p.sink = sink
}

// AddService adds a service segment
func (p *Pipeline) AddService(service pipelineServiceSegment) {
	p.registerSegment(service)
	p.services = append(p.services, service)
}

// registerSegment adds a segment to any relevant handlers
func (p *Pipeline) registerSegment(segment pipelineSegment) {
	if configSegment, ok := segment.(pipelineConfigSegment); ok {
		configChan := make(chan *config.Config)
		p.configSinks[configSegment] = configChan
		configSegment.SetConfigChan(configChan)
	}
}

// Run the pipeline, starting up each segment and then waiting for sink to finish or shutdown
func (p *Pipeline) Run(config *config.Config) {
	// Reseed rand
	rand.Seed(time.Now().UnixNano())

	if err := p.initRoutines(config); err != nil {
		log.Error("Pipeline failed: %s", err)
		return
	}

	// Wait for sink to complete
	log.Notice("Pipeline ready")
	p.group.Add(1)
	p.run(&p.group, p.sink.Run)
	// If the sink is not accepting events anymore then nothing is needed anymore
	p.shutdownAll()
}

// Run the pipeline, starting up each segment and then waiting for sink to finish or shutdown
func (p *Pipeline) initRoutines(config *config.Config) (err error) {
	if len(p.sources) == 0 {
		panic("Must have at least a single source segment")
	}
	if p.sink == nil {
		panic("Missing sink segment")
	}
	for _, service := range p.services {
		service.SetShutdownChan(p.signal)
		if err = service.Init(config); err != nil {
			p.shutdownAll()
			return
		}
		p.groupServices.Add(1)
		go p.run(&p.groupServices, service.Run)
	}
	if err = p.sink.Init(config); err != nil {
		p.shutdownAll()
		return
	}
	input := p.sink.Input()
	for i := len(p.processors) - 1; i >= 0; i-- {
		p.processors[i].SetOutput(input)
		if err = p.processors[i].Init(config); err != nil {
			p.shutdownAll()
			return
		}
		p.group.Add(1)
		go p.run(&p.group, p.processors[i].Run)
		input = p.processors[i].Input()
	}
	for _, source := range p.sources {
		source.SetOutput(input)
		source.SetShutdownChan(p.signalSources)
		if err = source.Init(config); err != nil {
			p.shutdownAll()
			return
		}
		p.groupSources.Add(1)
		go p.run(&p.groupSources, source.Run)
	}
	go func() {
		p.groupSources.Wait()
		close(input)
	}()
	return nil
}

// run wraps the pipeline segment Run method and closes the group when finished
func (p *Pipeline) run(group *sync.WaitGroup, run func()) {
	defer func() {
		group.Done()
	}()

	run()
}

// shutdownAll stops everything - should only run once sink dies
// or if an error occurs during startup (which means sink didn't start yet - as that's last)
func (p *Pipeline) shutdownAll() {
	p.Shutdown()
	p.Wait()
}

// Shutdown stops the pipeline, requesting sources to shutdown
func (p *Pipeline) Shutdown() {
	p.closeSinkOnce.Do(func() {
		log.Notice("Pipeline shutting down")
		close(p.signalSources)
	})
}

// Wait sleeps until the pipeline has completely shutdown
func (p *Pipeline) Wait() {
	p.groupSources.Wait()
	p.group.Wait()
	p.closeServiceOnce.Do(func() {
		log.Notice("Services shutting down")
		close(p.signal)
	})
	p.groupServices.Wait()
}

// SendConfig broadcasts the given configuration to all segments
// It will hang if a previous configuration broadcast has not yet completed so
// do not call this too often
func (p *Pipeline) SendConfig(config *config.Config) {
	for _, sink := range p.configSinks {
		sink <- config
	}
}
