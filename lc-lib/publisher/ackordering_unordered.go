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
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
)

// unorderedAcks dispatches each payload's acknowledgements as soon as they arrive, for acknowledgers such as
// the receiver pool that track each payload independently, so a slow payload never holds back the others
type unorderedAcks struct{}

// UnorderedAcks creates an AckOrdering that dispatches each payload's acknowledgements as soon as they arrive
func UnorderedAcks() AckOrdering {
	return &unorderedAcks{}
}

func (u *unorderedAcks) add(*payload.Payload) {}

func (u *unorderedAcks) acknowledged(pendingPayload *payload.Payload, firstAck bool) int64 {
	if !pendingPayload.HasAck() {
		return 0
	}
	event.DispatchAck(pendingPayload.Rollup())
	if pendingPayload.Complete() {
		return 1
	}
	return 0
}
