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

package tcp

import (
	"context"
	"fmt"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

const (
	defaultSSLVerifyPeers = true
)

// ReceiverTCPFactory holds the configuration from the configuration file
// It allows creation of ReceiverTCP instances that use this configuration
type ReceiverTCPFactory struct {
	// Constructor
	config         *config.Config
	transport      string
	hostportRegexp *regexp.Regexp

	// Configuration
	SSLClientCA    []string `config:"ssl client ca"`
	SSLVerifyPeers bool     `config:"verify_peers"`

	tlsConfiguration
}

// NewReceiverTCPFactory create a new ReceiverTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewReceiverTCPFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.ReceiverFactory, error) {
	ret := &ReceiverTCPFactory{
		config:         p.Config(),
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
		SSLVerifyPeers: defaultSSLVerifyPeers,
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// Validate the configuration
func (f *ReceiverTCPFactory) Validate(p *config.Parser, configPath string) (err error) {
	if f.transport != TransportTCPTLS {
		if len(f.SSLClientCA) > 0 {
			return fmt.Errorf("%[1]sssl client ca is not supported when the transport is tcp", configPath)
		}
		return nil
	}

	if err = f.tlsValidate(f.transport, p, configPath); err != nil {
		return err
	}

	for idx, clientCA := range f.SSLClientCA {
		if err = f.addCa(clientCA, fmt.Sprintf("%sssl client ca[%d]", configPath, idx)); err != nil {
			return err
		}
	}

	return nil
}

// NewReceiver returns a new Receiver interface using the settings from the ReceiverTCPFactory
func (f *ReceiverTCPFactory) NewReceiver(ctx context.Context, pool *addresspool.Pool, eventChan chan<- transports.Event) transports.Receiver {
	ctx, shutdownFunc := context.WithCancel(ctx)

	ret := &receiverTCP{
		ctx:          ctx,
		shutdownFunc: shutdownFunc,
		config:       f,
		pool:         pool,
		eventChan:    eventChan,
		connections:  make(map[*connection]*connection),
		// TODO: Own values
		backoff: core.NewExpBackoff(pool.Server()+" Receiver Reset", defaultReconnect, defaultReconnectMax),
	}

	ret.startController()
	return ret
}

// Register the transports
func init() {
	transports.RegisterReceiver(TransportTCPTCP, NewReceiverTCPFactory)
	transports.RegisterReceiver(TransportTCPTLS, NewReceiverTCPFactory)
}
