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

package publisher

import (
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/publisher/endpoint"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type methodLoadbalance struct {
	sink      *endpoint.Sink
	netConfig *transports.Config
}

func newMethodLoadbalance(sink *endpoint.Sink, netConfig *transports.Config) *methodLoadbalance {
	ret := &methodLoadbalance{
		sink: sink,
	}

	// Reload configuration to ensure all servers are present in the sink
	ret.reloadConfig(netConfig)

	// Schedule refresh of the available servers - we will do fresh DNS lookups
	sink.Scheduler.SetCallback(ret, 60*time.Second, func() {
		ret.refresh(false)
	})

	return ret
}

func (m *methodLoadbalance) onFail(endpoint *endpoint.Endpoint) {
	// All endpoints are maintained
}

func (m *methodLoadbalance) onFinish(endpoint *endpoint.Endpoint) bool {
	// All endpoints are maintained
	return true
}

func (m *methodLoadbalance) onStarted(endpoint *endpoint.Endpoint) {
	// All endpoints are maintained
}

func (m *methodLoadbalance) reloadConfig(netConfig *transports.Config) {
	m.netConfig = netConfig
	m.refresh(true)
}

func (m *methodLoadbalance) refresh(reloadConfig bool) {
	// Verify all servers are present and reload them
	entries, err := addresspool.GeneratePool(m.netConfig.Servers, m.netConfig.Rfc2782Srv, m.netConfig.Rfc2782Service, time.Second*60)
	if err != nil {
		log.Warning("[P Loadbalance] Server lookup failure: %s", err)
	}

	var last, foundEndpoint *endpoint.Endpoint
	for _, poolEntry := range entries {
		if foundEndpoint = m.sink.FindEndpoint(poolEntry); foundEndpoint == nil {
			// Add a new endpoint
			last = m.sink.AddEndpointAfter(
				poolEntry,
				last,
			)
			log.Info("[P Loadbalance] Initialised new endpoint: %s", last.PoolEntry().Desc)
			continue
		}

		// Ensure ordering
		m.sink.MoveEndpointAfter(foundEndpoint, last)
		if reloadConfig {
			foundEndpoint.ReloadConfig(m.netConfig)
		}
		last = foundEndpoint
	}

	m.sink.Scheduler.SetCallback(m, 60*time.Second, func() {
		m.refresh(false)
	})
}

func (m *methodLoadbalance) destroy() {
	m.sink.Scheduler.Remove(m)
}
