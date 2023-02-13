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
	*Factory `config:",embed"`

	EnableTls bool

	Reconnect    time.Duration `config:"reconnect backoff"`
	ReconnectMax time.Duration `config:"reconnect backoff max"`

	*transports.ClientTlsConfiguration `config:",embed"`
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

func (f *TransportFactory) Defaults() {
	f.Reconnect = defaultReconnect
	f.ReconnectMax = defaultReconnectMax
}

func (f *TransportFactory) Validate(p *config.Parser, configPath string) (err error) {
	return f.ClientTlsConfiguration.TlsValidate(f.EnableTls, p, configPath)
}

// NewTransportWithProtocol creates a new transport with the given protocol
func (f *TransportFactory) NewTransportWithProtocol(ctx context.Context, factory transports.TransportFactory, poolEntry *addresspool.PoolEntry, eventChan chan<- transports.Event, protocolFactory ProtocolFactory) transports.Transport {
	cancelCtx, shutdownFunc := context.WithCancel(ctx)

	backoffName := fmt.Sprintf("[T %s] Reconnect", poolEntry.Server)
	ret := &transportTCP{
		ctx:             cancelCtx,
		shutdownFunc:    shutdownFunc,
		config:          f,
		factory:         factory,
		netConfig:       transports.FetchConfig(f.config),
		poolEntry:       poolEntry,
		eventChan:       eventChan,
		backoff:         core.NewExpBackoff(backoffName, f.Reconnect, f.ReconnectMax),
		protocolFactory: protocolFactory,
	}

	ret.startController()
	return ret
}

func (f *TransportFactory) ShouldRestart(newFactory *TransportFactory) bool {
	if newFactory.Reconnect != f.Reconnect {
		return true
	}
	if newFactory.ReconnectMax != f.ReconnectMax {
		return true
	}
	if f.ClientTlsConfiguration.HasChanged(newFactory.ClientTlsConfiguration) {
		return true
	}
	return false
}
