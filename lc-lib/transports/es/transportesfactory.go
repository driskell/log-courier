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

package es

import (
	"context"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultRoutines int           = 4
	defaultRetry    time.Duration = 0 * time.Second
	defaultRetryMax time.Duration = 300 * time.Second
)

var (
	// TransportES is the transport name for ES HTTP
	TransportES = "es"
)

// TransportESFactory holds the configuration from the configuration file
// It allows creation of TransportES instances that use this configuration
type TransportESFactory struct {
	// Constructor
	config    *config.Config
	transport string

	// Configuration
	Routines int           `config:"routines"`
	Retry    time.Duration `config:"retry backoff"`
	RetryMax time.Duration `config:"retry backoff max"`
}

// NewTransportESFactory create a new TransportESFactory from the provided
// configuration data, reporting back any configuration errors it discovers
func NewTransportESFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	var err error

	ret := &TransportESFactory{
		config:    p.Config(),
		transport: name,
	}

	if err = p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}

	return ret, nil
}

// Defaults sets the default configuration values
func (f *TransportESFactory) Defaults() {
	f.Routines = defaultRoutines
	f.Retry = defaultRetry
	f.RetryMax = defaultRetryMax
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportESFactory) NewTransport(lcontext interface{}, pool *addresspool.Pool, eventChan chan<- transports.Event, finishOnFail bool) transports.Transport {
	shutdownContext, shutdownFunc := context.WithCancel(context.Background())

	ret := &transportES{
		config:          f,
		netConfig:       transports.FetchConfig(f.config),
		finishOnFail:    finishOnFail,
		context:         lcontext,
		pool:            pool,
		eventChan:       eventChan,
		shutdownContext: shutdownContext,
		shutdownFunc:    shutdownFunc,
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportES, NewTransportESFactory)
}
