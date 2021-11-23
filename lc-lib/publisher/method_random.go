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
	"math/rand"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/publisher/endpoint"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type methodRandom struct {
	sink         *endpoint.Sink
	netConfig    *transports.Config
	activeServer int
	generator    *rand.Rand
	backoff      *core.ExpBackoff
}

func newMethodRandom(sink *endpoint.Sink, netConfig *transports.Config) *methodRandom {
	ret := &methodRandom{
		sink:         sink,
		netConfig:    netConfig,
		activeServer: -1,
		generator:    rand.New(rand.NewSource(int64(time.Now().Nanosecond()))),
	}

	ret.backoff = core.NewExpBackoff("Random", ret.netConfig.Backoff, ret.netConfig.BackoffMax)

	if sink.Count() == 0 {
		// Empty sink, connect to a random endpoint
		ret.connectRandom()
		return ret
	}

	// We have existing endpoints, locate a suitable one to keep and discard any
	// others
	foundAcceptable := false
	for endpoint := sink.Front(); endpoint != nil; endpoint = endpoint.Next() {
		if endpoint.IsClosing() {
			// It's closing, we can ignore
			continue
		} else if endpoint.IsFailed() || foundAcceptable {
			// Failed endpoint or we've already found an acceptable one, get rid of
			// it
			sink.ShutdownEndpoint(endpoint.Server())
			continue
		}

		// Suitable endpoint, update activeServer
		for k, server := range ret.netConfig.Servers {
			if server == endpoint.Server() {
				ret.activeServer = k
				foundAcceptable = true

				log.Debug("[Random] Utilising existing endpoint connection: %s", server)

				// Reload it
				endpoint.ReloadConfig(netConfig)
				break
			}
		}

		// This should never happen, sink is reloaded before method is and sink
		// should have removed endpoints that don't exist in the configuration, or
		// at the very least placed them into a closing status
		if !foundAcceptable {
			log.Warning("[Random] Method reload discovered inconsistent Endpoint status: %s", endpoint.Server())
			sink.ShutdownEndpoint(endpoint.Server())
		}
	}

	return ret
}

func (m *methodRandom) connectRandom() {
	var server string
	var addressPool *addresspool.Pool
	if len(m.netConfig.Servers) == 1 {
		// Only one entry
		server = m.netConfig.Servers[0]
		addressPool = m.netConfig.AddressPools[0]
		m.activeServer = 0
	} else if len(m.netConfig.Servers) == 2 && m.activeServer != -1 {
		// Alternate between two endpoints
		m.activeServer = (m.activeServer + 1) % 2
		server = m.netConfig.Servers[m.activeServer]
		addressPool = m.netConfig.AddressPools[m.activeServer]
	} else {
		// Random selection that avoids the same endpoint twice in a row
		for {
			selected := m.generator.Intn(len(m.netConfig.Servers))
			if selected == m.activeServer {
				// Same server, try again
				continue
			}

			m.activeServer = selected
			server = m.netConfig.Servers[selected]
			addressPool = m.netConfig.AddressPools[selected]
			break
		}
	}

	log.Info("[Random] Randomly selected new endpoint: %s", server)

	m.sink.AddEndpoint(server, addressPool)
}

func (m *methodRandom) onFail(endpoint *endpoint.Endpoint) {
	// Failed endpoint - keep it alive until backoff triggers new connection
	// This way we still have an endpoint with a last error in monitor
	m.sink.Scheduler.SetCallback(m, m.backoff.Trigger(), func() {
		log.Warning("[Random] Giving up on failed endpoint: %s", endpoint.Server())
		m.sink.ShutdownEndpoint(endpoint.Server())
	})
}

func (m *methodRandom) onFinish(endpoint *endpoint.Endpoint) bool {
	// If shutdown, leave it out, but start another
	m.connectRandom()
	return false
}

func (m *methodRandom) onStarted(endpoint *endpoint.Endpoint) {
	// If we just recovered from failure before the shutdown occurred, prevent random timeout occuring
	m.sink.Scheduler.Remove(m)
}

func (m *methodRandom) reloadConfig(netConfig *transports.Config) {
	currentServer := m.netConfig.Servers[m.activeServer]
	m.netConfig = netConfig

	front := m.sink.Front()
	if front == nil {
		// No endpoints - skip reloading current endpoint
		return
	}

	// Find and reload the current endpoint
	for _, server := range m.netConfig.Servers {
		if server == currentServer {
			// Still present, all good, pass through the reload
			front.ReloadConfig(netConfig)
			return
		}
	}
}

func (m *methodRandom) destroy() {
	m.sink.Scheduler.Remove(m)
}
