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

package main

import (
	"flag"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/prospector"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"github.com/driskell/log-courier/lc-lib/registrar"
	"github.com/driskell/log-courier/lc-lib/spooler"
	"gopkg.in/op/go-logging.v1"
)

import _ "github.com/driskell/log-courier/lc-lib/codecs"
import _ "github.com/driskell/log-courier/lc-lib/transports/tcp"

// Generate platform-specific default configuration values
//go:generate go run lc-lib/config/generate/platform.go platform main config.DefaultConfigurationFile config.DefaultGeneralPersistDir admin.DefaultAdminBind
// TODO: This should be in lc-admin but we can't due to vendor failure on go generate in subpackages
//go:generate go run lc-lib/config/generate/platform.go lc-admin/platform main config.DefaultConfigurationFile config.DefaultGeneralPersistDir admin.DefaultAdminBind

var (
	log *logging.Logger
	app *core.App

	stdin         bool
	fromBeginning bool
)

func main() {
	app = core.NewApp("Log Courier", "log-courier", core.LogCourierVersion)
	startUp()
	setupPipeline()
	app.Run()
}

func startUp() {
	flag.BoolVar(&stdin, "stdin", false, "Read from stdin instead of files listed in the config file")
	flag.BoolVar(&fromBeginning, "from-beginning", false, "On first run, read new files from the beginning instead of the end")

	app.StartUp()

	if !stdin && len(*app.Config().Section("files").(*prospector.Config)) == 0 {
		log.Warning("No file groups were found in the configuration.")
	}
}

func setupPipeline() {
	var registrarImpl registrar.Registrator

	log.Info("Configuring Log Courier version %s pipeline", core.LogCourierVersion)

	// If reading from stdin, skip admin, and set up a null registrar
	if stdin {
		registrarImpl = newStdinRegistrar(app)
	} else {
		setupAdmin()
		registrarImpl = registrar.NewRegistrar(app)
	}
	app.AddToPipeline(registrarImpl)

	publisherImpl := publisher.NewPublisher(app, registrarImpl)
	app.AddToPipeline(publisherImpl)

	spoolerImpl := spooler.NewSpooler(app, publisherImpl)
	app.AddToPipeline(spoolerImpl)

	// If reading from stdin, don't start prospector, directly start a harvester
	if stdin {
		stdinHarvester := harvester.NewHarvester(nil, app, &app.Config().Stdin, 0)
		stdinHarvester.Start(spoolerImpl.Connect())
		go waitOnStdin(stdinHarvester, spoolerImpl, registrarImpl)
	} else {
		prospectorImpl, err := prospector.NewProspector(app, fromBeginning, registrarImpl, spoolerImpl)
		if err != nil {
			log.Fatalf("Failed to initialise: %s", err)
		}

		app.AddToPipeline(prospectorImpl)
	}
}

func setupAdmin() {
	if !app.Config().Section("admin").(*admin.Config).Enabled {
		return
	}

	server, err := admin.NewServer(app)
	if err != nil {
		log.Fatalf("Failed to initialise: %s", err)
	}

	app.AddToPipeline(server)
}

func waitOnStdin(stdinHarvester *harvester.Harvester, spoolerImpl *spooler.Spooler, registrarImpl registrar.Registrator) {
	finished := <-stdinHarvester.OnFinish()

	if finished.Error != nil {
		log.Notice("An error occurred reading from stdin at offset %d: %s", finished.LastReadOffset, finished.Error)
	} else {
		log.Notice("Finished reading from stdin at offset %d", finished.LastReadOffset)
	}

	// Flush the spooler in case it is still running and buffering
	spoolerImpl.Flush()

	// Wait for StdinRegistrar to receive ACK for the last event we sent or for it
	// to shutdown
	registrarImpl.(*StdinRegistrar).Wait(finished.LastEventOffset)

	app.Stop()
}

func init() {
	log = logging.MustGetLogger("log-courier")
}
