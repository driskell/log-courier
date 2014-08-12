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
  "fmt"
  "sync"
)

type Pipeline struct {
  pipes          []IPipelineSegment
  signal         chan interface{}
  group          sync.WaitGroup
  config_sinks   map[*PipelineConfigReceiver]chan *Config
  snapshot_chan  chan []*Snapshot
  snapshot_sinks map[*PipelineSnapshotProvider]chan interface{}
}

func NewPipeline() *Pipeline {
  return &Pipeline{
    pipes:          make([]IPipelineSegment, 0, 5),
    signal:         make(chan interface{}),
    config_sinks:   make(map[*PipelineConfigReceiver]chan *Config),
    snapshot_chan:  make(chan []*Snapshot),
    snapshot_sinks: make(map[*PipelineSnapshotProvider]chan interface{}),
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
    sink := make(chan *Config)
    p.config_sinks[pipe_ext] = sink
    pipe_ext.config_chan = sink
  }

  if ipipe_ext, ok := ipipe.(IPipelineSnapshotProvider); ok {
    pipe_ext := ipipe_ext.getSnapshotProviderStruct()
    sink := make(chan interface{}, 1)
    p.snapshot_sinks[pipe_ext] = sink
    pipe_ext.snapshot_chan = sink
    pipe_ext.sink = p.snapshot_chan
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
  for _, sink := range p.config_sinks {
    sink <- config
  }
}

func (p *Pipeline) Snapshot() ([]*Snapshot, error) {
  left := 0
  for _, sink := range p.snapshot_sinks {
    sink <- 1
    left++
  }

  var ret []*Snapshot

  snapshots := make([]*Snapshot, 0)
// TODO: Fix race condition here with shutdown that causes hang/crash
  for {
    left--
    select {
    case ret = <-p.snapshot_chan:
    case <-p.signal:
      return nil, fmt.Errorf("Log Courier is shutting down")
    }

    snapshots = append(snapshots, ret...)

    if left == 0 {
      break
    }
  }

  return snapshots, nil
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
  config_chan <-chan *Config
}

func (s *PipelineConfigReceiver) getConfigReceiverStruct() *PipelineConfigReceiver {
  return s
}

func (s *PipelineConfigReceiver) OnConfig() <-chan *Config {
  return s.config_chan
}

type IPipelineSnapshotProvider interface {
  getSnapshotProviderStruct() *PipelineSnapshotProvider
}

type PipelineSnapshotProvider struct {
  snapshot_chan <-chan interface{}
  sink          chan<- []*Snapshot
}

func (s *PipelineSnapshotProvider) getSnapshotProviderStruct() *PipelineSnapshotProvider {
  return s
}

func (s *PipelineSnapshotProvider) OnSnapshot() <-chan interface{} {
  return s.snapshot_chan
}

func (s *PipelineSnapshotProvider) SendSnapshot(snapshot []*Snapshot) {
  s.sink <- snapshot
}
