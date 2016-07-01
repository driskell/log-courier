/*
 * Copyright 2016 Jason Woods.
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

package simplejson

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"
)

var hex = "0123456789abcdef"

// InlineJSON contains an already JSON-encoded value that can be embedded
// directly into an encoding. This allows pre-encoding of values for faster
// processing
type InlineJSON string

// NewInlineJSON creates an InlineJSON from an existing string
func NewInlineJSON(value string) *InlineJSON {
	inline := InlineJSON(value)
	return &inline
}

// Marshal a value to JSON
//
// Supported types are:
//   map[string]interface{} -> JSON dictionary
//   []interface{} -> JSON array
//   string -> JSON string
//   *string -> JSON string
//   floats, uints, ints -> JSON number
//   InlineJSON -> The string is inlined as literal JSON (can be used to pre-encode values)
//   nil -> JSON null
//
// The InlineJSON type is special - the contents will be inlined directly into the
// JSON output. This can allow pre-encoded JSON to be reused over and over.
//
// Encoding performs at least 2 times faster than json.Marshal.
//
// NOTE: Values must NOT be modified during encoding as it may cause corrupt
//       output or even an out of bounds panic
//
// NOTE: MarshalJSON methods on types are ignored
func Marshal(value interface{}) ([]byte, error) {
	e := &encoder{}

	err := e.prepareBuffer(value)
	if err != nil {
		return nil, err
	}

	err = e.encodeValue(value)
	if err != nil {
		return nil, err
	}

	return e.dst, nil
}

type encoder struct {
	dst []byte
	p   int
}

func (e *encoder) len() int {
	return e.p
}

func (e *encoder) writeString(src string) {
	if e.dst != nil {
		copy(e.dst[e.p:], src)
	}
	e.p += len(src)
}

func (e *encoder) writeBytes(src []byte) {
	if e.dst != nil {
		copy(e.dst[e.p:], src)
	}
	e.p += len(src)
}

// Returns error to keep go vet clean
func (e *encoder) writeByte(src byte) {
	if e.dst != nil {
		e.dst[e.p] = src
	}
	e.p++
}

func (e *encoder) writeFloat(f float64, size int) {
	if e.dst != nil {
		result := strconv.AppendFloat(e.dst[e.p:e.p:cap(e.dst)], f, 'g', -1, size)
		e.p += len(result)
		return
	}

	e.p += len(strconv.FormatFloat(f, 'g', -1, size))
}

func (e *encoder) writeInt(i int64) {
	if e.dst != nil {
		result := strconv.AppendInt(e.dst[e.p:e.p:cap(e.dst)], i, 10)
		e.p += len(result)
		return
	}

	e.p += len(strconv.FormatInt(i, 10))
}

func (e *encoder) writeUint(i uint64) {
	if e.dst != nil {
		result := strconv.AppendUint(e.dst[e.p:e.p:cap(e.dst)], i, 10)
		e.p += len(result)
		return
	}

	e.p += len(strconv.FormatUint(i, 10))
}

func (e *encoder) prepareBuffer(value interface{}) error {
	e.dst = nil
	err := e.encodeValue(value)
	if err != nil {
		return err
	}

	e.dst = make([]byte, e.p)
	e.p = 0
	return nil
}

// encodeInto encodes the map into the given target byte array which should
// be pre-allocated to the correct size
func (e *encoder) encodeMap(value map[string]interface{}) error {
	separate := false

	// Open
	e.writeByte('{')

	for k, v := range value {
		// If we're not at the first key, add a separator
		if separate {
			e.writeByte(',')
		} else {
			separate = true
		}

		// Add the key
		if err := e.encodeString(k); err != nil {
			return err
		}

		// Separator
		e.writeByte(':')

		if err := e.encodeValue(v); err != nil {
			return err
		}
	}

	// Close
	e.writeByte('}')

	return nil
}

func (e *encoder) encodeValue(v interface{}) error {
	// Now add the value
	switch vt := v.(type) {
	case map[string]interface{}:
		// Another map, recurse
		err := e.encodeMap(vt)
		if err != nil {
			return err
		}
	case []interface{}:
		// Slice
		err := e.encodeSlice(vt)
		if err != nil {
			return err
		}
	case string:
		// String, just enclose in quotes
		e.encodeString(vt)
	case *string:
		// String, just enclose in quotes
		e.encodeString(*vt)
	case float64:
		e.writeFloat(vt, 64)
	case float32:
		e.writeFloat(float64(vt), 32)
	case int64:
		e.writeInt(vt)
	case int32:
		e.writeInt(int64(vt))
	case int16:
		e.writeInt(int64(vt))
	case int8:
		e.writeInt(int64(vt))
	case int:
		e.writeInt(int64(vt))
	case uint64:
		e.writeUint(vt)
	case uint32:
		e.writeUint(uint64(vt))
	case uint16:
		e.writeUint(uint64(vt))
	case uint8:
		e.writeUint(uint64(vt))
	case uint:
		e.writeUint(uint64(vt))
	case InlineJSON:
		// Embed directly
		e.writeString(string(vt))
	case *InlineJSON:
		// Embed directly
		e.writeString(string(*vt))
	default:
		fv := reflect.ValueOf(v)

		if !fv.IsValid() {
			e.writeString("null")
			break
		}

		return fmt.Errorf("Unsupported type: %t", v)
	}

	return nil
}

func (e *encoder) encodeSlice(s []interface{}) error {
	e.writeByte('[')

	total := len(s) - 1
	for c, v := range s {
		if err := e.encodeValue(v); err != nil {
			return err
		}

		if c != total {
			e.writeByte(',')
		}
	}

	e.writeByte(']')
	return nil
}

// This is a mirror of golang's json pkg encodeState.stringBytes
// Keep in sync with countString below
func (e *encoder) encodeString(s string) error {
	e.writeByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				e.writeString(s[start:i])
			}
			switch b {
			case '\\', '"':
				e.writeByte('\\')
				e.writeByte(b)
			case '\n':
				e.writeByte('\\')
				e.writeByte('n')
			case '\r':
				e.writeByte('\\')
				e.writeByte('r')
			case '\t':
				e.writeByte('\\')
				e.writeByte('t')
			default:
				// This encodes bytes < 0x20 except for \n and \r,
				// as well as <, >, and &. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				e.writeString(`\u00`)
				e.writeByte(hex[b>>4])
				e.writeByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				e.writeString(s[start:i])
			}
			e.writeString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				e.writeString(s[start:i])
			}
			e.writeString(`\u202`)
			e.writeByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		e.writeString(s[start:])
	}
	e.writeByte('"')
	return nil
}
