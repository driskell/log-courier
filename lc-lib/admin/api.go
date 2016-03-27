/*
 * Copyright 2014-2015 Jason Woods.
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

package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
)

// APIIndentation is a single indentation to be used in HumanReadable calls on
// API entries
const APIIndentation = "  "

var (
	// ErrNotImplemented is an API error when the call is not implemented
	ErrNotImplemented = errors.New("Not Implemented")

	// ErrNotFound is an API error when the requested information was not found
	ErrNotFound = errors.New("Not Found")

	// callMap is a list of commands known to be Call only, and the Client uses
	// this to automatically translate Request calls into Call calls to simplify
	// logic in clients
	callMap = map[string]interface{}{
		"reload": nil,
	}
)

// ErrUnknown represents a successful request where Log Courier returned an
// error, as opposed to an error processing a request
type ErrUnknown error

// APIEncodable is an encodable entry in the API, which can be a navigatable
// entry or just a piece of data
type APIEncodable interface {
	// HumanReadable returns the entry as a string in human-readable form
	// If it contains multiple lines, it should prefix each line with the indent
	// string passes to it. If the call to HumanReadable needs to recurse into
	// another HumanReadable call it should add APIIndentation to the indent
	// string, and check the returned string for new lines and render accordingly
	HumanReadable(indent string) ([]byte, error)
}

// APINavigatable is a navigatable entry in the API
type APINavigatable interface {
	APIEncodable

	// Get returns the child entry with the requested name, or nil there are no
	// children
	Get(name string) (APINavigatable, error)

	// Call happens in response to a POST request
	Call(params url.Values) (string, error)

	// Update updates the entry data
	Update() error
}

// APINode acts like a directory in the API, containing mappings from names to
// status information of various types
type APINode struct {
	children map[string]APINavigatable
}

// SetEntry adds a new path entry with the given name
func (n *APINode) SetEntry(path string, entry APINavigatable) {
	if n.children == nil {
		n.children = make(map[string]APINavigatable)
	}

	n.children[path] = entry
}

// RemoveEntry removes a path entry
func (n *APINode) RemoveEntry(path string) {
	if n.children == nil {
		return
	}

	delete(n.children, path)
}

// Get the child entry with the specified name
func (n *APINode) Get(path string) (APINavigatable, error) {
	entry, ok := n.children[path]
	if !ok {
		return nil, nil
	}
	return entry, nil
}

// Call an API
func (n *APINode) Call(url.Values) (string, error) {
	return "", ErrNotImplemented
}

// MarshalJSON returns the entire path structures in JSON form
func (n *APINode) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.children)
}

// HumanReadable returns the entire path structures in human-readable form
func (n *APINode) HumanReadable(indent string) ([]byte, error) {
	if n.children == nil || len(n.children) == 0 {
		return []byte("none"), nil
	}

	var result bytes.Buffer
	newIndent := indent + APIIndentation

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

		if bytes.IndexRune(part, '\n') != -1 {
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
func (n *APINode) Update() error {
	for _, entry := range n.children {
		if err := entry.Update(); err != nil {
			return err
		}
	}

	return nil
}

// APIDataEntry wraps an APIEncodable so it can be used as an APINavigatable
// It stubs the navigation methods so they are no-ops
type APIDataEntry struct {
	a APIEncodable
}

// NewAPIDataEntry creates a new APIDataEntry from an APIEncodable
func NewAPIDataEntry(a APIEncodable) *APIDataEntry {
	return &APIDataEntry{a: a}
}

// MarshalJSON returns the APIDataEntry in JSON form
func (d *APIDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.a)
}

// HumanReadable returns the APIDataEntry as a string
func (d *APIDataEntry) HumanReadable(indent string) ([]byte, error) {
	return d.a.HumanReadable(indent)
}

// Get always returns nil for an APIDataEntry
func (d *APIDataEntry) Get(path string) (APINavigatable, error) {
	return nil, nil
}

// Call always returns ErrNotImplemented for an APIDataEntry
func (d *APIDataEntry) Call(url.Values) (string, error) {
	return "", ErrNotImplemented
}

// Update does nothing for an APIDataEntry
func (d *APIDataEntry) Update() error {
	return nil
}

// APICallbackFunc is a functionthat can be called by the API
type APICallbackFunc func(url.Values) (string, error)

// APICallbackEntry is an entry that provides an API callback
type APICallbackEntry struct {
	f APICallbackFunc
}

// NewAPICallbackEntry creates a new APICallbackEntry from an APICallbackFunc
func NewAPICallbackEntry(f APICallbackFunc) *APICallbackEntry {
	return &APICallbackEntry{f: f}
}

// MarshalJSON returns the APICallbackEntry in JSON form
func (c *APICallbackEntry) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// HumanReadable returns the APICallbackEntry as a string
func (c *APICallbackEntry) HumanReadable(indent string) ([]byte, error) {
	return []byte("callback"), nil
}

// Get always returns nil for an APICallbackEntry
func (c *APICallbackEntry) Get(path string) (APINavigatable, error) {
	return nil, nil
}

// Call runs the callback function
func (c *APICallbackEntry) Call(values url.Values) (string, error) {
	return c.f(values)
}

// Update does nothing for an APICallbackEntry
func (c *APICallbackEntry) Update() error {
	return nil
}
