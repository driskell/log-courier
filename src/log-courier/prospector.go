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

type ProspectorInfo struct {
  file          string
  identity      FileIdentity
  last_seen     uint32
  harvester_cb  chan int64
  status        int
  running       bool
  finish_offset int64
}

const (
  Status_Ok     = iota
  Status_Resume = iota
  Status_Failed = iota
)

func NewProspectorInfoFromFileState(file string, filestate *FileState) *ProspectorInfo {
  return &ProspectorInfo{
    file:          file,
    identity:      filestate,
    harvester_cb:  make(chan int64, 1),
    running:       false,
    status:        Status_Resume,
    finish_offset: filestate.Offset,
  }
}

func NewProspectorInfoFromFileInfo(file string, fileinfo os.FileInfo) *ProspectorInfo {
  return &ProspectorInfo{
    file:         file,
    identity:     &FileInfo{fileinfo: fileinfo},
    harvester_cb: make(chan int64, 1),
    running:      false,
  }
}

func (pi *ProspectorInfo) IsRunning() bool {
  if pi.running {
    select {
    case pi.finish_offset = <-pi.harvester_cb:
      pi.running = false
    default:
    }
  }
  return pi.running
}

func (pi *ProspectorInfo) Update(fileinfo os.FileInfo, iteration uint32) {
  // Allow identity to replace itself with a new identity (this allows a FileState to promote itself to a FileInfo)
  pi.identity.Update(fileinfo, &pi.identity)
  pi.last_seen = iteration
}

type Prospector struct {
  generalconfig    *GeneralConfig
  fileconfigs      []FileConfig
  prospectorinfo   map[string]*ProspectorInfo
  orphanedinfo     map[*ProspectorInfo]*ProspectorInfo
  from_beginning   bool
  iteration        uint32
  lastscan         time.Time
  registrar_events []RegistrarEvent
}

func NewProspector(config *Config) *Prospector {
  return &Prospector{
    generalconfig: &config.General,
    fileconfigs: config.Files,
  }
}

func (p *Prospector) Prospect(resume map[string]*ProspectorInfo, registrar_chan chan<- []RegistrarEvent, output chan<- *FileEvent) {
  // Pre-populate prospectorinfo with what we had previously
  p.prospectorinfo = resume

  // Handle any "-" (stdin) paths - but only once
  stdin_started := false
  for _, config := range p.fileconfigs {
    for i, path := range config.Paths {
      if path == "-" {
        if !stdin_started {
          // Offset and Initial never get used when path is "-"
          harvester := NewHarvester(nil, &config, 0)
          go harvester.Harvest(output)
          stdin_started = true
        }

        // Remove it from the file list
        config.Paths = append(config.Paths[:i], config.Paths[i+1:]...)
      }
    }
  }

  p.registrar_events = make([]RegistrarEvent, 0)

  // To keep orphaned prospectors resulting from rename detections etc
  p.orphanedinfo = make(map[*ProspectorInfo]*ProspectorInfo)

  // On first scan use from_beginning setting, then always from beginning
  p.from_beginning = *from_beginning

  for {

    newlastscan := time.Now()
    p.iteration++ // Overflow is allowed

    for _, config := range p.fileconfigs {
      for _, path := range config.Paths {
        // Scan - flag false so new files always start at beginning
        p.scan(path, &config, registrar_chan, output)
      }
    }

    p.from_beginning = true

    // Clear out entries that no longer exist and we've stopped harvesting
    for file, info := range p.prospectorinfo {
      if !info.IsRunning() && info.last_seen < p.iteration {
        p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: info})
        delete(p.prospectorinfo, file)
      }
    }
    for _, info := range p.orphanedinfo {
      if !info.IsRunning() {
        p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: info})
        delete(p.orphanedinfo, info)
      }
    }

    // Flush the accumulated registrar events
    if len(p.registrar_events) != 0 {
      registrar_chan <- p.registrar_events
    }

    p.lastscan = newlastscan
    p.registrar_events = make([]RegistrarEvent, 0)

    // Defer next scan for a bit.
    time.Sleep(p.generalconfig.ProspectInterval)
  }
}

func (p *Prospector) scan(path string, config *FileConfig, registrar_chan chan<- []RegistrarEvent, output chan<- *FileEvent) {
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

    // Check the current info against p.prospectorinfo[file]
    info, is_known := p.prospectorinfo[file]

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
      } else if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > config.DeadTime {
        // Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
        log.Printf("Skipping file (older than dead time of %v): %s\n", config.DeadTime, file)
        info = NewProspectorInfoFromFileInfo(file, fileinfo)

        // Store the offset that we should resume from if we notice a modification
        info.finish_offset = fileinfo.Size()

        p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: file, Offset: fileinfo.Size(), fileinfo: fileinfo})
      } else {
        log.Printf("Launching harvester on new file: %s\n", file)
        info = NewProspectorInfoFromFileInfo(file, fileinfo)

        // Process new file
        p.startHarvester(info, output, config)
      }
    } else {
      if !info.identity.SameAs(fileinfo) {
        // Keep the old file in orphanedinfo in case we find it again shortly
        p.orphanedinfo[info] = info

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

    p.prospectorinfo[file] = info
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
  for kf, ki := range p.prospectorinfo {
    // We already know the prospector info for this file doesn't match, so don't check again
    if kf == file {
      continue
    }
    if ki.identity.SameAs(info) {
      // OK must be a rename, so remove this entry from prospector info and return it, it'll be added again
      delete(p.prospectorinfo, kf)
      return kf, ki
    }
  }

  // Now check the missingfiles
  for _, ki := range p.orphanedinfo {
    if ki.identity.SameAs(info) {
      delete(p.orphanedinfo, ki)
      return ki.file, ki
    }
  }

  return "", nil
}
