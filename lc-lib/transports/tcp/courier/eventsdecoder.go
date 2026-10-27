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
	"compress/zlib"
	"context"
	"encoding/binary"
	"io"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// eventsDecoder reads the size-prefixed events of a received events message from its compressed stream one at a time
type eventsDecoder struct {
	conn         tcp.Connection
	nonce        *string
	decompressor io.ReadCloser
	sizePrefix   [4]byte
	// skipped and largestSkipped describe the events too large to hold, reported once the message ends
	skipped        int
	largestSkipped int
}

// newEventsDecoder reads the nonce that starts an events message and prepares to read its events from source
func newEventsDecoder(conn tcp.Connection, source io.Reader) (*eventsDecoder, error) {
	var nonce [16]byte
	if _, err := io.ReadFull(source, nonce[:]); err != nil {
		return nil, err
	}

	decompressor, err := newDecompressor(source)
	if err != nil {
		return nil, err
	}

	nonceString := string(nonce[:])
	return &eventsDecoder{conn: conn, nonce: &nonceString, decompressor: decompressor}, nil
}

// Context returns the connection context
func (d *eventsDecoder) Context() context.Context {
	return d.conn.Context()
}

// Nonce returns the nonce of the payload
func (d *eventsDecoder) Nonce() *string {
	return d.nonce
}

// Next returns the next event, skipping without holding it any event larger than the receive window, or io.EOF once
// the compressed stream ends
func (d *eventsDecoder) Next(reserve func(int) error) ([]byte, map[string]interface{}, int, bool, error) {
	if d.decompressor == nil {
		return nil, nil, 0, false, io.EOF
	}

	if _, err := io.ReadFull(d.decompressor, d.sizePrefix[:]); err != nil {
		// The checksum trailing the stream is not verified, as the events before it have already been handed on
		if err == io.EOF || err == zlib.ErrChecksum {
			d.finish()
			return nil, nil, 0, false, io.EOF
		}
		return nil, nil, 0, false, d.fail(err)
	}
	eventSize := int(binary.BigEndian.Uint32(d.sizePrefix[:]))

	if eventSize > d.conn.ReceiveWindowSize() {
		d.skipped++
		d.largestSkipped = max(d.largestSkipped, eventSize)
		if _, err := io.CopyN(io.Discard, d.decompressor, int64(eventSize)); err != nil {
			return nil, nil, 0, false, d.fail(err)
		}
		return nil, nil, 0, true, nil
	}

	if err := reserve(eventSize); err != nil {
		return nil, nil, 0, false, d.fail(err)
	}
	data := make([]byte, eventSize)
	if _, err := io.ReadFull(d.decompressor, data); err != nil {
		return nil, nil, 0, false, d.fail(err)
	}
	return data, nil, eventSize, false, nil
}

// finish ends the message once its compressed stream ends, reporting any events skipped
func (d *eventsDecoder) finish() {
	d.release()
	if d.skipped != 0 {
		log.Warningf("[R %s < %s] Skipped %d events in payload %x as they exceeded the receive window size of %d bytes (largest %d bytes)", d.conn.LocalAddr().String(), d.conn.RemoteAddr().String(), d.skipped, *d.nonce, d.conn.ReceiveWindowSize(), d.largestSkipped)
	}
}

// fail ends the message with an error from reading it
func (d *eventsDecoder) fail(err error) error {
	d.release()
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return tcp.ErrUnexpectedEnd
	}
	return err
}

// release returns the decompressor for reuse once the message is finished with
func (d *eventsDecoder) release() {
	releaseDecompressor(d.decompressor)
	d.decompressor = nil
}
