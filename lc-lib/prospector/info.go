/*
* Copyright 2014 Jason Woods.
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
	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/harvester"
	"github.com/driskell/log-courier/src/lc-lib/registrar"
	"os"
)

const (
	Status_Ok = iota
	Status_Resume
	Status_Failed
	Status_Invalid
)

const (
	Orphaned_No = iota
	Orphaned_Maybe
	Orphaned_Yes
)

type prospectorInfo struct {
	file          string
	identity      registrar.FileIdentity
	last_seen     uint32
	status        int
	running       bool
	orphaned      int
	finish_offset int64
	harvester     *harvester.Harvester
	err           error
}

func newProspectorInfoFromFileState(file string, filestate *registrar.FileState) *prospectorInfo {
	return &prospectorInfo{
		file:          file,
		identity:      filestate,
		status:        Status_Resume,
		finish_offset: filestate.Offset,
	}
}

func newProspectorInfoFromFileInfo(file string, fileinfo os.FileInfo) *prospectorInfo {
	return &prospectorInfo{
		file:     file,
		identity: registrar.NewFileInfo(fileinfo), // fileinfo is nil for stdin
	}
}

func newProspectorInfoInvalid(file string, err error) *prospectorInfo {
	return &prospectorInfo{
		file:   file,
		err:    err,
		status: Status_Invalid,
	}
}

func (pi *prospectorInfo) Info() (string, os.FileInfo) {
	return pi.file, pi.identity.Stat()
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

func (pi *prospectorInfo) getSnapshot() *core.Snapshot {
	return pi.harvester.Snapshot()
}

func (pi *prospectorInfo) setHarvesterStopped(status *harvester.HarvesterFinish) {
	pi.running = false
	// Resume harvesting from the last event offset, not the last read, to allow codec to read from the last event
	// This ensures multiline codec populates correctly on resume
	pi.finish_offset = status.Last_Event_Offset
	if status.Error != nil {
		pi.status = Status_Failed
		pi.err = status.Error
	}
	if status.Last_Stat != nil {
		// Keep the last stat the harvester ran so we compare timestamps for potential resume
		pi.identity.Update(status.Last_Stat, &pi.identity)
	}
	pi.harvester = nil
}

func (pi *prospectorInfo) update(fileinfo os.FileInfo, iteration uint32) {
	if fileinfo != nil {
		// Allow identity to replace itself with a new identity (this allows a FileState to promote itself to a FileInfo)
		pi.identity.Update(fileinfo, &pi.identity)
	}

	pi.last_seen = iteration
}
