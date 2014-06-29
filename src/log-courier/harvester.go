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
  "bufio"
  "bytes"
  "io"
  "log"
  "os"
  "time"
)

type Harvester struct {
  info       *ProspectorInfo
  fileinfo   os.FileInfo
  path       string /* the file path to harvest */
  fileconfig *FileConfig
  offset     int64
  codec      Codec

  file *os.File /* the file being watched */
}

func NewHarvester(info *ProspectorInfo, fileconfig *FileConfig, offset int64) *Harvester {
  var fileinfo os.FileInfo
  var path string
  if info != nil {
    // Grab now so we can safely use them even if prospector changes them
    fileinfo = info.identity.Stat()
    path = info.file
  } else {
    // This is stdin
    fileinfo = nil
    path = "-"
  }
  return &Harvester{
    info:       info,
    fileinfo:   fileinfo,
    path:       path,
    fileconfig: fileconfig,
    offset:     offset,
  }
}

func (h *Harvester) Harvest(output chan<- *FileEvent) (int64, bool) {
  if !h.prepareHarvester() {
    return h.offset, true
  }

  h.codec = h.fileconfig.codec.NewCodec(h.path, h.fileconfig, h.info, h.offset, output)

  defer h.file.Close()

  // NOTE(driskell): How would we know line number if from_beginning is false and we SEEK_END? Or would we scan,count,skip?
  var line uint64 = 0 // Ask registrar about the line number

  // Get current offset in file
  // TODO: Check error?
  offset, _ := h.file.Seek(0, os.SEEK_CUR)
  log.Printf("Started harvester at position %d (requested %d): %s\n", offset, h.offset, h.path)
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
    case <-h.info.ShutdownSignal():
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
            log.Printf("File truncated, seeking to beginning: %s\n", h.path)
            h.file.Seek(0, os.SEEK_SET)
            h.offset = 0
            continue
          } else if age := time.Since(last_read_time); age > h.fileconfig.DeadTime {
            // if last_read_time was more than dead time, this file is probably dead. Stop watching it.
            log.Printf("Stopping harvest of %s; last change was %v ago\n", h.path, age-(age%time.Second))
            // TODO: We should return a Stat() from before we attempted to read
            // In prospector we use that for comparison to resume
            // This prevents a potential race condition if we stop just as the
            // file is modified with extra lines...
            return h.codec.Teardown(), false
          }

          continue
        } else {
          log.Printf("Unexpected error checking status of %s: %s\n", h.path, err)
        }
      } else {
        log.Printf("Unexpected error reading from %s: %s\n", h.path, err)
      }
      return h.codec.Teardown(), true
    }
    last_read_time = time.Now()

    line++
    line_offset := h.offset
    h.offset += int64(bytesread)

    // Codec is last - it saves harvester state for us such as offset for resume
    h.codec.Event(line_offset, h.offset, line, text)
  }

  log.Printf("Harvester shutdown for %s complete\n", h.path)
  return h.codec.Teardown(), false
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
    log.Printf("Failed opening %s: %s\n", h.path, err)
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

func (h *Harvester) readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (*string, int, error) {
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
        case <-h.info.ShutdownSignal():
          return nil, 0, err
        case <-time.After(1 * time.Second):
        }

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {
          return nil, 0, err
        }
        continue
      } else {
        log.Println(err)
        return nil, 0, err // TODO(sissel): don't do this?
      }
    }

    // If we got a full line, return the whole line without the EOL chars (CRLF or LF)
    if !is_partial {
      // Get the str length with the EOL chars (LF or CRLF)
      bufferSize := buffer.Len()
      str := new(string)
      *str = buffer.String()[:bufferSize-newline_length]
      // Reset the buffer for the next line
      buffer.Reset()
      return str, bufferSize, nil
    }
  } /* forever read chunks */

  return nil, 0, nil
}
