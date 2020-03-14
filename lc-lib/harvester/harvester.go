/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/registrar"
)

var (
	// Stdin is the filename that represents stdin
	// TODO: Could be improved
	Stdin = "stdin"

	errFileTruncated = errors.New("File truncation detected")
	errStopRequested = errors.New("Stop requested")
)

// FinishStatus contains the final file state, and any errors, from the point the
// harvester finished
type FinishStatus struct {
	LastEventOffset int64
	LastReadOffset  int64
	Error           error
	LastStat        os.FileInfo
}

// Harvester reads from a file, passes lines through a codec, and sends them
// for spooling
type Harvester struct {
	ctx   context.Context
	mutex sync.RWMutex

	stopChan        chan struct{}
	returnChan      chan *FinishStatus
	acker           event.Acknowledger
	fileinfo        os.FileInfo
	path            string
	genConfig       *General
	streamConfig    *StreamConfig
	eventStream     *codecs.Stream
	offset          int64
	output          chan<- []*event.Event
	file            *os.File
	backOffTimer    *time.Timer
	meterTimer      *time.Timer
	split           bool
	reader          *LineReader
	staleOffset     int64
	staleBytes      int64
	lastStaleOffset int64
	isStream        bool

	lastReadTime         time.Time
	lastMeasurement      time.Time
	lastCheck            time.Time
	lastLineCount        uint64
	lastByteCount        uint64
	secondsWithoutEvents int

	lineSpeed  float64
	byteSpeed  float64
	lineCount  uint64
	byteCount  uint64
	lastEOFOff *int64
	lastEOF    *time.Time
	lastSize   int64
	lastOffset int64
}

// SetOutput sets the harvester output
func (h *Harvester) SetOutput(output chan<- []*event.Event) {
	h.output = output
}

// Start runs the harvester, sending events to the output given, and returns immediately
func (h *Harvester) Start() {
	if h.output == nil {
		panic("Must call SetOutput before Start on Harvester")
	}

	if h.returnChan != nil {
		h.Stop()
		<-h.returnChan
	}

	h.returnChan = make(chan *FinishStatus, 1)

	go func() {
		status := &FinishStatus{}
		status.LastEventOffset, status.Error = h.harvest()
		status.LastReadOffset = h.offset
		status.LastStat = h.fileinfo
		h.returnChan <- status
		close(h.returnChan)
	}()
}

// Stop requests the harvester to stop
func (h *Harvester) Stop() {
	close(h.stopChan)
}

// OnFinish returns a channel which will receive a FinishStatus structure when
// the harvester stops
func (h *Harvester) OnFinish() <-chan *FinishStatus {
	return h.returnChan
}

