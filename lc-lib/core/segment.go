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
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

// pipelineSegment is the interface all segments implement
type pipelineSegment interface {
	Init(*config.Config) error
	Run()
}

// pipelineConfigSegment is implemented by segments that can receive configuration
// All segments have a Run routine
type pipelineConfigSegment interface {
	pipelineSegment
	SetConfigChan(<-chan *config.Config)
}

// pipelineServiceSegment is implemented by service segments that just need shutdown
type pipelineServiceSegment interface {
	pipelineSegment
	SetShutdownChan(<-chan struct{})
}

// pipelineSourceSegment is implemented by source segments
type pipelineSourceSegment interface {
	pipelineSegment
	SetShutdownChan(<-chan struct{})
	SetOutput(chan<- []*event.Event)
}

// pipelineSinkSegment is implemented by sink segments
type pipelineSinkSegment interface {
	pipelineSegment
	Input() chan<- []*event.Event
}

// pipelineProcessorSegment is implemented by processor segments
type pipelineProcessorSegment interface {
	pipelineSegment
	Input() chan<- []*event.Event
	SetOutput(chan<- []*event.Event)
}
