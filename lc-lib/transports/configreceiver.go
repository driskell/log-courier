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

package transports

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
)

const (
	defaultReceiverTransport = "tls"
)

// ReceiverConfig is the top level section configuration, and is an array of receivers
type ReceiverConfig []*ReceiverConfigEntry

// ReceiverConfigEntry contains configuration for a single receiver
type ReceiverConfigEntry struct {
	Factory ReceiverFactory

	Name               string   `config:"name"`
	Enabled            bool     `config:"enabled"`
	Transport          string   `config:"transport"`
	Listen             []string `config:"listen"`
	MaxPendingPayloads int64    `config:"max pending payloads"`
	MaxQueueSize       int64    `config:"max queue size"`

	Unused map[string]interface{} `json:",omitempty"`
}

// Defaults sets default receiver configuration
func (c *ReceiverConfigEntry) Defaults() {
	c.Enabled = true
	c.Transport = defaultReceiverTransport
	c.MaxPendingPayloads = defaultNetworkMaxPendingPayloads
	c.MaxQueueSize = defaultNetworkMaxQueueSize
}

// Init the receiver configuration
func (c *ReceiverConfigEntry) Init(p *config.Parser, path string) (err error) {
	registrarFunc, ok := registeredReceivers[c.Transport]
	if !ok {
		err = fmt.Errorf("unrecognised receiver transport %s", c.Transport)
		return
	}

	c.Factory, err = registrarFunc(p, path, c.Unused, c.Transport)
	return
}

// Validate the receiver configuration
func (c *ReceiverConfigEntry) Validate(p *config.Parser, path string) (err error) {
	if len(c.Listen) == 0 {
		err = fmt.Errorf("%slisten is required", path)
		return
	}

	return nil
}

// FetchReceiversConfig returns the network configuration from a Config structure
func FetchReceiversConfig(cfg *config.Config) ReceiverConfig {
	return cfg.Section("receivers").(ReceiverConfig)
}

func init() {
	config.RegisterSection("receivers", func() interface{} {
		return ReceiverConfig{}
	})
}
