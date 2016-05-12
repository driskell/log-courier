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

// IPipelineSegment is the interface a segment exposes
// Including PipelineSegment into the segment struct and defining a Run() method
// will provide this
type IPipelineSegment interface {
	Run()
	getStruct() *PipelineSegment
}

// PipelineSegment is included into pipeline segment structures and allows them
// to be registered with a Pipeline
type PipelineSegment struct {
	signal <-chan interface{}
	group  *sync.WaitGroup
}

// getStruct is a helper, made available on the parent segment structure, that
// returns back the pipeline segment
func (s *PipelineSegment) getStruct() *PipelineSegment {
	return s
}

// OnShutdown returns a channel, that when closed, signals the shutdown of the
// pipeline. All segments MUST pay attention to this channel and shutdown when
// it closes
func (s *PipelineSegment) OnShutdown() <-chan interface{} {
	return s.signal
}

// Done MUST be called by a segment to signal it has completed shutdown. The
// segment MUST exit immediately after calling this as the binary could exit at
// any moment
func (s *PipelineSegment) Done() {
	s.group.Done()
}

// IPipelineConfigReceiver is the interface a segment exposes when it wants
// notifying about configuration reload
// Including the PipelineConfigReceiver struct in the segment struct will expose
// this
type IPipelineConfigReceiver interface {
	getConfigReceiverStruct() *PipelineConfigReceiver
}

// PipelineConfigReceiver is included into a pipeline segment struct to provide
// configuration reload capabilities
type PipelineConfigReceiver struct {
	configChan <-chan *config.Config
}

// getConfigReceiverStruct is a helper, made available to the parent struct,
// that returns back the structure for config receiver
func (s *PipelineConfigReceiver) getConfigReceiverStruct() *PipelineConfigReceiver {
	return s
}

// OnConfig provides a channel which receives new configuration structures when
// a configuration reload happens
func (s *PipelineConfigReceiver) OnConfig() <-chan *config.Config {
	return s.configChan
}
