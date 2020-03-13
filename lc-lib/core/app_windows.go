// +build windows

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
	"os"
	"os/signal"

	"gopkg.in/op/go-logging.v1"
)

// registerSignals registers platform specific shutdown signals with the shutdown
// channel and reload signals with the reload channel
func (a *App) registerSignals() {
	// Windows only supports os.Interrupt
	signal.Notify(a.shutdownChan, os.Interrupt)

	// No reload signal for Windows - implementation will have to wait
}

// isShutdownSignal always returns true on windows as we do not support reload
func isShutdownSignal(signal os.Signal) bool {
	return true
}

// configureLoggingPlatform enables platform specific logging backends in the
// logging configuration
func (a *App) configureLoggingPlatform(backends *[]logging.Backend) error {
	return nil
}
