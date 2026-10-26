/*
 * Copyright 2026 Jason Woods and contributors
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
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/internallist"
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
)

// orderedAcks dispatches acknowledgements in the order payloads were sent, for acknowledgers such as the
// registrar that record a position and so must never see an earlier event acknowledged after a later one
type orderedAcks struct {
	payloadList internallist.List
	outOfSync   int
}

// OrderedAcks creates an AckOrdering that dispatches acknowledgements in the order payloads were sent
func OrderedAcks() AckOrdering {
	return &orderedAcks{}
}

func (o *orderedAcks) add(pendingPayload *payload.Payload) {
	o.payloadList.PushBack(&pendingPayload.Element)
}

func (o *orderedAcks) acknowledged(pendingPayload *payload.Payload, firstAck bool) int64 {
	numComplete := int64(0)

	// We potentially receive out-of-order ACKs due to payloads distributed across servers
	// This is where we enforce ordering again to ensure the handlers receive ACKs in order
	if pendingPayload == o.payloadList.Front().Value.(*payload.Payload) {
		// The out of sync count we have will never include the first payload, so
		// take the value +1
		outOfSync := o.outOfSync + 1

		// For each full payload we mark off, we decrease this count, the first we
		// mark off will always be the first payload - thus the +1. Subsequent
		// payloads are the out of sync ones - so if we mark them off we decrease
		// the out of sync count
		for pendingPayload.HasAck() {
			event.DispatchAck(pendingPayload.Rollup())

			if !pendingPayload.Complete() {
				break
			}

			o.payloadList.Remove(&pendingPayload.Element)
			outOfSync--
			o.outOfSync = outOfSync

			numComplete++

			if o.payloadList.Len() == 0 {
				break
			}

			pendingPayload = o.payloadList.Front().Value.(*payload.Payload)
		}
	} else if firstAck {
		// If this is NOT the first payload, and this is the first acknowledgement
		// for this payload, then increase out of sync payload count
		o.outOfSync++
	}

	return numComplete
}
