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

package endpoint

import (
	"sync"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
	"github.com/driskell/log-courier/lc-lib/scheduler"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// Sink structure contains the control channels that each endpoint
// will utilise. The newEndpoint method attaches new endpoints to this
type Sink struct {
	mutex sync.RWMutex

	endpoints map[string]*Endpoint
	config    *transports.Config
	eventChan chan transports.Event

	api *api.Array

	readyList   internallist.List
	failedList  internallist.List
	orderedList internallist.List

	// Scheduler manages scheduled callbacks and values - OnNext should be used to get a channel and Next called repeatedly until it returns nil
	// Endpoint sink will only schedule callbacks that are handled silently by Next - but Scheduler can have custom callbacks or just returns scheduled
	Scheduler *scheduler.Scheduler
	// OnAck is called when an acknowledgement response is received
	// The payload is given and the second argument is true if this ack is the
	// first ack for this payload
	OnAck func(*Endpoint, *payload.Payload, bool, int)
	// OnFail is called when the endpoint fails
	OnFail func(*Endpoint)
	// OnFinished is called when an endpoint finishes and is removed
	// Returning false prevents the endpoint from being recreated, which it will
	// be if it still exists in the configuration
	OnFinish func(*Endpoint) bool
	// OnPong is called when a pong response is received from the endpoint
	OnPong func(*Endpoint)
	// OnStarted is called when an endpoint starts up and is ready
	OnStarted func(*Endpoint)
}

// NewSink initialises a new message sink for endpoints
func NewSink(config *transports.Config) *Sink {
	// TODO: Make channel sizes configurable?
	ret := &Sink{
		endpoints: make(map[string]*Endpoint),
		config:    config,
		eventChan: make(chan transports.Event, 10),

		Scheduler: scheduler.NewScheduler(),
		OnAck:     func(*Endpoint, *payload.Payload, bool, int) {},
		OnFail:    func(*Endpoint) {},
		OnFinish:  func(*Endpoint) bool { return false },
		OnPong:    func(*Endpoint) {},
		OnStarted: func(*Endpoint) {},
	}

	ret.readyList.Init()
	ret.failedList.Init()
	ret.orderedList.Init()

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
		s.ShutdownEndpoint(endpoint)
	}
}

// Shutdown signals all associated endpoints to begin shutting down
func (s *Sink) Shutdown() {
	for _, endpoint := range s.endpoints {
		s.ShutdownEndpoint(endpoint)
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
