/*
 * Copyright 2014-2016 Jason Woods.
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

// ReceiverConfig contains extra section configuration values
type ReceiverConfig struct {
	Factory ReceiverFactory

	Enabled   bool   `config:"enabled"`
	Transport string `config:"transport"`

	Unused map[string]interface{}
}

// Init the receiver configuration
func (nc *ReceiverConfig) Init(p *config.Parser, path string) (err error) {
	registrarFunc, ok := registeredReceivers[nc.Transport]
	if !ok {
		err = fmt.Errorf("Unrecognised listener transport '%s'", nc.Transport)
		return
	}

	nc.Factory, err = registrarFunc(p, path+"/", nc.Unused, nc.Transport)

	return
}

// FetchReceiverConfig returns the network configuration from a Config structure
func FetchReceiverConfig(cfg *config.Config) *ReceiverConfig {
	return cfg.Section("listener").(*ReceiverConfig)
}

func init() {
	config.RegisterSection("listener", func() interface{} {
		return &ReceiverConfig{
			Transport: defaultNetworkTransport,
		}
	})
}
