package main

import (
  "errors"
  "fmt"
  "regexp"
  "strings"
  "sync"
  "time"
)

const codecMultiline_What_Previous = 0x00000001
const codecMultiline_What_Next = 0x00000002

type CodecMultilineRegistrar struct {
}

type CodecMultilineFactory struct {
  Pattern          string `json:"pattern"`
  matcher *regexp.Regexp
  What             string `json:"what"`
  what             int
  Negate           bool `json:"negate"`
  PreviousTimeout  time.Duration `json:"previous timeout"`
}

type CodecMultiline struct {
  config      *CodecMultilineFactory
  path string
  fileconfig *FileConfig
  info *ProspectorInfo
  last_offset int64
  output chan<- *FileEvent

  end_offset    int64
  start_offset      int64
  line        uint64
  buffer      []string
  timer_lock  *sync.Mutex
  timer_chan  chan bool
}

func (r *CodecMultilineRegistrar) NewFactory(config_path string, config map[string]interface{}) (CodecFactory, error) {
  var err error

  result := &CodecMultilineFactory{}
  if err = PopulateConfig(result, config_path, config); err != nil {
    return nil, err
  }

  if result.Pattern == "" {
    return nil, errors.New("Multiline codec pattern must be specified.")
  }

  result.matcher, err = regexp.Compile(result.Pattern)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("Failed to compile multiline codec pattern, '%s'.", err))
  }

  if result.What == "" || result.What == "previous" {
    result.what = codecMultiline_What_Previous
  } else if result.What == "next" {
    result.what = codecMultiline_What_Next
  }

  return result, nil
}

func (f *CodecMultilineFactory) NewCodec(path string, fileconfig *FileConfig, info *ProspectorInfo, offset int64, output chan<- *FileEvent) Codec {
  c := &CodecMultiline{
    config: f,
    path: path,
    fileconfig: fileconfig,
    info: info,
    last_offset: offset,
    output: output,
  }

  // TODO: Make this more performant - use similiar methodology to Go's internal network deadlines
  if f.PreviousTimeout != 0 {
    c.timer_lock = new(sync.Mutex)
    c.timer_chan = make(chan bool, 1)

    go func() {
      var active bool

      timer := time.NewTimer(0)

      for {
        select {
        case shutdown := <-c.timer_chan:
          timer.Stop()
          if shutdown {
            // Shutdown signal so end the routine
            break
          }
          timer.Reset(c.config.PreviousTimeout)
          active = true
        case <-timer.C:
          if active {
            // Surround flush in mutex to prevent data getting modified by a new line while we flush
            c.timer_lock.Lock()
            c.flush()
            c.timer_lock.Unlock()
            active = false
          }
        }
      }
    }()
  }
  return c
}

func (c *CodecMultiline) Teardown() int64 {
  return c.last_offset
}

func (c *CodecMultiline) Event(start_offset int64, end_offset int64, line uint64, text *string) {
  // TODO(driskell): If we are using previous and we match on the very first line read,
  // then this is because we've started in the middle of a multiline event (the first line
  // should never match) - so we could potentially offer an option to discard this.
  // The benefit would be that when using previous_timeout, we could discard any extraneous
  // event data that did not get written in time, if the user so wants it, in order to prevent
  // odd incomplete data. It would be a signal from the user, "I will worry about the buffering
  // issues my programs may have - you just make sure to write each event either completely or
  // partially, always with the FIRST line correct (which could be the important one)."
  match_failed := c.config.Negate == c.config.matcher.MatchString(*text)
  if c.config.what == codecMultiline_What_Previous {
    if c.config.PreviousTimeout != 0 {
      // Prevent a flush happening while we're modifying the stored data
      c.timer_lock.Lock()
    }
    if match_failed {
      c.flush()
    }
  }
  if len(c.buffer) == 0 {
    c.line = line
    c.start_offset = start_offset
  }
  c.end_offset = end_offset
  c.buffer = append(c.buffer, *text)
  if c.config.what == codecMultiline_What_Previous {
    if c.config.PreviousTimeout != 0 {
      // Reset the timer and unlock
      c.timer_chan <- false
      c.timer_lock.Unlock()
    }
  } else if c.config.what == codecMultiline_What_Next && match_failed {
    c.flush()
  }
}

func (c *CodecMultiline) flush() {
  if len(c.buffer) == 0 {
    return
  }

  text := strings.Join(c.buffer, "\n")

  // Set last offset - this is returned in Teardown so if we're mid multiline and crash, we start this multiline again
  c.last_offset = c.end_offset
  c.buffer = nil

  c.output <- &FileEvent{
    ProspectorInfo: c.info,
    Offset:         c.end_offset,
    Event:          NewEvent(c.fileconfig.Fields, &c.path, c.start_offset, c.line, &text),
  }
}

// Register the codec
func init() {
  RegisterCodec(&CodecMultilineRegistrar{}, "multiline")
}
