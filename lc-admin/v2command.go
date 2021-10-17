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
	"fmt"
	"net/url"
	"strings"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/admin/api"
)

type v2Command struct {
	client       *admin.Client
	quiet        bool
	adminConnect string
}

func newV2Command(quiet bool, adminConnect string) (*v2Command, error) {
	ret := &v2Command{
		quiet:        quiet,
		adminConnect: adminConnect,
	}

	return ret, nil
}

func printV2Help() {
	fmt.Printf("Available commands for v2+ remotes:\n")
	fmt.Printf("  help\n")
	fmt.Printf("    Show this information\n")
	fmt.Printf("  status\n")
	fmt.Printf("    Get a full status snapshot of all Log Courier internals\n")
	fmt.Printf("  prospector [status | files [id]]\n")
	fmt.Printf("    Get information on prospector state and running harvesters\n")
	fmt.Printf("  publisher [status | endpoints [id]]\n")
	fmt.Printf("    Get information on connectivity and endpoints\n")
	fmt.Printf("  reload\n")
	fmt.Printf("    Signals Log Courier to reload its configuration\n")
	fmt.Printf("  version\n")
	fmt.Printf("    Get the remote version\n")
	fmt.Printf("  debug\n")
	fmt.Printf("    Get a live goroutine trace for debugging purposes\n")
	fmt.Printf("  exit\n")
	fmt.Printf("    Exit\n")
}

func (a *v2Command) setupClient() error {
	if !a.quiet {
		fmt.Printf("Setting up client for %s...\n", a.adminConnect)
	}

	client, err := admin.NewClient(a.adminConnect)
	if err != nil {
		return err
	}

	a.client = client

	if !a.quiet {
		_, version := client.RemoteClient()
		fmt.Printf("Detected remote version %s\n", version)
	}

	return nil
}

func (a *v2Command) ProcessCommand(command string) bool {
	if a.client == nil {
		if err := a.setupClient(); err != nil {
			fmt.Printf("Failed to setup client: %s", err.Error())
			return false
		}
	}

	if command == "status" {
		// Simulate empty command so we grab full status
		command = ""
	}

	command = url.QueryEscape(command)

	path := strings.Map(func(r rune) rune {
		if r == '+' {
			return '/'
		}
		return r
	}, command)

	resp, err := a.client.RequestPretty(path)
	if err != nil {
		switch err {
		case api.ErrNotFound:
			fmt.Printf("Unknown command\n")
			return false
		}

		switch err := err.(type) {
		case api.ErrUnknown:
			fmt.Printf("Log Courier returned an error: %s\n", err.Error())
			return false
		}

		fmt.Printf("The API request failed: %s\n", err)
		return false
	}

	fmt.Println(resp)

	return true
}

func (a *v2Command) Monitor() error {
	if a.client == nil {
		if err := a.setupClient(); err != nil {
			return err
		}
	}

	monitor := NewMonitor(a.client)
	return monitor.Run()
}
