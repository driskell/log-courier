/*
 * Copyright 2012-2020 Jason Woods and contributors
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
)

const (
	// Event header is just uint32 at the moment
	eventHeaderSize = 4
)

// Spooler gathers events into groups and sends them to the publisher in bulk
type Spooler struct {
	genConfig    *General
	spool        []*event.Event
	spoolSize    int
	shutdownChan <-chan struct{}
	configChan   <-chan *config.Config
	input        chan []*event.Event
	output       chan<- []*event.Event
	timerStart   time.Time
	timer        *time.Timer
}

// NewSpooler creates a new event spooler
func NewSpooler(app *core.App) *Spooler {
	genConfig := app.Config().GeneralPart("spooler").(*General)
	return &Spooler{
		genConfig: genConfig,
		input:     make(chan []*event.Event, 16), // TODO: Make configurable?
		spool:     make([]*event.Event, 0, genConfig.SpoolSize),
	}
}

// Input returns the channel to send events to the spooler with
func (s *Spooler) Input() chan<- []*event.Event {
	return s.input
}

// SetOutput sets the output channel
func (s *Spooler) SetOutput(output chan<- []*event.Event) {
	s.output = output
}

// SetShutdownChan sets the shutdown channel
func (s *Spooler) SetShutdownChan(shutdownChan <-chan struct{}) {
	s.shutdownChan = shutdownChan
}

// SetConfigChan sets the config channel
func (s *Spooler) SetConfigChan(configChan <-chan *config.Config) {
	s.configChan = configChan
}

// Init does nothing as nothing to initialise
func (s *Spooler) Init(cfg *config.Config) error {
	return nil
}

// Run starts the spooling routine
func (s *Spooler) Run() {
	s.timerStart = time.Now()
	s.timer = time.NewTimer(s.genConfig.SpoolTimeout)

SpoolerLoop:
	for {
		select {
		case events := <-s.input:
			// If nil, shutting down
			if events == nil {
				break SpoolerLoop
			}

			// Empty events means flush
			if len(events) == 0 {
				if len(s.spool) > 0 {
					log.Debug("Spooler flushing %d events due to flush event", len(s.spool))

					if !s.sendSpool() {
						break SpoolerLoop
					}
				}

				continue
			}

			for _, newEvent := range events {
				if len(s.spool) > 0 && int64(s.spoolSize)+int64(len(newEvent.Bytes()))+eventHeaderSize >= s.genConfig.SpoolMaxBytes {
					log.Debug("Spooler flushing %d events due to spool max bytes (%d/%d - next is %d)", len(s.spool), s.spoolSize, s.genConfig.SpoolMaxBytes, len(newEvent.Bytes())+4)

					// Can't fit this event in the spool - flush and then queue
					if !s.sendSpool() {
						break SpoolerLoop
					}

					s.resetTimer()
					s.spoolSize += len(newEvent.Bytes()) + eventHeaderSize
					s.spool = append(s.spool, newEvent)

					continue
				}

				s.spoolSize += len(newEvent.Bytes()) + eventHeaderSize
				s.spool = append(s.spool, newEvent)

				// Flush if full
				if len(s.spool) >= cap(s.spool) {
					log.Debug("Spooler flushing %d events due to spool size reached", len(s.spool))

					if !s.sendSpool() {
						break SpoolerLoop
					}

					s.resetTimer()
				}
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
		case <-s.shutdownChan:
			break SpoolerLoop
		case config := <-s.configChan:
			if !s.reloadConfig(config) {
				break SpoolerLoop
			}
		}
	}

	// No more events
	close(s.output)

	log.Info("Spooler exiting")
}

// sendSpool flushes the current spool of events to the publisher
func (s *Spooler) sendSpool() bool {
	select {
	case <-s.shutdownChan:
		return false
	case config := <-s.configChan:
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
	if !s.timer.Stop() {
		<-s.timer.C
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
		s.resetTimer()
	} else {
		if !s.timer.Stop() {
			<-s.timer.C
		}
		s.timer.Reset(passed - s.genConfig.SpoolTimeout)
	}

	return true
}
