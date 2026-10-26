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
	"github.com/driskell/log-courier/lc-lib/publisher/payload"
)

// AckOrdering decides when acknowledged events are dispatched to their acknowledgers
type AckOrdering interface {
	// add tracks a payload as it is sent
	add(pendingPayload *payload.Payload)
	// acknowledged dispatches what it can after a payload is acknowledged, returning how many payloads it completed
	acknowledged(pendingPayload *payload.Payload, firstAck bool) int64
}
