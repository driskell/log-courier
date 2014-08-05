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
  pipes  []IPipelineSegment
  sinks  map[*PipelineConfigReceiver]chan *Config
  group  sync.WaitGroup
}

func NewPipeline() *Pipeline {
  return &Pipeline{
    signal: make(chan interface{}),
    sinks:  make(map[*PipelineConfigReceiver]chan *Config),
    pipes:  make([]IPipelineSegment, 0, 5),
  }
}

func (p *Pipeline) Register(ipipe IPipelineSegment) {
  p.group.Add(1)

  pipe := ipipe.getStruct()
  pipe.signal = p.signal
  pipe.group = &p.group

  p.pipes = append(p.pipes, ipipe)

  if ipipe_ext, ok := ipipe.(IPipelineConfigReceiver); ok {
    pipe_ext := ipipe_ext.getConfigReceiverStruct()
    config_chan := make(chan *Config)
    p.sinks[pipe_ext] = config_chan
    pipe_ext.sink = config_chan
  }
}

func (p *Pipeline) Start() {
  for _, ipipe := range p.pipes {
    go ipipe.Run()
  }
}

func (p *Pipeline) Shutdown() {
  close(p.signal)
}

func (p *Pipeline) Wait() {
  p.group.Wait()
}

func (p *Pipeline) SendConfig(config *Config) {
  for _, sink := range p.sinks {
    sink <- config
  }
}

type IPipelineSegment interface {
  Run()
  getStruct() *PipelineSegment
}

type PipelineSegment struct {
  signal <-chan interface{}
  group  *sync.WaitGroup
}

func (s *PipelineSegment) Run() {
  panic("Run() not implemented")
}

func (s *PipelineSegment) getStruct() *PipelineSegment {
  return s
}

func (s *PipelineSegment) ShutdownSignal() <-chan interface{} {
  return s.signal
}

func (s *PipelineSegment) Done() {
  s.group.Done()
}

type IPipelineConfigReceiver interface {
  getConfigReceiverStruct() *PipelineConfigReceiver
}

type PipelineConfigReceiver struct {
  sink <-chan *Config
}

func (s *PipelineConfigReceiver) getConfigReceiverStruct() *PipelineConfigReceiver {
  return s
}

func (s *PipelineConfigReceiver) RecvConfig() <-chan *Config {
  return s.sink
}
