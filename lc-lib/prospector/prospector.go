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
	"os"
	"path/filepath"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/registrar"
	"github.com/driskell/log-courier/lc-lib/spooler"
)

type Prospector struct {
	core.PipelineSegment
	core.PipelineConfigReceiver
	core.PipelineSnapshotProvider

	config          *config.Config
	prospectorindex map[string]*prospectorInfo
	prospectors     map[*prospectorInfo]*prospectorInfo
	fromBeginning   bool
	iteration       uint32
	lastscan        time.Time
	registrar       registrar.Registrator
	registrarSpool  registrar.EventSpooler
	snapshotChan    chan interface{}
	snapshotSink    chan []*core.Snapshot

	output chan<- *core.EventDescriptor
}

func NewProspector(pipeline *core.Pipeline, config *config.Config, fromBeginning bool, registrarImp registrar.Registrator, spoolerImp *spooler.Spooler) (*Prospector, error) {
	ret := &Prospector{
		config:          config,
		prospectorindex: make(map[string]*prospectorInfo),
		prospectors:     make(map[*prospectorInfo]*prospectorInfo),
		fromBeginning:   fromBeginning,
		registrar:       registrarImp,
		registrarSpool:  registrarImp.Connect(),
		snapshotChan:    make(chan interface{}),
		snapshotSink:    make(chan []*core.Snapshot),
		output:          spoolerImp.Connect(),
	}

	if err := ret.init(); err != nil {
		return nil, err
	}

	pipeline.Register(ret)

	return ret, nil
}

func (p *Prospector) init() (err error) {
	var havePrevious bool
	if havePrevious, err = p.registrar.LoadPrevious(p.loadCallback); err != nil {
		return
	}

	if havePrevious {
		// -from-beginning=false flag should only affect the very first run (no previous state)
		p.fromBeginning = true

		// Pre-populate prospectors with what we had previously
		for _, v := range p.prospectorindex {
			p.prospectors[v] = v
		}
	}

	return
}

func (p *Prospector) loadCallback(file string, state *registrar.FileState) (core.Stream, error) {
	p.prospectorindex[file] = newProspectorInfoFromFileState(file, state)
	return p.prospectorindex[file], nil
}

func (p *Prospector) Run() {
	defer func() {
		p.Done()
	}()

ProspectLoop:
	for {
		newlastscan := time.Now()
		p.iteration++ // Overflow is allowed

		for configKey, config := range p.config.Files {
			for _, path := range config.Paths {
				p.scan(path, &p.config.Files[configKey])
			}
		}

		// We only obey *fromBeginning (which is stored in this) on startup, if no
		// persist file exists. Afterwards we force from beginning
		p.fromBeginning = true

		// Clean up the prospector collections
		for _, info := range p.prospectors {
			if info.orphaned >= orphanedMaybe {
				if !info.isRunning() {
					delete(p.prospectors, info)
				}
			} else {
				if info.lastSeen >= p.iteration {
					continue
				}
				delete(p.prospectorindex, info.file)
				info.orphaned = orphanedMaybe
			}
			if info.orphaned == orphanedMaybe {
				info.orphaned = orphanedYes
				p.registrarSpool.Add(registrar.NewDeletedEvent(info))
			}
		}

		// Flush the accumulated registrar events
		p.registrarSpool.Send()

		p.lastscan = newlastscan

		// Defer next scan for a bit
		now := time.Now()
		scanDeadline := now.Add(p.config.General.ProspectInterval)

	DelayLoop:
		for {
			select {
			case <-time.After(scanDeadline.Sub(now)):
				break DelayLoop
			case <-p.OnShutdown():
				break ProspectLoop
			case <-p.snapshotChan:
				p.handleSnapshot()
			case config := <-p.OnConfig():
				p.config = config
			}

			now = time.Now()
			if now.After(scanDeadline) {
				break
			}
		}
	}

	// Send stop signal to all harvesters, then wait for them, for quick shutdown
	for _, info := range p.prospectors {
		info.stop()
	}
	for _, info := range p.prospectors {
		info.wait()
	}

	// Disconnect from the registrar
	p.registrarSpool.Close()

	log.Info("Prospector exiting")
}

