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
	"errors"
	"strconv"
	"time"
)

// MarshalJSONTo encodes the timestamp straight into the encoder, as MarshalJSON does but without allocating
func (e Timestamp) MarshalJSONTo(encoder *jsontext.Encoder) error {
	value := time.Time(e)
	// MarshalJSON reports years that RFC 3339 cannot represent as an error
	if year := value.Year(); year < 0 || year > 9999 {
		return errors.ErrUnsupported
	}
	buffer := encoder.AvailableBuffer()
	buffer = append(buffer, '"')
	buffer = value.AppendFormat(buffer, time.RFC3339Nano)
	buffer = append(buffer, '"')
	return encoder.WriteValue(buffer)
}

// MarshalJSONTo encodes the value straight into the encoder, as MarshalJSON does but without allocating
func (f FloatValue64) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encoder.WriteValue(appendFloatValue(encoder.AvailableBuffer(), float64(f), 64))
}

// MarshalJSONTo encodes the value straight into the encoder, as MarshalJSON does but without allocating
func (f FloatValue32) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encoder.WriteValue(appendFloatValue(encoder.AvailableBuffer(), float64(f), 32))
}

// appendFloatValue appends a float with a decimal point always present, so it is stored as a float
func appendFloatValue(buffer []byte, value float64, bitSize int) []byte {
	start := len(buffer)
	buffer = strconv.AppendFloat(buffer, value, 'f', -1, bitSize)
	if bytes.IndexByte(buffer[start:], '.') == -1 {
		buffer = append(buffer, '.', '0')
	}
	return buffer
}
