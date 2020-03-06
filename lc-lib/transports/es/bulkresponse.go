/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

package es

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	parseStateOutside int = iota
	parseStateInside
	parseStateTook
	parseStateItems
	parseStateItem
	parseStateResult
)

type bulkResponse struct {
	decoder     *json.Decoder
	bulkRequest *bulkRequest

	Took int
}

// newBulkResponse parses a bulk response and returns a structure representing it
// It also marks the successful events inside the bulkRequest
func newBulkResponse(response io.Reader, bulkRequest *bulkRequest) (*bulkResponse, error) {
	ret := &bulkResponse{
		decoder:     json.NewDecoder(response),
		bulkRequest: bulkRequest,
	}
	if err := ret.parse(); err != nil {
		return nil, err
	}
	return ret, nil
}

// parse parses a bulk response using the JSON tokeniser
func (b *bulkResponse) parse() error {
	// Expect {
	if err := b.consumeDelim('{'); err != nil {
		return err
	}

	for {
		// Expect } or key:value
		key, err := b.parseKeyOrEnd()
		if err != nil {
			return err
		}
		if key == nil {
			break
		}

		switch *key {
		case "took":
			if err := b.decoder.Decode(&b.Took); err != nil {
				return err
			}
			break
		case "items":
			if err := b.parseItems(); err != nil {
				return err
			}
			break
		default:
			if err := b.consumeValue(); err != nil {
				return err
			}
			break
		}
	}

	if b.decoder.More() {
		return errors.New("Unexpected tokens at end of stream")
	}

	return nil
}

// parseItems parses the items array
func (b *bulkResponse) parseItems() error {
	// Expect [
	if err := b.consumeDelim('['); err != nil {
		return err
	}

	var cursor *bulkRequestCursor
	var ended bool

	for {
		// Expect object for each item
		next, err := b.parseArrayNextOrEnd()
		if err != nil {
			return err
		}
		if next == nil {
			break
		}
		if delim, ok := next.(json.Delim); !ok || delim != json.Delim('{') {
			return errors.New("Expected 'items' entry to be an object")
		}

		// All bulk operations are index, so expect index object
		key, err := b.parseKeyOrEnd()
		if err != nil {
			return err
		}
		if key == nil {
			return errors.New("Unexpected end of an 'items' entry")
		}
		if *key != "index" {
			return errors.New("Expected only 'index' key within an 'items' entry")
		}

		// Expect another object for value
		if err := b.consumeDelim('{'); err != nil {
			return err
		}

		// Now we can discard everything except result
		for {
			key, err := b.parseKeyOrEnd()
			if err != nil {
				return err
			}
			if key == nil {
				break
			}
			if *key == "result" {
				if ended {
					return fmt.Errorf("Too many results received")
				}
				var result string
				if err := b.decoder.Decode(&result); err != nil {
					return err
				}
				// Mark status of the event in the request so we can possibly resend
				cursor, ended = b.bulkRequest.Mark(cursor, result == "created")
			} else {
				if err := b.consumeValue(); err != nil {
					return err
				}
			}
		}

		// Now end the inner object which should contain only "index"
		key, err = b.parseKeyOrEnd()
		if err != nil {
			return err
		}
		if key != nil {
			return fmt.Errorf("Unexpected additional key in an 'items' entry")
		}
	}

	if !ended {
		return fmt.Errorf("Too few results received")
	}

	return nil
}

// consumeDelim expects a delimiter and consumes it
func (b *bulkResponse) consumeDelim(expect rune) error {
	token, err := b.decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); !ok || delim != json.Delim(expect) {
		return fmt.Errorf("Expected delimiter: '%c'", expect)
	}
	return nil
}

// parseKey parses an object key, returning nil if end of object delimiter reached
func (b *bulkResponse) parseKeyOrEnd() (*string, error) {
	token, err := b.decoder.Token()
	if err != nil {
		return nil, err
	}
	key, ok := token.(string)
	if !ok {
		if delim, ok := token.(json.Delim); !ok || delim != json.Delim('}') {
			return nil, errors.New("Expected object key")
		}
		return nil, nil
	}
	return &key, nil
}

// parseArrayNextOrEnd parses a comma or end of array
func (b *bulkResponse) parseArrayNextOrEnd() (json.Token, error) {
	token, err := b.decoder.Token()
	if err != nil {
		return false, err
	}
	delim, ok := token.(json.Delim)
	if ok {
		if delim == json.Delim(']') {
			return nil, nil
		}
	}
	return delim, nil
}

// consumeValue takes away an entire value
func (b *bulkResponse) consumeValue() error {
	token, err := b.decoder.Token()
	if err != nil {
		return err
	}

	return b.consumeValueFrom(token)
}

// consumeValueFrom takes away an entire value, starting with the given token
func (b *bulkResponse) consumeValueFrom(token json.Token) error {
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	if delim == json.Delim('[') {
		for {
			next, err := b.parseArrayNextOrEnd()
			if err != nil {
				return err
			}
			if next == nil {
				break
			}
			if err := b.consumeValueFrom(next); err != nil {
				return err
			}
		}
	} else if delim == json.Delim('{') {
		for {
			key, err := b.parseKeyOrEnd()
			if err != nil {
				return err
			}
			if key == nil {
				break
			}
			if err := b.consumeValue(); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("Unrecognised delimiter: '%c'", delim)
	}
	return nil
}
