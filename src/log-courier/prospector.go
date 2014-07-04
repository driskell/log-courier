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

package main

import (
  "log"
  "os"
  "path/filepath"
  "time"
)

const (
  Status_Ok     = iota
  Status_Resume = iota
  Status_Failed = iota
)

type ProspectorInfo struct {
  file           string
  identity       FileIdentity
  last_seen      uint32
  harvester_cb   chan int64
  harvester_stop chan interface{}
  status         int
  running        bool
  orphaned       bool
  finish_offset  int64
}

func NewProspectorInfoFromFileState(file string, filestate *FileState) *ProspectorInfo {
  return &ProspectorInfo{
    file:           file,
    identity:       filestate,
    harvester_cb:   make(chan int64, 1),
    harvester_stop: make(chan interface{}),
    status:         Status_Resume,
    finish_offset:  filestate.Offset,
  }
}

func NewProspectorInfoFromFileInfo(file string, fileinfo os.FileInfo) *ProspectorInfo {
  return &ProspectorInfo{
    file:           file,
    identity:       &FileInfo{fileinfo: fileinfo}, // fileinfo is nil for stdin
    harvester_cb:   make(chan int64, 1),
    harvester_stop: make(chan interface{}),
  }
}

func (pi *ProspectorInfo) IsRunning() bool {
  if !pi.running {
    return false
  }
  select {
  case pi.finish_offset = <-pi.harvester_cb:
    pi.running = false
  default:
  }
  return pi.running
}

func (pi *ProspectorInfo) ShutdownSignal() <-chan interface{} {
  return pi.harvester_stop
}

func (pi *ProspectorInfo) Stop() {
  if !pi.running {
    return
  }
  if pi.file == "-" {
    // Just in case someone started us outside a pipeline with stdin
    // to stop confusion at why we don't exit after Ctrl+C
    // There's no deadline on Stdin reads :-(
    log.Printf("Waiting for Stdin to close (EOF or Ctrl+D)\n")
  }
  close(pi.harvester_stop)
}

func (pi *ProspectorInfo) Wait() {
  if !pi.running {
    return
  }
  pi.finish_offset = <-pi.harvester_cb
  pi.running = false
}

func (pi *ProspectorInfo) Update(fileinfo os.FileInfo, iteration uint32) {
  // Allow identity to replace itself with a new identity (this allows a FileState to promote itself to a FileInfo)
  pi.identity.Update(fileinfo, &pi.identity)
  pi.last_seen = iteration
}

type Prospector struct {
  control          *LogCourierControl
  generalconfig    *GeneralConfig
  fileconfigs      []FileConfig
  prospectorindex  map[string]*ProspectorInfo
  prospectors      map[*ProspectorInfo]*ProspectorInfo
  from_beginning   bool
  iteration        uint32
  lastscan         time.Time
  registrar        *Registrar
  registrar_chan   chan<- []RegistrarEvent
  registrar_events []RegistrarEvent
}

func NewProspector(config *Config, from_beginning bool, registrar *Registrar, control *LogCourierMasterControl) *Prospector {
  return &Prospector{
    control: control.RegisterWithRecvConfig(),
    generalconfig: &config.General,
    fileconfigs: config.Files,
    from_beginning: from_beginning,
    registrar: registrar,
    registrar_chan: registrar.Connect(),
    registrar_events: make([]RegistrarEvent, 0),
  }
}

func (p *Prospector) Prospect(output chan<- *FileEvent) {
  defer func() {
    p.control.Done()
  }()

  p.prospectors = make(map[*ProspectorInfo]*ProspectorInfo)
  p.prospectorindex = p.registrar.LoadPrevious()
  if p.prospectorindex == nil {
    // No previous state to follow, obey -from-beginning and start empy
    p.prospectorindex = make(map[string]*ProspectorInfo)
  } else {
    // -from-beginning=false flag should only affect the very first run (no previous state)
    p.from_beginning = true

    // Pre-populate prospectors with what we had previously
    for _, v := range p.prospectorindex {
      p.prospectors[v] = v
    }
  }

  // Handle any "-" (stdin) paths - but only once
  stdin_started := false
  for config_k, config := range p.fileconfigs {
    for i, path := range config.Paths {
      if path == "-" {
        if !stdin_started {
          // We need to check err - we cannot allow a nil stat
          stat, err := os.Stdin.Stat()
          if err != nil {
            log.Printf("stat(Stdin) failed: %s\n", err)
            continue
          }

          // Stdin is implicitly an orphaned fileinfo
          info := NewProspectorInfoFromFileInfo("-", stat)
          info.orphaned = true

          // Store the reference so we can shut it down later
          p.prospectors[info] = info

          // Start the harvester
          p.startHarvesterWithOffset(info, output, &p.fileconfigs[config_k], 0)

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
        p.scan(path, &p.fileconfigs[config_k], output)
      }
    }

    // We only obey *from_beginning (which is stored in this) on startup,
    // afterwards we force from beginning
    p.from_beginning = true

    // Clear out entries that no longer exist and we've stopped harvesting
    for _, info := range p.prospectors {
      if info.IsRunning() {
        continue
      }
      if !info.orphaned {
        if info.last_seen >= p.iteration {
          continue
        }
        delete(p.prospectorindex, info.file)
        p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: info})
      }
    }

    // Flush the accumulated registrar events
    if len(p.registrar_events) != 0 {
      p.registrar_chan <- p.registrar_events
      p.registrar_events = make([]RegistrarEvent, 0)
    }

    p.lastscan = newlastscan

    // Defer next scan for a bit
    select {
      case <-time.After(p.generalconfig.ProspectInterval):
      case <-p.control.ShutdownSignal():
        break ProspectLoop
      case config := <-p.control.RecvConfig():
        p.generalconfig = &config.General
        p.fileconfigs = config.Files
    }
  }

  // Send stop signal to all harvesters, then wait for them, for quick shutdown
  for _, info := range p.prospectors {
    info.Stop()
  }
  for _, info := range p.prospectors {
    info.Wait()
  }

  // Disconnect from the registrar
  p.registrar.Disconnect()

  log.Printf("Prospector shutdown complete\n")
}

