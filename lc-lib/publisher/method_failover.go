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

type methodFailover struct {
	sink             *endpoint.Sink
	netConfig        *transports.Config
	currentEndpoint  *endpoint.Endpoint
	failoverPosition int
}

func newMethodFailover(sink *endpoint.Sink, netConfig *transports.Config) *methodFailover {
	ret := &methodFailover{
		sink:             sink,
		failoverPosition: 0,
	}

	// reloadConfig will fix up existing endpoints in the sink as well as setting
	// up the failover method and reloading endpoint configurations
	ret.reloadConfig(netConfig)

	return ret
}

func (m *methodFailover) onFail(endpoint *endpoint.Endpoint) {
	if m.currentEndpoint != endpoint {
		// Not the current endpoint, ignore it
		return
	}

	// Current endpoint failed, are all failed? We'd have to ignore
	// Do a fresh lookup first
	entries, err := addresspool.GeneratePool(m.netConfig.Servers, m.netConfig.Rfc2782Srv, m.netConfig.Rfc2782Service, time.Second*60)
	if err != nil {
		log.Warning("[P Failover] Server lookup failure: %s", err)
	}
	if m.sink.Count() == len(entries) {
		log.Warning("[P Failover] All endpoints have failed, awaiting recovery")
		return
	}

	// Add on extra endpoints
	m.failoverPosition++
	newServer := entries[m.failoverPosition]
	log.Warning("[P Failover] Initiating failover to: %s", newServer)

	// Check it's not already there (it may be still shutting down from a previous
	// recovery)
	if endpoint := m.sink.FindEndpoint(newServer); endpoint != nil {
		// Mark as the current and onFinish will trigger a respawn of it
		// Just ensure the position as any reload would not have since it was
		// shutting down
		m.sink.MoveEndpointAfter(endpoint, m.currentEndpoint)
		m.currentEndpoint = endpoint
	} else {
		m.currentEndpoint = m.sink.AddEndpoint(newServer)
	}
}

func (m *methodFailover) onFinish(endpoint *endpoint.Endpoint) bool {
	// If the current endpoint, or one better, keep it
	for previous := m.currentEndpoint; previous != nil; previous = previous.Prev() {
		if previous == endpoint {
			return true
		}
	}

	// The endpoint is not the current or any better, so let it close
	return false
}

func (m *methodFailover) onStarted(endpoint *endpoint.Endpoint) {
	// Is this the current endpoint? Nothing to do if it is
	if endpoint == m.currentEndpoint {
		return
	}

	// Is the current endpoint better?
	for previous := endpoint.Prev(); previous != nil; previous = previous.Prev() {
		if previous == m.currentEndpoint {
			return
		}
	}

	// This is the best endpoint, use it, close all later endpoints
	m.currentEndpoint = endpoint
	log.Info("[P Failover] A higher priority endpoint has recovered: %s", endpoint.Server())

	for next := endpoint.Next(); next != nil; next = next.Next() {
		m.sink.ShutdownEndpoint(next)
		m.failoverPosition--
	}
}

func (m *methodFailover) reloadConfig(netConfig *transports.Config) {
	m.netConfig = netConfig

	// Verify server ordering and if any better current server now available
	// We also use reloadConfig on first load of this method to cleanup what any
	// other method may have left behind
	entries, err := addresspool.GeneratePool(m.netConfig.Servers, m.netConfig.Rfc2782Srv, m.netConfig.Rfc2782Service, time.Second*60)
	if err != nil {
		log.Warning("[P Failover] Server lookup failure: %s", err)
	}

	var last, foundEndpoint *endpoint.Endpoint
	foundCurrent := false
	for _, poolEntry := range entries {
		if m.currentEndpoint != nil && m.currentEndpoint.PoolEntry().Desc == poolEntry.Desc {
			foundCurrent = true
		}

		if foundEndpoint = m.sink.FindEndpoint(poolEntry); foundEndpoint == nil {
			// If we've not found the current endpoint yet we should add this new
			// endpoint as it's better than the current!
			if foundCurrent {
				continue
			}

			last = m.sink.AddEndpointAfter(poolEntry, last)

			// If there was no current, we're initialising, use this one
			if m.currentEndpoint == nil {
				log.Info("[P Failover] Initialised priority endpoint: %s", last.Server())
				m.currentEndpoint = last
				foundCurrent = true
			}
			continue
		} else if m.currentEndpoint != nil && m.currentEndpoint == foundEndpoint {
			foundCurrent = true
		} else if foundCurrent {
			// Anything after the current is guaranteed to be shutting down so we
			// ignore them, but in case we took over from another method, call its
			// shutdown so we can be sure
			m.sink.ShutdownEndpoint(foundEndpoint)
			continue
		}

		// Ensure ordering and reload the configuration
		m.sink.MoveEndpointAfter(foundEndpoint, last)
		foundEndpoint.ReloadConfig(netConfig)
		last = foundEndpoint
	}
}

func (m *methodFailover) destroy() {
}
