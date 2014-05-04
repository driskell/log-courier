package main

import (
  "log"
  "os"
  "path/filepath"
  "time"
)

type ProspectorResume struct {
  files map[string]*FileState
  sync chan bool
}

type ProspectorInfo struct {
  fileinfo  os.FileInfo /* the file info */
  harvester chan int64  /* the harvester will send an event with its offset when it closes */
  last_seen uint32      /* int number of the last iterations in which we saw this file */
}

type Prospector struct {
  FileConfig     FileConfig
  prospectorinfo map[string]*ProspectorInfo
  iteration      uint32
  lastscan       time.Time
}

func (p *Prospector) Prospect(resume *ProspectorResume, registrar_chan chan []*RegistrarEvent, output chan *FileEvent) {
  p.prospectorinfo = make(map[string]*ProspectorInfo)

  // Handle any "-" (stdin) paths
  for i, path := range p.FileConfig.Paths {
    if path == "-" {
      // Offset and Initial never get used when path is "-"
      harvester := Harvester{Path: path, FileConfig: p.FileConfig}
      go harvester.Harvest(output)

      // Remove it from the file list
      p.FileConfig.Paths = append(p.FileConfig.Paths[:i], p.FileConfig.Paths[i+1:]...)
    }
  }

  // Seed last scan time
  p.lastscan = time.Now()

  // Now let's do one quick scan to pick up new files - flag true so new files obey from-beginning
  for _, path := range p.FileConfig.Paths {
    p.scan(path, registrar_chan, output, resume.files)
  }

  // Wait for registrar to save the first bulk
  <- resume.sync

  for {
    newlastscan := time.Now()

    for _, path := range p.FileConfig.Paths {
      // Scan - flag false so new files always start at beginning
      p.scan(path, registrar_chan, output, nil)
    }

    p.lastscan = newlastscan

    // Defer next scan for a bit.
    time.Sleep(10 * time.Second) // Make this tunable
  }
} /* Prospect */

