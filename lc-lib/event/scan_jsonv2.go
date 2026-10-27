//go:build go1.27 && goexperiment.jsonv2

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

package event

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"sync"
	"time"
)

// eventScanner is a decoder reused to scan encoded events strictly, rejecting duplicate keys and invalid UTF-8 that
// decoding would otherwise resolve
type eventScanner struct {
	reader  bytes.Reader
	decoder *jsontext.Decoder
}

// eventScanners holds scanners for reuse
var eventScanners = sync.Pool{
	New: func() interface{} {
		return &eventScanner{}
	},
}

// scan prepares a scanner for the encoded event, which must be returned to eventScanners once done
func scan(encoded []byte) *eventScanner {
	scanner := eventScanners.Get().(*eventScanner)
	scanner.reader.Reset(encoded)
	if scanner.decoder == nil {
		scanner.decoder = jsontext.NewDecoder(&scanner.reader)
	} else {
		scanner.decoder.Reset(&scanner.reader)
	}
	return scanner
}

// scanTimestamp reads a valid top-level @timestamp from the encoded event, stopping as soon as it is found, which
// encoding places first; with a duplicated @timestamp it is the first that is read, though decoding keeps the last
func scanTimestamp(encoded []byte) (Timestamp, bool) {
	scanner := scan(encoded)
	defer eventScanners.Put(scanner)
	decoder := scanner.decoder
	if decoder.PeekKind() != '{' {
		return Timestamp{}, false
	}
	if _, err := decoder.ReadToken(); err != nil {
		return Timestamp{}, false
	}
	for decoder.PeekKind() == '"' {
		name, err := decoder.ReadToken()
		if err != nil {
			return Timestamp{}, false
		}
		if name.String() == TimestampKey {
			timestamp, ok, _ := readTimestamp(decoder)
			return timestamp, ok
		}
		if err := decoder.SkipValue(); err != nil {
			return Timestamp{}, false
		}
	}
	return Timestamp{}, false
}

// readTimestamp reads a @timestamp value, which is valid only as an RFC 3339 string, also returning whether it is
// written exactly as encoding writes it, as parsing accepts forms such as single digit hours
func readTimestamp(decoder *jsontext.Decoder) (Timestamp, bool, bool) {
	if decoder.PeekKind() != '"' {
		return Timestamp{}, false, false
	}
	token, err := decoder.ReadToken()
	if err != nil {
		return Timestamp{}, false, false
	}
	value := token.String()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return Timestamp{}, false, false
	}
	var buffer [64]byte
	return Timestamp(parsed), true, string(parsed.AppendFormat(buffer[:0], time.RFC3339Nano)) == value
}

// encodedNormalised returns true if the encoded event is a valid JSON object that decoding and encoding would leave
// as it is: a valid @timestamp, tags of strings, no @metadata, no duplicate keys and valid UTF-8; numbers are not
// checked against the float64 range decoding requires, so an out of range number passes through unchanged
func encodedNormalised(encoded []byte) bool {
	scanner := scan(encoded)
	defer eventScanners.Put(scanner)
	decoder := scanner.decoder
	if decoder.PeekKind() != '{' {
		return false
	}
	if _, err := decoder.ReadToken(); err != nil {
		return false
	}
	hasTimestamp, hasTags := false, false
	for decoder.PeekKind() == '"' {
		name, err := decoder.ReadToken()
		if err != nil {
			return false
		}
		switch name.String() {
		case MetadataKey:
			return false
		case TimestampKey:
			if _, ok, canonical := readTimestamp(decoder); !ok || !canonical {
				return false
			}
			hasTimestamp = true
		case TagsKey:
			if !skipTags(decoder) {
				return false
			}
			hasTags = true
		default:
			if err := decoder.SkipValue(); err != nil {
				return false
			}
		}
	}
	if _, err := decoder.ReadToken(); err != nil {
		return false
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		return false
	}
	return hasTimestamp && hasTags
}

// skipTags reads a tags value, returning true if it is an array of strings
func skipTags(decoder *jsontext.Decoder) bool {
	if decoder.PeekKind() != '[' {
		return false
	}
	if _, err := decoder.ReadToken(); err != nil {
		return false
	}
	for decoder.PeekKind() == '"' {
		if _, err := decoder.ReadToken(); err != nil {
			return false
		}
	}
	if decoder.PeekKind() != ']' {
		return false
	}
	_, err := decoder.ReadToken()
	return err == nil
}
