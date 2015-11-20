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
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

type HarvesterFinish struct {
	Last_Event_Offset int64
	Last_Read_Offset  int64
	Error             error
	Last_Stat         os.FileInfo
}

type Harvester struct {
	sync.RWMutex

	stop_chan     chan interface{}
	return_chan   chan *HarvesterFinish
	stream        core.Stream
	fileinfo      os.FileInfo
	path          string
	config        *config.Config
	stream_config *config.Stream
	offset        int64
	output        chan<- *core.EventDescriptor
	codec         codecs.Codec
	codecChain    []codecs.Codec
	file          *os.File
	split         bool
	timezone      string

	line_speed   float64
	byte_speed   float64
	line_count   uint64
	byte_count   uint64
	last_eof_off *int64
	last_eof     *time.Time
}

func NewHarvester(stream core.Stream, config *config.Config, stream_config *config.Stream, offset int64) *Harvester {
	var fileinfo os.FileInfo
	var path string

	if stream != nil {
		// Grab now so we can safely use them even if prospector changes them
		path, fileinfo = stream.Info()
	} else {
		// This is stdin
		path, fileinfo = "-", nil
	}

	ret := &Harvester{
		stop_chan:     make(chan interface{}),
		stream:        stream,
		fileinfo:      fileinfo,
		path:          path,
		config:        config,
		stream_config: stream_config,
		offset:        offset,
		timezone:      time.Now().Format("-0700 MST"),
		last_eof:      nil,
		codecChain:    make([]codecs.Codec, len(stream_config.Codecs)-1),
	}

	// Build the codec chain
	var entry codecs.Codec
	callback := ret.eventCallback
	for i := len(stream_config.Codecs) - 1; i >= 0; i-- {
		entry = codecs.NewCodec(stream_config.Codecs[i].Factory, callback, ret.offset)
		callback = entry.Event
		if i != 0 {
			ret.codecChain[i-1] = entry
		}
	}
	ret.codec = entry

	return ret
}

func (h *Harvester) Start(output chan<- *core.EventDescriptor) {
	if h.return_chan != nil {
		h.Stop()
		<-h.return_chan
	}

	h.return_chan = make(chan *HarvesterFinish, 1)

	go func() {
		status := &HarvesterFinish{}
		status.Last_Event_Offset, status.Error = h.harvest(output)
		status.Last_Read_Offset = h.offset
		status.Last_Stat = h.fileinfo
		h.return_chan <- status
		close(h.return_chan)
	}()
}

func (h *Harvester) Stop() {
	close(h.stop_chan)
}

func (h *Harvester) OnFinish() <-chan *HarvesterFinish {
	return h.return_chan
}

func (h *Harvester) codecTeardown() int64 {
	for _, codec := range h.codecChain {
		codec.Teardown()
	}

	return h.codec.Teardown()
}

func (h *Harvester) harvest(output chan<- *core.EventDescriptor) (int64, error) {
	if err := h.prepareHarvester(); err != nil {
		return h.offset, err
	}

	defer h.file.Close()

	h.output = output

	if h.path == "-" {
		log.Info("Started stdin harvester")
		h.offset = 0
	} else {
		// Get current offset in file
		offset, err := h.file.Seek(0, os.SEEK_CUR)
		if err != nil {
			log.Warning("Failed to determine start offset for %s: %s", h.path, err)
			return h.offset, err
		}

		if h.offset != offset {
			log.Warning("Started harvester at position %d (requested %d): %s", offset, h.offset, h.path)
		} else {
			log.Info("Started harvester at position %d (requested %d): %s", offset, h.offset, h.path)
		}

		h.offset = offset
	}

	// The buffer size limits the maximum line length we can read, including terminator
	reader := NewLineReader(h.file, int(h.config.General.LineBufferBytes), int(h.config.General.MaxLineBytes))

	// TODO: Make configurable?
	read_timeout := 10 * time.Second

	last_read_time := time.Now()
	last_line_count := uint64(0)
	last_byte_count := uint64(0)
	last_measurement := last_read_time
	seconds_without_events := 0

ReadLoop:
	for {
		text, bytesread, err := h.readline(reader)

		if duration := time.Since(last_measurement); duration >= time.Second {
			h.Lock()

			h.line_speed = core.CalculateSpeed(duration, h.line_speed, float64(h.line_count-last_line_count), &seconds_without_events)
			h.byte_speed = core.CalculateSpeed(duration, h.byte_speed, float64(h.byte_count-last_byte_count), &seconds_without_events)

			last_byte_count = h.byte_count
			last_line_count = h.line_count
			last_measurement = time.Now()

			h.codec.Meter()

			h.last_eof = nil

			h.Unlock()

			// Check shutdown
			select {
			case <-h.stop_chan:
				break ReadLoop
			default:
			}
		}

		if err == nil {
			line_offset := h.offset
			h.offset += int64(bytesread)

			// Codec is last - it forwards harvester state for us such as offset for resume
			h.codec.Event(line_offset, h.offset, text)

			last_read_time = time.Now()
			h.line_count++
			h.byte_count += uint64(bytesread)

			continue
		}

		if err != io.EOF {
			if h.path == "-" {
				log.Error("Unexpected error reading from stdin: %s", err)
			} else {
				log.Error("Unexpected error reading from %s: %s", h.path, err)
			}
			return h.codecTeardown(), err
		}

		if h.path == "-" {
			// Stdin has finished - stdin blocks permanently until the stream ends
			// Once the stream ends, finish the harvester
			log.Info("Stopping harvest of stdin; EOF reached")
			return h.codecTeardown(), nil
		}

		// Check shutdown
		select {
		case <-h.stop_chan:
			break ReadLoop
		default:
		}

		h.Lock()
		if h.last_eof_off == nil {
			h.last_eof_off = new(int64)
		}
		*h.last_eof_off = h.offset

		if h.last_eof == nil {
			h.last_eof = new(time.Time)
		}
		*h.last_eof = last_read_time
		h.Unlock()

		// Don't check for truncation until we hit the full read_timeout
		if time.Since(last_read_time) < read_timeout {
			continue
		}

		info, err := h.file.Stat()
		if err != nil {
			log.Error("Unexpected error checking status of %s: %s", h.path, err)
			return h.codecTeardown(), err
		}

		if info.Size() < h.offset {
			log.Warning("Unexpected file truncation, seeking to beginning: %s", h.path)
			h.file.Seek(0, os.SEEK_SET)
			h.offset = 0

			// TODO: Should we be allowing truncation to lose buffer data? Or should
			//       we be flushing what we have?
			// Reset line buffer and codec buffers
			reader.Reset()
			h.codec.Reset()
			continue
		}

		// If last_read_time was more than dead time, this file is probably dead.
		// Stop only if the mtime did not change since last check - this stops a
		// race where we hit EOF but as we Stat() the mtime is updated - this mtime
		// is the one we monitor in order to resume checking, so we need to check it
		// didn't already update
		if age := time.Since(last_read_time); age > h.stream_config.DeadTime && h.fileinfo.ModTime() == info.ModTime() {
			log.Info("Stopping harvest of %s; last change was %v ago", h.path, age-(age%time.Second))
			return h.codecTeardown(), nil
		}

		// Store latest stat()
		h.fileinfo = info
	}

	log.Info("Harvester for %s exiting", h.path)
	return h.codecTeardown(), nil
}

