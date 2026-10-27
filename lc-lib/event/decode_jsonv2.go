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
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"sync"
	"time"
)

// decodeTopLevelSize is the room made for an event's top-level fields, including the @metadata added after decoding
const decodeTopLevelSize = 16

// eventDecoder is a decoder reused to decode events, reading with the options encoding/json decodes with
type eventDecoder struct {
	reader  bytes.Reader
	decoder *jsontext.Decoder
}

// eventDecoders holds decoders for reuse
var eventDecoders = sync.Pool{
	New: func() interface{} {
		return &eventDecoder{}
	},
}

// decodeData decodes event data, which must be a JSON object, building its values directly rather than through
// reflection, with tags and a valid @timestamp decoded straight to the types events hold them as
func decodeData(encoded []byte) (map[string]interface{}, error) {
	state := eventDecoders.Get().(*eventDecoder)
	defer eventDecoders.Put(state)
	state.reader.Reset(encoded)
	if state.decoder == nil {
		state.decoder = jsontext.NewDecoder(&state.reader, json.DefaultOptionsV1())
	} else {
		state.decoder.Reset(&state.reader, json.DefaultOptionsV1())
	}
	decoder := state.decoder

	if kind := decoder.PeekKind(); kind != '{' {
		if _, err := decoder.ReadToken(); err != nil {
			return nil, err
		}
		return nil, errNotObject
	}
	data, err := decodeObject(decoder, true)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("invalid data after top-level value")
		}
		return nil, err
	}
	return data, nil
}

// decodeObject decodes an object, decoding the top-level tags and @timestamp of an event to the types events hold
func decodeObject(decoder *jsontext.Decoder, topLevel bool) (map[string]interface{}, error) {
	if _, err := decoder.ReadToken(); err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if topLevel {
		data = make(map[string]interface{}, decodeTopLevelSize)
	} else {
		data = make(map[string]interface{})
	}
	for decoder.PeekKind() != '}' {
		name, err := decoder.ReadToken()
		if err != nil {
			return nil, err
		}
		key := name.String()
		var value interface{}
		switch {
		case topLevel && key == TagsKey && decoder.PeekKind() == '[':
			value, err = decodeTags(decoder)
		case topLevel && key == TimestampKey && decoder.PeekKind() == '"':
			value, err = decodeTimestamp(decoder)
		default:
			value, err = decodeValue(decoder)
		}
		if err != nil {
			return nil, err
		}
		data[key] = value
	}
	if _, err := decoder.ReadToken(); err != nil {
		return nil, err
	}
	return data, nil
}

// decodeArray decodes the rest of an array into items, after its opening bracket was read
func decodeArray(decoder *jsontext.Decoder, items []interface{}) ([]interface{}, error) {
	for decoder.PeekKind() != ']' {
		value, err := decodeValue(decoder)
		if err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	if _, err := decoder.ReadToken(); err != nil {
		return nil, err
	}
	return items, nil
}

// decodeTags decodes an event's tags, falling back to a plain array if any is not a string so it fails normalisation
// as it would otherwise
func decodeTags(decoder *jsontext.Decoder) (interface{}, error) {
	if _, err := decoder.ReadToken(); err != nil {
		return nil, err
	}
	tags := make(Tags, 0, 4)
	for decoder.PeekKind() != ']' {
		if decoder.PeekKind() != '"' {
			items := make([]interface{}, len(tags))
			for idx, tag := range tags {
				items[idx] = tag
			}
			return decodeArray(decoder, items)
		}
		tag, err := decoder.ReadToken()
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag.String())
	}
	if _, err := decoder.ReadToken(); err != nil {
		return nil, err
	}
	return tags, nil
}

// decodeTimestamp decodes an event's @timestamp, leaving it as a string if invalid so it fails normalisation as it
// would otherwise
func decodeTimestamp(decoder *jsontext.Decoder) (interface{}, error) {
	token, err := decoder.ReadToken()
	if err != nil {
		return nil, err
	}
	value := token.String()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value, nil
	}
	return Timestamp(parsed), nil
}

// decodeValue decodes any JSON value to the types encoding/json decodes them to
func decodeValue(decoder *jsontext.Decoder) (interface{}, error) {
	switch decoder.PeekKind() {
	case '{':
		return decodeObject(decoder, false)
	case '[':
		if _, err := decoder.ReadToken(); err != nil {
			return nil, err
		}
		return decodeArray(decoder, []interface{}{})
	}
	token, err := decoder.ReadToken()
	if err != nil {
		return nil, err
	}
	switch token.Kind() {
	case 'n':
		return nil, nil
	case 't', 'f':
		return token.Bool(), nil
	case '"':
		return token.String(), nil
	default:
		return token.Float()
	}
}
