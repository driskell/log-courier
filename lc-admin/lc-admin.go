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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"gopkg.in/op/go-logging.v1"
)

// Generate platform-specific default configuration values
//go:generate go run -mod=vendor ../lc-lib/config/generate/platform.go platform main config.DefaultConfigurationFile admin.DefaultAdminBind defaultCarverConfigurationFile defaultCarverAdminBind

var (
	defaultCarverConfigurationFile string
	defaultCarverAdminBind         string
)

type commandProcessor interface {
	ProcessCommand(string) bool
	Monitor() error
}

type lcAdmin struct {
	quiet        bool
	watch        bool
	legacy       bool
	adminConnect string
	configFile   string
	configDebug  bool
}

func main() {
	(&lcAdmin{}).Run()
}

func (a *lcAdmin) startUp() {
	var version bool
	var carver bool

	flag.BoolVar(&version, "version", false, "display the lc-admin version")
	if defaultCarverAdminBind != "" {
		flag.BoolVar(&carver, "carver", false, "connect to log-carver instead of log-courier")
	}
	flag.BoolVar(&a.quiet, "quiet", false, "quietly execute the command line argument and output only the result")
	flag.BoolVar(&a.watch, "watch", false, "repeat the command specified on the command line every second")
	flag.BoolVar(&a.legacy, "legacy", false, "connect to v1 Log Courier instances")
	flag.StringVar(&a.adminConnect, "connect", "", "the remote instance to connect to")
	flag.StringVar(&a.configFile, "config", config.DefaultConfigurationFile, "read the connection address from the given configuration file (ignored if connect specified)")
	flag.BoolVar(&a.configDebug, "config-debug", false, "Enable configuration parsing debug logs on the console")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		if a.legacy {
			printV1Help()
		} else {
			printV2Help()
			fmt.Fprintf(flag.CommandLine.Output(), "\nRun %s -legacy -help to show available commands for v1 remotes\n", os.Args[0])
		}
	}

	flag.Parse()

	if !a.quiet || version {
		fmt.Printf("Admin version %s\n", core.LogCourierVersion)
		if version {
			os.Exit(0)
		}
	}

	// If connecting to carver - change admin.DefaultAdminBind
	if carver {
		if a.adminConnect != "" {
			fmt.Printf("Cannot use both -carver and -connect at the same time")
		}
		config.DefaultConfigurationFile = defaultCarverConfigurationFile
		admin.DefaultAdminBind = defaultCarverAdminBind
	}

	// Enable config logging if requested
	if a.configDebug {
		logging.SetLevel(logging.DEBUG, "config")
	}
	a.loadConfig()
	// Enable config logging if requested
	if a.configDebug {
		logging.SetLevel(logging.INFO, "config")
	}

	fmt.Println("")
}

func (a *lcAdmin) loadConfig() {
	if a.configFile != "" && a.adminConnect != "" {
		fmt.Printf("Cannot use both -config and -connect at the same time")
		os.Exit(1)
	}

	if a.adminConnect != "" {
		return
	}

	if a.configFile == "" {
		if config.DefaultConfigurationFile == "" {
			fmt.Printf("Either -connect or -config must be specified\n")
			flag.PrintDefaults()
			os.Exit(1)
		} else {
			a.configFile = config.DefaultConfigurationFile
		}
	}

	fmt.Printf("Loading configuration: %s\n", a.configFile)

	// Load admin connect address from the configuration file
	config := config.NewConfig()
	if err := config.Load(a.configFile, false); err != nil {
		fmt.Printf("Configuration error: %s\n", err)
		os.Exit(1)
	}

	adminConfig := config.Section("admin").(*admin.Config)
	if !adminConfig.Enabled {
		fmt.Printf("Unable to connect: the admin interface is disabled\n")
		os.Exit(1)
	}

	a.adminConnect = adminConfig.Bind
}

func (a *lcAdmin) Run() {
	a.startUp()

	processor, err := a.newCommandProcessor()
	if err != nil {
		fmt.Printf("Failed to initialise: %s\n", err)
		os.Exit(1)
		return
	}

	args := flag.Args()
	if len(args) != 0 {
		if a.argsCommand(processor, args, a.watch) {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if a.quiet {
		fmt.Printf("No command specified on the command line for quiet execution\n")
		os.Exit(1)
	}

	if a.watch {
		fmt.Printf("No command specified on the command line to watch\n")
		os.Exit(1)
	}

	if err := processor.Monitor(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func (a *lcAdmin) newCommandProcessor() (commandProcessor, error) {
	if a.legacy {
		// Create the old V1 legacy processor
		return newV1Command(a.quiet, a.adminConnect)
	}

	return newV2Command(a.quiet, a.adminConnect)
}

func (a *lcAdmin) argsCommand(processor commandProcessor, args []string, watch bool) bool {
	var signalChan chan os.Signal

	if watch {
		signalChan = make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
	}

WatchLoop:
	for {
		if !processor.ProcessCommand(strings.Join(args, " ")) {
			if !watch {
				return false
			}
		}

		if !watch {
			break
		}

		// Gap between repeats
		fmt.Printf("\n")

		select {
		case <-signalChan:
			break WatchLoop
		case <-time.After(time.Second):
		}
	}

	return true
}
