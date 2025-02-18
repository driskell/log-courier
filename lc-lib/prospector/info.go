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
	"context"
	"os"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/registrar"
)

const (
	statusOk = iota
	statusResume
	statusFailed
	statusInvalid
)

const (
	orphanedNo = iota
	orphanedMaybe
	orphanedYes
)

type prospectorInfo struct {
	ctx          context.Context
	file         string
	identity     registrar.FileIdentity
	lastSeen     uint32
	status       int
	running      bool
	orphaned     int
	finishOffset int64
	harvester    *harvester.Harvester
	err          error
	backoff      *core.ExpBackoff
	failedUntil  time.Time
}

func newProspectorInfoFromFileState(file string, filestate *registrar.FileState) *prospectorInfo {
	info := &prospectorInfo{
		file:         file,
		identity:     filestate,
		status:       statusResume,
		finishOffset: filestate.Offset,
		// TODO: Make configurable
		backoff: core.NewExpBackoff(file, 0, 300),
	}
	info.ctx = context.WithValue(context.Background(), registrar.ContextEntry, registrar.Entry(info))
	return info
}

func newProspectorInfoFromFileInfo(file string, fileinfo os.FileInfo) *prospectorInfo {
	info := &prospectorInfo{
		file:     file,
		identity: registrar.NewFileInfo(fileinfo), // fileinfo is nil for stdin
		// TODO: Make configurable
		backoff: core.NewExpBackoff(file, 0, 300),
	}
	info.ctx = context.WithValue(context.Background(), registrar.ContextEntry, registrar.Entry(info))
	return info
}

func newProspectorInfoInvalid(file string, err error) *prospectorInfo {
	info := &prospectorInfo{
		file:   file,
		err:    err,
		status: statusInvalid,
		// TODO: Make configurable
		backoff: core.NewExpBackoff(file, 0, 300),
	}
	info.ctx = context.WithValue(context.Background(), registrar.ContextEntry, registrar.Entry(info))
	return info
}

func (pi *prospectorInfo) isRunning() bool {
	if !pi.running {
		return false
	}

	select {
	case status := <-pi.harvester.OnFinish():
		pi.setHarvesterStopped(status)
	default:
	}

	return pi.running
}

func (pi *prospectorInfo) stop() {
	if !pi.running {
		return
	}
	pi.harvester.Stop()
}

func (pi *prospectorInfo) wait() {
	if !pi.running {
		return
	}
	status := <-pi.harvester.OnFinish()
	pi.setHarvesterStopped(status)
}

func (pi *prospectorInfo) apiEncodable() api.Encodable {
	return pi.harvester.APIEncodable()
}

func (pi *prospectorInfo) setHarvesterStopped(status *harvester.FinishStatus) {
	pi.running = false
	// Resume harvesting from the last event offset, not the last read, to allow codec to read from the last event
	// This ensures multiline codec populates correctly on resume
	pi.finishOffset = status.LastEventOffset
	if status.Error != nil {
		pi.status = statusFailed
		pi.err = status.Error
		if _, ok := pi.err.(*harvester.PermanentError); !ok {
			pi.failedUntil = time.Now().Add(pi.backoff.Trigger())
		}
	}
	if status.LastStat != nil {
		// Keep the last stat the harvester ran so we compare timestamps for potential resume
		pi.identity.Update(status.LastStat, &pi.identity)
	}
	pi.harvester = nil
}

func (pi *prospectorInfo) update(fileinfo os.FileInfo, iteration uint32) {
	if fileinfo != nil {
		// Allow identity to replace itself with a new identity (this allows a FileState to promote itself to a FileInfo)
		pi.identity.Update(fileinfo, &pi.identity)
	}

	pi.lastSeen = iteration
}

func (pi *prospectorInfo) seenInIteration(iteration uint32) bool {
	return pi.lastSeen == iteration
}

func (pi *prospectorInfo) canRestartFailed() bool {
	if pi.status != statusFailed {
		return true
	}
	if pi.failedUntil.IsZero() {
		// Permanent error if still zero
		return false
	}
	return time.Now().After(pi.failedUntil)
}

func (pi *prospectorInfo) maybeOrphaned() {
	pi.orphaned = orphanedMaybe
}

func (pi *prospectorInfo) setOrphaned() {
	pi.orphaned = orphanedYes
	if pi.running {
		pi.harvester.SetOrphaned()
	}
}
