/*
 * Copyright 2014 Jason Woods.
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

package spooler

import (
	"github.com/driskell/log-courier/src/lc-lib/config"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/publisher"
	"time"
)

const (
	// Event header is just uint32 at the moment
	event_header_size = 4
)

type Spooler struct {
	core.PipelineSegment
	core.PipelineConfigReceiver

	config      *config.General
	spool       []*core.EventDescriptor
	spool_size  int
	input       chan *core.EventDescriptor
	output      chan<- []*core.EventDescriptor
	timer_start time.Time
	timer       *time.Timer
}

func NewSpooler(pipeline *core.Pipeline, config *config.General, publisher_imp *publisher.Publisher) *Spooler {
	ret := &Spooler{
		config: config,
		spool:  make([]*core.EventDescriptor, 0, config.SpoolSize),
		input:  make(chan *core.EventDescriptor, 16), // TODO: Make configurable?
		output: publisher_imp.Connect(),
	}

	pipeline.Register(ret)

	return ret
}

func (s *Spooler) Connect() chan<- *core.EventDescriptor {
	return s.input
}

func (s *Spooler) Flush() {
	s.input <- nil
}

func (s *Spooler) Run() {
	defer func() {
		s.Done()
	}()

	s.timer_start = time.Now()
	s.timer = time.NewTimer(s.config.SpoolTimeout)

SpoolerLoop:
	for {
		select {
		case event := <-s.input:
			// Nil event means flush
			if event == nil {
				if len(s.spool) > 0 {
					log.Debug("Spooler flushing %d events due to flush event", len(s.spool))

					if !s.sendSpool() {
						break SpoolerLoop
					}
				}

				continue
			}

			if len(s.spool) > 0 && int64(s.spool_size)+int64(len(event.Event))+event_header_size >= s.config.SpoolMaxBytes {
				log.Debug("Spooler flushing %d events due to spool max bytes (%d/%d - next is %d)", len(s.spool), s.spool_size, s.config.SpoolMaxBytes, len(event.Event)+4)

				// Can't fit this event in the spool - flush and then queue
				if !s.sendSpool() {
					break SpoolerLoop
				}

				s.resetTimer()
				s.spool_size += len(event.Event) + event_header_size
				s.spool = append(s.spool, event)

				continue
			}

			s.spool_size += len(event.Event) + event_header_size
			s.spool = append(s.spool, event)

			// Flush if full
			if len(s.spool) >= cap(s.spool) {
				log.Debug("Spooler flushing %d events due to spool size reached", len(s.spool))

				if !s.sendSpool() {
					break SpoolerLoop
				}

				s.resetTimer()
			}
		case <-s.timer.C:
			// Flush what we have, if anything
			if len(s.spool) > 0 {
				log.Debug("Spooler flushing %d events due to spool timeout exceeded", len(s.spool))

				if !s.sendSpool() {
					break SpoolerLoop
				}
			}

			s.resetTimer()
		case <-s.OnShutdown():
			break SpoolerLoop
		case config := <-s.OnConfig():
			if !s.reloadConfig(config) {
				break SpoolerLoop
			}
		}
	}

	log.Info("Spooler exiting")
}

func (s *Spooler) sendSpool() bool {
	select {
	case <-s.OnShutdown():
		return false
	case config := <-s.OnConfig():
		if !s.reloadConfig(config) {
			return false
		}
	case s.output <- s.spool:
	}

	s.spool = make([]*core.EventDescriptor, 0, s.config.SpoolSize)
	s.spool_size = 0

	return true
}

func (s *Spooler) resetTimer() {
	s.timer_start = time.Now()

	// Stop the timer, and ensure the channel is empty before restarting it
	s.timer.Stop()
	select {
	case <-s.timer.C:
	default:
	}
	s.timer.Reset(s.config.SpoolTimeout)
}

func (s *Spooler) reloadConfig(config *config.Config) bool {
	s.config = &config.General

	// Immediate flush?
	passed := time.Now().Sub(s.timer_start)
	if passed >= s.config.SpoolTimeout || len(s.spool) >= int(s.config.SpoolSize) {
		if !s.sendSpool() {
			return false
		}
		s.timer_start = time.Now()
		s.timer.Reset(s.config.SpoolTimeout)
	} else {
		s.timer.Reset(passed - s.config.SpoolTimeout)
	}

	return true
}
