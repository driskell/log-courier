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

package main

import (
	"flag"
	"fmt"
	"github.com/driskell/log-courier/src/lc-lib/admin"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/harvester"
	"github.com/driskell/log-courier/src/lc-lib/prospector"
	"github.com/driskell/log-courier/src/lc-lib/publisher"
	"github.com/driskell/log-courier/src/lc-lib/registrar"
	"github.com/driskell/log-courier/src/lc-lib/spooler"
	"github.com/op/go-logging"
	stdlog "log"
	"os"
	"runtime/pprof"
	"time"
)

import _ "github.com/driskell/log-courier/src/lc-lib/codecs"
import _ "github.com/driskell/log-courier/src/lc-lib/transports"

func main() {
	logcourier := NewLogCourier()
	logcourier.Run()
}

type LogCourier struct {
	pipeline       *core.Pipeline
	config         *core.Config
	shutdown_chan  chan os.Signal
	reload_chan    chan os.Signal
	config_file    string
	stdin          bool
	from_beginning bool
	harvester      *harvester.Harvester
	log_file       *DefaultLogBackend
	last_snapshot  time.Time
	snapshot       *core.Snapshot
}

func NewLogCourier() *LogCourier {
	ret := &LogCourier{
		pipeline: core.NewPipeline(),
	}
	return ret
}

func (lc *LogCourier) Run() {
	var admin_listener *admin.Listener
	var on_command <-chan string
	var harvester_wait <-chan *harvester.HarvesterFinish
	var registrar_imp registrar.Registrator

	lc.startUp()

	log.Info("Log Courier version %s pipeline starting", core.Log_Courier_Version)

	// If reading from stdin, skip admin, and set up a null registrar
	if lc.stdin {
		registrar_imp = newStdinRegistrar(lc.pipeline)
	} else {
		if lc.config.General.AdminEnabled {
			var err error

			admin_listener, err = admin.NewListener(lc.pipeline, &lc.config.General)
			if err != nil {
				log.Fatalf("Failed to initialise: %s", err)
			}

			on_command = admin_listener.OnCommand()
		}

		registrar_imp = registrar.NewRegistrar(lc.pipeline, lc.config.General.PersistDir)
	}

	publisher_imp := publisher.NewPublisher(lc.pipeline, &lc.config.Network, registrar_imp)

	spooler_imp := spooler.NewSpooler(lc.pipeline, &lc.config.General, publisher_imp)

	// If reading from stdin, don't start prospector, directly start a harvester
	if lc.stdin {
		lc.harvester = harvester.NewHarvester(nil, lc.config, &lc.config.Stdin, 0)
		lc.harvester.Start(spooler_imp.Connect())
		harvester_wait = lc.harvester.OnFinish()
	} else {
		if _, err := prospector.NewProspector(lc.pipeline, lc.config, lc.from_beginning, registrar_imp, spooler_imp); err != nil {
			log.Fatalf("Failed to initialise: %s", err)
		}
	}

	// Start the pipeline
	lc.pipeline.Start()

	log.Notice("Pipeline ready")

	lc.shutdown_chan = make(chan os.Signal, 1)
	lc.reload_chan = make(chan os.Signal, 1)
	lc.registerSignals()

SignalLoop:
	for {
		select {
		case <-lc.shutdown_chan:
			lc.cleanShutdown()
			break SignalLoop
		case <-lc.reload_chan:
			lc.reloadConfig()
		case command := <-on_command:
			admin_listener.Respond(lc.processCommand(command))
		case finished := <-harvester_wait:
			if finished.Error != nil {
				log.Notice("An error occurred reading from stdin at offset %d: %s", finished.Last_Read_Offset, finished.Error)
			} else {
				log.Notice("Finished reading from stdin at offset %d", finished.Last_Read_Offset)
			}
			lc.harvester = nil

			// Flush the spooler
			spooler_imp.Flush()

			// Wait for StdinRegistrar to receive ACK for the last event we sent
			registrar_imp.(*StdinRegistrar).Wait(finished.Last_Event_Offset)

			lc.cleanShutdown()
			break SignalLoop
		}
	}

	log.Notice("Exiting")

	if lc.log_file != nil {
		lc.log_file.Close()
	}
}