func (p *Prospector) scan(path string, config *config.File) {
	// Evaluate the path as a wildcards/shell glob
	matches, err := filepath.Glob(path)
	if err != nil {
		log.Error("glob(%s) failed: %v", path, err)
		return
	}

	// Check any matched files to see if we need to start a harvester
	for _, file := range matches {
		// Check the current info against our index
		info, isKnown := p.prospectorindex[file]

		// Stat the file, following any symlinks
		// TODO: Low priority. Trigger loadFileId here for Windows instead of
		//       waiting for Harvester or Registrar to do it
		fileinfo, err := os.Stat(file)
		if err == nil {
			if fileinfo.IsDir() {
				err = newProspectorSkipError("Directory")
			}
		}

		if err != nil {
			// Do we know this entry?
			if isKnown {
				if info.status != statusInvalid {
					// The current entry is not an error, orphan it so we can log one
					info.orphaned = orphanedMaybe
				} else if info.err.Error() == err.Error() {
					// The same error occurred - don't log it again
					info.update(nil, p.iteration)
					continue
				}
			}

			// This is a new error
			info = newProspectorInfoInvalid(file, err)
			info.update(nil, p.iteration)

			// Print a friendly log message
			if _, ok := err.(*ProspectorSkipError); ok {
				log.Info("Skipping %s: %s", file, err)
			} else {
				log.Error("Error prospecting %s: %s", file, err)
			}

			p.prospectors[info] = info
			p.prospectorindex[file] = info
			continue
		} else if isKnown && info.status == statusInvalid {
			// We have an error stub and we've just successfully got fileinfo
			// Mark isKnown so we treat as a new file still
			isKnown = false
		}

		// Conditions for starting a new harvester:
		// - file path hasn't been seen before
		// - the file's inode or device changed
		if !isKnown {
			// Is this a rename/move?
			if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
				// Symlinks could mean we see the same file twice - skip if we have
				if previousinfo == nil {
					p.flagDuplicateError(file, info)
					continue
				}

				// This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
				log.Info("File rename was detected: %s -> %s", previous, file)
				info = previousinfo
				info.file = file

				p.registrarSpool.Add(registrar.NewRenamedEvent(info, file))
			} else {
				// This is a new entry
				info = newProspectorInfoFromFileInfo(file, fileinfo)

				// Check for dead time, but only if the file modification time is before the last scan started
				// This ensures we don't skip genuine creations with dead times less than 10s
				if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > config.DeadTime {
					// Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
					log.Info("Skipping file (older than dead time of %v): %s", config.DeadTime, file)

					// Store the offset that we should resume from if we notice a modification
					info.finishOffset = fileinfo.Size()
					p.registrarSpool.Add(registrar.NewDiscoverEvent(info, file, fileinfo.Size(), fileinfo))
				} else {
					// Process new file
					log.Info("Launching harvester on new file: %s", file)
					p.startHarvester(info, config)
				}
			}

			// Store the new entry
			p.prospectors[info] = info
		} else {
			if !info.identity.SameAs(fileinfo) {
				// Keep the old file in case we find it again shortly
				info.orphaned = orphanedMaybe

				if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
					// Symlinks could mean we see the same file twice - skip if we have
					if previousinfo == nil {
						p.flagDuplicateError(file, nil)
						continue
					}

					// This file was renamed from another file we know - link the same harvester channel as the old file
					log.Info("File rename was detected: %s -> %s", previous, file)
					info = previousinfo
					info.file = file

					p.registrarSpool.Add(registrar.NewRenamedEvent(info, file))
				} else {
					// File is not the same file we saw previously, it must have rotated and is a new file
					log.Info("Launching harvester on rotated file: %s", file)

					// Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
					info = newProspectorInfoFromFileInfo(file, fileinfo)

					// Process new file
					p.startHarvester(info, config)
				}

				// Store it
				p.prospectors[info] = info
			}
		}

		// Resume stopped harvesters
		resume := !info.isRunning()
		if resume {
			if info.status == statusResume {
				if info.finishOffset == fileinfo.Size() && time.Since(fileinfo.ModTime()) > config.DeadTime {
					// Old file with an unchanged offset, skip it
					log.Info("Skipping file (older than dead time of %v): %s", config.DeadTime, file)
					info.status = statusOk
					resume = false
				} else {
					// This is a filestate that was saved, resume the harvester
					log.Info("Resuming harvester on a previously harvested file: %s", file)
				}
			} else if info.status == statusFailed {
				// Last attempt we failed to start, try again
				log.Info("Attempting to restart failed harvester: %s", file)
			} else if info.identity.Stat().ModTime() != fileinfo.ModTime() {
				// Resume harvesting of an old file we've stopped harvesting from
				log.Info("Resuming harvester on an old file that was just modified: %s", file)
			} else {
				resume = false
			}
		}

		info.update(fileinfo, p.iteration)

		if resume {
			p.startHarvesterWithOffset(info, config, info.finishOffset)
		}

		p.prospectorindex[file] = info
	} // for each file matched by the glob
}

