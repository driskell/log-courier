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

package courier

import (
	"fmt"
	"math"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// protocolEVNTStream reads the events of a received EVNT message as they arrive
type protocolEVNTStream struct {
	*eventsDecoder
}

var _ tcp.StreamMessage = (*protocolEVNTStream)(nil)

// newProtocolEVNT starts reading the events of a received EVNT message, whose compressed stream ends itself
func newProtocolEVNT(conn tcp.Connection, bodyLength uint32) (receivedMessage, error) {
	if bodyLength != math.MaxUint32 {
		return nil, fmt.Errorf("protocol error: Corrupt message (EVNT size %d != %d)", bodyLength, uint32(math.MaxUint32))
	}
	decoder, err := newEventsDecoder(conn, conn)
	if err != nil {
		return nil, err
	}
	return &protocolEVNTStream{eventsDecoder: decoder}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolEVNTStream) Type() string {
	return "EVNT"
}
