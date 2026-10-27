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
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
)

// eventData encodes event data without its @metadata, writing the value types events hold directly rather than
// through reflection, with @timestamp first so it is found quickly without decoding, then the keys sorted as
// encoding/json sorts them
type eventData map[string]interface{}

// encodeKeys holds key slices reused to sort the keys of each map encoded
var encodeKeys = sync.Pool{
	New: func() interface{} {
		return new([]string)
	},
}

// encodeData encodes event data without its @metadata
func encodeData(data map[string]interface{}) ([]byte, error) {
	return json.Marshal(eventData(data))
}

// MarshalJSONTo encodes the event data, leaving out @metadata
func (d eventData) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encodeObject(encoder, d, true)
}

// encodeObject encodes a map, leaving out @metadata and writing @timestamp first when it is the top level of an event
func encodeObject(encoder *jsontext.Encoder, data map[string]interface{}, topLevel bool) error {
	keys := encodeKeys.Get().(*[]string)
	defer func() {
		clear(*keys)
		*keys = (*keys)[:0]
		encodeKeys.Put(keys)
	}()
	for key := range data {
		if topLevel && (key == MetadataKey || key == TimestampKey) {
			continue
		}
		*keys = append(*keys, key)
	}
	sort.Strings(*keys)

	if err := encoder.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if timestamp, ok := data[TimestampKey]; topLevel && ok {
		if err := encoder.WriteToken(jsontext.String(TimestampKey)); err != nil {
			return err
		}
		if err := encodeValue(encoder, timestamp); err != nil {
			return err
		}
	}
	for _, key := range *keys {
		if err := encoder.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		if err := encodeValue(encoder, data[key]); err != nil {
			return err
		}
	}
	return encoder.WriteToken(jsontext.EndObject)
}

// encodeValue encodes the value types events hold directly, and any other through the encoder's own marshaling
func encodeValue(encoder *jsontext.Encoder, value interface{}) error {
	switch typed := value.(type) {
	case string:
		return encoder.WriteToken(jsontext.String(typed))
	case float64:
		// The encoder writes these as strings, but they are not valid JSON numbers so encoding must fail
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return fmt.Errorf("unsupported value: %v", typed)
		}
		return encoder.WriteToken(jsontext.Float(typed))
	case bool:
		return encoder.WriteToken(jsontext.Bool(typed))
	case nil:
		return encoder.WriteToken(jsontext.Null)
	case map[string]interface{}:
		if typed == nil {
			return encoder.WriteToken(jsontext.Null)
		}
		return encodeObject(encoder, typed, false)
	case []interface{}:
		if typed == nil {
			return encoder.WriteToken(jsontext.Null)
		}
		if err := encoder.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, item := range typed {
			if err := encodeValue(encoder, item); err != nil {
				return err
			}
		}
		return encoder.WriteToken(jsontext.EndArray)
	case Tags:
		if typed == nil {
			return encoder.WriteToken(jsontext.Null)
		}
		if err := encoder.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, tag := range typed {
			if err := encoder.WriteToken(jsontext.String(tag)); err != nil {
				return err
			}
		}
		return encoder.WriteToken(jsontext.EndArray)
	case Timestamp:
		if err := typed.MarshalJSONTo(encoder); !errors.Is(err, errors.ErrUnsupported) {
			return err
		}
	case FloatValue64:
		return typed.MarshalJSONTo(encoder)
	case FloatValue32:
		return typed.MarshalJSONTo(encoder)
	}
	return jsonv2.MarshalEncode(encoder, value)
}
