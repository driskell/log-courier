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

package prospector

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/admin"
)

type apiStatus struct {
	admin.APIKeyValue

	p *Prospector
}

// Update updates the prospector status information
func (a *apiStatus) Update() error {
	// Update the values and pass through to node
	a.p.mutex.RLock()
	a.SetEntry("watchedFiles", admin.APINumber(len(a.p.prospectorindex)))
	a.SetEntry("activeStates", admin.APINumber(len(a.p.prospectors)))
	a.p.mutex.RUnlock()

	return nil
}

type apiFiles struct {
	admin.APIArray

	p *Prospector
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
	var fileType, orphaned, status admin.APIString
	var errString admin.APIEncodable

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
		errString = admin.APINull
	case statusResume:
		status = "resuming"
		errString = admin.APINull
	case statusFailed:
		status = "failed"
		errString = admin.APIString(info.err.Error())
	case statusInvalid:
		if _, ok := info.err.(*ProspectorSkipError); ok {
			status = "skipped"
		} else {
			status = "error"
		}
		errString = admin.APIString(info.err.Error())
	}

	apiEntry := &admin.APIKeyValue{}
	key := fmt.Sprintf("%p", info)
	apiEntry.SetEntry("id", admin.APIString(key))
	apiEntry.SetEntry("path", admin.APIString(info.file))
	apiEntry.SetEntry("type", fileType)
	apiEntry.SetEntry("orphaned", orphaned)
	apiEntry.SetEntry("status", status)
	apiEntry.SetEntry("error", errString)

	if info.running {
		apiEntry.SetEntry("harvester", info.apiEncodable())
	}

	a.AddEntry(key, apiEntry)
}

type api struct {
	admin.APINode

	p *Prospector
}

// Get processes a prospector API path
func (a *api) Get(path string) (admin.APIEntry, error) {
	if path == "files" {
		// Return a new apiFiles with empty array
		return &apiFiles{p: a.p}, nil
	}

	return a.APINode.Get(path)
}
