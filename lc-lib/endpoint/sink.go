/*
 * Copyright 2014-2015 Jason Woods.
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

package endpoint

import (
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	mutex sync.RWMutex

	endpoints    map[string]*Endpoint
	config       *transports.Config
	eventChan    chan transports.Event
	timeoutTimer *time.Timer

	api *api.Array

	timeoutList internallist.List
	readyList   internallist.List
	failedList  internallist.List
	orderedList internallist.List
}

// NewSink initialises a new message sink for endpoints
func NewSink(config *transports.Config) *Sink {
	// TODO: Make channel sizes configurable?
	ret := &Sink{
		endpoints:    make(map[string]*Endpoint),
		config:       config,
		eventChan:    make(chan transports.Event, 10),
		timeoutTimer: time.NewTimer(1 * time.Second),
	}

	ret.timeoutTimer.Stop()

	return ret
}

// ReloadConfig loads in a new configuration, endpoints will be shutdown if they
// are no longer in the configuration
func (s *Sink) ReloadConfig(config *transports.Config) {
EndpointLoop:
	for endpoint := s.Front(); endpoint != nil; endpoint = endpoint.Next() {
		var server string
		for _, server = range config.Servers {
			if server == endpoint.Server() {
				continue EndpointLoop
			}
		}

		// Not present in server list anymore, shut down
		s.ShutdownEndpoint(server)
	}
}

// Shutdown signals all associated endpoints to begin shutting down
func (s *Sink) Shutdown() {
	for server := range s.endpoints {
		s.ShutdownEndpoint(server)
	}
}

// APINavigatable returns an APINavigatable that exposes status information for this sink
// It should be called BEFORE adding any endpoints as existing endpoints will
// not automatically become monitored
func (s *Sink) APINavigatable() api.Navigatable {
	if s.api == nil {
		s.api = &api.Array{}
	}

	return s.api
}
