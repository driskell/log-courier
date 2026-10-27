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
	"io"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// protocolJDATStream reads the events of a received JDAT message as they arrive
type protocolJDATStream struct {
	*eventsDecoder
	body *bodyReader
}

var _ tcp.StreamMessage = (*protocolJDATStream)(nil)

// newProtocolJDAT starts reading the events of a received JDAT message, whose body has the given length
func newProtocolJDAT(conn tcp.Connection, bodyLength uint32) (receivedMessage, error) {
	if bodyLength < 17 {
		return nil, fmt.Errorf("protocol error: Corrupt message (JDAT size %d < 17)", bodyLength)
	}
	body := newBodyReader(conn, int(bodyLength))
	decoder, err := newEventsDecoder(conn, body)
	if err != nil {
		return nil, err
	}
	return &protocolJDATStream{eventsDecoder: decoder, body: body}, nil
}

// Type returns a human-readable name for the message type
func (p *protocolJDATStream) Type() string {
	return "JDAT"
}

// Next returns the next event, discarding anything left in the body once the events end so the next message is read
// from where it starts
func (p *protocolJDATStream) Next(reserve func(int) error) ([]byte, map[string]interface{}, int, bool, error) {
	data, decoded, size, skipped, err := p.eventsDecoder.Next(reserve)
	if err == io.EOF {
		if _, discardErr := io.Copy(io.Discard, p.body); discardErr != nil {
			return nil, nil, 0, false, discardErr
		}
	}
	return data, decoded, size, skipped, err
}

// bodyReader reads no further than the end of a message body, reading single bytes so decompression stops exactly
// at the end of the compressed stream
type bodyReader struct {
	source    io.Reader
	remaining int
}

func newBodyReader(conn tcp.Connection, length int) *bodyReader {
	return &bodyReader{source: conn, remaining: length}
}

// Read reads from the body, returning io.EOF at its end
func (b *bodyReader) Read(data []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, io.EOF
	}
	if len(data) > b.remaining {
		data = data[:b.remaining]
	}
	n, err := b.source.Read(data)
	b.remaining -= n
	return n, err
}

// ReadByte reads one byte from the body, returning io.EOF at its end
func (b *bodyReader) ReadByte() (byte, error) {
	if b.remaining <= 0 {
		return 0, io.EOF
	}
	if byteSource, ok := b.source.(io.ByteReader); ok {
		value, err := byteSource.ReadByte()
		if err == nil {
			b.remaining--
		}
		return value, err
	}
	var value [1]byte
	if _, err := b.Read(value[:]); err != nil {
		return 0, err
	}
	return value[0], nil
}
