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
	"fmt"
	stdlog "log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/config"
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

func main() {
	logcourier := NewLogCourier()
	logcourier.Run()
}

// LogCourier is the root structure for the log-courier binary
type LogCourier struct {
	pipeline      *core.Pipeline
	config        *config.Config
	shutdownChan  chan os.Signal
	reloadChan    chan os.Signal
	configFile    string
	stdin         bool
	fromBeginning bool
	harvester     *harvester.Harvester
	logFile       *DefaultLogBackend
	lastSnapshot  time.Time
	snapshot      *core.Snapshot
}

// NewLogCourier creates a new LogCourier structure for the log-courier binary
func NewLogCourier() *LogCourier {
	ret := &LogCourier{
		pipeline: core.NewPipeline(),
	}
	return ret
}

// Run starts the log-courier binary
func (lc *LogCourier) Run() {
	var adminListener *admin.Listener
	var onCommand <-chan string
	var harvesterWait <-chan *harvester.FinishStatus
	var registrarImp registrar.Registrator

	lc.startUp()

	log.Info("Log Courier version %s pipeline starting", core.LogCourierVersion)

	// If reading from stdin, skip admin, and set up a null registrar
	if lc.stdin {
		registrarImp = newStdinRegistrar(lc.pipeline)
	} else {
		if lc.config.General.AdminEnabled {
			var err error

			adminListener, err = admin.NewListener(lc.pipeline, &lc.config.General)
			if err != nil {
				log.Fatalf("Failed to initialise: %s", err)
			}

			onCommand = adminListener.OnCommand()
		}

		registrarImp = registrar.NewRegistrar(lc.pipeline, lc.config.General.PersistDir)
	}

	publisherImp := publisher.NewPublisher(lc.pipeline, &lc.config.Network, registrarImp)

	spoolerImp := spooler.NewSpooler(lc.pipeline, &lc.config.General, publisherImp)

	// If reading from stdin, don't start prospector, directly start a harvester
	if lc.stdin {
		lc.harvester = harvester.NewHarvester(nil, lc.config, &lc.config.Stdin, 0)
		lc.harvester.Start(spoolerImp.Connect())
		harvesterWait = lc.harvester.OnFinish()
	} else {
		if _, err := prospector.NewProspector(lc.pipeline, lc.config, lc.fromBeginning, registrarImp, spoolerImp); err != nil {
			log.Fatalf("Failed to initialise: %s", err)
		}
	}

	// Start the pipeline
	lc.pipeline.Start()

	log.Notice("Pipeline ready")

	lc.shutdownChan = make(chan os.Signal, 1)
	lc.reloadChan = make(chan os.Signal, 1)
	lc.registerSignals()

SignalLoop:
	for {
		select {
		case <-lc.shutdownChan:
			lc.cleanShutdown()
			break SignalLoop
		case <-lc.reloadChan:
			lc.reloadConfig()
		case command := <-onCommand:
			adminListener.Respond(lc.processCommand(command))
		case finished := <-harvesterWait:
			if finished.Error != nil {
				log.Notice("An error occurred reading from stdin at offset %d: %s", finished.LastReadOffset, finished.Error)
			} else {
				log.Notice("Finished reading from stdin at offset %d", finished.LastReadOffset)
			}
			lc.harvester = nil

			// Flush the spooler
			spoolerImp.Flush()

			// Wait for StdinRegistrar to receive ACK for the last event we sent
			registrarImp.(*StdinRegistrar).Wait(finished.LastEventOffset)

			lc.cleanShutdown()
			break SignalLoop
		}
	}

	log.Notice("Exiting")

	if lc.logFile != nil {
		lc.logFile.Close()
	}
}

