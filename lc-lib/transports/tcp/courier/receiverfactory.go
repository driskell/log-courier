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

package courier

import (
	"context"
	"regexp"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// ReceiverFactory holds the configuration from the configuration file
// It allows creation of ReceiverTCP instances that use this configuration
type ReceiverFactory struct {
	*tcp.ReceiverFactory `,config:"embed"`

	// Constructor
	config         *config.Config
	transport      string
	hostportRegexp *regexp.Regexp
}

// NewReceiverFactory create a new ReceiverFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewReceiverFactory(p *config.Parser, configPath string, unUsed map[string]interface{}, name string) (transports.ReceiverFactory, error) {
	factory, err := tcp.NewReceiverFactory(p, configPath, unUsed, name == TransportCourierTLS)
	if err != nil {
		return nil, err
	}

	ret := &ReceiverFactory{
		ReceiverFactory: factory,
		config:          p.Config(),
		transport:       name,
		hostportRegexp:  regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}
	if err := p.Populate(ret, unUsed, configPath, true); err != nil {
		return nil, err
	}
	return ret, nil
}

// NewReceiver returns a new Receiver interface using the settings from the ReceiverFactory
func (f *ReceiverFactory) NewReceiver(ctx context.Context, listen string, eventChan chan<- transports.Event) transports.Receiver {
	return f.ReceiverFactory.NewReceiverWithProtocol(ctx, f, listen, eventChan, &protocolFactory{isClient: false})
}

func (f *ReceiverFactory) ShouldRestart(newFactory transports.ReceiverFactory) bool {
	return f.ReceiverFactory.ShouldRestart(newFactory.(*ReceiverFactory).ReceiverFactory)
}

// Register the transports
func init() {
	transports.RegisterReceiver(TransportCourier, NewReceiverFactory)
	transports.RegisterReceiver(TransportCourierTLS, NewReceiverFactory)
}
