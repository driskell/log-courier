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

package tcp

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultReconnect    time.Duration = 0 * time.Second
	defaultReconnectMax time.Duration = 300 * time.Second
)

// TransportTCPFactory holds the configuration from the configuration file
// It allows creation of TransportTCP instances that use this configuration
type TransportTCPFactory struct {
	// Constructor
	config         *config.Config
	transport      string
	hostportRegexp *regexp.Regexp

	// Configuration
	Reconnect    time.Duration `config:"reconnect backoff"`
	ReconnectMax time.Duration `config:"reconnect backoff max"`
	SSLCA        string        `config:"ssl ca"`

	*TlsConfiguration `config:",embed"`
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTCPFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &TransportTCPFactory{
		config:         p.Config(),
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// Validate the configuration
func (f *TransportTCPFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.transport == TransportTCPTLS {
		if len(f.SSLCA) == 0 {
			return fmt.Errorf("%sssl ca is required when the transport is tls", configPath)
		}
		if f.caList, err = transports.AddCertificates(f.caList, f.SSLCA); err != nil {
			return fmt.Errorf("failure loading %sssl ca: %s", configPath, err)
		}
	} else {
		if len(f.SSLCA) > 0 {
			return fmt.Errorf("%[1]sssl ca is not supported when the transport is tcp", configPath)
		}
	}

	return f.tlsValidate(f.transport, p, configPath)
}

// Defaults sets the default configuration values
func (f *TransportTCPFactory) Defaults() {
	f.Reconnect = defaultReconnect
	f.ReconnectMax = defaultReconnectMax
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportTCPFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Transport {
	cancelCtx, shutdownFunc := context.WithCancel(ctx)

	ret := &transportTCP{
		ctx:          cancelCtx,
		shutdownFunc: shutdownFunc,
		config:       f,
		netConfig:    transports.FetchConfig(f.config),
		pool:         pool,
		eventChan:    eventChan,
		backoff:      core.NewExpBackoff(pool.Server()+" Reconnect", f.Reconnect, f.ReconnectMax),
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportTCPTCP, NewTransportTCPFactory)
	transports.RegisterTransport(TransportTCPTLS, NewTransportTCPFactory)
}
