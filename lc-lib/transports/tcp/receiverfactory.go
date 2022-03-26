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
	"regexp"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultSSLVerifyPeers = true
)

// Factory holds common TCP factory settings
type Factory struct {
	config         *config.Config
	hostportRegexp *regexp.Regexp
}

func newFactory(config *config.Config) *Factory {
	return &Factory{
		config:         config,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}
}

type ReceiverFactory struct {
	*Factory

	EnableTls bool

	SSLClientCA    []string `config:"ssl client ca"`
	SSLVerifyPeers bool     `config:"verify_peers"`

	*transports.TlsConfiguration `config:",embed"`
}

func NewReceiverFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, enableTls bool) (*ReceiverFactory, error) {
	ret := &ReceiverFactory{
		Factory:   newFactory(p.Config()),
		EnableTls: enableTls,
	}
	if err := p.Populate(ret, unUsed, configPath, false); err != nil {
		return nil, err
	}
	return ret, nil
}

// NewTransport for this factory base is null and just so we can cast down
func (f *ReceiverFactory) NewReceiver(context.Context, *addresspool.Pool, chan<- transports.Event) transports.Receiver {
	panic("Not implemented")
}

// NewReceiverWithProtocol creates a new receiver with the given protocol
func (f *ReceiverFactory) NewReceiverWithProtocol(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event, protocolFactory ProtocolFactory) transports.Receiver {
	ret := &receiverTCP{
		config:       f,
		pool:         pool,
		eventChan:    eventChan,
		connections:  make(map[*connection]*connection),
		shutdownChan: make(chan struct{}),
		// TODO: Own values
		backoff:         core.NewExpBackoff(pool.Server()+" Receiver Reset", 0, 300*time.Second),
		protocolFactory: protocolFactory,
	}

	ret.ctx, ret.shutdownFunc = context.WithCancel(context.WithValue(ctx, transports.ContextReceiver, ret))

	ret.startController()
	return ret
}

// Defaults sets the default configuration values
func (f *ReceiverFactory) Defaults() {
	f.SSLVerifyPeers = defaultSSLVerifyPeers
}

// Validate the configuration
func (f *ReceiverFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.EnableTls {
		for idx, clientCA := range f.SSLClientCA {
			if f.CaList, err = transports.AddCertificates(f.CaList, clientCA); err != nil {
				return fmt.Errorf("failure loading %sssl client ca[%d]: %s", configPath, idx, err)
			}
		}
	} else {
		if len(f.SSLClientCA) > 0 {
			return fmt.Errorf("%sssl client ca is not supported when the transport is tcp", configPath)
		}
	}

	if err = f.TlsValidate(f.EnableTls, p, configPath); err != nil {
		return err
	}

	return nil
}

func (f *ReceiverFactory) ShouldRestart(newFactory transports.ReceiverFactory) bool {
	newFactoryImpl := newFactory.(*ReceiverFactory)

	if !reflect.DeepEqual(newFactoryImpl.SSLClientCA, f.SSLClientCA) {
		return true
	}
	if f.TlsConfiguration.HasChanged(newFactoryImpl.TlsConfiguration) {
		return true
	}
	return false
}
