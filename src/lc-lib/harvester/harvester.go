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
  "fmt"
  "io"
  "lc-lib/core"
  "math"
  "os"
  "time"
)

type HarvesterFinish struct {
  Last_Offset int64
  Error       error
}

type Harvester struct {
  stop_chan     chan interface{}
  return_chan   chan *HarvesterFinish
  snapshot_chan chan interface{}
  snapshot_sink chan *core.Snapshot
  stream        core.Stream
  fileinfo      os.FileInfo
  path          string
  fileconfig    *core.FileConfig
  offset        int64
  codec         core.Codec
  output        chan<- *core.EventDescriptor
  file          *os.File
  line_speed    float64
  byte_speed    float64
  line_count    uint64
  byte_count    uint64
  last_eof      time.Time
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
    stop_chan:     make(chan interface{}),
    return_chan:   make(chan *HarvesterFinish, 1),
    stream:        stream,
    fileinfo:      fileinfo,
    path:          path,
    fileconfig:    fileconfig,
    offset:        offset,
  }
}

func (h *Harvester) Start(output chan<- *core.EventDescriptor) {
  // Reset these channels on each harvester restart to ensure that any pending
  // snapshot request or response that was ignored/interrupted is cleared
  h.snapshot_chan = make(chan interface{}, 1)
  h.snapshot_sink = make(chan *core.Snapshot, 1)

  go func() {
    status := &HarvesterFinish{}
    status.Last_Offset, status.Error = h.harvest(output)
    h.return_chan <- status
  }()
}

func (h *Harvester) Stop() {
  close(h.stop_chan)
}

func (h *Harvester) OnFinish() <-chan *HarvesterFinish {
  return h.return_chan
}

func (h *Harvester) RequestSnapshot() {
  h.snapshot_chan <- 1
}

func (h *Harvester) OnSnapshot() <-chan *core.Snapshot {
  return h.snapshot_sink
}

func (h *Harvester) harvest(output chan<- *core.EventDescriptor) (int64, error) {
  if err := h.prepareHarvester(); err != nil {
    return h.offset, err
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

  // TODO: Make configurable?
  read_timeout := 10 * time.Second

  last_read_time := time.Now()
  last_line_count := uint64(0)
  last_byte_count := uint64(0)
  last_measurement := last_read_time
  seconds_without_events := 0

ReadLoop:
  for {
    text, bytesread, err := h.readline(reader, buffer)

    if duration := time.Since(last_measurement); duration >= time.Second {
      count := float64(h.line_count - last_line_count)

      if count == 0 {
        if seconds_without_events != 5 {
          seconds_without_events++
        }
      } else {
        seconds_without_events = 0
      }

      h.line_speed = h.calculateSpeed(duration, h.line_speed, count, seconds_without_events)
      h.byte_speed = h.calculateSpeed(duration, h.byte_speed, float64(h.byte_count - last_byte_count), seconds_without_events)

      last_byte_count = h.byte_count
      last_line_count = h.line_count
      last_measurement = time.Now()

      // Check shutdown
      select {
      case <-h.stop_chan:
        break ReadLoop
      case <-h.snapshot_chan:
        h.handleSnapshot()
      default:
      }
    }

    if err != nil {
      if err == io.EOF {
        // Check shutdown
        select {
        case <-h.stop_chan:
          break ReadLoop
        default:
        }

        h.last_eof = time.Now()

        // Timed out waiting for data, got EOF
        if h.path == "-" {
          // This wouldn't make sense on stdin so lets not risk anything strange happening
          continue
        }

        // Don't check for truncation until we hit the full read_timeout
        if time.Since(last_read_time) < read_timeout {
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
            return h.codec.Teardown(), nil
          }

          continue
        } else {
          log.Error("Unexpected error checking status of %s: %s", h.path, err)
        }
      } else {
        log.Error("Unexpected error reading from %s: %s", h.path, err)
      }
      return h.codec.Teardown(), err
    }

    line++
    line_offset := h.offset
    h.offset += int64(bytesread)

    // Codec is last - it forwards harvester state for us such as offset for resume
    h.codec.Event(line_offset, h.offset, line, text)

    last_read_time = time.Now()
    h.line_count++
    h.byte_count += uint64(bytesread)
  }

  log.Info("Harvester for %s exiting", h.path)
  return h.codec.Teardown(), nil
}

func (h *Harvester) calculateSpeed(duration time.Duration, speed float64, count float64, seconds_without_events int) float64 {
  if speed == 0. {
    return count
  }

  if seconds_without_events == 5 {
    return 0.
  }

  // Calculate a moving average over 5 seconds - use similiar weight as load average
  return count + math.Exp(float64(duration) / float64(time.Second) / -5.) * (speed - count)
}

func (h *Harvester) eventCallback(start_offset int64, end_offset int64, line uint64, text string) {
  event := &core.EventDescriptor{
    Stream: h.stream,
    Offset: end_offset,
    Event:  core.NewEvent(h.fileconfig.Fields, h.path, start_offset, line, text),
  }

EventLoop:
  for {
    select {
    case <-h.stop_chan:
      break EventLoop
    case <-h.snapshot_chan:
      h.handleSnapshot()
    case h.output <- event:
      break EventLoop
    }
  }
}

func (h *Harvester) prepareHarvester() error {
  // Special handling that "-" means to read from standard input
  if h.path == "-" {
    h.file = os.Stdin
    return nil
  }

  var err error
  h.file, err = h.openFile(h.path)
  if err != nil {
    log.Error("Failed opening %s: %s", h.path, err)
    return err
  }

  // Check we opened the right file
  info, err := h.file.Stat()
  if err != nil {
    h.file.Close()
    return err
  }

  if !os.SameFile(info, h.fileinfo) {
    h.file.Close()
    return fmt.Errorf("Not the same file")
  }

  // TODO: Check error?
  h.file.Seek(h.offset, os.SEEK_SET)

  return nil
}

func (h *Harvester) readline(reader *bufio.Reader, buffer *bytes.Buffer) (string, int, error) {
  var is_partial bool = true
  var newline_length int = 1

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

        return "", 0, err
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

func (h *Harvester) handleSnapshot() {
  ret := core.NewSnapshot("Harvester")
  ret.AddEntry("Speed (Lps)", h.line_speed)
  ret.AddEntry("Speed (Bps)", h.byte_speed)
  ret.AddEntry("Processed lines", h.line_count)
  ret.AddEntry("Last offset", h.offset)
  if h.last_eof.IsZero() {
    ret.AddEntry("Last EOF", "Never")
  } else {
    ret.AddEntry("Last EOF", h.last_eof)
  }

  if sub_snap := h.codec.Snapshot(); sub_snap != nil {
    ret.AddSub(sub_snap)
  }

  h.snapshot_sink <- ret
}
