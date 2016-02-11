// +build !windows

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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"github.com/driskell/log-courier/Godeps/_workspace/src/github.com/op/go-logging"
)

// registerSignals registers platform specific shutdown signals with the shutdown
// channel and reload signals with the reload channel
func (lc *LogCourier) registerSignals() {
	// *nix systems support SIGTERM so handle shutdown on that too
	signal.Notify(lc.shutdownChan, os.Interrupt, syscall.SIGTERM)

	// *nix has SIGHUP for reload
	signal.Notify(lc.reloadChan, syscall.SIGHUP)
}

// configureLoggingPlatform enables platform specific logging backends in the
// logging configuration
func (lc *LogCourier) configureLoggingPlatform(backends *[]logging.Backend) error {
	// Make it color if it's a TTY
	// TODO: This could be prone to problems when updating logging in future
	if lc.isatty(os.Stdout) && lc.config.General.LogStdout {
		(*backends)[0].(*logging.LogBackend).Color = true
	}

	if lc.config.General.LogSyslog {
		syslogBackend, err := logging.NewSyslogBackend("log-courier")
		if err != nil {
			return fmt.Errorf("Failed to open syslog: %s", err)
		}
		newBackends := append(*backends, syslogBackend)
		*backends = newBackends
	}

	return nil
}

// isatty is used to detect a console terminal so that coloured logging can be
// enabled
func (lc *LogCourier) isatty(f *os.File) bool {
	var pgrp int64
	// NOTE(Driskell): Most real isatty implementations use TIOCGETA
	// However, TIOCGPRGP is easier than TIOCGETA as it only requires an int and not a termios struct
	// There is a possibility it may not have the exact same effect - but seems fine to me
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)))
	return err == 0
}
