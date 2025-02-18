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

package prospector

import (
	"fmt"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
)

type apiStatus struct {
	api.KeyValue

	p *Prospector
}

// Update updates the prospector status information
func (a *apiStatus) Update() error {
	// Update the values and pass through to node
	a.p.mutex.RLock()
	a.SetEntry("watchedFiles", api.Number(len(a.p.prospectorindex)))
	a.SetEntry("activeStates", api.Number(len(a.p.prospectors)))
	a.p.mutex.RUnlock()

	return nil
}

type apiFiles struct {
	api.Array

	p *Prospector
}

func (a *apiFiles) Get(path string) (api.Navigatable, error) {
	if err := a.Update(); err != nil {
		return nil, err
	}

	return a.Array.Get(path)
}

func (a *apiFiles) Update() error {
	// Update the values and pass through to node
	a.p.mutex.RLock()

	for _, info := range a.p.prospectorindex {
		a.processEntry(info)
	}

	for _, info := range a.p.prospectors {
		if info.orphaned == orphanedNo {
			continue
		}
		a.processEntry(info)
	}

	a.p.mutex.RUnlock()

	return nil
}

// processEntry generates the status information for a single watched file
func (a *apiFiles) processEntry(info *prospectorInfo) {
	var fileType, orphaned, status api.String
	var errString api.Encodable

	if info.file == "-" {
		fileType = "stdin"
		orphaned = "no"
	} else {
		fileType = "file"
		switch info.orphaned {
		case orphanedMaybe:
			orphaned = "maybe"
		case orphanedYes:
			orphaned = "yes"
		default:
			orphaned = "no"
		}
	}

	switch info.status {
	default:
		if info.running {
			status = "running"
		} else {
			status = "dead"
		}
		errString = api.Null
	case statusResume:
		status = "resuming"
		errString = api.Null
	case statusFailed:
		if info.failedUntil.IsZero() {
			status = "failed (permanent)"
		} else {
			status = api.String(fmt.Sprintf("failed (retry at %s)", info.failedUntil.Format(time.RFC3339)))
		}
		errString = api.String(info.err.Error())
	case statusInvalid:
		if _, ok := info.err.(*prospectorSkipError); ok {
			status = "skipped"
		} else {
			status = "error"
		}
		errString = api.String(info.err.Error())
	}

	apiEntry := &api.KeyValue{}
	// TODO: Memory location leakage - replace with array instead or lookup by path
	key := fmt.Sprintf("%p", info)
	apiEntry.SetEntry("id", api.String(key))
	apiEntry.SetEntry("path", api.String(info.file))
	apiEntry.SetEntry("type", fileType)
	apiEntry.SetEntry("orphaned", orphaned)
	apiEntry.SetEntry("status", status)
	apiEntry.SetEntry("error", errString)

	if info.running {
		apiEntry.SetEntry("harvester", info.apiEncodable())
	}

	a.AddEntry(key, apiEntry)
}

type apiNode struct {
	api.Node

	p *Prospector
}

// Get processes a prospector API path
func (a *apiNode) Get(path string) (api.Navigatable, error) {
	if path == "files" {
		// Return a new apiFiles with empty array
		return &apiFiles{p: a.p}, nil
	}

	return a.Node.Get(path)
}

// MarshalJSON encodes the status in json form
func (a *apiNode) MarshalJSON() ([]byte, error) {
	// Add on the ephemeral files entry
	// TODO: This should be managed as part of adding/removing file tracking
	files := &apiFiles{p: a.p}
	if err := files.Update(); err != nil {
		return nil, err
	}

	a.SetEntry("files", files)
	result, err := a.Node.MarshalJSON()
	a.RemoveEntry("files")
	return result, err
}

// HumanReadable encodes the status as a readable string
func (a *apiNode) HumanReadable(indent string) ([]byte, error) {
	// Add on the ephemeral files entry
	// TODO: This should be managed as part of adding/removing file tracking
	files := &apiFiles{p: a.p}
	if err := files.Update(); err != nil {
		return nil, err
	}

	a.SetEntry("files", files)
	result, err := a.Node.HumanReadable(indent)
	a.RemoveEntry("files")
	return result, err
}
