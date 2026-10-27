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
	"context"
	"sort"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/core"
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
	ctx   context.Context
	nonce *string
	// count is the number of the payload's events received so far, and final is set once all have been
	count uint32
	final bool
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

// complete returns true once every event in the payload is received and acknowledged
func (p *poolEventProgress) complete() bool {
	return p.final && p.ackedThrough == p.count
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
	// window is the connection's receive window, known once it first asks for its events to be taken
	window *transports.ReceiveWindow
	// closed is set once the connection has gone, with failure set to why if it did not close cleanly
	closed  bool
	failure error
	// closing is set once the connection has finished sending, so it closes when progress empties
	closing bool
	lines   int64
	// lineSpeed measures how fast lines are completed
	lineSpeed   core.SpeedMeter
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

// receiving returns the payload still being received, or nil if every payload has been received in full
func (p *poolConnectionStatus) receiving() *poolEventProgress {
	if len(p.progress) == 0 || p.progress[len(p.progress)-1].final {
		return nil
	}
	return p.progress[len(p.progress)-1]
}

func (p *poolConnectionStatus) Update() error {
	p.p.connectionLock.RLock()
	defer p.p.connectionLock.RUnlock()

	p.SetEntry("remote", api.String(p.remote))
	p.SetEntry("listener", api.String(p.listener))
	p.SetEntry("description", api.String(p.desc))
	p.SetEntry("completedLines", api.Number(p.lines))
	p.SetEntry("speed", api.Float(p.lineSpeed.Speed()))
	p.SetEntry("receivedBytes", api.Bytes(p.bytes))
	if p.supportsAck {
		p.SetEntry("pendingPayloads", api.Number(len(p.progress)))
	}
	if p.window != nil {
		p.SetEntry("windowBytes", api.Bytes(p.window.Held()))
		p.SetEntry("windowSize", api.Bytes(p.window.Size()))
	}
	switch {
	case p.failure != nil:
		p.SetEntry("status", api.String("failed"))
		p.SetEntry("error", api.String(p.failure.Error()))
	case p.closed:
		p.SetEntry("status", api.String("closed"))
	case p.closing:
		p.SetEntry("status", api.String("closing"))
	default:
		p.SetEntry("status", api.String("active"))
	}
	return nil
}

type poolReceiverStatus struct {
	config *transports.ReceiverConfigEntry
	listen string
	active bool
}
