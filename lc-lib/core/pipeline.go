/*
 * Copyright 2014-2015 Jason Woods.
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
	"sync"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Pipeline represents a shipper instance
// It holds references to each "pipeline worker" and provides start/stop and
// config reload and other features to them
type Pipeline struct {
	pipes       []IPipelineSegment
	signal      chan interface{}
	group       sync.WaitGroup
	configSinks map[*PipelineConfigReceiver]chan *config.Config
}

// NewPipeline creates a new pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		pipes:       make([]IPipelineSegment, 0, 5),
		signal:      make(chan interface{}),
		configSinks: make(map[*PipelineConfigReceiver]chan *config.Config),
	}
}

// Add a pipe segment to the pipeline
// A pipe segment is any struct that includes a PipelineSegment structure
// Segments can also include PipelineConfigReceiver structures and others to
// provide additional functionality to them
func (p *Pipeline) Add(ipipe IPipelineSegment) {
	p.group.Add(1)

	pipe := ipipe.getStruct()
	pipe.signal = p.signal
	pipe.group = &p.group

	p.pipes = append(p.pipes, ipipe)

	if ipipeExt, ok := ipipe.(IPipelineConfigReceiver); ok {
		pipeExt := ipipeExt.getConfigReceiverStruct()
		sink := make(chan *config.Config)
		p.configSinks[pipeExt] = sink
		pipeExt.configChan = sink
	}
}

// Start runs the pipeline, starting up each segment
func (p *Pipeline) Start() {
	for _, ipipe := range p.pipes {
		go ipipe.Run()
	}
}

// Shutdown stops the pipeline, requesting each segment to shutdown
func (p *Pipeline) Shutdown() {
	close(p.signal)
}

// Wait sleeps until the pipeline has completely shutdown
func (p *Pipeline) Wait() {
	p.group.Wait()
}

// SendConfig broadcasts the given configuration to all segments
// It will hang if a previous configuration broadcast has not yet completed so
// do not call this too often
func (p *Pipeline) SendConfig(config *config.Config) {
	for _, sink := range p.configSinks {
		sink <- config
	}
}
