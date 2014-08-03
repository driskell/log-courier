/*
* Copyright 2014 Jason Woods.
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

import "sync"

type Pipeline struct {
	signal chan interface{}
	sinks  map[*PipelineSegment]chan *Config
	group  sync.WaitGroup
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		signal: make(chan interface{}),
		sinks:  make(map[*PipelineSegment]chan *Config),
	}
}

func (p *Pipeline) Shutdown() {
	close(p.signal)
}

func (p *Pipeline) SendConfig(config *Config) {
	for _, sink := range p.sinks {
		sink <- config
	}
}

func (p *Pipeline) Register(s interface{}) {
	p.group.Add(1)

	var pipe *PipelineSegment = s.(*PipelineSegment)
	pipe.signal = p.signal
	pipe.group = &p.group

	if pipe_config, ok := s.(*PipelineConfigReceiver); ok {
		config_chan := make(chan *Config)
		p.sinks[pipe] = config_chan
		pipe_config.sink = config_chan
	}
}

func (p *Pipeline) Wait() {
	p.group.Wait()
}

type PipelineSegment struct {
	signal <-chan interface{}
	group  *sync.WaitGroup
}

func (s *PipelineSegment) ShutdownSignal() <-chan interface{} {
	return s.signal
}

func (s *PipelineSegment) Done() {
	s.group.Done()
}

type PipelineConfigReceiver struct {
	sink <-chan *Config
}

func (s *PipelineConfigReceiver) RecvConfig() <-chan *Config {
	return s.sink
}
