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
  ProspectorInfo *ProspectorInfo
  FileInfo       os.FileInfo
  Path           string /* the file path to harvest */
  FileConfig     FileConfig
  Offset         int64
  Initial        bool
  Codec          Codec

  file *os.File /* the file being watched */
}

func (h *Harvester) Harvest(output chan *FileEvent) (int64, bool) {
  if !h.open() {
    return h.Offset, true
  }

  h.Codec = h.FileConfig.codec.Create(h, output)

  defer h.file.Close()

  // NOTE(driskell): How would we know line number if from_beginning is false and we SEEK_END? Or would we scan,count,skip?
  var line uint64 = 0 // Ask registrar about the line number

  // get current offset in file
  offset, _ := h.file.Seek(0, os.SEEK_CUR)

  if h.Initial {
    if *from_beginning {
      log.Printf("Started harvester at position %d (requested beginning): %s\n", offset, h.Path)
    } else {
      log.Printf("Started harvester at position %d (requested end): %s\n", offset, h.Path)
    }
  } else {
    log.Printf("Started harvester at position %d (requested %d): %s\n", offset, h.Offset, h.Path)
  }

  h.Offset = offset

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(h.file, 16<<10) // 16kb buffer by default
  buffer := new(bytes.Buffer)

  var read_timeout = 10 * time.Second
  last_read_time := time.Now()
  for {
    text, bytesread, err := h.readline(reader, buffer, read_timeout)

    if err != nil {
      if err == io.EOF {
        // Timed out waiting for data, got EOF, check to see if the file was truncated
        if h.Path == "-" {
          // This wouldn't make sense on stdin so lets not risk anything strange happening
          continue
        }

        info, err := h.file.Stat()
        if err == nil {
          if info.Size() < h.Offset {
            log.Printf("File truncated, seeking to beginning: %s\n", h.Path)
            h.file.Seek(0, os.SEEK_SET)
            h.Offset = 0
            continue
          } else if age := time.Since(last_read_time); age > h.FileConfig.deadtime {
            // if last_read_time was more than dead time, this file is probably dead. Stop watching it.
            log.Printf("Stopping harvest of %s; last change was %v ago\n", h.Path, age-(age%time.Second))
            return h.Codec.Teardown(), false
          }
          continue
        } else {
          log.Printf("Unexpected error checking status of %s: %s\n", h.Path, err)
        }
      } else {
        log.Printf("Unexpected error reading from %s: %s\n", h.Path, err)
      }
      return h.Codec.Teardown(), true
    }
    last_read_time = time.Now()

    line++
    h.Offset += int64(bytesread)

    h.Codec.Event(line, text)
  } /* forever */
}

func (h *Harvester) open() bool {
  // Special handling that "-" means to read from standard input
  if h.Path == "-" {
    h.file = os.Stdin
    return true
  }

  var err error
  h.file, err = open_file_no_lock(h.Path)

  if err != nil {
    log.Printf("Failed opening %s: %s\n", h.Path, err)
    return false
  }

  // Check we opened the right file
  info, err := h.file.Stat()
  if err != nil || !os.SameFile(h.FileInfo, info) {
    h.file.Close()
    return false
  }

  // TODO(sissel): Only seek if the file is a file, not a pipe or socket.
  if h.Initial {
    // This is a new file detected during startup so calculate offset based on from-beginning
    if *from_beginning {
      h.file.Seek(0, os.SEEK_SET)
    } else {
      h.file.Seek(0, os.SEEK_END)
    }
  } else {
    // This is a new file detected after startup, or one that was resumed from the state file; in both cases we must obey the given offset
    // For new files offset will be 0 so this ensures we always start from the beginning on files created while we're running
    // This matches the LogStash behaviour
    h.file.Seek(h.Offset, os.SEEK_SET)
  }

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
        time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

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
