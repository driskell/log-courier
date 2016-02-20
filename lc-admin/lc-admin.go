/*
 * Copyright 2014-2015 Jason Woods.
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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

func printHelp() {
	fmt.Printf("Available commands:\n")
	fmt.Printf("  reload    Reload configuration\n")
	fmt.Printf("  status    Display the current shipping status\n")
	fmt.Printf("  exit      Exit\n")
}

func main() {
	var version bool
	var quiet bool
	var watch bool
	var adminConnect string
	var configFile string

	flag.BoolVar(&version, "version", false, "display the Log Courier client version")
	flag.BoolVar(&quiet, "quiet", false, "quietly execute the command line argument and output only the result")
	flag.BoolVar(&watch, "watch", false, "repeat the command specified on the command line every second")
	flag.StringVar(&adminConnect, "connect", "", "the Log Courier instance to connect to")
	flag.StringVar(&configFile, "config", config.DefaultConfigurationFile, "read the Log Courier connection address from the given configuration file (ignored if connect specified)")

	flag.Parse()

	if version {
		fmt.Printf("Log Courier version %s\n", core.LogCourierVersion)
		os.Exit(0)
	}

	if !quiet {
		fmt.Printf("Log Courier version %s client\n\n", core.LogCourierVersion)
	}

	if configFile == "" && adminConnect == "" {
		if config.DefaultGeneralAdminBind == "" {
			fmt.Printf("Either connect or config parameter must be specified\n")
			flag.PrintDefaults()
			os.Exit(1)
		} else {
			adminConnect = config.DefaultGeneralAdminBind
		}
	}

	if adminConnect == "" {
		// Load admin connect address from the configuration file
		config := config.NewConfig()
		if err := config.Load(configFile); err != nil {
			fmt.Printf("Configuration error: %s\n", err)
			os.Exit(1)
		}

		adminConnect = config.General.AdminBind
	}

	args := flag.Args()

	if len(args) != 0 {
		// Don't require a connection to display the help message
		if args[0] == "help" {
			printHelp()
			os.Exit(0)
		}

		admin := newLcAdmin(quiet, adminConnect)
		if admin.argsCommand(args, watch) {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if quiet {
		fmt.Printf("No command specified on the command line for quiet execution\n")
		os.Exit(1)
	}

	if watch {
		fmt.Printf("No command specified on the command line to watch\n")
		os.Exit(1)
	}

	admin := newLcAdmin(quiet, adminConnect)
	if err := admin.connect(); err != nil {
		return
	}

	admin.run()
}
