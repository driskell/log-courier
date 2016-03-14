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
)

// APIIndentation is a single indentation to be used in HumanReadable calls on
// API entries
const APIIndentation = "  "

// An API error when the call is not implemented
var ErrNotImplemented = errors.New("Not Implemented")

// An API error when the requested information was not found
var ErrNotFound = errors.New("Not Found")

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

// APIEntry is a navigatable entry in the API
type APIEntry interface {
	APIEncodable

	// Get returns the child entry with the requested name, or nil there are no
	// children
	Get(name string) (APIEntry, error)

	// Call happens in response to a POST request
	Call(params url.Values) error

	// Update updates the entry data
	Update() error
}

// APINode acts like a directory in the API, containing mappings from names to
// status information of various types
type APINode struct {
	children map[string]APIEntry
}

// SetEntry adds a new path entry with the given name
func (n *APINode) SetEntry(path string, entry APIEntry) {
	if n.children == nil {
		n.children = make(map[string]APIEntry)
	}

	n.children[path] = entry
}

// Get the child entry with the specified name
func (n *APINode) Get(path string) (APIEntry, error) {
	entry, ok := n.children[path]
	if !ok {
		return nil, nil
	}
	return entry, nil
}

// Call an API
func (n *APINode) Call(params url.Values) error {
	return ErrNotImplemented
}

// MarshalJSON returns the entire path structures in JSON form
func (n *APINode) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.children)
}

// HumanReadable returns the entire path structures in human-readable form
func (n *APINode) HumanReadable(indent string) ([]byte, error) {
	if n.children == nil {
		return nil, nil
	}

	var result bytes.Buffer

	newIndent := indent + APIIndentation

	for name, entry := range n.children {
		part, err := entry.HumanReadable(newIndent)
		if err != nil {
			return nil, err
		}

		if bytes.IndexRune(part, '\n') != -1 {
			result.WriteString(name)
			result.WriteString(":\n")
			result.Write(part)
			continue
		}

		result.WriteString(name)
		result.WriteString(": ")
		result.Write(part)
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