// startUp processes the command line arguments and sets up logging
func (lc *LogCourier) startUp() {
	var version bool
	var configTest bool
	var listSupported bool
	var cpuProfile string

	flag.BoolVar(&version, "version", false, "show version information")
	flag.BoolVar(&configTest, "config-test", false, "Test the configuration specified by -config and exit")
	flag.BoolVar(&listSupported, "list-supported", false, "List supported transports and codecs")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to file")

	flag.StringVar(&lc.configFile, "config", config.DefaultConfigurationFile, "The config file to load")
	flag.BoolVar(&lc.stdin, "stdin", false, "Read from stdin instead of files listed in the config file")
	flag.BoolVar(&lc.fromBeginning, "from-beginning", false, "On first run, read new files from the beginning instead of the end")

	flag.Parse()

	if version {
		fmt.Printf("Log Courier version %s\n", core.LogCourierVersion)
		os.Exit(0)
	}

	if listSupported {
		fmt.Printf("Available transports:\n")
		for _, transport := range config.AvailableTransports() {
			fmt.Printf("  %s\n", transport)
		}

		fmt.Printf("Available codecs:\n")
		for _, codec := range config.AvailableCodecs() {
			fmt.Printf("  %s\n", codec)
		}
		os.Exit(0)
	}

	if lc.configFile == "" {
		fmt.Fprintf(os.Stderr, "Please specify a configuration file with -config.\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	err := lc.loadConfig()

	if configTest {
		if err == nil {
			fmt.Printf("Configuration OK\n")
			os.Exit(0)
		}
		fmt.Printf("Configuration test failed: %s\n", err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Configuration error: %s\n", err)
		os.Exit(1)
	}

	if err = lc.configureLogging(); err != nil {
		fmt.Printf("Failed to initialise logging: %s", err)
		os.Exit(1)
	}

	if cpuProfile != "" {
		log.Notice("Starting CPU profiler")
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		go func() {
			time.Sleep(60 * time.Second)
			pprof.StopCPUProfile()
			log.Panic("CPU profile completed")
		}()
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
}

// configureLogging enables the available logging backends
func (lc *LogCourier) configureLogging() (err error) {
	backends := make([]logging.Backend, 0, 1)

	// First, the stdout backend
	if lc.config.General.LogStdout {
		backends = append(backends, logging.NewLogBackend(os.Stdout, "", stdlog.LstdFlags|stdlog.Lmicroseconds))
	}

	// Log file?
	if lc.config.General.LogFile != "" {
		lc.logFile, err = NewDefaultLogBackend(lc.config.General.LogFile, "", stdlog.LstdFlags|stdlog.Lmicroseconds)
		if err != nil {
			return
		}

		backends = append(backends, lc.logFile)
	}

	if err = lc.configureLoggingPlatform(&backends); err != nil {
		return
	}

	// Set backends BEFORE log level (or we reset log level)
	logging.SetBackend(backends...)

	// Set the logging level
	logging.SetLevel(lc.config.General.LogLevel, "")

	return nil
}

// loadConfig loads the configuration data
func (lc *LogCourier) loadConfig() error {
	lc.config = config.NewConfig()
	if err := lc.config.Load(lc.configFile); err != nil {
		return err
	}

	if lc.stdin {
		// TODO: Where to find stdin config for codec and fields?
	} else if len(lc.config.Files) == 0 {
		log.Warning("No file groups were found in the configuration.")
	}

	return nil
}

// reloadConfig reloads the configuration data and submits to all running
// routines in the pipeline that are subscribed to it, so they may update their
// runtime configuration
func (lc *LogCourier) reloadConfig() error {
	if err := lc.loadConfig(); err != nil {
		return err
	}

	log.Notice("Configuration reload successful")

	// Update the log level
	logging.SetLevel(lc.config.General.LogLevel, "")

	// Reopen the log file if we specified one
	if lc.logFile != nil {
		lc.logFile.Reopen()
		log.Notice("Log file reopened")
	}

	// Pass the new config to the pipeline workers
	lc.pipeline.SendConfig(lc.config)

	return nil
}

// processCommand is called from the admin routine in response to commands from
// a connected lc-admin compatible utility
// TODO: Replace with a REST API
func (lc *LogCourier) processCommand(command string) *admin.Response {
	switch command {
	case "RELD":
		if err := lc.reloadConfig(); err != nil {
			return &admin.Response{&admin.ErrorResponse{Message: fmt.Sprintf("Configuration error, reload unsuccessful: %s", err.Error())}}
		}
		return &admin.Response{&admin.ReloadResponse{}}
	case "SNAP":
		if lc.snapshot == nil || time.Since(lc.lastSnapshot) >= time.Second {
			lc.snapshot = lc.pipeline.Snapshot()
			lc.snapshot.Sort()
			lc.lastSnapshot = time.Now()
		}
		return &admin.Response{lc.snapshot}
	}

	return &admin.Response{&admin.ErrorResponse{Message: "Unknown command"}}
}

// cleanShutdown initiates a clean shutdown of log-courier
func (lc *LogCourier) cleanShutdown() {
	log.Notice("Initiating shutdown")

	if lc.harvester != nil {
		lc.harvester.Stop()
		finished := <-lc.harvester.OnFinish()
		log.Notice("Aborted reading from stdin at offset %d", finished.LastReadOffset)
	}

	lc.pipeline.Shutdown()
	lc.pipeline.Wait()
}
