// +build windows

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
	"os"
	"os/signal"

	"github.com/driskell/log-courier/Godeps/_workspace/src/github.com/op/go-logging"
)

// registerSignals registers platform specific shutdown signals with the shutdown
// channel and reload signals with the reload channel
func (lc *LogCourier) registerSignals() {
	// Windows only supports os.Interrupt
	signal.Notify(lc.shutdownChan, os.Interrupt)

	// No reload signal for Windows - implementation will have to wait
}

// configureLoggingPlatform enables platform specific logging backends in the
// logging configuration
func (lc *LogCourier) configureLoggingPlatform(backends *[]logging.Backend) error {
	return nil
}
