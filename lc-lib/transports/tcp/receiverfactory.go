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

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type ReceiverFactory struct {
	*Factory `config:",embed"`

	EnableTls bool

	*transports.ServerTlsConfiguration `config:",embed"`
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
func (f *ReceiverFactory) NewReceiver(context.Context, string, chan<- transports.Event) transports.Receiver {
	panic("Not implemented")
}

// NewReceiverWithProtocol creates a new receiver with the given protocol
func (f *ReceiverFactory) NewReceiverWithProtocol(ctx context.Context, factory transports.ReceiverFactory, bind string, eventChan chan<- transports.Event, protocolFactory ProtocolFactory) transports.Receiver {
	backoffName := fmt.Sprintf("[R %s] Receiver Reset", bind)
	ret := &receiverTCP{
		config:       f,
		factory:      factory,
		bind:         bind,
		eventChan:    eventChan,
		connections:  make(map[*connection]*connection),
		shutdownChan: make(chan struct{}),
		// TODO: Own values
		backoff:         core.NewExpBackoff(backoffName, 0, 300*time.Second),
		protocolFactory: protocolFactory,
	}

	ret.ctx, ret.shutdownFunc = context.WithCancel(context.WithValue(ctx, transports.ContextReceiver, ret))

	ret.startController()
	return ret
}

func (f *ReceiverFactory) Defaults() {
}

func (f *ReceiverFactory) Validate(p *config.Parser, configPath string) (err error) {
	return f.ServerTlsConfiguration.TlsValidate(f.EnableTls, p, configPath)
}

func (f *ReceiverFactory) ShouldRestart(newFactory *ReceiverFactory) bool {
	return f.ServerTlsConfiguration.HasChanged(newFactory.ServerTlsConfiguration)
}