func (lc *LogCourier) startUp() {
	var version bool
	var config_test bool
	var list_supported bool
	var cpu_profile string

	flag.BoolVar(&version, "version", false, "show version information")
	flag.BoolVar(&config_test, "config-test", false, "Test the configuration specified by -config and exit")
	flag.BoolVar(&list_supported, "list-supported", false, "List supported transports and codecs")
	flag.StringVar(&cpu_profile, "cpuprofile", "", "write cpu profile to file")

	flag.StringVar(&lc.config_file, "config", "", "The config file to load")
	flag.BoolVar(&lc.stdin, "stdin", false, "Read from stdin instead of files listed in the config file")
	flag.BoolVar(&lc.from_beginning, "from-beginning", false, "On first run, read new files from the beginning instead of the end")

	flag.Parse()

	if version {
		fmt.Printf("Log Courier version %s\n", core.Log_Courier_Version)
		os.Exit(0)
	}

	if list_supported {
		fmt.Printf("Available transports:\n")
		for _, transport := range core.AvailableTransports() {
			fmt.Printf("  %s\n", transport)
		}

		fmt.Printf("Available codecs:\n")
		for _, codec := range core.AvailableCodecs() {
			fmt.Printf("  %s\n", codec)
		}
		os.Exit(0)
	}

	if lc.config_file == "" {
		fmt.Fprintf(os.Stderr, "Please specify a configuration file with -config.\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	err := lc.loadConfig()

	if config_test {
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

	if cpu_profile != "" {
		log.Notice("Starting CPU profiler")
		f, err := os.Create(cpu_profile)
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
}

func (lc *LogCourier) configureLogging() (err error) {
	backends := make([]logging.Backend, 0, 1)

	// First, the stdout backend
	if lc.config.General.LogStdout {
		backends = append(backends, logging.NewLogBackend(os.Stdout, "", stdlog.LstdFlags|stdlog.Lmicroseconds))
	}

	// Log file?
	if lc.config.General.LogFile != "" {
		lc.log_file, err = NewDefaultLogBackend(lc.config.General.LogFile, "", stdlog.LstdFlags|stdlog.Lmicroseconds)
		if err != nil {
			return
		}

		backends = append(backends, lc.log_file)
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

func (lc *LogCourier) loadConfig() error {
	lc.config = core.NewConfig()
	if err := lc.config.Load(lc.config_file); err != nil {
		return err
	}

	if lc.stdin {
		// TODO: Where to find stdin config for codec and fields?
	} else if len(lc.config.Files) == 0 {
		log.Warning("No file groups were found in the configuration.")
	}

	return nil
}

func (lc *LogCourier) reloadConfig() error {
	if err := lc.loadConfig(); err != nil {
		return err
	}

	log.Notice("Configuration reload successful")

	// Update the log level
	logging.SetLevel(lc.config.General.LogLevel, "")

	// Reopen the log file if we specified one
	if lc.log_file != nil {
		lc.log_file.Reopen()
		log.Notice("Log file reopened")
	}

	// Pass the new config to the pipeline workers
	lc.pipeline.SendConfig(lc.config)

	return nil
}

func (lc *LogCourier) processCommand(command string) *admin.Response {
	switch command {
	case "RELD":
		if err := lc.reloadConfig(); err != nil {
			return &admin.Response{&admin.ErrorResponse{Message: fmt.Sprintf("Configuration error, reload unsuccessful: %s", err.Error())}}
		}
		return &admin.Response{&admin.ReloadResponse{}}
	case "SNAP":
		if lc.snapshot == nil || time.Since(lc.last_snapshot) >= time.Second {
			lc.snapshot = lc.pipeline.Snapshot()
			lc.snapshot.Sort()
			lc.last_snapshot = time.Now()
		}
		return &admin.Response{lc.snapshot}
	}

	return &admin.Response{&admin.ErrorResponse{Message: "Unknown command"}}
}

func (lc *LogCourier) cleanShutdown() {
	log.Notice("Initiating shutdown")

	if lc.harvester != nil {
		lc.harvester.Stop()
		finished := <-lc.harvester.OnFinish()
		log.Notice("Aborted reading from stdin at offset %d", finished.Last_Read_Offset)
	}

	lc.pipeline.Shutdown()
	lc.pipeline.Wait()
}
