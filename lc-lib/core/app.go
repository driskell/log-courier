/*
 * Copyright 2014-2016 Jason Woods.
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
	"flag"
	"fmt"
	golog "log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
	"gopkg.in/op/go-logging.v1"
)

// App represents a courier application
type App struct {
	name       string
	binName    string
	version    string
	configFile string
	pipeline   *Pipeline
	config     *config.Config
	signalChan chan os.Signal
	logFile    *defaultLogBackend
}

// NewApp creates a new courier application
func NewApp(name, binName, version string) *App {
	return &App{
		name:     name,
		version:  version,
		pipeline: NewPipeline(),
	}
}

// StartUp processes the command line arguments and sets up logging
func (a *App) StartUp() {
	var version bool
	var configDebug bool
	var listSupported bool
	var configTest bool
	var cpuProfile string

	flag.BoolVar(&version, "version", false, "Show version information")
	flag.BoolVar(&configDebug, "config-debug", false, "Enable configuration parsing debug logs on the console")
	flag.BoolVar(&listSupported, "list-supported", false, "List the supported transports and codecs")
	flag.BoolVar(&configTest, "config-test", false, "Test the configuration specified by -config")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "Write a cpu profile to the specified file")
	flag.StringVar(&a.configFile, "config", config.DefaultConfigurationFile, "The configuration file to load")

	flag.Parse()

	if version {
		fmt.Printf("%s version %s\n", a.name, a.version)
		os.Exit(0)
	}

	if listSupported {
		fmt.Printf("Available transports:\n")
		for _, transport := range transports.Available() {
			fmt.Printf("  %s\n", transport)
		}

		fmt.Printf("Available codecs:\n")
		for _, codec := range codecs.Available() {
			fmt.Printf("  %s\n", codec)
		}
		os.Exit(0)
	}

	if a.configFile == "" {
		fmt.Fprintf(os.Stderr, "Please specify a configuration file with -config.\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Enable config logging if requested
	if configDebug {
		logging.SetLevel(logging.DEBUG, "config")
	}

	err := a.loadConfig()

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

	if err = a.configureLogging(); err != nil {
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

// Run the application
func (a *App) Run() {
	a.pipeline.Start()

	log.Notice("Pipeline ready")

	a.signalChan = make(chan os.Signal, 1)
	a.registerSignals()

SignalLoop:
	for {
		select {
		case signal := <-a.signalChan:
			if signal == nil || isShutdownSignal(signal) {
				a.cleanShutdown()
				break SignalLoop
			}

			a.ReloadConfig()
		}
	}

	log.Notice("Exiting")

	if a.logFile != nil {
		a.logFile.Close()
	}
}

// Stop requests the application to start shutting down
func (a *App) Stop() {
	close(a.signalChan)
}

// AddToPipeline adds a pipeline segment to the pipeline
func (a *App) AddToPipeline(segment IPipelineSegment) {
	a.pipeline.Add(segment)
}

// Config returns the configuration
func (a *App) Config() *config.Config {
	return a.config
}

// configureLogging enables the available logging backends
func (a *App) configureLogging() (err error) {
	backends := make([]logging.Backend, 0, 1)

	// First, the stdout backend
	if a.config.General().LogStdout {
		backends = append(backends, logging.NewLogBackend(os.Stdout, "", golog.LstdFlags|golog.Lmicroseconds))
	}

	// Log file?
	if a.config.General().LogFile != "" {
		a.logFile, err = newDefaultLogBackend(a.config.General().LogFile, "", golog.LstdFlags|golog.Lmicroseconds)
		if err != nil {
			return
		}

		backends = append(backends, a.logFile)
	}

	if err = a.configureLoggingPlatform(&backends); err != nil {
		return
	}

	// Set backends BEFORE log level (or we reset log level)
	logging.SetBackend(backends...)

	// Set the logging level
	logging.SetLevel(a.config.General().LogLevel, "")

	return nil
}

// loadConfig loads the configuration data
func (a *App) loadConfig() error {
	a.config = config.NewConfig()
	return a.config.Load(a.configFile, true)
}

// ReloadConfig reloads the configuration data and submits to all running
// routines in the pipeline that are subscribed to it, so they may update their
// runtime configuration
func (a *App) ReloadConfig() error {
	if err := a.loadConfig(); err != nil {
		return err
	}

	log.Notice("Configuration reload successful")

	// Update the log level
	logging.SetLevel(a.config.General().LogLevel, "")

	// Reopen the log file if we specified one
	if a.logFile != nil {
		a.logFile.Reopen()
		log.Notice("Log file reopened")
	}

	// Pass the new config to the pipeline workers
	a.pipeline.SendConfig(a.config)

	return nil
}

// cleanShutdown initiates a clean shutdown of log-courier
func (a *App) cleanShutdown() {
	log.Notice("Initiating shutdown")

	a.pipeline.Shutdown()
	a.pipeline.Wait()
}
