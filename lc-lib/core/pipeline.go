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

import (
	"github.com/driskell/log-courier/src/lc-lib/config"
	"sync"
)

type Pipeline struct {
	pipes          []IPipelineSegment
	signal         chan interface{}
	group          sync.WaitGroup
	config_sinks   map[*PipelineConfigReceiver]chan *config.Config
	snapshot_chan  chan []*Snapshot
	snapshot_pipes map[IPipelineSnapshotProvider]IPipelineSnapshotProvider
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		pipes:          make([]IPipelineSegment, 0, 5),
		signal:         make(chan interface{}),
		config_sinks:   make(map[*PipelineConfigReceiver]chan *config.Config),
		snapshot_chan:  make(chan []*Snapshot),
		snapshot_pipes: make(map[IPipelineSnapshotProvider]IPipelineSnapshotProvider),
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
		sink := make(chan *config.Config)
		p.config_sinks[pipe_ext] = sink
		pipe_ext.config_chan = sink
	}

	if ipipe_ext, ok := ipipe.(IPipelineSnapshotProvider); ok {
		p.snapshot_pipes[ipipe_ext] = ipipe_ext
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

func (p *Pipeline) SendConfig(config *config.Config) {
	for _, sink := range p.config_sinks {
		sink <- config
	}
}

func (p *Pipeline) Snapshot() *Snapshot {
	snap := NewSnapshot("Log Courier")

	for _, sink := range p.snapshot_pipes {
		for _, sub := range sink.Snapshot() {
			snap.AddSub(sub)
		}
	}

	return snap
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

func (s *PipelineSegment) OnShutdown() <-chan interface{} {
	return s.signal
}

func (s *PipelineSegment) Done() {
	s.group.Done()
}

type IPipelineConfigReceiver interface {
	getConfigReceiverStruct() *PipelineConfigReceiver
}

type PipelineConfigReceiver struct {
	config_chan <-chan *config.Config
}

func (s *PipelineConfigReceiver) getConfigReceiverStruct() *PipelineConfigReceiver {
	return s
}

func (s *PipelineConfigReceiver) OnConfig() <-chan *config.Config {
	return s.config_chan
}

type IPipelineSnapshotProvider interface {
	Snapshot() []*Snapshot
}

type PipelineSnapshotProvider struct {
}

func (s *PipelineSnapshotProvider) getSnapshotProviderStruct() *PipelineSnapshotProvider {
	return s
}

func (s *PipelineSnapshotProvider) Snapshot() []*Snapshot {
	ret := NewSnapshot("Unknown")
	ret.AddEntry("Error", "NotImplemented")
	return []*Snapshot{ret}
}
