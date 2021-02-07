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
	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"github.com/driskell/log-courier/lc-lib/spooler"
	"gopkg.in/op/go-logging.v1"

	_ "github.com/driskell/log-courier/lc-lib/codecs"

	_ "github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// Generate platform-specific default configuration values
//go:generate go run -mod=vendor ../lc-lib/config/generate/platform.go platform main config.DefaultConfigurationFile DefaultMuninConfigBase

var (
	log *logging.Logger
	app *core.App
)

func main() {
	app = core.NewApp("Fact Courier", "fact-courier", core.LogCourierVersion)
	app.StartUp()

	if app.Config().Section("admin").(*admin.Config).Enabled {
		app.Pipeline().AddService(admin.NewServer(app))
	}

	// TODO: Support arbitary scripts, not just Munin
	app.Pipeline().AddSource(NewMuninCollector(app))

	// Add spooler as first processor, it combines into larger chunks as needed
	app.Pipeline().AddProcessor(spooler.NewSpooler(app))
	// Create sink
	app.Pipeline().SetSink(publisher.NewPublisher())
	// Go!
	app.Run()
}

func init() {
	log = logging.MustGetLogger("fact-courier")
}
