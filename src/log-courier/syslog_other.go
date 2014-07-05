// +build !windows

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
  "github.com/op/go-logging"
  stdlog "log"
  "os"
  "syscall"
  "unsafe"
)

func (lc *LogCourier) configureLogging(syslog bool) {
  // First, the stdout backend
  backends := make([]logging.Backend, 1)
  stderr_backend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lmicroseconds)
  backends[0] = stderr_backend

  // Make it color if it's a TTY
  if lc.isatty(os.Stderr) {
    stderr_backend.Color = true
  }

  if syslog {
    syslog_backend, err := logging.NewSyslogBackend("log-courier")
    if err != nil {
      log.Fatalf("Failed to open syslog: %s", err)
    }
    backends = append(backends, syslog_backend)
  }

  logging.SetBackend(backends...)
}

func (lc *LogCourier) isatty(f *os.File) bool {
  var pgrp int64
  // Most real isatty implementations use TIOCGETA
  // However, TIOCGPRGP is easier than TIOCGETA as it only requires an int and not a termios struct
  // There is a possibility it may not have the exact same effect - but seems fine to me
  _, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)));
  return err == 0
}
