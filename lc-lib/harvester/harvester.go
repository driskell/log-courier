/*
 * Copyright 2014-2015 Jason Woods.
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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/admin"
	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

var (
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
	mutex sync.RWMutex

	stopChan     chan interface{}
	returnChan   chan *FinishStatus
	stream       core.Stream
	fileinfo     os.FileInfo
	path         string
	config       *config.Config
	streamConfig *config.Stream
	offset       int64
	output       chan<- *core.EventDescriptor
	codec        codecs.Codec
	codecChain   []codecs.Codec
	file         *os.File
	backOffTimer *time.Timer
	meterTimer   *time.Timer
	split        bool
	timezone     string
	reader       *LineReader

	lastReadTime         time.Time
	lastMeasurement      time.Time
	lastStatCheck        time.Time
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

// NewHarvester creates a new harvester with the given configuration for the given stream identifier
func NewHarvester(stream core.Stream, config *config.Config, streamConfig *config.Stream, offset int64) *Harvester {
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
		stopChan:     make(chan interface{}),
		stream:       stream,
		fileinfo:     fileinfo,
		path:         path,
		config:       config,
		streamConfig: streamConfig,
		offset:       offset,
		timezone:     time.Now().Format("-0700 MST"),
		lastEOF:      nil,
		codecChain:   make([]codecs.Codec, len(streamConfig.Codecs)-1),
		backOffTimer: time.NewTimer(0),
		// TODO: Configurable meter timer? Use same as statCheck timer
		meterTimer: time.NewTimer(10 * time.Second),
	}

	ret.backOffTimer.Stop()

	// Build the codec chain
	var entry codecs.Codec
	callback := ret.eventCallback
	for i := len(streamConfig.Codecs) - 1; i >= 0; i-- {
		entry = codecs.NewCodec(streamConfig.Codecs[i].Factory, callback, ret.offset)
		callback = entry.Event
		if i != 0 {
			ret.codecChain[i-1] = entry
		}
	}
	ret.codec = entry

	return ret
}

// Start runs the harvester, sending events to the output given, and returns
// immediately
func (h *Harvester) Start(output chan<- *core.EventDescriptor) {
	if h.returnChan != nil {
		h.Stop()
		<-h.returnChan
	}

	h.returnChan = make(chan *FinishStatus, 1)

	go func() {
		status := &FinishStatus{}
		status.LastEventOffset, status.Error = h.harvest(output)
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

// codecTeardown shuts down all codecs in the order they are used
func (h *Harvester) codecTeardown() int64 {
	for _, codec := range h.codecChain {
		codec.Teardown()
	}

	return h.codec.Teardown()
}

// harvest runs in its own routine, opening the file and starting the read loop
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
	h.reader = NewLineReader(h.file, int(h.config.General.LineBufferBytes), int(h.config.General.MaxLineBytes))

	// Prepare internal data
	h.lastReadTime = time.Now()
	h.lastMeasurement = h.lastReadTime
	h.lastStatCheck = h.lastReadTime

	for {
		if err := h.performRead(); err != nil {
			if err == errStopRequested {
				break
			}
			return h.codecTeardown(), err
		}
	}

	log.Info("Harvester for %s exiting", h.path)
	return h.codecTeardown(), nil
}

// performRead performs a single read operation
func (h *Harvester) performRead() error {
	text, bytesread, err := h.readline()

	// Is a measurement due?
	if duration := time.Since(h.lastMeasurement); duration >= time.Second {
		if measureErr := h.takeMeasurements(duration); measureErr != nil {
			if measureErr == errFileTruncated {
				log.Warning("Unexpected file truncation, seeking to beginning: %s", h.path)
				h.file.Seek(0, os.SEEK_SET)
				h.offset = 0

				// TODO: Should we be allowing truncation to lose buffer data? Or should
				//       we be flushing what we have?
				// Reset line buffer and codec buffers
				h.reader.Reset()
				h.codec.Reset()
				return nil
			}
			return measureErr
		}
	}

	if err == nil {
		lineOffset := h.offset
		h.offset += int64(bytesread)

		// Codec is last - it forwards harvester state for us such as offset for resume
		h.codec.Event(lineOffset, h.offset, text)

		h.lastReadTime = time.Now()
		h.lineCount++
		h.byteCount += uint64(bytesread)
		return nil
	}

	if err != io.EOF {
		if h.path == "-" {
			log.Error("Unexpected error reading from stdin: %s", err)
		} else {
			log.Error("Unexpected error reading from %s: %s", h.path, err)
		}
		return err
	}

	if h.path == "-" {
		// Stdin has finished - stdin blocks permanently until the stream ends
		// Once the stream ends, finish the harvester
		log.Info("Stopping harvest of stdin; EOF reached")
		return nil
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

func (h *Harvester) takeMeasurements(duration time.Duration) error {
	h.lastMeasurement = time.Now()

	if h.path != "-" {
		// Has enough time passed for a truncation / deletion check?
		// TODO: Make time configurable?
		if duration := time.Since(h.lastStatCheck); duration >= 10*time.Second {
			h.lastStatCheck = h.lastMeasurement

			var err error
			if err = h.statCheck(); err != nil {
				return err
			}
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
	h.codec.Meter()
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
		log.Error("Unexpected error checking status of %s: %s", h.path, err)
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
func (h *Harvester) eventCallback(startOffset int64, endOffset int64, text string) {
	event := core.Event{
		"message": text,
	}

	if h.streamConfig.AddHostField {
		event["host"] = h.config.General.Host
	}
	if h.streamConfig.AddPathField {
		event["path"] = h.path
	}
	if h.streamConfig.AddOffsetField {
		event["offset"] = startOffset
	}
	if h.streamConfig.AddTimezoneField {
		event["timezone"] = h.timezone
	}

	for k := range h.config.General.GlobalFields {
		event[k] = h.config.General.GlobalFields[k]
	}

	for k := range h.streamConfig.Fields {
		event[k] = h.streamConfig.Fields[k]
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
		log.Warning("Skipping line in %s at offset %d due to encoding failure: %s", h.path, startOffset, err)
		return
	}

	desc := &core.EventDescriptor{
		Stream: h.stream,
		Offset: endOffset,
		Event:  encoded,
	}

EventLoop:
	for {
		select {
		case <-h.stopChan:
			break EventLoop
		case h.output <- desc:
			break EventLoop
		case <-h.meterTimer.C:
			// TODO: Configurable meter timer? Same as statCheck?
			h.meterTimer.Reset(10 * time.Second)

			// Take measurements if enough time has elapsed since the last measurement
			if duration := time.Since(h.lastMeasurement); duration >= time.Second {
				if measureErr := h.takeMeasurements(duration); measureErr == errStopRequested {
					break EventLoop
				}
			}
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

// APIEncodeable returns an admin API entry with harvester status
func (h *Harvester) APIEncodable() admin.APIEncodable {
	h.mutex.RLock()

	apiEncodable := &admin.APIKeyValue{}
	apiEncodable.SetEntry("speed_lps", admin.APIFloat(h.lineSpeed))
	apiEncodable.SetEntry("speed_bps", admin.APIFloat(h.byteSpeed))
	apiEncodable.SetEntry("processed_lines", admin.APINumber(h.lineCount))
	apiEncodable.SetEntry("current_offset", admin.APINumber(h.lastOffset))
	apiEncodable.SetEntry("last_known_size", admin.APINumber(h.lastSize))

	if h.offset >= h.lastSize {
		apiEncodable.SetEntry("completion", admin.APIFloat(100.))
	} else {
		completion := float64(h.offset) * 100 / float64(h.lastSize)
		apiEncodable.SetEntry("completion", admin.APIFloat(completion))
	}
	if h.lastEOFOff == nil {
		apiEncodable.SetEntry("last_eof_offset", admin.APINull)
	} else {
		apiEncodable.SetEntry("last_eof_offset", admin.APINumber(*h.lastEOFOff))
	}
	if h.lastEOF == nil {
		apiEncodable.SetEntry("status", admin.APIString("alive"))
	} else {
		apiEncodable.SetEntry("status", admin.APIString("idle"))
		if age := time.Since(*h.lastEOF); age < h.streamConfig.DeadTime {
			apiEncodable.SetEntry("dead_timer", admin.APIString(fmt.Sprintf("%s", h.streamConfig.DeadTime-age)))
		} else {
			apiEncodable.SetEntry("dead_timer", admin.APIString("0s"))
		}
	}

	h.mutex.RUnlock()

	return apiEncodable
}
