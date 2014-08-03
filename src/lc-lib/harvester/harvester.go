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

package harvester

import (
  "bufio"
  "bytes"
  "io"
  "lc-lib/core"
  "os"
  "time"
)

type HarvesterStatus struct {
  Last_Offset int64
  Failed bool
}

type Harvester struct {
  stop_chan   chan interface{}
  return_chan chan *HarvesterStatus
  stream      core.Stream
  fileinfo    os.FileInfo
  path        string
  fileconfig  *core.FileConfig
  offset      int64
  codec       core.Codec
  output      chan<- *core.EventDescriptor
  file        *os.File
}

func NewHarvester(stream core.Stream, fileconfig *core.FileConfig, offset int64) *Harvester {
  var fileinfo os.FileInfo
  var path string
  if stream != nil {
    // Grab now so we can safely use them even if prospector changes them
    path, fileinfo = stream.Info()
  } else {
    // This is stdin
    path, fileinfo = "-", nil
  }
  return &Harvester{
    stop_chan:   make(chan interface{}),
    return_chan: make(chan *HarvesterStatus, 1),
    stream:      stream,
    fileinfo:    fileinfo,
    path:        path,
    fileconfig:  fileconfig,
    offset:      offset,
  }
}

func (h *Harvester) Start(output chan<- *core.EventDescriptor) {
  go func() {
    status := &HarvesterStatus{}
    status.Last_Offset, status.Failed = h.harvest(output)
    h.return_chan <- status
  }()
}

func (h *Harvester) Stop() {
  close(h.stop_chan)
}

func (h *Harvester) Status() <-chan *HarvesterStatus {
  return h.return_chan
}

func (h *Harvester) harvest(output chan<- *core.EventDescriptor) (int64, bool) {
  if !h.prepareHarvester() {
    return h.offset, true
  }

  h.output = output
  h.codec = h.fileconfig.CodecFactory.NewCodec(h.eventCallback, h.offset)

  defer h.file.Close()

  // NOTE(driskell): How would we know line number if from_beginning is false and we SEEK_END? Or would we scan,count,skip?
  var line uint64 = 0 // Ask registrar about the line number

  // Get current offset in file
  // TODO: Check error?
  offset, _ := h.file.Seek(0, os.SEEK_CUR)
  log.Info("Started harvester at position %d (requested %d): %s", offset, h.offset, h.path)
  h.offset = offset

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(h.file, 16<<10) // 16kb buffer by default
  buffer := new(bytes.Buffer)

  var read_timeout = 10 * time.Second
  last_read_time := time.Now()

ReadLoop:
  for {
    // Check shutdown
    select {
    case <-h.stop_chan:
      break ReadLoop
    default:
    }

    text, bytesread, err := h.readline(reader, buffer, read_timeout)

    if err != nil {
      if err == io.EOF {
        // Timed out waiting for data, got EOF, check to see if the file was truncated
        if h.path == "-" {
          // This wouldn't make sense on stdin so lets not risk anything strange happening
          continue
        }

        info, err := h.file.Stat()
        if err == nil {
          if info.Size() < h.offset {
            log.Warning("Unexpected file truncation, seeking to beginning: %s", h.path)
            h.file.Seek(0, os.SEEK_SET)
            h.offset = 0
            continue
          } else if age := time.Since(last_read_time); age > h.fileconfig.DeadTime {
            // if last_read_time was more than dead time, this file is probably dead. Stop watching it.
            log.Info("Stopping harvest of %s; last change was %v ago", h.path, age-(age%time.Second))
            // TODO: We should return a Stat() from before we attempted to read
            // In prospector we use that for comparison to resume
            // This prevents a potential race condition if we stop just as the
            // file is modified with extra lines...
            return h.codec.Teardown(), false
          }

          continue
        } else {
          log.Error("Unexpected error checking status of %s: %s", h.path, err)
        }
      } else {
        log.Error("Unexpected error reading from %s: %s", h.path, err)
      }
      return h.codec.Teardown(), true
    }
    last_read_time = time.Now()

    line++
    line_offset := h.offset
    h.offset += int64(bytesread)

    // Codec is last - it forwards harvester state for us such as offset for resume
    h.codec.Event(line_offset, h.offset, line, text)
  }

  log.Info("Harvester for %s exiting", h.path)
  return h.codec.Teardown(), false
}

func (h *Harvester) eventCallback(start_offset int64, end_offset int64, line uint64, text string) {
  event := &core.EventDescriptor{
    Stream: h.stream,
    Offset: end_offset,
    Event:  core.NewEvent(h.fileconfig.Fields, h.path, start_offset, line, text),
  }

  select {
  case <-h.stop_chan:
  case h.output <- event:
  }
}

func (h *Harvester) prepareHarvester() bool {
  // Special handling that "-" means to read from standard input
  if h.path == "-" {
    h.file = os.Stdin
    return true
  }

  var err error
  h.file, err = h.openFile(h.path)
  if err != nil {
    log.Error("Failed opening %s: %s", h.path, err)
    return false
  }

  // Check we opened the right file
  info, err := h.file.Stat()
  if err != nil || !os.SameFile(info, h.fileinfo) {
    h.file.Close()
    return false
  }

  // TODO: Check error?
  h.file.Seek(h.offset, os.SEEK_SET)

  return true
}

func (h *Harvester) readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (string, int, error) {
  var is_partial bool = true
  var newline_length int = 1
  start_time := time.Now()

  for {
    segment, err := reader.ReadBytes('\n')

    if segment != nil && len(segment) > 0 {
      if segment[len(segment)-1] == '\n' {
        // Found a complete line
        is_partial = false

        // Check if also a CR present
        if len(segment) > 1 && segment[len(segment)-2] == '\r' {
          newline_length++
        }
      }

      // TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
      buffer.Write(segment)
    }

    if err != nil {
      if err == io.EOF && is_partial {
        // Backoff
        select {
        case <-h.stop_chan:
          return "", 0, err
        case <-time.After(1 * time.Second):
        }

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {
          return "", 0, err
        }
        continue
      } else {
        log.Warning("%s", err)
        return "", 0, err // TODO(sissel): don't do this?
      }
    }

    // If we got a full line, return the whole line without the EOL chars (CRLF or LF)
    if !is_partial {
      // Get the str length with the EOL chars (LF or CRLF)
      buffer_size := buffer.Len()
      str := buffer.String()[:buffer_size-newline_length]
      // Reset the buffer for the next line
      buffer.Reset()
      return str, buffer_size, nil
    }
  } /* forever read chunks */

  return "", 0, nil
}
