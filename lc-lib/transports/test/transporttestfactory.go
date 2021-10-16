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

package test

import (
	"context"
	"fmt"
	"math"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	TransportTest string = "test"
)

// TransportTestFactory holds the configuration from the configuration file
// It allows creation of TransportTest instances that use this configuration
type TransportTestFactory struct {
	// Constructor
	config    *config.Config
	transport string

	// Config
	MinDelay int64 `config:"min delay"`
	MaxDelay int64 `config:"max delay"`
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTestFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.TransportFactory, error) {
	ret := &TransportTestFactory{
		config:    p.Config(),
		transport: name,
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportTestFactory) NewTransport(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Transport {
	ret := &transportTest{
		ctx:       ctx,
		config:    f,
		eventChan: eventChan,
		server:    pool.Server(),
	}
	eventChan <- transports.NewStatusEvent(ctx, transports.Started, nil)
	return ret
}

// Validate the configuration
func (f *TransportTestFactory) Validate(p *config.Parser, path string) (err error) {
	if f.MinDelay == math.MaxInt64 {
		f.MinDelay = 0
	}
	if f.MaxDelay == math.MaxInt64 {
		f.MaxDelay = f.MinDelay
	}
	if f.MaxDelay < f.MinDelay {
		return fmt.Errorf("`%s/max delay` cannot be higher than `%s/min delay`", path, path)
	}
	return nil
}

// Defaults sets the default configuration values
func (f *TransportTestFactory) Defaults() {
	f.MinDelay = math.MaxInt64
	f.MaxDelay = math.MaxInt64
}

// Register the transports
func init() {
	transports.RegisterTransport(TransportTest, NewTransportTestFactory)
}
