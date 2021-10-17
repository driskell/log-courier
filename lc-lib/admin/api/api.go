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
	"errors"
	"fmt"
	"net/url"
	"sort"
	"sync"
)

// Indentation is a single indentation to be used in HumanReadable calls on
// API entries
const Indentation = "  "

var (
	// ErrNotImplemented is an API error when the call is not implemented
	ErrNotImplemented = errors.New("not implemented")

	// ErrNotFound is an API error when the requested information was not found
	ErrNotFound = errors.New("not found")
)

// ErrUnknown represents a successful request where Log Courier returned an
// error, as opposed to an error processing a request
type ErrUnknown error

// Encodable is an encodable entry in the API, which can be a navigatable
// entry or just a piece of data
type Encodable interface {
	// HumanReadable returns the entry as a string in human-readable form
	// If it contains multiple lines, it should prefix each line with the indent
	// string passes to it. If the call to HumanReadable needs to recurse into
	// another HumanReadable call it should add Indentation to the indent
	// string, and check the returned string for new lines and render accordingly
	HumanReadable(indent string) ([]byte, error)
}

// Nested represents an entry in the API that nests other entries
// It allows us to request instead a "summary" of just the keys
type Nested interface {
	Summary() map[string]interface{}
}

// Navigatable is a navigatable entry in the API
type Navigatable interface {
	Encodable

	// Get returns the child entry with the requested name, or nil there are no
	// children
	Get(name string) (Navigatable, error)

	// Call happens in response to a POST request
	Call(params url.Values) (string, error)

	// Update updates the entry data
	Update() error
}

// Node acts like a directory in the API, containing mappings from names to
// status information of various types
// Thread-safe
type Node struct {
	mutex    sync.RWMutex
	children map[string]Navigatable
}

// SetEntry adds a new path entry with the given name
func (n *Node) SetEntry(path string, entry Navigatable) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.children == nil {
		n.children = make(map[string]Navigatable)
	}

	n.children[path] = entry
}

// RemoveEntry removes a path entry
func (n *Node) RemoveEntry(path string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.children == nil {
		return
	}

	delete(n.children, path)
}

// Get the child entry with the specified name
func (n *Node) Get(path string) (Navigatable, error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	entry, ok := n.children[path]
	if !ok {
		return nil, nil
	}
	return entry, nil
}

// Call an API
func (n *Node) Call(url.Values) (string, error) {
	return "", ErrNotImplemented
}

// MarshalJSON returns the entire path structures in JSON form
func (n *Node) MarshalJSON() ([]byte, error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return json.Marshal(n.children)
}

// HumanReadable returns the entire path structures in human-readable form
func (n *Node) HumanReadable(indent string) ([]byte, error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	if n.children == nil || len(n.children) == 0 {
		return []byte("none"), nil
	}

	var result bytes.Buffer
	newIndent := indent + Indentation

	mapOrder := make([]string, 0, len(n.children))
	for key := range n.children {
		mapOrder = append(mapOrder, key)
	}
	sort.Strings(mapOrder)

	for _, name := range mapOrder {
		entry := n.children[name]

		part, err := entry.HumanReadable(newIndent)
		if err != nil {
			return nil, err
		}

		result.WriteString(indent)
		result.WriteString(name)

		if bytes.ContainsRune(part, '\n') {
			result.WriteString(":\n")
			result.Write(part)
			continue
		}

		result.WriteString(": ")
		result.Write(part)
		result.WriteString("\n")
	}

	return result.Bytes(), nil
}

// Update ensures the data we have is up to date - should be overriden by users
// if required to keep the contents up to date on each request
// Default behaviour is to update each of the navigatable entries
func (n *Node) Update() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for _, entry := range n.children {
		if err := entry.Update(); err != nil {
			return err
		}
	}

	return nil
}

// Summary returns just the children names and their types as a summary
// That way we could request the root just to get the names and not
// cause it to grab everything all the way down the tree
func (n *Node) Summary() map[string]interface{} {
	summary := make(map[string]interface{})
	for key, entry := range n.children {
		if _, ok := entry.(Nested); ok {
			summary[key] = struct{ Type string }{Type: fmt.Sprintf("%T", entry)}
		} else {
			summary[key] = entry
		}
	}
	return summary
}

// DataEntry wraps an Encodable so it can be used as an Navigatable
// It stubs the navigation methods so they are no-ops
type DataEntry struct {
	a Encodable
}

// NewDataEntry creates a new DataEntry from an Encodable
func NewDataEntry(a Encodable) *DataEntry {
	return &DataEntry{a: a}
}

// MarshalJSON returns the DataEntry in JSON form
func (d *DataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.a)
}

// HumanReadable returns the DataEntry as a string
func (d *DataEntry) HumanReadable(indent string) ([]byte, error) {
	return d.a.HumanReadable(indent)
}

// Get always returns nil for an DataEntry
func (d *DataEntry) Get(path string) (Navigatable, error) {
	return nil, nil
}

// Call always returns ErrNotImplemented for an DataEntry
func (d *DataEntry) Call(url.Values) (string, error) {
	return "", ErrNotImplemented
}

// Update does nothing for an DataEntry
func (d *DataEntry) Update() error {
	return nil
}

// CallbackFunc is a function that can be called by the API
type CallbackFunc func(url.Values) (string, error)

// CallbackEntry is an entry that provides an API callback
type CallbackEntry struct {
	f CallbackFunc
}

// NewCallbackEntry creates a new CallbackEntry from an CallbackFunc
func NewCallbackEntry(f CallbackFunc) *CallbackEntry {
	return &CallbackEntry{f: f}
}

// MarshalJSON returns the CallbackEntry in JSON form
func (c *CallbackEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{ Type string }{Type: fmt.Sprintf("%T", c)})
}

// HumanReadable returns the CallbackEntry as a string
func (c *CallbackEntry) HumanReadable(indent string) ([]byte, error) {
	return []byte("callback"), nil
}

// Get always returns nil for an CallbackEntry
func (c *CallbackEntry) Get(path string) (Navigatable, error) {
	return nil, nil
}

// Call runs the callback function
func (c *CallbackEntry) Call(values url.Values) (string, error) {
	return c.f(values)
}

// Update does nothing for an CallbackEntry
func (c *CallbackEntry) Update() error {
	return nil
}
