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
	"reflect"
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

type TransportFactory struct {
	*Factory

	EnableTls bool

	Reconnect    time.Duration `config:"reconnect backoff"`
	ReconnectMax time.Duration `config:"reconnect backoff max"`
	SSLCA        string        `config:"ssl ca"`

	*transports.TlsConfiguration `config:",embed"`
}

func NewTransportFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, enableTls bool) (*TransportFactory, error) {
	ret := &TransportFactory{
		Factory:   newFactory(p.Config()),
		EnableTls: enableTls,
	}
	if err := p.Populate(ret, unUsed, configPath, false); err != nil {
		return nil, err
	}
	return ret, nil
}

// NewTransport for this factory base is null and just so we can cast down
func (f *TransportFactory) NewTransport(context.Context, *addresspool.Pool, chan<- transports.Event) transports.Transport {
	panic("Not implemented")
}

// NewTransportWithProtocol creates a new transport with the given protocol
func (f *TransportFactory) NewTransportWithProtocol(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event, protocolFactory ProtocolFactory) transports.Transport {
	cancelCtx, shutdownFunc := context.WithCancel(ctx)

	ret := &transportTCP{
		ctx:             cancelCtx,
		shutdownFunc:    shutdownFunc,
		config:          f,
		netConfig:       transports.FetchConfig(f.config),
		pool:            pool,
		eventChan:       eventChan,
		backoff:         core.NewExpBackoff(pool.Server()+" Reconnect", f.Reconnect, f.ReconnectMax),
		protocolFactory: protocolFactory,
	}

	ret.startController()
	return ret
}

// Defaults sets the default configuration values
func (f *TransportFactory) Defaults() {
	f.Reconnect = defaultReconnect
	f.ReconnectMax = defaultReconnectMax
}

// Validate the configuration
func (f *TransportFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.EnableTls {
		if len(f.SSLCA) == 0 {
			return fmt.Errorf("%sssl ca is required when the transport is tls", configPath)
		}
		if f.CaList, err = transports.AddCertificates(f.CaList, f.SSLCA); err != nil {
			return fmt.Errorf("failure loading %sssl ca: %s", configPath, err)
		}
	} else {
		if len(f.SSLCA) > 0 {
			return fmt.Errorf("%[1]sssl ca is not supported when the transport is tcp", configPath)
		}
	}

	return f.TlsValidate(f.EnableTls, p, configPath)
}

func (f *TransportFactory) ShouldRestart(newFactory transports.TransportFactory) bool {
	newFactoryImpl := newFactory.(*TransportFactory)
	if newFactoryImpl.Reconnect != f.Reconnect {
		return true
	}
	if newFactoryImpl.ReconnectMax != f.ReconnectMax {
		return true
	}
	if !reflect.DeepEqual(newFactoryImpl.SSLCA, f.SSLCA) {
		return true
	}
	if f.TlsConfiguration.HasChanged(newFactoryImpl.TlsConfiguration) {
		return true
	}
	return false
}