// harvest runs in its own routine, opening the file and starting the read loop
func (h *Harvester) harvest() (int64, error) {
	if err := h.prepareHarvester(); err != nil {
		return h.offset, err
	}

	defer h.file.Close()

	if h.isStream {
		log.Info("Started harvester: %s", h.path)
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
	h.reader = NewLineReader(h.file, int(h.genConfig.LineBufferBytes), int(h.genConfig.MaxLineBytes))

	// Prepare internal data
	h.lastReadTime = time.Now()
	h.lastMeasurement = h.lastReadTime
	h.lastCheck = h.lastReadTime

	for {
		if err := h.performRead(); err != nil {
			if err == errStopRequested {
				break
			}
			return h.eventStream.Close(), err
		}
	}

	log.Info("Harvester for %s exiting", h.path)
	return h.eventStream.Close(), nil
}

// performRead performs a single read operation
func (h *Harvester) performRead() error {
	text, bytesread, err := h.readline()

	// Is a measurement due?
	if duration := time.Since(h.lastMeasurement); duration >= time.Second {
		if measureErr := h.takeMeasurements(duration, false); measureErr != nil {
			if measureErr == errFileTruncated {
				h.handleTruncation()
				return nil
			}
			return measureErr
		}
	}

	if err == nil {
		lineOffset := h.offset
		h.offset += int64(bytesread)

		// Codec is last - it forwards harvester state for us such as offset for resume
		h.eventStream.ProcessEvent(lineOffset, h.offset, text)

		h.lastReadTime = time.Now()
		h.lineCount++
		h.byteCount += uint64(bytesread)
		return nil
	}

	if err != io.EOF {
		log.Errorf("Unexpected error reading from %s: %s", h.path, err)
		return err
	}

	if h.isStream {
		// Stream has finished
		log.Info("Stopping harvest of %s; EOF reached", h.path)
		return errStopRequested
	}

	h.mutex.Lock()
	if h.lastEOF == nil {
		h.lastEOF = new(time.Time)
		h.lastEOFOff = new(int64)
	}
	*h.lastEOF = h.lastReadTime
	*h.lastEOFOff = h.offset
	h.mutex.Unlock()

	// Check shutdown
	select {
	case <-h.stopChan:
		return errStopRequested
	default:
	}

	return nil
}

func (h *Harvester) handleTruncation() {
	log.Warning("Unexpected file truncation, seeking to beginning: %s", h.path)

	h.file.Seek(0, os.SEEK_SET)
	h.offset = 0
	h.staleOffset = 0
	h.lastStaleOffset = 0

	// TODO: Should we be allowing truncation to lose buffer data? Or should
	//       we be flushing what we have?
	if h.reader.BufferedLen() != 0 {
		log.Errorf("%d bytes of incomplete log data was lost due to file truncation", h.reader.BufferedLen())
	}

	// Reset line buffer and codec buffers
	h.reader.Reset()
	h.eventStream.Reset()
}

func (h *Harvester) takeMeasurements(duration time.Duration, isPipelineBlocked bool) error {
	h.lastMeasurement = time.Now()

	// Has enough time passed for periodic checks?
	// TODO: Make time configurable? Bear in mind this does a stale buffer check
	//       and reports an error saying "stale data for more than 10s"
	doChecks := false
	if checksDuration := time.Since(h.lastCheck); checksDuration >= 10*time.Second {
		h.lastCheck = h.lastMeasurement
		doChecks = true
	}

	// Check for stale data in the buffer
	if doChecks {
		if !isPipelineBlocked && h.reader.BufferedLen() != 0 {
			if h.staleOffset == h.offset && h.lastStaleOffset != h.offset+int64(h.reader.BufferedLen()) {
				log.Warningf(
					"%s has had %d stale byte(s) at the end with no line ending for over 10 seconds, please check the application",
					h.path,
					h.reader.BufferedLen(),
				)

				h.lastStaleOffset = h.offset + int64(h.reader.BufferedLen())
			}

			h.staleOffset = h.offset
		}
	}

	if doChecks && !h.isStream {
		var err error
		if err = h.statCheck(); err != nil {
			return err
		}
	}

	h.mutex.Lock()
	h.lineSpeed = core.CalculateSpeed(duration, h.lineSpeed, float64(h.lineCount-h.lastLineCount), &h.secondsWithoutEvents)
	h.byteSpeed = core.CalculateSpeed(duration, h.byteSpeed, float64(h.byteCount-h.lastByteCount), &h.secondsWithoutEvents)
	h.lastByteCount = h.byteCount
	h.lastLineCount = h.lineCount
	h.lastOffset = h.offset
	if h.fileinfo != nil {
		h.lastSize = h.fileinfo.Size()
	}
	if h.offset > h.lastSize {
		h.lastSize = h.offset
	}
	if h.lastStaleOffset > h.offset {
		h.staleBytes = h.lastStaleOffset - h.offset
	} else {
		h.staleBytes = 0
	}
	h.eventStream.Meter()
	h.mutex.Unlock()

	// Check shutdown
	select {
	case <-h.stopChan:
		return errStopRequested
	default:
	}

	return nil
}

// statCheck checks for truncation and returns the file size of the file
func (h *Harvester) statCheck() error {
	info, err := h.file.Stat()
	if err != nil {
		log.Errorf("Unexpected error checking status of %s: %s", h.path, err)
		return err
	}

	if info.Size() < h.offset {
		return errFileTruncated
	}

	// If lastReadTime was more than dead time, this file is probably dead.
	// Stop only if the mtime did not change since last check - this stops a
	// race where we hit EOF but as we Stat() the mtime is updated - this mtime
	// is the one we monitor in order to resume checking, so we need to check it
	// didn't already update
	if age := time.Since(h.lastReadTime); age > h.streamConfig.DeadTime && h.fileinfo.ModTime() == info.ModTime() {
		log.Info("Stopping harvest of %s; last change was %v ago", h.path, age-(age%time.Second))
		// TODO: dead_action implementation here
		return errStopRequested
	}

	// Store latest stat()
	h.fileinfo = info

	return nil
}

// eventCallback receives events from the final codec and ships them to the output
func (h *Harvester) eventCallback(startOffset int64, endOffset int64, data map[string]interface{}) {
	if h.streamConfig.AddPathField {
		data["path"] = h.path
	}

	// If we split any of the line data, tag it
	if h.split {
		if v, ok := data["tags"]; ok {
			va, ok := v.([]string)
			if ok {
				va = append(va, "splitline")
				data["tags"] = va
			}
		} else {
			data["tags"] = []string{"splitline"}
		}
		h.split = false
	}

	ctx := context.WithValue(h.ctx, registrar.ContextEndOffset, endOffset)
	newEvent := h.streamConfig.NewEvent(ctx, h.acker, data)

EventLoop:
	for {
		select {
		case <-h.stopChan:
			break EventLoop
		case h.output <- []*event.Event{newEvent}:
			break EventLoop
		case <-h.meterTimer.C:
			// TODO: Configurable meter timer? Same as statCheck?
			h.meterTimer.Reset(10 * time.Second)

			// Take measurements if enough time has elapsed since the last measurement
			if duration := time.Since(h.lastMeasurement); duration >= time.Second {
				if measureErr := h.takeMeasurements(duration, true); measureErr == errStopRequested {
					break EventLoop
				}
			}
		}
	}
}

func (h *Harvester) prepareHarvester() error {
	// Streams don't need opening or checking
	if h.isStream {
		return nil
	}

	var err error
	h.file, err = h.openFile(h.path)
	if err != nil {
		log.Errorf("Failed opening %s: %s", h.path, err)
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

// readline reads a single line from the file, handling mixed line endings
// and detecting where lines were split due to being too big for the buffer
func (h *Harvester) readline() (string, int, error) {
	var newline int

	line, err := h.reader.ReadSlice()

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
		h.backOffTimer.Reset(1 * time.Second)
		select {
		case <-h.stopChan:
		case <-h.backOffTimer.C:
		}
	}

	return "", 0, io.EOF
}

// APIEncodable returns an admin API entry with harvester status
func (h *Harvester) APIEncodable() api.Encodable {
	h.mutex.RLock()

	apiEncodable := &api.KeyValue{}
	apiEncodable.SetEntry("speed_lps", api.Float(h.lineSpeed))
	apiEncodable.SetEntry("speed_bps", api.Float(h.byteSpeed))
	apiEncodable.SetEntry("processed_lines", api.Number(h.lineCount))
	apiEncodable.SetEntry("current_offset", api.Number(h.lastOffset))
	apiEncodable.SetEntry("stale_bytes", api.Number(h.staleBytes))
	apiEncodable.SetEntry("last_known_size", api.Number(h.lastSize))

	if h.lastOffset >= h.lastSize {
		apiEncodable.SetEntry("completion", api.Float(100.))
	} else {
		completion := float64(h.lastOffset) * 100 / float64(h.lastSize)
		apiEncodable.SetEntry("completion", api.Float(completion))
	}
	if h.lastEOFOff == nil {
		apiEncodable.SetEntry("last_eof_offset", api.Null)
	} else {
		apiEncodable.SetEntry("last_eof_offset", api.Number(*h.lastEOFOff))
	}
	if h.lastEOF == nil {
		apiEncodable.SetEntry("status", api.String("alive"))
	} else {
		apiEncodable.SetEntry("status", api.String("idle"))
		if age := time.Since(*h.lastEOF); age < h.streamConfig.DeadTime {
			apiEncodable.SetEntry("dead_timer", api.String(fmt.Sprintf("%s", h.streamConfig.DeadTime-age)))
		} else {
			apiEncodable.SetEntry("dead_timer", api.String("0s"))
		}
	}
	apiEncodable.SetEntry("codecs", h.eventStream.APIEntry())

	h.mutex.RUnlock()

	return apiEncodable
}
