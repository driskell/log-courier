/*
 * Copyright 2014 Jason Woods.
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
  "lc-lib/core"
  "lc-lib/harvester"
  "lc-lib/registrar"
  "lc-lib/spooler"
  "os"
  "path/filepath"
  "time"
)

type Prospector struct {
  core.PipelineSegment
  core.PipelineConfigReceiver
  core.PipelineSnapshotProvider

  generalconfig   *core.GeneralConfig
  fileconfigs     []core.FileConfig
  prospectorindex map[string]*prospectorInfo
  prospectors     map[*prospectorInfo]*prospectorInfo
  from_beginning  bool
  iteration       uint32
  lastscan        time.Time
  registrar       *registrar.Registrar
  registrar_spool *registrar.RegistrarEventSpool

  output chan<- *core.EventDescriptor
}

func NewProspector(pipeline *core.Pipeline, config *core.Config, from_beginning bool, registrar_imp *registrar.Registrar, spooler_imp *spooler.Spooler) (*Prospector, error) {
  ret := &Prospector{
    generalconfig:   &config.General,
    fileconfigs:     config.Files,
    prospectorindex: make(map[string]*prospectorInfo),
    prospectors:     make(map[*prospectorInfo]*prospectorInfo),
    from_beginning:  from_beginning,
    registrar:       registrar_imp,
    registrar_spool: registrar_imp.Connect(),
    output:          spooler_imp.Connect(),
  }

  if err := ret.init(); err != nil {
    return nil, err
  }

  pipeline.Register(ret)

  return ret, nil
}

func (p *Prospector) init() (err error) {
  var have_previous bool
  if have_previous, err = p.registrar.LoadPrevious(p.loadCallback); err != nil {
    return
  }

  if have_previous {
    // -from-beginning=false flag should only affect the very first run (no previous state)
    p.from_beginning = true

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

  // Handle any "-" (stdin) paths - but only once
  stdin_started := false
  for config_k, config := range p.fileconfigs {
    for i, path := range config.Paths {
      if path == "-" {
        if !stdin_started {
          // We need to check err - we cannot allow a nil stat
          stat, err := os.Stdin.Stat()
          if err != nil {
            log.Error("stat(Stdin) failed: %s", err)
            continue
          }

          // Stdin is implicitly an orphaned fileinfo
          info := newProspectorInfoFromFileInfo("-", stat)
          info.orphaned = Orphaned_Yes

          // Store the reference so we can shut it down later
          p.prospectors[info] = info

          // Start the harvester
          p.startHarvesterWithOffset(info, &p.fileconfigs[config_k], 0)

          stdin_started = true
        }

        // Remove it from the file list
        config.Paths = append(config.Paths[:i], config.Paths[i+1:]...)
      }
    }
  }

ProspectLoop:
  for {

    newlastscan := time.Now()
    p.iteration++ // Overflow is allowed

    for config_k, config := range p.fileconfigs {
      for _, path := range config.Paths {
        // Scan - flag false so new files always start at beginning
        p.scan(path, &p.fileconfigs[config_k])
      }
    }

    // We only obey *from_beginning (which is stored in this) on startup,
    // afterwards we force from beginning
    p.from_beginning = true

    // Clean up the prospector collections
    for _, info := range p.prospectors {
      if info.orphaned >= Orphaned_Maybe {
        if !info.isRunning() {
          delete(p.prospectors, info)
        }
      } else {
        if info.last_seen >= p.iteration {
          continue
        }
        delete(p.prospectorindex, info.file)
        info.orphaned = Orphaned_Maybe
      }
      if info.orphaned == Orphaned_Maybe {
        info.orphaned = Orphaned_Yes
        p.registrar_spool.Add(registrar.NewDeletedEvent(info))
      }
    }

    // Flush the accumulated registrar events
    p.registrar_spool.Send()

    p.lastscan = newlastscan

    // Defer next scan for a bit
    select {
    case <-time.After(p.generalconfig.ProspectInterval):
    case <-p.OnShutdown():
      break ProspectLoop
    case <-p.OnSnapshot():
      p.handleSnapshot()
    case config := <-p.OnConfig():
      p.generalconfig = &config.General
      p.fileconfigs = config.Files
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
  p.registrar_spool.Close()

  log.Info("Prospector exiting")
}

func (p *Prospector) scan(path string, config *core.FileConfig) {
  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Error("glob(%s) failed: %v", path, err)
    return
  }

  // Check any matched files to see if we need to start a harvester
  for _, file := range matches {
    // Stat the file, following any symlinks
    // TODO: Low priority. Trigger loadFileId here for Windows instead of
    //       waiting for Harvester or Registrar to do it
    fileinfo, err := os.Stat(file)
    if err != nil {
      log.Error("stat(%s) failed: %s", file, err)
      continue
    }

    if fileinfo.IsDir() {
      log.Info("Skipping directory: %s", file)
      continue
    }

    // Check the current info against our index
    info, is_known := p.prospectorindex[file]

    // Conditions for starting a new harvester:
    // - file path hasn't been seen before
    // - the file's inode or device changed
    if !is_known {
      // Check for dead time, but only if the file modification time is before the last scan started
      // This ensures we don't skip genuine creations with dead times less than 10s
      if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
        // This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
        log.Info("File rename was detected: %s -> %s", previous, file)
        info = previousinfo
        info.file = file

        p.registrar_spool.Add(registrar.NewRenamedEvent(info, file))
      } else {
        // This is a new entry
        info = newProspectorInfoFromFileInfo(file, fileinfo)

        if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > config.DeadTime {
          // Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
          log.Info("Skipping file (older than dead time of %v): %s", config.DeadTime, file)

          // Store the offset that we should resume from if we notice a modification
          info.finish_offset = fileinfo.Size()
          p.registrar_spool.Add(registrar.NewDiscoverEvent(info, file, fileinfo.Size(), fileinfo))
        } else {
          // Process new file
          log.Info("Launching harvester on new file: %s", file)
          p.startHarvester(info, config)
        }

        // Store the new entry
        p.prospectors[info] = info
      }
    } else {
      if !info.identity.SameAs(fileinfo) {
        // Keep the old file in case we find it again shortly
        info.orphaned = Orphaned_Maybe

        if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Info("File rename was detected: %s -> %s", previous, file)
          info = previousinfo
          info.file = file

          p.registrar_spool.Add(registrar.NewRenamedEvent(info, file))
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Info("Launching harvester on rotated file: %s", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          info = newProspectorInfoFromFileInfo(file, fileinfo)

          // Process new file
          p.startHarvester(info, config)

          // Store it
          p.prospectors[info] = info
        }
      }
    }

    // Resume stopped harvesters
    resume := !info.isRunning()
    if resume {
      if info.status == Status_Resume {
        // This is a filestate that was saved, resume the harvester
        log.Info("Resuming harvester on a previously harvested file: %s", file)
      } else if info.status == Status_Failed {
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
      p.startHarvesterWithOffset(info, config, info.finish_offset)
    }

    p.prospectorindex[file] = info
  } // for each file matched by the glob
}

func (p *Prospector) startHarvester(info *prospectorInfo, config *core.FileConfig) {
  var offset int64

  if p.from_beginning {
    offset = 0
  } else {
    offset = info.identity.Stat().Size()
  }

  // Send a new file event to allow registrar to begin persisting for this harvester
  p.registrar_spool.Add(registrar.NewDiscoverEvent(info, info.file, offset, info.identity.Stat()))

  p.startHarvesterWithOffset(info, config, offset)
}

func (p *Prospector) startHarvesterWithOffset(info *prospectorInfo, config *core.FileConfig, offset int64) {
  // TODO - hook in a shutdown channel
  info.harvester = harvester.NewHarvester(info, config, offset)
  info.running = true
  info.status = Status_Ok
  info.harvester.Start(p.output)
}

func (p *Prospector) lookupFileIds(file string, info os.FileInfo) (string, *prospectorInfo) {
  for _, ki := range p.prospectors {
    if ki.orphaned == Orphaned_No && ki.file == file {
      // We already know the prospector info for this file doesn't match, so don't check again
      continue
    }
    if ki.identity.SameAs(info) {
      // Found previous information, remove it and return it (it will be added again)
      delete(p.prospectors, ki)
      if ki.orphaned == Orphaned_No {
        delete(p.prospectorindex, ki.file)
      } else {
        ki.orphaned = Orphaned_No
      }
      return ki.file, ki
    }
  }

  return "", nil
}

func (p *Prospector) handleSnapshot() {
  snapshots := make([]*core.Snapshot, 0)

  for _, info := range p.prospectors {
    if !info.running {
      continue
    }
    info.requestSnapshot()
  }

  for _, info := range p.prospectorindex {
    var status string

    if info.status == Status_Failed {
      status = "Failed"
    } else {
      if info.running {
        status = "Running"
      } else {
        status = "Dead"
      }
    }

    snap := core.NewSnapshot(info.file)
    snap.AddEntry("Status", status)

    if info.running {
      if sub_snap := info.getSnapshot(); sub_snap != nil {
        snap.AddSub(sub_snap)
      }
    }

    snapshots = append(snapshots, snap)
  }

  var snap *core.Snapshot

  for _, info := range p.prospectors {
    if info.orphaned != Orphaned_No {
      continue
    }

    if snap == nil {
      snap = core.NewSnapshot(info.file + " (Orphaned)")
    }

    if sub_snap := info.getSnapshot(); sub_snap != nil {
      snap.AddSub(sub_snap)
    }
  }

  p.SendSnapshot(snapshots)
}
