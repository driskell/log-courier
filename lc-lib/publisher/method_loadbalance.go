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

package publisher

import (
	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/endpoint"
)

type methodLoadbalance struct {
	sink      *endpoint.Sink
	netConfig *config.Network
}

func newMethodLoadbalance(sink *endpoint.Sink, cfg *config.Config) *methodLoadbalance {
	ret := &methodLoadbalance{
		sink: sink,
	}

	// Reload configuration to ensure all servers are present in the sink
	ret.reloadConfig(cfg)

	return ret
}

func (m *methodLoadbalance) onFail(endpoint *endpoint.Endpoint) {
	// All endpoints are maintained
	return
}

func (m *methodLoadbalance) onFinish(endpoint *endpoint.Endpoint) bool {
	// All endpoints are maintained
	return true
}

func (m *methodLoadbalance) onStarted(endpoint *endpoint.Endpoint) {
	// All endpoints are maintained
	return
}

func (m *methodLoadbalance) reloadConfig(cfg *config.Config) {
	m.netConfig = cfg.Network()

	// Verify all servers are present and reload them
	var last, foundEndpoint *endpoint.Endpoint
	for _, server := range m.netConfig.Servers {
		if foundEndpoint = m.sink.FindEndpoint(server); foundEndpoint == nil {
			// Add a new endpoint
			last = m.sink.AddEndpointAfter(server, addresspool.NewPool(server), false, last)
			log.Debug("[Loadbalance] Initialised new endpoint: %s", last.Server())
			continue
		}

		// Ensure ordering
		m.sink.MoveEndpointAfter(foundEndpoint, last)
		foundEndpoint.ReloadConfig(cfg, false)
		last = foundEndpoint
	}
}