func (h *Harvester) eventCallback(start_offset int64, end_offset int64, text string) {
	event := core.Event{
		"message": text,
	}

	if h.stream_config.AddHostField {
		event["host"] = h.config.General.Host
	}
	if h.stream_config.AddPathField {
		event["path"] = h.path
	}
	if h.stream_config.AddOffsetField {
		event["offset"] = start_offset
	}
	if h.stream_config.AddTimezoneField {
		event["timezone"] = h.timezone
	}

	for k := range h.config.General.GlobalFields {
		event[k] = h.config.General.GlobalFields[k]
	}

	for k := range h.stream_config.Fields {
		event[k] = h.stream_config.Fields[k]
	}

	// If we split any of the line data, tag it
	if h.split {
		if v, ok := event["tags"]; ok {
			if v, ok := v.([]string); ok {
				v = append(v, "splitline")
			}
		} else {
			event["tags"] = []string{"splitline"}
		}
		h.split = false
	}

	encoded, err := event.Encode()
	if err != nil {
		// This should never happen - log and skip if it does
		log.Warning("Skipping line in %s at offset %d due to encoding failure: %s", h.path, start_offset, err)
		return
	}

	desc := &core.EventDescriptor{
		Stream: h.stream,
		Offset: end_offset,
		Event:  encoded,
	}

EventLoop:
	for {
		select {
		case <-h.stop_chan:
			break EventLoop
		case h.output <- desc:
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

	// Store latest stat()
	h.fileinfo = info

	// TODO: Check error?
	h.file.Seek(h.offset, os.SEEK_SET)

	return nil
}

func (h *Harvester) readline(reader *LineReader) (string, int, error) {
	var newline int

	line, err := reader.ReadSlice()

	if line != nil {
		if err == nil {
			// Line will always end in '\n' if no error, but check also for CR
			if len(line) > 1 && line[len(line)-2] == '\r' {
				newline = 2
			} else {
				newline = 1
			}
		} else if err == ErrLineTooLong {
			h.split = true
			err = nil
		}

		// Return the line along with the length including line ending
		length := len(line)
		// We use string() to copy the memory, which is a slice of the line buffer we need to re-use
		return string(line[:length-newline]), length, err
	}

	if err != nil {
		if err != io.EOF {
			// Pass back error to tear down harvester
			return "", 0, err
		}

		// Backoff
		select {
		case <-h.stop_chan:
		case <-time.After(1 * time.Second):
		}
	}

	return "", 0, io.EOF
}

func (h *Harvester) Snapshot() *core.Snapshot {
	h.RLock()

	ret := core.NewSnapshot("Harvester")
	ret.AddEntry("Speed (Lps)", h.line_speed)
	ret.AddEntry("Speed (Bps)", h.byte_speed)
	ret.AddEntry("Processed lines", h.line_count)
	ret.AddEntry("Current offset", h.offset)
	if h.last_eof_off == nil {
		ret.AddEntry("Last EOF Offset", "Never")
	} else {
		ret.AddEntry("Last EOF Offset", h.last_eof_off)
	}
	if h.last_eof == nil {
		ret.AddEntry("Status", "Alive")
	} else {
		ret.AddEntry("Status", "Idle")
		if age := time.Since(*h.last_eof); age < h.stream_config.DeadTime {
			ret.AddEntry("Dead timer", h.stream_config.DeadTime-age)
		} else {
			ret.AddEntry("Dead timer", "0s")
		}
	}

	if sub_snap := h.codec.Snapshot(); sub_snap != nil {
		ret.AddSub(sub_snap)
	}

	h.RUnlock()

	return ret
}
