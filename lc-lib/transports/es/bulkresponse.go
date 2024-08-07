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

package es

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type bulkResponseError struct {
	Type      string                 `json:"type"`
	Reason    string                 `json:"reason"`
	CausedBy  *bulkResponseError     `json:"caused_by"`
	EventData map[string]interface{} `json:"-"`
}

func (e *bulkResponseError) SameAs(other *bulkResponseError) bool {
	return e.Type == other.Type && e.Reason == other.Reason
}

func (e *bulkResponseError) Error() string {
	data, err := json.Marshal(e.EventData)
	if err != nil {
		data = []byte(err.Error())
	}
	if e.CausedBy != nil {
		return fmt.Sprintf("[%s] %s; Caused by %s; Event: %s", e.Type, strings.TrimRight(e.Reason, ";"), e.CausedBy.Error(), data)
	}
	return fmt.Sprintf("[%s] %s; Event: %s", e.Type, strings.TrimRight(e.Reason, ";"), data)
}

type bulkResponse struct {
	decoder     *json.Decoder
	bulkRequest *bulkRequest

	Took   int
	Errors []*bulkResponseError
}

// newBulkResponse parses a bulk response and returns a structure representing it
// It also marks the successful events inside the bulkRequest
func newBulkResponse(body []byte, bulkRequest *bulkRequest) (*bulkResponse, error) {
	ret := &bulkResponse{
		decoder:     json.NewDecoder(bytes.NewReader(body)),
		bulkRequest: bulkRequest,
	}
	if err := ret.parse(); err != nil {
		return nil, fmt.Errorf("%s [Body: %s]", err, body)
	}
	return ret, nil
}

// parse parses a bulk response using the JSON tokeniser
func (b *bulkResponse) parse() error {
	// Expect {
	if err := b.consumeDelim('{'); err != nil {
		return err
	}

	var hasTook, hasItems bool

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
			if hasTook {
				return errors.New("unexpected duplicate key \"took\"")
			}
			hasTook = true
			if err := b.decoder.Decode(&b.Took); err != nil {
				return err
			}
		case "items":
			if hasItems {
				return errors.New("unexpected duplicate key \"items\"")
			}
			hasItems = true
			if err := b.parseItems(); err != nil {
				return err
			}
		default:
			if err := b.consumeValue(); err != nil {
				return err
			}
		}
	}

	if b.decoder.More() {
		return errors.New("unexpected tokens at end of stream")
	}

	if !hasTook {
		return errors.New("response is missing \"took\" key")
	}
	if !hasItems {
		return errors.New("response is missing \"items\" key")
	}

	return nil
}

// parseItems parses the items array
func (b *bulkResponse) parseItems() error {
	// Expect [
	if err := b.consumeDelim('['); err != nil {
		return err
	}

	var (
		cursor *bulkRequestCursor
		ended  bool
	)

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
			return errors.New("expected 'items' entry to be an object")
		}

		// Should have met the end?
		if ended {
			return errors.New("too many results received")
		}

		// All bulk operations are index, so expect index object
		key, err := b.parseKeyOrEnd()
		if err != nil {
			return err
		}
		if key == nil {
			return errors.New("unexpected end of an 'items' entry")
		}
		if *key != "index" {
			return errors.New("expected only 'index' key within an 'items' entry")
		}

		// Expect another object for value
		if err := b.consumeDelim('{'); err != nil {
			return err
		}

		// Now we can discard everything except result
		var (
			status     uint16 = 200
			errorValue *bulkResponseError
		)
		for {
			key, err := b.parseKeyOrEnd()
			if err != nil {
				return err
			}
			if key == nil {
				break
			}
			if *key == "result" {
				if err := b.consumeValue(); err != nil {
					return err
				}
			} else if *key == "error" {
				errorValue = &bulkResponseError{}
				if err := b.decoder.Decode(errorValue); err != nil {
					return err
				}
				// I believe older versions of ES do not have status field, so simulate 400 just in case which will not retry
				if status == 200 {
					status = 400
				}
			} else if *key == "status" {
				if err := b.decoder.Decode(&status); err != nil {
					return err
				}
			} else {
				if err := b.consumeValue(); err != nil {
					return err
				}
			}
		}

		// Status?
		discard := true
		if status < 200 || status > 299 {
			// Should have an error
			if errorValue == nil {
				errorValue = &bulkResponseError{Type: fmt.Sprintf("Status %d with no error", status), Reason: "none"}
			}
			errorValue.EventData = b.bulkRequest.Event(cursor)
			b.Errors = append(b.Errors, errorValue)
			// 400 are not retryable so leave it as error we discard for, retry everything else
			if status != 400 {
				discard = false
			}
		}

		// Mark status of the event in the request so we can possibly resend
		cursor, ended = b.bulkRequest.Mark(cursor, discard)

		// Now end the inner object which should contain only "index"
		key, err = b.parseKeyOrEnd()
		if err != nil {
			return err
		}
		if key != nil {
			return errors.New("unexpected additional key in an 'items' entry")
		}
	}

	if !ended {
		return errors.New("too few results received")
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
		return fmt.Errorf("expected delimiter: '%c'", expect)
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
			return nil, errors.New("expected object key")
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
		return delim, nil
	}
	return token, nil
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
		return fmt.Errorf("unrecognised delimiter: '%c'", delim)
	}
	return nil
}
