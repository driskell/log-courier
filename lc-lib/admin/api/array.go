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
	"strconv"
	"sync"
)

type apiArrayEntry struct {
	row   int
	entry Navigatable
}

// MarshalJSON forwards to the inner array entry
func (a *apiArrayEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.entry)
}

// Array represents an array of entries in the API accessible through a
// primary key
// Thread safe
type Array struct {
	mutex    sync.RWMutex
	entryMap map[string]*apiArrayEntry
	entries  []*apiArrayEntry
}

// AddEntry a new array entry
func (a *Array) AddEntry(key string, entry Navigatable) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.entryMap != nil {
		if _, ok := a.entryMap[key]; ok {
			panic("Key already exists")
		}
	}

	a.addEntry(key, entry)
}

// ReplaceEntry replaces an array entry, adding it if it does not exist
func (a *Array) ReplaceEntry(key string, entry Navigatable) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if existingEntry, ok := a.entryMap[key]; ok {
		a.removeEntry(key, existingEntry)
	}

	a.addEntry(key, entry)
}

// RemoveEntry removes an array entry
// Pass in non-nil entry to only remove if matches the given value
func (a *Array) RemoveEntry(key string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.entryMap == nil {
		panic("Array has no entries")
	}

	entry, ok := a.entryMap[key]
	if !ok {
		panic("Entry not found")
	}

	a.removeEntry(key, entry)
}

// RemoveEntry removes an array entry
// Pass in non-nil entry to only remove if matches the given value
func (a *Array) RemoveEntryIfValue(key string, ifValue Navigatable) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.entryMap == nil {
		panic("Array has no entries")
	}

	entry, ok := a.entryMap[key]
	if !ok {
		panic("Entry not found")
	}

	if ifValue != nil && entry.entry != ifValue {
		return
	}

	a.removeEntry(key, entry)
}

// addEntry adds an entry
// This is internal and performs no checks
func (a *Array) addEntry(key string, entry Navigatable) {
	arrayEntry := &apiArrayEntry{
		row:   len(a.entries),
		entry: entry,
	}

	if a.entryMap == nil {
		a.entryMap = make(map[string]*apiArrayEntry)
	}

	a.entryMap[key] = arrayEntry

	a.entries = append(a.entries, arrayEntry)
}

// removeEntry removes an entry
// This is internal and performs no checks
func (a *Array) removeEntry(key string, entry *apiArrayEntry) {
	delete(a.entryMap, key)
	for i := entry.row; i+1 < len(a.entries); i++ {
		a.entries[i] = a.entries[i+1]
		a.entries[i].row--
	}
	a.entries = a.entries[:len(a.entries)-1]
}

// Get returns an entry using it's primary key name or row number
func (a *Array) Get(path string) (Navigatable, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.entryMap == nil {
		return nil, nil
	}

	if entry, ok := a.entryMap[path]; ok {
		return entry.entry, nil
	}

	// Try to parse as a number
	entryNum, err := strconv.ParseInt(path, 10, 0)
	if err != nil {
		return nil, err
	}

	if entryNum < 0 || entryNum >= int64(len(a.entries)) {
		return nil, nil
	}

	return a.entries[entryNum].entry, nil
}

// Call an API
func (a *Array) Call(params url.Values) (string, error) {
	return "", ErrNotImplemented
}

// MarshalJSON returns the Array in JSON form
func (a *Array) MarshalJSON() ([]byte, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return json.Marshal(a.entries)
}

// HumanReadable returns the Array as a string
func (a *Array) HumanReadable(indent string) ([]byte, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.entryMap == nil || len(a.entryMap) == 0 {
		return []byte("none"), nil
	}

	var result bytes.Buffer
	newIndent := indent + Indentation

	mapOrder := make([]string, 0, len(a.entryMap))
	for key := range a.entryMap {
		mapOrder = append(mapOrder, key)
	}
	sort.Strings(mapOrder)

	for _, key := range mapOrder {
		arrayEntry := a.entryMap[key]

		subResult, err := arrayEntry.entry.HumanReadable(newIndent)
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
// Default behaviour is to update each of the array entries
func (a *Array) Update() error {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for _, entry := range a.entries {
		if err := entry.entry.Update(); err != nil {
			return err
		}
	}

	return nil
}
