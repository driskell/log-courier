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

package harvester

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

const (
	defaultStreamAddPathField bool          = true
	defaultStreamDeadTime     time.Duration = 1 * time.Hour

	defaultGeneralLineBufferBytes int64 = 16384
	defaultGeneralMaxLineBytes    int64 = 1048576
)

// StreamConfig holds the configuration for a stream of logs produced by a
// harvester
type StreamConfig struct {
	codecs.StreamConfig `config:",embed"`

	AddPathField bool          `config:"add path field"`
	DeadTime     time.Duration `config:"dead time"`
}

// Defaults sets the default harvester stream configuration
// Ensure we override the one from codecs.StreamConfig
func (sc *StreamConfig) Defaults() {
	sc.AddPathField = defaultStreamAddPathField
	sc.DeadTime = defaultStreamDeadTime
}

// Init initialises the configuration
// Ensure we override the one from codecs.StreamConfig
func (sc *StreamConfig) Init(p *config.Parser, path string) error {
	return nil
}

// Validate does nothing for a harvester stream
// This is here to prevent double validation of event.StreamConfig whose
// validation function would otherwise be inherited
// Ensure we override the one from codecs.StreamConfig
func (sc *StreamConfig) Validate(p *config.Parser, path string) (err error) {
	return nil
}

// NewHarvester creates a new harvester with the given configuration for the given stream identifier
func (sc *StreamConfig) NewHarvester(ctx context.Context, path string, fileinfo os.FileInfo, cfg *config.Config, acker event.Acknowledger, offset int64) *Harvester {
	ret := &Harvester{
		ctx:          ctx,
		path:         path,
		fileinfo:     fileinfo,
		stopChan:     make(chan struct{}),
		acker:        acker,
		genConfig:    cfg.GeneralPart("harvester").(*General),
		streamConfig: sc,
		offset:       offset,
		lastEOF:      nil,
		backOffTimer: time.NewTimer(0),
		// TODO: Configurable meter timer? Use same as statCheck timer
		meterTimer: time.NewTimer(10 * time.Second),
	}

	ret.eventStream = sc.NewStream(ret.eventCallback, offset)
	<-ret.backOffTimer.C

	if path == Stdin {
		// This is stdin
		ret.file = os.Stdin
		ret.isStream = true
	} else {
		// Grab now so we can safely use them even if prospector changes them
		ret.isStream = false
	}

	return ret
}

// General contains extra general section configuration values for the
// harvester
type General struct {
	LineBufferBytes int64 `config:"line buffer bytes"`
	MaxLineBytes    int64 `config:"max line bytes"`
}

// Validate the additional general configuration
func (gc *General) Validate(p *config.Parser, path string) (err error) {
	if gc.LineBufferBytes < 1 {
		err = fmt.Errorf("%s/line buffer bytes must be greater than 1", path)
		return
	}

	if gc.MaxLineBytes < 1 {
		err = fmt.Errorf("%s/max line bytes must be greater than 1", path)
		return
	}

	return
}

func init() {
	config.RegisterGeneral("harvester", func() interface{} {
		return &General{
			LineBufferBytes: defaultGeneralLineBufferBytes,
			MaxLineBytes:    defaultGeneralMaxLineBytes,
		}
	})
}
