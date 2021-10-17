// +build !windows

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

package core

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"unsafe"

	"gopkg.in/op/go-logging.v1"
)

// registerSignals registers platform specific shutdown signals with the shutdown
// channel and reload signals with the reload channel
func (a *App) registerSignals() {
	// *nix systems support SIGTERM so handle shutdown on that too
	signal.Notify(a.signalChan, os.Interrupt, syscall.SIGTERM)

	// *nix has SIGHUP for reload
	signal.Notify(a.signalChan, syscall.SIGHUP)
}

// isShutdownSignal returns true if the signal provided is a shutdown signal
func isShutdownSignal(signal os.Signal) bool {
	return signal != syscall.SIGHUP
}

// configureLoggingPlatform enables platform specific logging backends in the
// logging configuration
func (a *App) configureLoggingPlatform(backends *[]logging.Backend) error {
	// Make it color if it's a TTY
	// TODO: This could be prone to problems when updating logging in future
	if isatty(os.Stdout) && a.config.General().LogStdout {
		(*backends)[0].(*logging.LogBackend).Color = true
	}

	if a.config.General().LogSyslog {
		syslogBackend, err := logging.NewSyslogBackend(path.Base(os.Args[0]))
		if err != nil {
			return fmt.Errorf("failed to open syslog: %s", err)
		}
		newBackends := append(*backends, syslogBackend)
		*backends = newBackends
	}

	return nil
}

// isatty is used to detect a console terminal so that coloured logging can be
// enabled
func isatty(f *os.File) bool {
	var pgrp int64
	// NOTE(Driskell): Most real isatty implementations use TIOCGETA
	// However, TIOCGPRGP is easier than TIOCGETA as it only requires an int and not a termios struct
	// There is a possibility it may not have the exact same effect - but seems fine to me
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)))
	return err == 0
}
