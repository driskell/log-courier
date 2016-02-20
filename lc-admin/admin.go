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
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/scanner"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/core"
)

type lcAdmin struct {
	client       *admin.Client
	connected    bool
	quiet        bool
	adminConnect string
	scanner      scanner.Scanner
	scannerErr   error
}

func newLcAdmin(quiet bool, adminConnect string) *lcAdmin {
	return &lcAdmin{
		quiet:        quiet,
		adminConnect: adminConnect,
	}
}

func (a *lcAdmin) connect() error {
	if !a.connected {
		var err error

		if !a.quiet {
			fmt.Printf("Attempting connection to %s...\n", a.adminConnect)
		}

		if a.client, err = admin.NewClient(a.adminConnect); err != nil {
			fmt.Printf("Failed to connect: %s\n", err)
			return err
		}

		if !a.quiet {
			fmt.Printf("Connected\n\n")
		}

		a.connected = true
	}

	return nil
}

func (a *lcAdmin) processCommand(command string) bool {
	var reconnected bool

	for {
		if !a.connected {
			if err := a.connect(); err != nil {
				return false
			}

			reconnected = true
		}

		var err error

		a.initScanner(command)
		if command, err = a.scanIdent(); err != nil {
			goto Error
		}

		switch command {
		case "reload":
			if !a.scanEOF() {
				err = commandTooManyArgs
				break
			}

			err = a.client.Reload()
			if err != nil {
				break
			}

			fmt.Printf("Configuration reload successful\n")
		case "status":
			var format string
			format, err = a.scanIdent()
			if err != nil && err != commandEOF {
				break
			}

			if !a.scanEOF() {
				err = commandTooManyArgs
				break
			}

			var snaps *core.Snapshot
			snaps, err = a.client.FetchSnapshot()
			if err != nil {
				break
			}

			a.renderSnap(format, snaps)
		case "help":
			if !a.scanEOF() {
				err = commandTooManyArgs
				break
			}

			printHelp()
		default:
			err = &commandError{fmt.Sprintf("Unknown command: %s", command)}
		}

		if err == nil {
			return true
		}

	Error:
		if _, ok := err.(*commandError); ok {
			fmt.Printf("Parse error: %s\n", err)
			return false
		} else if _, ok := err.(*admin.ErrorResponse); ok {
			fmt.Printf("Log Courier returned an error: %s\n", err)
			return false
		} else {
			a.connected = false
			fmt.Printf("Connection error: %s\n", err)
		}

		if reconnected {
			break
		}
	}

	return false
}

func (a *lcAdmin) initScanner(command string) {
	a.scanner.Init(strings.NewReader(command))
	a.scanner.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings
	a.scanner.Whitespace = 1 << ' '

	a.scanner.Error = func(s *scanner.Scanner, msg string) {
		a.scannerErr = &commandError{msg}
	}
}

func (a *lcAdmin) scanIdent() (string, error) {
	r := a.scanner.Scan()
	if a.scannerErr != nil {
		return "", a.scannerErr
	}
	switch r {
	case scanner.Ident:
		return a.scanner.TokenText(), nil
	case scanner.EOF:
		return "", commandEOF
	}
	return "", &commandError{"Invalid token"}
}

func (a *lcAdmin) scanEOF() bool {
	r := a.scanner.Scan()
	if a.scannerErr == nil && r == scanner.EOF {
		return true
	}
	return false
}

func (a *lcAdmin) renderSnap(format string, snap *core.Snapshot) {
	switch format {
	case "json":
		fmt.Printf("{\n")
		a.renderSnapJSON("\t", snap)
		fmt.Printf("}\n")
	default:
		a.renderSnapYAML("", snap)
	}
}

func (a *lcAdmin) renderSnapJSON(indent string, snap *core.Snapshot) {
	if snap.NumEntries() != 0 {
		for i, j := 0, snap.NumEntries(); i < j; i = i + 1 {
			k, v := snap.Entry(i)
			switch t := v.(type) {
			case string:
				fmt.Printf(indent+"%q: %q", k, t)
			case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
				fmt.Printf(indent+"%q: %d", k, t)
			case float32, float64:
				fmt.Printf(indent+"%q: %.2f", k, t)
			case time.Time:
				fmt.Printf(indent+"%q: %q", k, t.Format("_2 Jan 2006 15.04.05"))
			case time.Duration:
				fmt.Printf(indent+"%q: %q", k, (t - (t % time.Second)).String())
			default:
				fmt.Printf(indent+"%q: %q", k, fmt.Sprintf("%v", t))
			}
			if i+1 < j || snap.NumSubs() != 0 {
				fmt.Printf(",\n")
			} else {
				fmt.Printf("\n")
			}
		}
	}
	if snap.NumSubs() != 0 {
		for i, j := 0, snap.NumSubs(); i < j; i = i + 1 {
			subSnap := snap.Sub(i)
			fmt.Printf(indent+"%q: {\n", subSnap.Description())
			a.renderSnapJSON(indent+"\t", subSnap)
			if i+1 < j {
				fmt.Printf(indent + "},\n")
			} else {
				fmt.Printf(indent + "}\n")
			}
		}
	}
}

func (a *lcAdmin) renderSnapYAML(indent string, snap *core.Snapshot) {
	if snap.NumEntries() != 0 {
		for i, j := 0, snap.NumEntries(); i < j; i = i + 1 {
			k, v := snap.Entry(i)
			switch t := v.(type) {
			case string:
				fmt.Printf(indent+"%s: %s\n", k, t)
			case int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
				fmt.Printf(indent+"%s: %d\n", k, t)
			case float32, float64:
				fmt.Printf(indent+"%s: %.2f\n", k, t)
			case time.Time:
				fmt.Printf(indent+"%s: %s\n", k, t.Format("_2 Jan 2006 15.04.05"))
			case time.Duration:
				fmt.Printf(indent+"%s: %s\n", k, (t - (t % time.Second)).String())
			default:
				fmt.Printf(indent+"%s: %v\n", k, t)
			}
		}
	}
	if snap.NumSubs() != 0 {
		for i, j := 0, snap.NumSubs(); i < j; i = i + 1 {
			subSnap := snap.Sub(i)
			fmt.Printf(indent+"%s:\n", subSnap.Description())
			a.renderSnapYAML(indent+"  ", subSnap)
		}
	}
}

func (a *lcAdmin) run() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	commandChan := make(chan string)
	go func() {
		var discard bool
		reader := bufio.NewReader(os.Stdin)
		for {
			line, prefix, err := reader.ReadLine()
			if err != nil {
				break
			} else if prefix {
				discard = true
			} else if discard {
				fmt.Printf("Line too long!\n")
				discard = false
			} else {
				commandChan <- string(line)
			}
		}
	}()

CommandLoop:
	for {
		fmt.Printf("> ")
		select {
		case command := <-commandChan:
			if command == "exit" {
				break CommandLoop
			}
			a.processCommand(command)
		case <-signalChan:
			fmt.Printf("\n> exit\n")
			break CommandLoop
		}
	}
}

func (a *lcAdmin) argsCommand(args []string, watch bool) bool {
	var signalChan chan os.Signal

	if watch {
		signalChan = make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
	}

WatchLoop:
	for {
		if !a.processCommand(strings.Join(args, " ")) {
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
