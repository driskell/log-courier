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

package courier

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type protocolACKN struct {
	ctx      context.Context
	nonce    *string
	sequence uint32
}

var _ transports.AckEvent = (*protocolACKN)(nil)

// newProtocolACKN reads a new protocolACKN
func newProtocolACKN(conn tcp.Connection, bodyLength uint32) (tcp.ProtocolMessage, error) {
	if bodyLength != 20 {
		return nil, fmt.Errorf("protocol error: Corrupt message (ACKN size %d != 20)", bodyLength)
	}

	message := make([]byte, 20)
	if _, err := conn.Read(message); err != nil {
		return nil, err
	}

	nonce := string(message[:16])
	sequence := binary.BigEndian.Uint32(message[16:])
	return &protocolACKN{ctx: conn.Context(), nonce: &nonce, sequence: sequence}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolACKN) Type() string {
	return "ACKN"
}

// Context returns the connection context
func (p *protocolACKN) Context() context.Context {
	return p.ctx
}

// Nonce returns the nonce being acknowledged
func (p *protocolACKN) Nonce() *string {
	return p.nonce
}

// Sequence returns the sequence being acknowledged
func (p *protocolACKN) Sequence() uint32 {
	return p.sequence
}

// Write writes a payload to the connection
func (p *protocolACKN) Write(conn tcp.Connection) error {
	// Encapsulate the ack into a message
	// 4-byte message header (ACKN)
	// 4-byte message length
	// 16-byte nonce
	// 4-byte uint32 sequence
	if _, err := conn.Write([]byte{'A', 'C', 'K', 'N', 0, 0, 0, 20}); err != nil {
		return err
	}

	if _, err := conn.Write([]byte(*p.nonce)); err != nil {
		return err
	}

	var sequence [4]byte
	binary.BigEndian.PutUint32(sequence[:], uint32(p.sequence))
	if _, err := conn.Write(sequence[:]); err != nil {
		return err
	}
	return conn.Flush()
}
