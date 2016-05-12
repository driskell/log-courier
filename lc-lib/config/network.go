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

package config

import (
	"fmt"
	"time"
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

// Network holds network related configuration
type Network struct {
	Factory interface{}

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

// InitDefaults initiases default values for the network configuration
func (nc *Network) InitDefaults() {
	nc.Backoff = defaultNetworkBackoff
	nc.BackoffMax = defaultNetworkBackoffMax
	nc.MaxPendingPayloads = defaultNetworkMaxPendingPayloads
	nc.Method = defaultNetworkMethod
	nc.Rfc2782Service = defaultNetworkRfc2782Service
	nc.Rfc2782Srv = defaultNetworkRfc2782Srv
	nc.Timeout = defaultNetworkTimeout
	nc.Transport = defaultNetworkTransport
}

// Validate configuration
func (nc *Network) Validate(cfg *Config, buildMetadata bool) (err error) {
	if nc.Method == "" {
		nc.Method = defaultNetworkMethod
	}
	if nc.Method != "random" && nc.Method != "failover" && nc.Method != "loadbalance" {
		err = fmt.Errorf("The network method (/network/method) is not recognised: %s", nc.Method)
		return
	}

	if len(nc.Servers) == 0 {
		err = fmt.Errorf("No network servers were specified (/network/servers)")
		return
	}

	servers := make(map[string]bool)
	for _, server := range nc.Servers {
		if _, exists := servers[server]; exists {
			err = fmt.Errorf("The list of network servers (/network/servers) must be unique: %s appears multiple times", server)
			return
		}
		servers[server] = true
	}

	if buildMetadata {
		if registrarFunc, ok := registeredTransports[nc.Transport]; ok {
			if nc.Factory, err = registrarFunc(cfg, "/network/", nc.Unused, nc.Transport); err != nil {
				return
			}
		} else {
			err = fmt.Errorf("Unrecognised transport '%s'", nc.Transport)
			return
		}
	}

	return
}

// Network returns the network configuration
func (c *Config) Network() *Network {
	return c.Sections["network"].(*Network)
}

func init() {
	RegisterConfigSection("network", func() Section {
		return &Network{}
	})
}
