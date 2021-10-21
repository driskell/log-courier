/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/registrar"
)

// Prospector handles the crawling of paths and starting and stopping of
// harvester routines against discovered files
type Prospector struct {
	mutex           sync.RWMutex
	config          *config.Config
	adminConfig     *admin.Config
	genConfig       *General
	fileConfigs     Config
	prospectorindex map[string]*prospectorInfo
	prospectors     map[*prospectorInfo]*prospectorInfo
	fromBeginning   bool
	iteration       uint32
	lastscan        time.Time
	registrar       *registrar.Registrar
	registrarSpool  *registrar.EventSpooler
	shutdownChan    <-chan struct{}
	configChan      <-chan *config.Config
	output          chan<- []*event.Event
}

// NewProspector creates a new path crawler with the given configuration
// If fromBeginning is true and registrar reports no state was loaded, all new
// files on the FIRST scan will be started from the beginning, as opposed to
// from the end
func NewProspector(app *core.App, fromBeginning bool) *Prospector {
	cfg := app.Config()
	genConfig := cfg.GeneralPart("prospector").(*General)

	registrarImpl := registrar.NewRegistrar(genConfig.PersistDir)
	app.Pipeline().AddService(registrarImpl)

	return &Prospector{
		prospectorindex: make(map[string]*prospectorInfo),
		prospectors:     make(map[*prospectorInfo]*prospectorInfo),
		config:          cfg,
		adminConfig:     admin.FetchConfig(cfg),
		genConfig:       genConfig,
		fileConfigs:     cfg.Section("files").(Config),
		registrar:       registrarImpl,
		registrarSpool:  registrar.NewEventSpooler(registrarImpl),
		fromBeginning:   fromBeginning,
	}
}

// SetOutput sets the output channel
func (p *Prospector) SetOutput(output chan<- []*event.Event) {
	p.output = output
}

// SetShutdownChan sets the shutdown channel
func (p *Prospector) SetShutdownChan(shutdownChan <-chan struct{}) {
	p.shutdownChan = shutdownChan
}

// SetConfigChan sets the config channel
func (p *Prospector) SetConfigChan(configChan <-chan *config.Config) {
	p.configChan = configChan
}

// Init prepares the Prospector
func (p *Prospector) Init(cfg *config.Config) error {
	havePrevious, err := p.registrar.LoadPrevious(p.loadCallback)
	if err != nil {
		return err
	}

	if havePrevious {
		// -from-beginning=false flag should only affect the very first run (no previous state)
		p.fromBeginning = true

		// Pre-populate prospectors with what we had previously
		for _, v := range p.prospectorindex {
			p.prospectors[v] = v
		}
	}

	p.initAPI()

	return nil
}

// loadCallback receives existing file offsets from the registrar
func (p *Prospector) loadCallback(file string, state *registrar.FileState) (registrar.Entry, error) {
	p.prospectorindex[file] = newProspectorInfoFromFileState(file, state)
	return p.prospectorindex[file], nil
}

// Run begins the prospector loop
func (p *Prospector) Run() {
	for {
		if p.runOnce() {
			break
		}
	}

	// Send stop signal to all harvesters, then wait for them, for quick shutdown
	// TODO: Lock duration too long?
	p.mutex.Lock()
	for _, info := range p.prospectors {
		info.stop()
	}
	for _, info := range p.prospectors {
		info.wait()
	}
	p.mutex.Unlock()

	log.Info("Prospector exiting")
}

// runOnce handles a single prospector iteration
// Returns true if shutdown is necessary
func (p *Prospector) runOnce() bool {
	newlastscan := time.Now()
	p.iteration++ // Overflow is allowed

	for configKey, config := range p.fileConfigs {
		for _, path := range config.Paths {
			log.Debug("Scanning %s", path)
			p.scan(path, p.fileConfigs[configKey])
		}
	}

	// We only obey *fromBeginning (which is stored in this) on startup, if no
	// persist file exists. Afterwards we force from beginning
	p.fromBeginning = true

	// Clean up the prospector collections
	p.mutex.Lock()
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
			info.maybeOrphaned()
		}
		if info.orphaned == orphanedMaybe {
			info.setOrphaned()
			p.registrarSpool.Add(registrar.NewDeletedEvent(info))
		}
	}
	p.mutex.Unlock()

	// Flush the accumulated registrar events
	p.registrarSpool.Send()

	p.lastscan = newlastscan

	// Defer next scan for a bit
	now := time.Now()
	scanDeadline := now.Add(p.genConfig.ProspectInterval)

DelayLoop:
	for {
		select {
		case <-time.After(scanDeadline.Sub(now)):
			break DelayLoop
		case <-p.shutdownChan:
			return true
		case cfg := <-p.configChan:
			p.genConfig = cfg.GeneralPart("prospector").(*General)
			p.fileConfigs = cfg.Section("files").(Config)
		}

		now = time.Now()
		if now.After(scanDeadline) {
			break
		}
	}

	return false
}

