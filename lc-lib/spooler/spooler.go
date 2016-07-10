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

package spooler

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/publisher"
)

const (
	// Event header is just uint32 at the moment
	eventHeaderSize = 4
)

// Spooler gathers events into groups and sends them to the publisher in bulk
type Spooler struct {
	core.PipelineSegment
	core.PipelineConfigReceiver

	genConfig  *General
	spool      []*event.Event
	spoolSize  int
	input      chan *event.Event
	output     chan<- []*event.Event
	timerStart time.Time
	timer      *time.Timer
}

// NewSpooler creates a new event spooler
func NewSpooler(app *core.App, publisherImpl *publisher.Publisher) *Spooler {
	genConfig := app.Config().GeneralPart("spooler").(*General)
	ret := &Spooler{
		genConfig: genConfig,
		spool:     make([]*event.Event, 0, genConfig.SpoolSize),
		input:     make(chan *event.Event, 16), // TODO: Make configurable?
		output:    publisherImpl.Connect(),
	}

	app.AddToPipeline(ret)

	return ret
}

// Connect returns the channel to send events to the spooler with
func (s *Spooler) Connect() chan<- *event.Event {
	return s.input
}

// Flush requests the spooler to flush its events immediately
func (s *Spooler) Flush() {
	select {
	case s.input <- nil:
	case <-s.OnShutdown():
	}
}

// Run starts the spooling routine
func (s *Spooler) Run() {
	defer func() {
		s.Done()
	}()

	s.timerStart = time.Now()
	s.timer = time.NewTimer(s.genConfig.SpoolTimeout)

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

			if len(s.spool) > 0 && int64(s.spoolSize)+int64(len(event.Bytes()))+eventHeaderSize >= s.genConfig.SpoolMaxBytes {
				log.Debug("Spooler flushing %d events due to spool max bytes (%d/%d - next is %d)", len(s.spool), s.spoolSize, s.genConfig.SpoolMaxBytes, len(event.Bytes())+4)

				// Can't fit this event in the spool - flush and then queue
				if !s.sendSpool() {
					break SpoolerLoop
				}

				s.resetTimer()
				s.spoolSize += len(event.Bytes()) + eventHeaderSize
				s.spool = append(s.spool, event)

				continue
			}

			s.spoolSize += len(event.Bytes()) + eventHeaderSize
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

// sendSpool flushes the current spool of events to the publisher
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

	s.spool = make([]*event.Event, 0, s.genConfig.SpoolSize)
	s.spoolSize = 0

	return true
}

// resetTimer resets the time needed to wait before automatically flushing
func (s *Spooler) resetTimer() {
	s.timerStart = time.Now()

	// Stop the timer, and ensure the channel is empty before restarting it
	s.timer.Stop()
	select {
	case <-s.timer.C:
	default:
	}
	s.timer.Reset(s.genConfig.SpoolTimeout)
}

// reloadConfig updates the spooler configuration after a reload
func (s *Spooler) reloadConfig(cfg *config.Config) bool {
	s.genConfig = cfg.GeneralPart("spooler").(*General)

	// Immediate flush?
	passed := time.Now().Sub(s.timerStart)
	if passed >= s.genConfig.SpoolTimeout || len(s.spool) >= int(s.genConfig.SpoolSize) {
		if !s.sendSpool() {
			return false
		}
		s.timerStart = time.Now()
		s.timer.Reset(s.genConfig.SpoolTimeout)
	} else {
		s.timer.Reset(passed - s.genConfig.SpoolTimeout)
	}

	return true
}
