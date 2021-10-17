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

package api

import (
	"bytes"
	"encoding/json"
	"net/url"
	"sort"
)

// KeyValue represents a set of data
type KeyValue struct {
	entryMap map[string]Encodable
}

// Get always returns nil for an KeyValue as it is not navigatable
func (d *KeyValue) Get(string) (Navigatable, error) {
	return nil, nil
}

// Call an API
func (d *KeyValue) Call(params url.Values) (string, error) {
	return "", ErrNotImplemented
}

// SetEntry sets a new data entry
func (d *KeyValue) SetEntry(key string, entry Encodable) {
	if d.entryMap == nil {
		d.entryMap = make(map[string]Encodable)
	}

	d.entryMap[key] = entry
}

// RemoveEntry removes a data entry
func (d *KeyValue) RemoveEntry(key string, entry Encodable) {
	if _, ok := d.entryMap[key]; !ok {
		return
	}

	delete(d.entryMap, key)
}

// MarshalJSON returns the KeyValue data in JSON form
func (d *KeyValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.entryMap)
}

// HumanReadable returns the KeyValue as a string
func (d *KeyValue) HumanReadable(indent string) ([]byte, error) {
	if d.entryMap == nil || len(d.entryMap) == 0 {
		return []byte("none"), nil
	}

	var result bytes.Buffer
	newIndent := indent + Indentation

	mapOrder := make([]string, 0, len(d.entryMap))
	for key := range d.entryMap {
		mapOrder = append(mapOrder, key)
	}
	sort.Strings(mapOrder)

	for _, key := range mapOrder {
		entry := d.entryMap[key]

		subResult, err := entry.HumanReadable(newIndent)
		if err != nil {
			return nil, err
		}

		result.WriteString(indent)
		result.WriteString(key)

		if bytes.ContainsRune(subResult, '\n') {
			result.WriteString(":\n")
			result.Write(subResult)
			continue
		}

		result.WriteString(": ")
		result.Write(subResult)
		result.WriteString("\n")
	}

	return result.Bytes(), nil
}

// Update ensures the data we have is up to date - should be overriden by users
// if required to keep the contents up to date on each request
func (d *KeyValue) Update() error {
	return nil
}