// scan crawls a path for file movements
func (p *Prospector) scan(path string, cfg *FileConfig) {
	// Evaluate the path as a wildcards/shell glob
	matches, err := filepath.Glob(path)
	if err != nil {
		log.Errorf("glob(%s) failed: %v", path, err)
		return
	}

	// Check any matched files to see if we need to start a harvester
	for _, file := range matches {
		p.processFile(file, cfg)
	}
}

// processFile works out if a single discovered file has moved or is new etc.
func (p *Prospector) processFile(file string, cfg *FileConfig) {
	defer func() {
		p.mutex.Unlock()
	}()

	// TODO: More granular locking?
	p.mutex.Lock()

	// Check the current info against our index
	info, isKnown := p.prospectorindex[file]
	// Have we already processed this file in an earlier prospector declaration?
	// We do not support merging as it requires a full rewrite of how we handle file status
	if info.seenInIteration(p.iteration) {
		return
	}

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
				info.maybeOrphaned()
			} else if info.err.Error() == err.Error() {
				// The same error occurred - don't log it again
				info.update(nil, p.iteration)
				return
			}
		}

		// This is a new error
		info = newProspectorInfoInvalid(file, err)
		info.update(nil, p.iteration)

		// Print a friendly log message
		if _, ok := err.(*prospectorSkipError); ok {
			log.Info("Skipping %s: %s", file, err)
		} else {
			log.Errorf("Error prospecting %s: %s", file, err)
		}

		p.prospectors[info] = info
		p.prospectorindex[file] = info
		return
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
				return
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
			if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > cfg.DeadTime {
				// Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
				log.Info("Skipping file (older than dead time of %v): %s", cfg.DeadTime, file)

				// Store the offset that we should resume from if we notice a modification
				info.finishOffset = fileinfo.Size()
				p.registrarSpool.Add(registrar.NewDiscoverEvent(info, file, fileinfo.Size(), fileinfo))
			} else {
				// Process new file
				log.Info("Launching harvester on new file: %s", file)
				p.startHarvester(info, cfg)
			}
		}

		// Store the new entry
		p.prospectors[info] = info
	} else {
		if !info.identity.SameAs(fileinfo) {
			// Keep the old file in case we find it again shortly
			info.maybeOrphaned()

			if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
				// Symlinks could mean we see the same file twice - skip if we have
				if previousinfo == nil {
					p.flagDuplicateError(file, nil)
					return
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
				p.startHarvester(info, cfg)
			}

			// Store it
			p.prospectors[info] = info
		}
	}

	// Resume stopped harvesters
	resume := !info.isRunning()
	if resume {
		if info.status == statusResume {
			if info.finishOffset == fileinfo.Size() && time.Since(fileinfo.ModTime()) > cfg.DeadTime {
				// Old file with an unchanged offset, skip it
				log.Info("Skipping file (older than dead time of %v): %s", cfg.DeadTime, file)
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
		p.startHarvesterWithOffset(info, cfg, info.finishOffset)
	}

	p.prospectorindex[file] = info
}

// flagDuplicateError notes a file as a duplicate of another file (symlink?)
// and only reports an error to the log if it wasn't already noted before
func (p *Prospector) flagDuplicateError(file string, info *prospectorInfo) {
	// Have we already logged this error?
	if info != nil {
		if info.status == statusInvalid {
			if skipErr, ok := info.err.(*prospectorSkipError); ok && skipErr.message == "Duplicate" {
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

// startHarvester starts a new harvester against a file
func (p *Prospector) startHarvester(info *prospectorInfo, fileConfig *FileConfig) {
	var offset int64

	if p.fromBeginning {
		offset = 0
	} else {
		offset = info.identity.Stat().Size()
	}

	// Send a new file event to allow registrar to begin persisting for this harvester
	p.registrarSpool.Add(registrar.NewDiscoverEvent(info, info.file, offset, info.identity.Stat()))

	p.startHarvesterWithOffset(info, fileConfig, offset)
}

// startHarvesterWithOffset starts a new harvester against a file starting at
// the given offset
func (p *Prospector) startHarvesterWithOffset(info *prospectorInfo, fileConfig *FileConfig, offset int64) {
	// TODO: Hook in a shutdown channel (via context?)
	info.harvester = fileConfig.StreamConfig.NewHarvester(info.ctx, info.file, info.identity.Stat(), p.config, p.registrar, offset)
	info.running = true
	info.status = statusOk
	info.harvester.SetOutput(p.output)
	info.harvester.Start()
}

// lookupFileIds checks a file's filesystem identifiers against all other known
// files so we can handle file movements and renames
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

// initAPI sets up admin connectivity
func (p *Prospector) initAPI() {
	// Is admin loaded into the pipeline?
	if !p.adminConfig.Enabled {
		return
	}

	prospectorAPI := &apiNode{p: p}
	prospectorAPI.SetEntry("status", &apiStatus{p: p})

	p.adminConfig.SetEntry("prospector", prospectorAPI)
}
