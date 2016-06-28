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

package tcp

import (
	"compress/zlib"
	"encoding/binary"

	"github.com/driskell/log-courier/lc-lib/payload"
)

type cachedJDAT []byte

type protocolJDAT struct {
	payload *payload.Payload
}

// writeEvents writes a payload to the socket
func (p *protocolJDAT) write(t *TransportTCP) error {
	// Cached?
	if cached, ok := p.payload.Cache.(cachedJDAT); ok && cached != nil {
		_, err := t.socket.Write(cached)
		return err
	}

	t.buffer.Reset()

	// Encapsulate the data into the message
	// 4-byte message header (JDAT = JSON Data, Compressed)
	// 4-byte uint32 data length
	// Then the data
	if _, err := t.buffer.Write([]byte("JDAT")); err != nil {
		return err
	}

	// False length as we don't know it yet
	if _, err := t.buffer.Write([]byte("----")); err != nil {
		return err
	}

	// Create the compressed data payload
	// 16-byte Nonce, followed by the compressed event data
	// The event data is each event, prefixed with a 4-byte uint32 length, one
	// after the other
	if _, err := t.buffer.Write([]byte(p.payload.Nonce)); err != nil {
		return err
	}

	if t.compressor == nil {
		var err error
		t.compressor, err = zlib.NewWriterLevel(&t.buffer, 3)
		if err != nil {
			return err
		}
	} else {
		t.compressor.Reset(&t.buffer)
	}

	for _, singleEvent := range p.payload.Events() {
		if err := binary.Write(t.compressor, binary.BigEndian, uint32(len(singleEvent.Bytes()))); err != nil {
			return err
		}

		if _, err := t.compressor.Write(singleEvent.Bytes()); err != nil {
			return err
		}
	}

	if err := t.compressor.Close(); err != nil {
		return err
	}

	// Fill in the size
	// TODO: This prevents us bypassing buffer and just sending...
	//       New JDA2? With FFFF size? Means stream message?
	messageBytes := t.buffer.Bytes()
	binary.BigEndian.PutUint32(messageBytes[4:8], uint32(t.buffer.Len()-8))

	// Cache the payload data
	p.payload.Cache = messageBytes

	_, err := t.socket.Write(messageBytes)
	return err
}
