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

package receiver

import (
	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/transports"
)

type poolEventPosition struct {
	nonce    *string
	sequence uint32
}

type poolEventProgress struct {
	event    transports.EventsEvent
	sequence uint32
}

type poolConnectionStatus struct {
	p                *Pool
	name             string
	listener         string
	remote           string
	desc             string
	metadataReceiver MetadataReceiver
	progress         []*poolEventProgress
	lines            int64
	bytes            int

	api.KeyValue
}

func newPoolConnectionStatus(p *Pool, name string, listener string, remote string, desc string) *poolConnectionStatus {
	return &poolConnectionStatus{
		p:        p,
		name:     name,
		listener: listener,
		remote:   remote,
		desc:     desc,
		metadataReceiver: MetadataReceiver{
			"name":     name,
			"listener": listener,
			"remote":   remote,
			"desc":     desc,
		},
		progress: make([]*poolEventProgress, 0),
	}
}

func (p *poolConnectionStatus) Update() error {
	p.p.connectionLock.RLock()
	defer p.p.connectionLock.RUnlock()

	p.SetEntry("remote", api.String(p.remote))
	p.SetEntry("listener", api.String(p.listener))
	p.SetEntry("description", api.String(p.desc))
	p.SetEntry("completedLines", api.Number(p.lines))
	p.SetEntry("receivedBytes", api.Bytes(p.bytes))
	p.SetEntry("pendingPayloads", api.Number(len(p.progress)))
	return nil
}

type poolReceiverStatus struct {
	config *transports.ReceiverConfigEntry
	listen string
	active bool
}
