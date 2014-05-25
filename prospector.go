package main

import (
  "log"
  "os"
  "path/filepath"
  "time"
)

type ProspectorInfo struct {
  fileinfo    os.FileInfo /* the file info */
  last_offset chan int64  /* the harvester will send an event with its offset when it closes */
  failed      bool        /* whether the harvester failed to start or finished successfully */
  last_seen   uint32      /* the last iteration ID in which we saw this file */
}

type ProspectorResume struct {
  prospectorinfo *ProspectorInfo /* the prospectorinfo the registrar is using */
  filestate      *FileState      /* the saved file state */
}

type Prospector struct {
  FileConfigs      []FileConfig
  prospectorinfo   map[string]*ProspectorInfo
  missinginfo      map[string]*ProspectorInfo
  iteration        uint32
  lastscan         time.Time
  registrar_events []RegistrarEvent
}

func (p *Prospector) Prospect(resume map[string]*ProspectorResume, registrar_chan chan []RegistrarEvent, output chan *FileEvent) {
  p.prospectorinfo = make(map[string]*ProspectorInfo)

  // Handle any "-" (stdin) paths - but only once
  stdin_started := false
  for _, config := range p.FileConfigs {
    for i, path := range config.Paths {
      if path == "-" {
        if !stdin_started {
          // Offset and Initial never get used when path is "-"
          harvester := Harvester{Path: path, FileConfig: config}
          go harvester.Harvest(output)
          stdin_started = true
        }

        // Remove it from the file list
        config.Paths = append(config.Paths[:i], config.Paths[i+1:]...)
      }
    }
  }

  newlastscan := time.Now()
  p.registrar_events = make([]RegistrarEvent, 0)

  // To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
  p.missinginfo = make(map[string]*ProspectorInfo)

  // Now let's do one quick scan to pick up new files - flag true so new files obey from-beginning
  for _, config := range p.FileConfigs {
    for _, path := range config.Paths {
      p.scan(path, config, registrar_chan, output, resume)
    }
  }

  // Send deletion events for resumes that weren't used
  for _, resume_entry := range resume {
    p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: resume_entry.prospectorinfo})
  }

  for {
    // Flush registrar events
    if len(p.registrar_events) != 0 {
      registrar_chan <- p.registrar_events
    }

    p.lastscan = newlastscan
    p.registrar_events = make([]RegistrarEvent, 0)

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable

    newlastscan = time.Now()
    p.iteration++ // Overflow is allowed

    // To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
    p.missinginfo = make(map[string]*ProspectorInfo)

    for _, config := range p.FileConfigs {
      for _, path := range config.Paths {
        // Scan - flag false so new files always start at beginning
        p.scan(path, config, registrar_chan, output, nil)
      }
    }

    // Clear out files that disappeared and we've stopped harvesting
    for file, info := range p.prospectorinfo {
      if len(info.last_offset) != 0 && info.last_seen < p.iteration {
        p.registrar_events = append(p.registrar_events, &DeletedEvent{ProspectorInfo: info})
        delete(p.prospectorinfo, file)
      }
    }
  }
} /* Prospect */

