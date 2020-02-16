/*
 * Copyright 2014-2019 Jason Woods.
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

package main

import (
	"flag"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/prospector"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"github.com/driskell/log-courier/lc-lib/spooler"
	"github.com/driskell/log-courier/lc-lib/stdinharvester"
	"github.com/driskell/log-courier/lc-lib/transports"

	_ "github.com/driskell/log-courier/lc-lib/codecs/filter"

	_ "github.com/driskell/log-courier/lc-lib/codecs/multiline"

	_ "github.com/driskell/log-courier/lc-lib/codecs/plain"

	_ "github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// Generate platform-specific default configuration values
//go:generate go run lc-lib/config/generate/platform.go platform main config.DefaultConfigurationFile prospector.DefaultGeneralPersistDir admin.DefaultAdminBind
// TODO: This should be in lc-admin but we can't due to vendor failure on go generate in subpackages
//go:generate go run lc-lib/config/generate/platform.go lc-admin/platform main config.DefaultConfigurationFile prospector.DefaultGeneralPersistDir admin.DefaultAdminBind
// TODO: This should be in fact-courier but we can't due to vendor failure on go generate in subpackages
//go:generate go run lc-lib/config/generate/platform.go fact-courier/platform main config.DefaultConfigurationFile:LC_FACT_DEFAULT_CONFIGURATION_FILE

var (
	app *core.App

	stdin         bool
	fromBeginning bool
)

func main() {
	app = core.NewApp("Log Courier", "log-courier", core.LogCourierVersion)
	flag.BoolVar(&stdin, "stdin", false, "Read from stdin instead of files listed in the config file")
	flag.BoolVar(&fromBeginning, "from-beginning", false, "On first run, read new files from the beginning instead of the end")
	app.StartUp()

	// Skip admin if reading from stdin
	if !stdin && app.Config().Section("admin").(*admin.Config).Enabled {
		app.Pipeline().AddService(admin.NewServer(app))
	}

	if stdin {
		// If reading from stdin, don't start prospector, directly start a harvester
		app.Pipeline().AddSource(stdinharvester.New(app))
	} else {
		// Prospector will handle new files, start harvesters, and own the registrar
		app.Pipeline().AddSource(prospector.NewProspector(app, fromBeginning))

		// Receiver?
		receiverConfig := transports.FetchReceiverConfig(app.Config())
		if receiverConfig.Enabled {
			segment := &receiverSegment{cfg: receiverConfig}
			app.Pipeline().AddSource(segment)
		}
	}

	// Add spooler as first processor, it combines into larger chunks as needed
	app.Pipeline().AddProcessor(spooler.NewSpooler(app))
	// Create sink
	app.Pipeline().SetSink(publisher.NewPublisher())
	// Go!
	app.Run()
}

type receiverSegment struct {
	cfg          *transports.ReceiverConfig
	output       chan<- []*event.Event
	shutdownChan <-chan struct{}
	eventChan    chan transports.Event
}

// SetOutput sets the output channel
func (r *receiverSegment) SetOutput(output chan<- []*event.Event) {
	r.output = output
}

// SetShutdownChan sets the shutdown channel
func (r *receiverSegment) SetShutdownChan(shutdownChan <-chan struct{}) {
	r.shutdownChan = shutdownChan
}

// Init sets up the listener
func (r *receiverSegment) Init(cfg *config.Config) error {
	r.eventChan = make(chan transports.Event)
	return nil
}

// Run starts listening
func (r *receiverSegment) Run() {
	pool := addresspool.NewPool("0.0.0.0")
	receiver := r.cfg.Factory.NewReceiver(nil, pool, r.eventChan)

	for {
		select {
		case <-r.shutdownChan:
			receiver.Shutdown()
		case receiverEvent := <-r.eventChan:
			switch eventImpl := receiverEvent.(type) {
			case *transports.EventsEvent:
				// TODO: Congestion handling
				r.output <- eventImpl.Events()
			}
		}
	}
}
