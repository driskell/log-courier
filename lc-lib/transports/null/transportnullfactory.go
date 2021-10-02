/*
 * Copyright 2012-2020 Jason Woods and contributors
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

package null

import (
	"context"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	TransportNull string = "null"
)

// TransportNullFactory holds the configuration from the configuration file
// It allows creation of TransportNull instances that use this configuration
type TransportNullFactory struct {
	// Constructor
	config    *config.Config
	transport string
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportNullFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &TransportNullFactory{
		config:    p.Config(),
		transport: name,
	}
	return ret, nil
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportNullFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event, finishOnFail bool) transports.Transport {
	ret := &transportNull{
		ctx:       ctx,
		eventChan: eventChan,
	}
	eventChan <- transports.NewStatusEvent(ctx, transports.Started)
	return ret
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportNull, NewTransportNullFactory)
}
