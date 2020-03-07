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
	"math/rand"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/endpoint"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type methodRandom struct {
	sink         *endpoint.Sink
	netConfig    *transports.Config
	activeServer int
	generator    *rand.Rand
	backoff      *core.ExpBackoff

	endpoint.Timeout
}

func newMethodRandom(sink *endpoint.Sink, netConfig *transports.Config) *methodRandom {
	ret := &methodRandom{
		sink:         sink,
		netConfig:    netConfig,
		activeServer: -1,
		generator:    rand.New(rand.NewSource(int64(time.Now().Nanosecond()))),
	}

	ret.backoff = core.NewExpBackoff("Random", ret.netConfig.Backoff, ret.netConfig.BackoffMax)

	ret.InitTimeout()

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
				endpoint.ReloadConfig(netConfig, true)
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
		m.activeServer = (m.activeServer + 1) % 2
		server = m.netConfig.Servers[m.activeServer]
		addressPool = m.netConfig.AddressPools[m.activeServer]
	} else {
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

	log.Debug("[Random] Randomly selected new endpoint: %s", server)

	m.sink.AddEndpoint(server, addressPool, true)
}

func (m *methodRandom) onFail(endpoint *endpoint.Endpoint) {
	// Should never happen - we initiate transports with finishOnFail
	return
}

func (m *methodRandom) onFinish(endpoint *endpoint.Endpoint) bool {
	// Due to finishOnFail we have no backoff after failure, so start one now to
	// call connectRandom after the backoff
	m.sink.RegisterTimeout(
		&m.Timeout,
		m.backoff.Trigger(),
		func() {
			m.connectRandom()
		},
	)
	return false
}

func (m *methodRandom) onStarted(endpoint *endpoint.Endpoint) {
	return
}

func (m *methodRandom) reloadConfig(netConfig *transports.Config) {
	currentServer := m.netConfig.Servers[m.activeServer]
	m.netConfig = netConfig

	front := m.sink.Front()
	if front == nil {
		// No endpoints - skip reloading current endpoint
		return
	}

	// If the current active endpoint is no longer present, shut it down
	// onFinish will trigger a new endpoint connection
	for _, server := range m.netConfig.Servers {
		if server == currentServer {
			// Still present, all good, pass through the reload
			front.ReloadConfig(netConfig, true)
			return
		}
	}

	// Not present in server list, shut down
	m.sink.ShutdownEndpoint(currentServer)
}