func (p *Prospector) flagDuplicateError(file string, info *prospectorInfo) {
	// Have we already logged this error?
	if info != nil {
		if info.status == statusInvalid {
			if skipErr, ok := info.err.(*ProspectorSkipError); ok && skipErr.message == "Duplicate" {
				return
			}
		}
	}

	// Flag duplicate error and save it
	info = newProspectorInfoInvalid(file, newProspectorSkipError("Duplicate"))
	info.update(nil, p.iteration)
	p.prospectors[info] = info
	p.prospectorindex[file] = info
}

func (p *Prospector) startHarvester(info *prospectorInfo, fileconfig *config.File) {
	var offset int64

	if p.fromBeginning {
		offset = 0
	} else {
		offset = info.identity.Stat().Size()
	}

	// Send a new file event to allow registrar to begin persisting for this harvester
	p.registrarSpool.Add(registrar.NewDiscoverEvent(info, info.file, offset, info.identity.Stat()))

	p.startHarvesterWithOffset(info, fileconfig, offset)
}

func (p *Prospector) startHarvesterWithOffset(info *prospectorInfo, fileconfig *config.File, offset int64) {
	// TODO - hook in a shutdown channel
	info.harvester = harvester.NewHarvester(info, p.config, &fileconfig.Stream, offset)
	info.running = true
	info.status = statusOk
	info.harvester.Start(p.output)
}

func (p *Prospector) lookupFileIds(file string, info os.FileInfo) (string, *prospectorInfo) {
	for _, ki := range p.prospectors {
		if ki.status == statusInvalid {
			// Don't consider error placeholders
			continue
		}
		if ki.orphaned == orphanedNo && ki.file == file {
			// We already know the prospector info for this file doesn't match, so don't check again
			continue
		}
		if ki.identity.SameAs(info) {
			// Already seen?
			if ki.lastSeen == p.iteration {
				return ki.file, nil
			}

			// Found previous information, remove it and return it (it will be added again)
			delete(p.prospectors, ki)
			if ki.orphaned == orphanedNo {
				delete(p.prospectorindex, ki.file)
			} else {
				ki.orphaned = orphanedNo
			}
			return ki.file, ki
		}
	}

	return "", nil
}

// Snapshot returns a snapshot structure containing information about all
// monitored and open files
func (p *Prospector) Snapshot() []*core.Snapshot {
	// TODO: Use Lock/Unlock and RLock/RUnlock instead of a channel - snapshots
	// will be slightly faster and code simpler and no need for timeout
	select {
	case p.snapshotChan <- 1:
	// Timeout after 5 seconds
	case <-time.After(5 * time.Second):
		ret := core.NewSnapshot("Prospector")
		ret.AddEntry("Error", "Timeout")
		return []*core.Snapshot{ret}
	}

	return <-p.snapshotSink
}

// handleSnapshot is called by the prospector routine when it receives a signal
// that a snapshot is required. It creates the snapshot and sends it to the requester
func (p *Prospector) handleSnapshot() {
	snapshots := make([]*core.Snapshot, 1)

	snapshots[0] = core.NewSnapshot("Prospector")
	snapshots[0].AddEntry("Watched files", len(p.prospectorindex))
	snapshots[0].AddEntry("Active states", len(p.prospectors))

	for _, info := range p.prospectorindex {
		snapshots = append(snapshots, p.snapshotInfo(info))
	}

	for _, info := range p.prospectors {
		if info.orphaned == orphanedNo {
			continue
		}
		snapshots = append(snapshots, p.snapshotInfo(info))
	}

	p.snapshotSink <- snapshots
}

// snapshotInfo generates the snapshot information for a single watched file
func (p *Prospector) snapshotInfo(info *prospectorInfo) *core.Snapshot {
	var extra string
	var status string

	if info.file == "-" {
		extra = "Stdin / "
	} else {
		switch info.orphaned {
		case orphanedMaybe:
			extra = "Orphan? / "
		case orphanedYes:
			extra = "Orphan / "
		}
	}

	switch info.status {
	default:
		if info.running {
			status = "Running"
		} else {
			status = "Dead"
		}
	case statusResume:
		status = "Resuming"
	case statusFailed:
		status = fmt.Sprintf("Failed: %s", info.err)
	case statusInvalid:
		if _, ok := info.err.(*ProspectorSkipError); ok {
			status = fmt.Sprintf("Skipped (%s)", info.err)
		} else {
			status = fmt.Sprintf("Error: %s", info.err)
		}
	}

	snap := core.NewSnapshot(fmt.Sprintf("\"State: %s (%s%p)\"", info.file, extra, info))
	snap.AddEntry("Status", status)

	if info.running {
		if subSnap := info.getSnapshot(); subSnap != nil {
			snap.AddSub(subSnap)
		}
	}

	return snap
}
