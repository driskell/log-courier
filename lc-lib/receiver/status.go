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
	"sort"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// poolEventPosition is the position of an event within the payload it was received in
type poolEventPosition struct {
	progress *poolEventProgress
	sequence uint32
}

// poolAckRange is an inclusive range of acknowledged event sequences
type poolAckRange struct {
	from, to uint32
}

// poolEventProgress tracks the acknowledgement of a received payload, whose events may be acknowledged in any order
type poolEventProgress struct {
	event transports.EventsEvent
	// ackedThrough is the end of the contiguous acknowledged prefix, which is the sequence the protocol reports
	ackedThrough uint32
	// pending holds sorted, non-overlapping ranges acknowledged beyond a gap after ackedThrough
	pending []poolAckRange
}

// acknowledge records events from through to as acknowledged, returning true if ackedThrough advanced
func (p *poolEventProgress) acknowledge(from, to uint32) bool {
	if to <= p.ackedThrough {
		return false
	}
	if from <= p.ackedThrough {
		from = p.ackedThrough + 1
	}

	p.pending = append(p.pending, poolAckRange{from: from, to: to})
	sort.Slice(p.pending, func(i, j int) bool { return p.pending[i].from < p.pending[j].from })
	merged := p.pending[:1]
	for _, entry := range p.pending[1:] {
		last := &merged[len(merged)-1]
		if entry.from <= last.to+1 {
			last.to = max(last.to, entry.to)
		} else {
			merged = append(merged, entry)
		}
	}
	p.pending = merged

	if p.pending[0].from != p.ackedThrough+1 {
		return false
	}
	p.ackedThrough = p.pending[0].to
	p.pending = p.pending[1:]
	return true
}

// complete returns true once every event in the payload is acknowledged
func (p *poolEventProgress) complete() bool {
	return p.ackedThrough == p.event.Count()
}

type poolConnectionStatus struct {
	p                *Pool
	name             string
	listener         string
	remote           string
	desc             string
	metadataReceiver MetadataReceiver
	// progress holds payloads pending acknowledgement in the order they were received
	progress []*poolEventProgress
	// closing is set once the connection has finished sending, so it closes when progress empties
	closing     bool
	lines       int64
	bytes       int
	supportsAck bool

	api.KeyValue
}

func newPoolConnectionStatus(p *Pool, name string, listener string, remote string, desc string, supportsAck bool) *poolConnectionStatus {
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
		progress:    make([]*poolEventProgress, 0),
		supportsAck: supportsAck,
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
	if p.supportsAck {
		p.SetEntry("pendingPayloads", api.Number(len(p.progress)))
	}
	return nil
}

type poolReceiverStatus struct {
	config *transports.ReceiverConfigEntry
	listen string
	active bool
}