func (p *Prospector) scan(path string, registrar_chan chan []*RegistrarEvent, output chan *FileEvent, resume map[string]*FileState) {
  // Rename / new file events to pass to registrar
  registrar_events := make([]*RegistrarEvent, 0)

  // Clear out files that disappeared and we've stopped harvesting
  for file, info := range p.prospectorinfo {
    if len(info.harvester) != 0 && info.last_seen < p.iteration {
      registrar_events = append(registrar_events, &RegistrarEvent{
        ProspectorInfo: info,
        Type:           EVENT_DELETED,
      })
      delete(p.prospectorinfo, file)
    }
  }

  p.iteration++ // Overflow is allowed

  // Evaluate the path as a wildcards/shell glob
  matches, err := filepath.Glob(path)
  if err != nil {
    log.Printf("glob(%s) failed: %v\n", path, err)
    return
  }

  // To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
  missinginfo := make(map[string]*ProspectorInfo)

  // Check any matched files to see if we need to start a harvester
  for _, file := range matches {
    // Stat the file, following any symlinks.
    fileinfo, err := os.Stat(file)
    // TODO(sissel): check err
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
      // Create a new prospector info with the stat info for comparison
      info = &ProspectorInfo{harvester: make(chan int64, 1)}

      // Check for dead time, but only if the file modification time is before the last scan started
      // This ensures we don't skip genuine creations with dead times less than 10s
      if previous, previousinfo := lookup_file_ids(file, fileinfo, p.prospectorinfo, missinginfo); previous != "" {
        // This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
        log.Printf("File rename was detected: %s -> %s\n", previous, file)

        info = previousinfo
        delete(p.prospectorinfo, previous)
        registrar_events = append(registrar_events, &RegistrarEvent{
          ProspectorInfo: info,
          Type:           EVENT_RENAMED,
          Source:         file,
          fileinfo:       &fileinfo,
        })
      } else if fileinfo.ModTime().Before(p.lastscan) && time.Since(fileinfo.ModTime()) > p.FileConfig.deadtime {
        var offset int64 = 0
        var is_resuming bool = false

        if resume != nil {
          // Call the calculator - it will process resume state if there is one
          offset, is_resuming = p.calculate_resume(file, fileinfo, resume)
        }

        // Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
        // This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
        // Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)
          registrar_events = append(registrar_events, &RegistrarEvent{
            ProspectorInfo: info,
            Type:           EVENT_NEWFILE,
            Source:         file,
            Offset:         offset,
            fileinfo:       &fileinfo,
          })

          harvester := &Harvester{ProspectorInfo: info, Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: info.harvester}
          go harvester.Harvest(output)
        } else {
          // Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
          log.Printf("Skipping file (older than dead time of %v): %s\n", p.FileConfig.deadtime, file)
          info.harvester <- fileinfo.Size()
          registrar_events = append(registrar_events, &RegistrarEvent{
            ProspectorInfo: info,
            Type:           EVENT_NEWFILE,
            Source:         file,
            Offset:         fileinfo.Size(),
            fileinfo:       &fileinfo,
          })
        }
      } else {
        var initial bool = false
        var offset int64 = 0
        var is_resuming bool = false

        if resume != nil {
          // Call the calculator - it will process resume state if there is one
          offset, is_resuming = p.calculate_resume(file, fileinfo, resume)
          initial = true
        }

        // Are we resuming a file or is this a completely new file?
        if is_resuming {
          log.Printf("Resuming harvester on a previously harvested file: %s\n", file)

          // By setting initial to false we ensure that offset is always obeyed, even on first scan, which is necessary for resume
          initial = false
        } else {
          log.Printf("Launching harvester on new file: %s\n", file)
        }

        // Launch the harvester - if initial is true it means ignore offset and choose end if this is first scan, and choose beginning if subsequence scan
        // This ensures we always pick up new logs from start - and only skip to end if we've just started up
        registrar_events = append(registrar_events, &RegistrarEvent{
          ProspectorInfo: info,
          Type:           EVENT_NEWFILE,
          Source:         file,
          Offset:         offset,
          fileinfo:       &fileinfo,
        })

        harvester := &Harvester{ProspectorInfo: info, Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: info.harvester, Initial: initial}
        go harvester.Harvest(output)
      }
    } else {
      if !os.SameFile(info.fileinfo, fileinfo) {
        // Keep the old file in missinginfo so we don't rescan it if it was renamed and we've not yet reached the new filename
        // We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
        missinginfo[file] = info

        if previous, previousinfo := lookup_file_ids(file, fileinfo, p.prospectorinfo, missinginfo); previous != "" {
          // This file was renamed from another file we know - link the same harvester channel as the old file
          log.Printf("File rename was detected: %s -> %s\n", previous, file)

          info = previousinfo
          registrar_events = append(registrar_events, &RegistrarEvent{
            ProspectorInfo: info,
            Type:           EVENT_RENAMED,
            Source:         file,
            fileinfo:       &fileinfo,
          })
        } else {
          // File is not the same file we saw previously, it must have rotated and is a new file
          log.Printf("Launching harvester on rotated file: %s\n", file)

          // Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
          info = &ProspectorInfo{harvester: make(chan int64, 1)}
          registrar_events = append(registrar_events, &RegistrarEvent{
            ProspectorInfo: info,
            Type:           EVENT_NEWFILE,
            Source:         file,
            Offset:         0,
            fileinfo:       &fileinfo,
          })

          // Start a harvester on the path
          harvester := &Harvester{ProspectorInfo: info, Path: file, FileConfig: p.FileConfig, FinishChan: info.harvester, Initial: (resume != nil)}
          go harvester.Harvest(output)
        }
      } else if len(info.harvester) != 0 && info.fileinfo.ModTime() != fileinfo.ModTime() {
        // Resume harvesting of an old file we've stopped harvesting from
        log.Printf("Resuming harvester on an old file that was just modified: %s\n", file)

        // Start a harvester on the path; an old file was just modified and it doesn't have a harvester
        // The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
        // Don't pass initial, we are resuming based on last offset
        harvester := &Harvester{ProspectorInfo: info, Path: file, FileConfig: p.FileConfig, Offset: <-info.harvester, FinishChan: info.harvester}
        go harvester.Harvest(output)
      }
    }

    // Update the fileinfo information used for future comparisons, and the last_seen counter
    info.fileinfo = fileinfo
    info.last_seen = p.iteration

    // Track the stat data for this file for later comparison to check for
    // rotation/etc
    p.prospectorinfo[file] = info
  } // for each file matched by the glob

  // Flush registrar events
  if len(registrar_events) != 0 {
    registrar_chan <- registrar_events
  }
}

func (p *Prospector) calculate_resume(file string, fileinfo os.FileInfo, resume map[string]*FileState) (int64, bool) {
  last_state, is_found := resume[file]

  if is_found && is_filestate_same(file, fileinfo, last_state) {
    // We're resuming - throw the last state back downstream so we resave it
    // And return the offset - also force harvest in case the file is old and we're about to skip it
    return last_state.Offset, true
  }

  if previous := lookup_file_ids_resume(file, fileinfo, resume); previous != "" {
    // File has rotated between shutdown and startup
    // We return last state downstream, with a modified event source with the new file name
    // And return the offset - also force harvest in case the file is old and we're about to skip it
    log.Printf("Detected rename of a previously harvested file: %s -> %s\n", previous, file)
    return last_state.Offset, true
  }

  if is_found {
    log.Printf("Not resuming rotated file: %s\n", file)
  }

  // New file so just start from an automatic position
  return 0, false
}
