/*
 * Copyright 2014-2015 Jason Woods.
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
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/publisher"
	"gopkg.in/op/go-logging.v1"
)

import _ "github.com/driskell/log-courier/lc-lib/codecs"
import _ "github.com/driskell/log-courier/lc-lib/transports/tcp"

var (
	log *logging.Logger
	app *core.App
)

func main() {
	app = core.NewApp("Fact Courier", "fact-courier", core.LogCourierVersion)
	app.StartUp()
	setupPipeline()
	app.Run()
}

func setupPipeline() {
	log.Info("Configuring Fact Courier version %s pipeline", core.LogCourierVersion)

	publisherImpl := publisher.NewPublisher(app)
	app.AddToPipeline(publisherImpl)

	// TODO: Support arbitary scripts, not just Munin
	collector, err := NewMuninCollector(app, publisherImpl)
	if err != nil {
		log.Fatalf("Failed to initialise: %s", err)
	}

	app.AddToPipeline(collector)
}

func init() {
	log = logging.MustGetLogger("fact-courier")
}