func (p *Prospector) scan(path string, config FileConfig, registrar_chan chan []RegistrarEvent, output chan *FileEvent, resume map[string]*ProspectorResume) {
  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

  // Check any matched files to see if we need to start a harvester
  for _, file := range matches {
    // Stat the file, following any symlinks.
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
        p.registrar_events = append(p.registrar_events, &RenamedEvent{ProspectorInfo: info, Source: file})
      } else if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > config.deadtime {
        var offset int64 = 0
        var is_resuming bool = false

        if resume != nil {
          // Call the calculator - it will process resume state if there is one
          offset, info = p.calculateResume(file, fileinfo, resume)
        }

        if info == nil {
          info = &ProspectorInfo{last_offset: make(chan int64, 1)}
        } else {
          is_resuming = true
        }

        // Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
        // This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
        // Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)

          p.spawnHarvester(info, output, &Harvester{ProspectorInfo: info, FileInfo: fileinfo, Path: file, FileConfig: config, Offset: offset})
        } else {
          // Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", config.deadtime, file)
          info.last_offset <- fileinfo.Size()
          info.failed = false
          p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: file, Offset: fileinfo.Size(), fileinfo: &fileinfo})
        }
      } else {
        var initial bool = false
        var offset int64 = 0
        var is_resuming bool = false

        if resume != nil {
          // Call the calculator - it will process resume state if there is one
          offset, info = p.calculateResume(file, fileinfo, resume)
          initial = true
        }

        if info == nil {
          info = &ProspectorInfo{last_offset: make(chan int64, 1)}
        } else {
          is_resuming = true
        }

        // Are we resuming a file or is this a completely new file?
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)

          // By setting initial to false we ensure that offset is always obeyed, even on first scan, which is necessary for resume
          initial = false
        } else {
          log.Printf("Launching harvester on new file: %s\n", file)
          p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: file, Offset: offset, fileinfo: &fileinfo})
        }

        // Launch the harvester - if initial is true it means ignore offset and choose end if this is first scan, and choose beginning if subsequence scan
        // This ensures we always pick up new logs from start - and only skip to end if we've just started up
        p.spawnHarvester(info, output, &Harvester{ProspectorInfo: info, FileInfo: fileinfo, Path: file, FileConfig: config, Offset: offset, Initial: initial})
      }
    } else {
      if !os.SameFile(info.fileinfo, fileinfo) {
        // Keep the old file in missinginfo so we don't rescan it if it was renamed and we've not yet reached the new filename
        // We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
        p.missinginfo[file] = info

        if previous, previousinfo := p.lookupFileIds(file, fileinfo); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Printf("File rename was detected: %s -> %s\n", previous, file)

          info = previousinfo
          p.registrar_events = append(p.registrar_events, &RenamedEvent{ProspectorInfo: info, Source: file})
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Printf("Launching harvester on rotated file: %s\n", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          info = &ProspectorInfo{last_offset: make(chan int64, 1)}
          p.registrar_events = append(p.registrar_events, &NewFileEvent{ProspectorInfo: info, Source: file, Offset: 0, fileinfo: &fileinfo})

          // Start a harvester on the path
          p.spawnHarvester(info, output, &Harvester{ProspectorInfo: info, FileInfo: fileinfo, Path: file, FileConfig: config, Initial: (resume != nil)})
        }
      } else if len(info.last_offset) != 0 {  // This should be safe since we are the only receiver
        restart := false

        if info.fileinfo.ModTime() != fileinfo.ModTime() {
          // Resume harvesting of an old file we've stopped harvesting from
          log.Printf("Resuming harvester on an old file that was just modified: %s\n", file)

          restart = true
        } else if info.failed {
          // Last attempt we failed to start, try again
          log.Printf("Attempting to restart failed harvester: %s\n", file)

          restart = true
        }

        if (restart) {
          // The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
          p.spawnHarvester(info, output, &Harvester{ProspectorInfo: info, FileInfo: fileinfo, Path: file, FileConfig: config, Offset: <-info.last_offset})
        }
      }
    }

    // Update the fileinfo information used for future comparisons, and the last_seen counter
    info.fileinfo = fileinfo
    info.last_seen = p.iteration

    // Track the stat data for this file for later comparison to check for
    // rotation/etc
    p.prospectorinfo[file] = info
  } // for each file matched by the glob
}

func (p *Prospector) spawnHarvester(info *ProspectorInfo, output chan *FileEvent, harvester *Harvester) {
  go func() {
    var offset int64
    offset, info.failed = harvester.Harvest(output)
    info.last_offset <- offset
  }()
}

func (p *Prospector) calculateResume(file string, fileinfo os.FileInfo, resume map[string]*ProspectorResume) (int64, *ProspectorInfo) {
  last_state, is_found := resume[file]

  if is_found && is_filestate_same(file, fileinfo, last_state.filestate) {
    // We're resuming, return the offset
    delete(resume, file)
    return last_state.filestate.Offset, last_state.prospectorinfo
  }

  if previous := p.lookupFileIdsResume(file, fileinfo, resume); previous != "" {
    // File has rotated between shutdown and startup, return offset to resume
    log.Printf("Detected rename of a previously harvested file: %s -> %s\n", previous, file)
    delete(resume, previous)
    return last_state.filestate.Offset, last_state.prospectorinfo
  }

  if is_found {
    log.Printf("Not resuming rotated file: %s\n", file)
  }

  // New file so just start from an automatic position
  return 0, nil
}

func (p *Prospector) lookupFileIds(file string, info os.FileInfo) (string, *ProspectorInfo) {
  for kf, ki := range p.prospectorinfo {
    // We already know the prospector info for this file doesn't match, so don't check again
    if kf == file {
      continue
    }
    if os.SameFile(ki.fileinfo, info) {
      // OK must be a rename, so remove this entry from prospector info and return it, it'll be added again
      delete(p.prospectorinfo, kf)
      return kf, ki
    }
  }

  // Now check the missingfiles
  for kf, ki := range p.missinginfo {
    if os.SameFile(ki.fileinfo, info) {
      delete(p.missinginfo, kf)
      return kf, ki
    }
  }

  return "", nil
}

func (p *Prospector) lookupFileIdsResume(file string, info os.FileInfo, initial map[string]*ProspectorResume) string {
  for kf, ki := range initial {
    if kf == file {
      continue
    }
    if is_filestate_same(file, info, ki.filestate) {
      return kf
    }
  }

  return ""
}
