/*
 * Copyright 2014-2015 Jason Woods.
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

package transports

import (
	"fmt"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
)

const (
	defaultNetworkBackoff            time.Duration = 5 * time.Second
	defaultNetworkBackoffMax         time.Duration = 300 * time.Second
	defaultNetworkMaxPendingPayloads int64         = 10
	defaultNetworkMethod             string        = "random"
	defaultNetworkRfc2782Service     string        = "courier"
	defaultNetworkRfc2782Srv         bool          = true
	defaultNetworkTimeout            time.Duration = 15 * time.Second
	defaultNetworkTransport          string        = "tls"
)

// Config holds network related configuration
type Config struct {
	Factory      TransportFactory
	AddressPools []*addresspool.Pool

	Backoff            time.Duration `config:"failure backoff"`
	BackoffMax         time.Duration `config:"failure backoff max"`
	MaxPendingPayloads int64         `config:"max pending payloads"`
	Method             string        `config:"method"`
	Rfc2782Service     string        `config:"rfc 2782 service"`
	Rfc2782Srv         bool          `config:"rfc 2782 srv"`
	Servers            []string      `config:"servers"`
	Timeout            time.Duration `config:"timeout"`
	Transport          string        `config:"transport"`

	Unused map[string]interface{}
}

// Init the transport configuration based on which was chosen
func (nc *Config) Init(p *config.Parser, path string) (err error) {
	registrarFunc, ok := registeredTransports[nc.Transport]
	if !ok {
		err = fmt.Errorf("Unrecognised transport '%s'", nc.Transport)
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
		err = fmt.Errorf("The network method (%s/method) is not recognised: %s", path, nc.Method)
		return
	}

	if len(nc.Servers) == 0 {
		err = fmt.Errorf("No network servers were specified (%s/servers)", path)
		return
	}

	servers := make(map[string]bool)
	nc.AddressPools = make([]*addresspool.Pool, len(nc.Servers))
	for n, server := range nc.Servers {
		if _, exists := servers[server]; exists {
			err = fmt.Errorf("The list of network servers (%s/servers) must be unique: %s appears multiple times", path, server)
			return
		}
		servers[server] = true
		nc.AddressPools[n] = addresspool.NewPool(server)
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
