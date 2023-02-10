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

package courier

import (
	"context"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// TransportFactory holds the configuration from the configuration file
// It allows creation of TransportTCP instances that use this configuration
type TransportFactory struct {
	*tcp.TransportFactory `,config:"embed"`

	// Constructor
	config         *config.Config
	transport      string
	hostportRegexp *regexp.Regexp
}

// NewTransportFactory create a new TransportFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	factory, err := tcp.NewTransportFactory(p, configPath, unUsed, name == TransportCourierTLS)
	if err != nil {
		return nil, err
	}

	ret := &TransportFactory{
		TransportFactory: factory,
		config:           p.Config(),
		transport:        name,
		hostportRegexp:   regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// NewTransport returns a new Transport interface using the settings from the
// TransportFactory.
func (f *TransportFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Transport {
	return f.TransportFactory.NewTransportWithProtocol(ctx, pool, eventChan, &protocolFactory{isClient: true})
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportCourier, NewTransportFactory)
	transports.RegisterTransport(TransportCourierTLS, NewTransportFactory)
}
