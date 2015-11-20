/*
* Copyright 2014 Jason Woods.
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
	"flag"
	"fmt"
	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/core"
	"os"
	"os/signal"
	"strings"
	"text/scanner"
	"time"
)

type CommandError struct {
	message string
}

func (c *CommandError) Error() string {
	return c.message
}

var CommandEOF *CommandError = &CommandError{"EOF"}
var CommandTooManyArgs *CommandError = &CommandError{"Too many arguments"}

type Admin struct {
	client        *admin.Client
	connected     bool
	quiet         bool
	admin_connect string
	scanner       scanner.Scanner
	scanner_err   error
}

func NewAdmin(quiet bool, admin_connect string) *Admin {
	return &Admin{
		quiet:         quiet,
		admin_connect: admin_connect,
	}
}

func (a *Admin) connect() error {
	if !a.connected {
		var err error

		if !a.quiet {
			fmt.Printf("Attempting connection to %s...\n", a.admin_connect)
		}

		if a.client, err = admin.NewClient(a.admin_connect); err != nil {
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

func (a *Admin) ProcessCommand(command string) bool {
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
				err = CommandTooManyArgs
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
			if err != nil && err != CommandEOF {
				break
			}

			if !a.scanEOF() {
				err = CommandTooManyArgs
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
				err = CommandTooManyArgs
				break
			}

			PrintHelp()
		default:
			err = &CommandError{fmt.Sprintf("Unknown command: %s", command)}
		}

		if err == nil {
			return true
		}

	Error:
		if _, ok := err.(*CommandError); ok {
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

func (a *Admin) initScanner(command string) {
	a.scanner.Init(strings.NewReader(command))
	a.scanner.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings
	a.scanner.Whitespace = 1 << ' '

	a.scanner.Error = func(s *scanner.Scanner, msg string) {
		a.scanner_err = &CommandError{msg}
	}
}

func (a *Admin) scanIdent() (string, error) {
	r := a.scanner.Scan()
	if a.scanner_err != nil {
		return "", a.scanner_err
	}
	switch r {
	case scanner.Ident:
		return a.scanner.TokenText(), nil
	case scanner.EOF:
		return "", CommandEOF
	}
	return "", &CommandError{"Invalid token"}
}

func (a *Admin) scanEOF() bool {
	r := a.scanner.Scan()
	if a.scanner_err == nil && r == scanner.EOF {
		return true
	}
	return false
}

func (a *Admin) renderSnap(format string, snap *core.Snapshot) {
	switch format {
	case "json":
		fmt.Printf("{\n")
		a.renderSnapJSON("\t", snap)
		fmt.Printf("}\n")
	default:
		a.renderSnapYAML("", snap)
	}
}

func (a *Admin) renderSnapJSON(indent string, snap *core.Snapshot) {
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
			sub_snap := snap.Sub(i)
			fmt.Printf(indent+"%q: {\n", sub_snap.Description())
			a.renderSnapJSON(indent+"\t", sub_snap)
			if i+1 < j {
				fmt.Printf(indent + "},\n")
			} else {
				fmt.Printf(indent + "}\n")
			}
		}
	}
}

func (a *Admin) renderSnapYAML(indent string, snap *core.Snapshot) {
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
			sub_snap := snap.Sub(i)
			fmt.Printf(indent+"%s:\n", sub_snap.Description())
			a.renderSnapYAML(indent+"  ", sub_snap)
		}
	}
}

func (a *Admin) Run() {
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, os.Interrupt)

	command_chan := make(chan string)
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
				command_chan <- string(line)
			}
		}
	}()

CommandLoop:
	for {
		fmt.Printf("> ")
		select {
		case command := <-command_chan:
			if command == "exit" {
				break CommandLoop
			}
			a.ProcessCommand(command)
		case <-signal_chan:
			fmt.Printf("\n> exit\n")
			break CommandLoop
		}
	}
}

func (a *Admin) argsCommand(args []string, watch bool) bool {
	var signal_chan chan os.Signal

	if watch {
		signal_chan = make(chan os.Signal, 1)
		signal.Notify(signal_chan, os.Interrupt)
	}

WatchLoop:
	for {
		if !a.ProcessCommand(strings.Join(args, " ")) {
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
		case <-signal_chan:
			break WatchLoop
		case <-time.After(time.Second):
		}
	}

	return true
}

func PrintHelp() {
	fmt.Printf("Available commands:\n")
	fmt.Printf("  reload    Reload configuration\n")
	fmt.Printf("  status    Display the current shipping status\n")
	fmt.Printf("  exit      Exit\n")
}

func main() {
	var version bool
	var quiet bool
	var watch bool
	var admin_connect string

	flag.BoolVar(&version, "version", false, "display the Log Courier client version")
	flag.BoolVar(&quiet, "quiet", false, "quietly execute the command line argument and output only the result")
	flag.BoolVar(&watch, "watch", false, "repeat the command specified on the command line every second")
	flag.StringVar(&admin_connect, "connect", "tcp:127.0.0.1:1234", "the Log Courier instance to connect to (default tcp:127.0.0.1:1234)")

	flag.Parse()

	if version {
		fmt.Printf("Log Courier version %s\n", core.Log_Courier_Version)
		os.Exit(0)
	}

	if !quiet {
		fmt.Printf("Log Courier version %s client\n\n", core.Log_Courier_Version)
	}

	args := flag.Args()

	if len(args) != 0 {
		// Don't require a connection to display the help message
		if args[0] == "help" {
			PrintHelp()
			os.Exit(0)
		}

		admin := NewAdmin(quiet, admin_connect)
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

	admin := NewAdmin(quiet, admin_connect)
	if err := admin.connect(); err != nil {
		return
	}

	admin.Run()
}