func (p *Prospector) scan(path string, config *FileConfig, output chan<- *FileEvent) {
  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

  // Check any matched files to see if we need to start a harvester
  for _, file := range matches {
    // Stat the file, following any symlinks
    // TODO: Low priority. Trigger loadFileId here for Windows instead of
    //       waiting for Harvester or Registrar to do it
    fileinfo, err := os.Stat(file)
    if err != nil {
      log.Printf("stat(%s) failed: %s\n", file, err)
      continue
    }

    if fileinfo.IsDir() {
      log.Printf("Skipping directory: %s\n", file)
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
        log.Printf("File rename was detected: %s -> %s\n", previous, file)
        info = previousinfo
        info.file = file

        p.registrar_events = append(p.registrar_events, &RenamedEvent{ProspectorInfo: info, Source: file})
      } else {
        // This is a new entry
        info = NewProspectorInfoFromFileInfo(file, fileinfo)

        if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > config.DeadTime {
          // Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", config.DeadTime, file)

          // Store the offset that we should resume from if we notice a modification
          info.finish_offset = fileinfo.Size()
          p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: file, Offset: fileinfo.Size(), fileinfo: fileinfo})
        } else {
          // Process new file
          log.Printf("Launching harvester on new file: %s\n", file)
          p.startHarvester(info, output, config)
        }

        // Store the new entry
        p.prospectors[info] = info
      }
    } else {
      if !info.identity.SameAs(fileinfo) {
        // Keep the old file in case we find it again shortly
        info.orphaned = true

        // Remove the orphan from registrar to prevent its updated overwriting saved state of this new file
        p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: info})

        if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Printf("File rename was detected: %s -> %s\n", previous, file)
          info = previousinfo
          info.file = file

          p.registrar_events = append(p.registrar_events, &RenamedEvent{ProspectorInfo: info, Source: file})
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Printf("Launching harvester on rotated file: %s\n", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          info = NewProspectorInfoFromFileInfo(file, fileinfo)

          // Process new file
          p.startHarvester(info, output, config)

          // Store it
          p.prospectors[info] = info
        }
      }
    }

    // Resume stopped harvesters
    resume := !info.IsRunning()
    if resume {
      if info.status == Status_Resume {
        // This is a filestate that was saved, resume the harvester
        log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
      } else if info.status == Status_Failed {
        // Last attempt we failed to start, try again
        log.Printf("Attempting to restart failed harvester: %s\n", file)
      } else if info.identity.Stat().ModTime() != fileinfo.ModTime() {
        // Resume harvesting of an old file we've stopped harvesting from
        log.Printf("Resuming harvester on an old file that was just modified: %s\n", file)
      } else {
        resume = false
      }
    }

    info.Update(fileinfo, p.iteration)

    if resume {
      p.startHarvesterWithOffset(info, output, config, info.finish_offset)
    }

    p.prospectorindex[file] = info
  } // for each file matched by the glob
}

func (p *Prospector) startHarvester(info *ProspectorInfo, output chan<- *FileEvent, config *FileConfig) {
  var offset int64

  if p.from_beginning {
    offset = 0
  } else {
    offset = info.identity.Stat().Size()
  }

  // Send a new file event to allow registrar to begin persisting for this harvester
  p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: info.file, Offset: offset, fileinfo: info.identity.Stat()})

  p.startHarvesterWithOffset(info, output, config, offset)
}

func (p *Prospector) startHarvesterWithOffset(info *ProspectorInfo, output chan<- *FileEvent, config *FileConfig, offset int64) {
  // TODO - hook in a shutdown channel
  harvester := NewHarvester(info, config, offset)
  info.running = true
  info.status = Status_Ok
  go func() {
    offset, failed := harvester.Harvest(output)
    if failed {
      info.status = Status_Failed
    }
    info.harvester_cb <- offset
  }()
}

func (p *Prospector) lookupFileIds(file string, info os.FileInfo) (string, *ProspectorInfo) {
  for _, ki := range p.prospectors {
    if !ki.orphaned && ki.file == file {
      // We already know the prospector info for this file doesn't match, so don't check again
      continue
    }
    if ki.identity.SameAs(info) {
      // Found previous information, remove it and return it (it will be added again)
      delete(p.prospectors, ki)
      if !ki.orphaned {
        delete(p.prospectorindex, ki.file)
      }
      return ki.file, ki
    }
  }

  return "", nil
}
