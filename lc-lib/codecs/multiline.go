/*
 * Copyright 2014-2015 Jason Woods.
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

package codecs

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
)

const (
	codecMultilineWhatPrevious = 0x00000001
	codecMultilineWhatNext     = 0x00000002
)

// codecMultilinePatternInstance holds the regular expression matcher for a
// single pattern in the configuration file, along with any pattern specific
// configurations
type codecMultilinePatternInstance struct {
	matcher *regexp.Regexp
	negate  bool
}

// CodecMultilineFactory holds the configuration for a multiline codec
type CodecMultilineFactory struct {
	Patterns          []string      `config:"patterns"`
	What              string        `config:"what"`
	PreviousTimeout   time.Duration `config:"previous timeout"`
	MaxMultilineBytes int64         `config:"max multiline bytes"`

	patterns []*codecMultilinePatternInstance
	what     int
}

// CodecMultiline is an instance of a multiline codec that is used by the
// Harvester for multiline processing
type CodecMultiline struct {
	config       *CodecMultilineFactory
	lastOffset   int64
	callbackFunc CallbackFunc

	endOffset     int64
	startOffset   int64
	buffer        []string
	bufferLines   int64
	bufferLen     int64
	timerLock     sync.Mutex
	timerStop     chan interface{}
	timerWait     sync.WaitGroup
	timerDeadline time.Time

	meterLines int64
	meterBytes int64
}

// NewMultilineCodecFactory creates a new MultilineCodecFactory for a codec
// definition in the configuration file. This factory can be used to create
// instances of a multiline codec for use by harvesters
func NewMultilineCodecFactory(config *config.Config, configPath string, unused map[string]interface{}, name string) (interface{}, error) {
	var err error

	result := &CodecMultilineFactory{}
	if err = config.PopulateConfig(result, unused, configPath); err != nil {
		return nil, err
	}

	if len(result.Patterns) == 0 {
		return nil, errors.New("At least one multiline codec pattern must be specified.")
	}

	result.patterns = make([]*codecMultilinePatternInstance, len(result.Patterns))
	for k, pattern := range result.Patterns {
		patternInstance := &codecMultilinePatternInstance{}

		switch pattern[0] {
		case '!':
			patternInstance.negate = true
			pattern = pattern[1:]
		case '=':
			pattern = pattern[1:]
		}

		patternInstance.matcher, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("Failed to compile multiline codec pattern, '%s'.", err)
		}

		result.patterns[k] = patternInstance
	}

	if result.What == "" || result.What == "previous" {
		result.what = codecMultilineWhatPrevious
	} else if result.What == "next" {
		result.what = codecMultilineWhatNext
	}

	if result.MaxMultilineBytes == 0 {
		result.MaxMultilineBytes = config.General.SpoolMaxBytes
	}

	// We conciously allow a line 4 bytes longer what we would normally have as the limit
	// This 4 bytes is the event header size. It's not worth considering though
	if result.MaxMultilineBytes > config.General.SpoolMaxBytes {
		return nil, fmt.Errorf("max multiline bytes cannot be greater than /general/spool max bytes")
	}

	return result, nil
}

// NewCodec returns a new codec instance that will send events to the callback
// function provided upon completion of processing
func (f *CodecMultilineFactory) NewCodec(callbackFunc CallbackFunc, offset int64) Codec {
	c := &CodecMultiline{
		config:       f,
		endOffset:    offset,
		lastOffset:   offset,
		callbackFunc: callbackFunc,
	}

	// Start the "previous timeout" routine that will auto flush at deadline
	if f.PreviousTimeout != 0 {
		c.timerStop = make(chan interface{})
		c.timerWait.Add(1)

		c.timerDeadline = time.Now().Add(f.PreviousTimeout)

		go c.deadlineRoutine()
	}
	return c
}

// Teardown ends the codec and returns the last offset shipped to the callback
func (c *CodecMultiline) Teardown() int64 {
	if c.config.PreviousTimeout != 0 {
		close(c.timerStop)
		c.timerWait.Wait()
	}

	return c.lastOffset
}

// Reset restores the codec to a blank state so it can be reused on a new file
// stream
func (c *CodecMultiline) Reset() {
	c.lastOffset = 0
	c.buffer = nil
	c.bufferLen = 0
	c.bufferLines = 0
}

// Event is called by a Harvester when a new line event occurs on a file.
// Multiline processing takes place and when a complete multiline event is found
// as described by the configuration it is shipped to the callback
func (c *CodecMultiline) Event(startOffset int64, endOffset int64, text string) {
	// TODO(driskell): If we are using previous and we match on the very first line read,
	// then this is because we've started in the middle of a multiline event (the first line
	// should never match) - so we could potentially offer an option to discard this.
	// The benefit would be that when using previoustimeout, we could discard any extraneous
	// event data that did not get written in time, if the user so wants it, in order to prevent
	// odd incomplete data. It would be a signal from the user, "I will worry about the buffering
	// issues my programs may have - you just make sure to write each event either completely or
	// partially, always with the FIRST line correct (which could be the important one)."
	var matchFailed bool
	for _, pattern := range c.config.patterns {
		if matchFailed = pattern.negate == pattern.matcher.MatchString(text); !matchFailed {
			break
		}
	}

	if c.config.what == codecMultilineWhatPrevious {
		if c.config.PreviousTimeout != 0 {
			// Prevent a flush happening while we're modifying the stored data
			c.timerLock.Lock()
		}
		if matchFailed {
			c.flush()
		}
	}

	textLen := int64(len(text))

	if len(c.buffer) == 0 {
		c.startOffset = startOffset
	}

	// Check we don't exceed the max multiline bytes
	checkLen := c.bufferLen + textLen + c.bufferLines
	for checkLen >= c.config.MaxMultilineBytes {
		// Store partial and flush
		overflow := checkLen - c.config.MaxMultilineBytes
		cut := textLen - overflow

		c.endOffset = endOffset - overflow

		c.buffer = append(c.buffer, text[:cut])
		c.bufferLines++
		c.bufferLen += cut

		c.flush()

		// Append the remaining data to the buffer
		c.startOffset = c.endOffset
		text = text[cut:]
		textLen -= cut

		// Reset check length in case we're still over the max
		checkLen = textLen
	}

	c.endOffset = endOffset

	c.buffer = append(c.buffer, text)
	c.bufferLines++
	c.bufferLen += textLen

	if c.config.what == codecMultilineWhatPrevious {
		if c.config.PreviousTimeout != 0 {
			// Reset the timer and unlock
			c.timerDeadline = time.Now().Add(c.config.PreviousTimeout)
			c.timerLock.Unlock()
		}
	} else if c.config.what == codecMultilineWhatNext && matchFailed {
		c.flush()
	}
}

// flush is called internally when a multiline event is ready.
// It combines the lines collected and passes the new event to the callback
func (c *CodecMultiline) flush() {
	if len(c.buffer) == 0 {
		return
	}

	text := strings.Join(c.buffer, "\n")

	// Set last offset - this is returned in Teardown so if we're mid multiline and crash, we start this multiline again
	c.lastOffset = c.endOffset
	c.buffer = nil
	c.bufferLen = 0
	c.bufferLines = 0

	c.callbackFunc(c.startOffset, c.endOffset, text)
}

// Meter is called by the Harvester to request accounting
func (c *CodecMultiline) Meter() {
	c.meterLines = c.bufferLines
	c.meterBytes = c.endOffset - c.lastOffset
}

// Snapshot is called when lc-admin tool requests a snapshot and the accounting
// data is returned in a snapshot structure
func (c *CodecMultiline) Snapshot() *core.Snapshot {
	snap := core.NewSnapshot("Multiline Codec")
	snap.AddEntry("Pending lines", c.meterLines)
	snap.AddEntry("Pending bytes", c.meterBytes)
	return snap
}

func (c *CodecMultiline) deadlineRoutine() {
	timer := time.NewTimer(0)

DeadlineLoop:
	for {
		select {
		case <-c.timerStop:
			timer.Stop()

			// Shutdown signal so end the routine
			break DeadlineLoop
		case now := <-timer.C:
			c.timerLock.Lock()

			// Have we reached the target time?
			if !now.After(c.timerDeadline) {
				// Deadline moved, update the timer
				timer.Reset(c.timerDeadline.Sub(now))
				c.timerLock.Unlock()
				continue
			}

			c.flush()
			timer.Reset(c.config.PreviousTimeout)
			c.timerLock.Unlock()
		}
	}

	c.timerWait.Done()
}

// Register the codec
func init() {
	config.RegisterCodec("multiline", NewMultilineCodecFactory)
}
