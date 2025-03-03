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
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
)

// Config holds network related configuration
type Config struct {
	Factory TransportFactory

	Backoff            time.Duration `config:"failure backoff"`
	BackoffMax         time.Duration `config:"failure backoff max"`
	MaxPendingPayloads int64         `config:"max pending payloads"`
	Method             string        `config:"method"`
	Rfc2782Service     string        `config:"rfc 2782 service"`
	Rfc2782Srv         bool          `config:"rfc 2782 srv"`
	Servers            []string      `config:"servers"`
	Timeout            time.Duration `config:"timeout"`
	Transport          string        `config:"transport"`

	Unused map[string]interface{} `json:",omitempty"`
}

// Init the transport configuration based on which was chosen
func (nc *Config) Init(p *config.Parser, path string) (err error) {
	registrarFunc, ok := registeredTransports[nc.Transport]
	if !ok {
		err = fmt.Errorf("unrecognised transport %s", nc.Transport)
		return
	}

	nc.Factory, err = registrarFunc(p, path, nc.Unused, nc.Transport)
	return
}

// Validate configuration
func (nc *Config) Validate(p *config.Parser, path string) (err error) {
	if nc.Method == "" {
		nc.Method = defaultNetworkMethod
	}
	if nc.Method != "random" && nc.Method != "failover" && nc.Method != "loadbalance" {
		err = fmt.Errorf("%smethod is not a recognised value: %s", path, nc.Method)
		return
	}

	if len(nc.Servers) == 0 {
		err = fmt.Errorf("%sservers is required", path)
		return
	}

	servers := make(map[string]bool)
	for _, server := range nc.Servers {
		if _, exists := servers[server]; exists {
			err = fmt.Errorf("%sservers must be unique: %s appears multiple times", path, server)
			return
		}
		servers[server] = true
	}

	return
}

// FetchConfig returns the network configuration from a Config structure
func FetchConfig(cfg *config.Config) *Config {
	return cfg.Section("network").(*Config)
}

func init() {
	config.RegisterSection("network", func() interface{} {
		return &Config{
			Backoff:            defaultNetworkBackoff,
			BackoffMax:         defaultNetworkBackoffMax,
			MaxPendingPayloads: defaultNetworkMaxPendingPayloads,
			Method:             defaultNetworkMethod,
			Rfc2782Service:     defaultNetworkRfc2782Service,
			Rfc2782Srv:         defaultNetworkRfc2782Srv,
			Timeout:            defaultNetworkTimeout,
			Transport:          defaultNetworkTransport,
		}
	})
}
