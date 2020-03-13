/*
 * Copyright 2014-2019 Jason Woods.
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
	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/processor"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"github.com/driskell/log-courier/lc-lib/receiver"
	"github.com/driskell/log-courier/lc-lib/spooler"

	_ "github.com/driskell/log-courier/lc-lib/codecs/filter"
	_ "github.com/driskell/log-courier/lc-lib/codecs/multiline"
	_ "github.com/driskell/log-courier/lc-lib/codecs/plain"

	_ "github.com/driskell/log-courier/lc-lib/transports/es"
	_ "github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// Generate platform-specific default configuration values
//go:generate go run ../lc-lib/config/generate/platform.go platform main config.DefaultConfigurationFile prospector.DefaultGeneralPersistDir admin.DefaultAdminBind

var (
	app *core.App

	stdin         bool
	fromBeginning bool
)

func main() {
	app = core.NewApp("Log Carver", "log-carver", core.LogCourierVersion)
	app.StartUp()

	if app.Config().Section("admin").(*admin.Config).Enabled {
		app.Pipeline().AddService(admin.NewServer(app))
	}

	// Receivers will receive over the network
	app.Pipeline().AddSource(receiver.NewPool(app))

	// Add spooler as first processor, it combines into larger chunks as needed
	app.Pipeline().AddProcessor(spooler.NewSpooler(app))

	// Add processors
	app.Pipeline().AddProcessor(processor.NewPool(app))

	// Create sink
	app.Pipeline().SetSink(publisher.NewPublisher())

	// Go!
	app.Run()
}
