// +build !windows

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
	"flag"
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

func (lc *LogCourier) registerSignals() {
	// *nix systems support SIGTERM so handle shutdown on that too
	signal.Notify(lc.shutdown_chan, os.Interrupt, syscall.SIGTERM)

	// *nix has SIGHUP for reload
	signal.Notify(lc.reload_chan, syscall.SIGHUP)
}

type LogCourierPlatformOther struct {
	syslog bool
}

func NewLogCourierPlatform() LogCourierPlatform {
	return &LogCourierPlatformOther{}
}

func (lcp *LogCourierPlatformOther) Init() {
	flag.BoolVar(&lcp.syslog, "log-to-syslog", false, "Log to syslog as well as stdout")
}

func (lcp *LogCourierPlatformOther) ConfigureLogging(backends []logging.Backend) {
	// Make it color if it's a TTY
	if lcp.isatty(os.Stdout) {
		backends[0].(*logging.LogBackend).Color = true
	}

	if lcp.syslog {
		syslog_backend, err := logging.NewSyslogBackend("log-courier")
		if err != nil {
			log.Fatalf("Failed to open syslog: %s", err)
		}
		backends = append(backends, syslog_backend)
	}
}

func (lcp *LogCourierPlatformOther) isatty(f *os.File) bool {
	var pgrp int64
	// Most real isatty implementations use TIOCGETA
	// However, TIOCGPRGP is easier than TIOCGETA as it only requires an int and not a termios struct
	// There is a possibility it may not have the exact same effect - but seems fine to me
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)));
	return err == 0
}
