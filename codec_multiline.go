package main

import (
  "errors"
  "fmt"
  "regexp"
  "strings"
)

const CODECMULTILINE_PREVIOUS = 0x00000001
const CODECMULTILINE_NEXT = 0x00000002

type CodecMultilineFactory struct {
  pattern string
  what    uint32
  negate  bool
}

type CodecMultiline struct {
  config      *CodecMultilineFactory
  harvester   *Harvester
  output      chan *FileEvent
  last_offset int64

  offset  int64
  line    uint64
  matcher *regexp.Regexp
  buffer  []string
}

func CreateCodecMultilineFactory(config map[string]interface{}) (*CodecMultilineFactory, error) {
  var ok bool
  result := &CodecMultilineFactory{}
  for key, value := range config {
    if key == "name" {
    } else if key == "pattern" {
      result.pattern, ok = value.(string)
      if !ok {
        return nil, errors.New("Invalid value for 'pattern'. Must be a string.")
      }
    } else if key == "what" {
      var what string
      what, ok = value.(string)
      if !ok {
        return nil, errors.New("Invalid value for 'what'. Must be a string.")
      }
      if what == "previous" {
        result.what = CODECMULTILINE_PREVIOUS
      } else if what == "next" {
        result.what = CODECMULTILINE_NEXT
      } else {
        return nil, errors.New("Invalid value for 'what'. Must be either 'previous' or 'next'.")
      }
    } else if key == "negate" {
      result.negate, ok = value.(bool)
      if !ok {
        return nil, errors.New("Invalid value for 'negate'. Must be true or false.")
      }
    } else {
      return nil, errors.New(fmt.Sprintf("Invalid property for multiline codec, '%s'.", key))
    }
  }
  if result.pattern == "" {
    return nil, errors.New("Multiline codec pattern must be specified.")
  }
  return result, nil
}

func (cf *CodecMultilineFactory) Create(harvester *Harvester, output chan *FileEvent) Codec {
  return &CodecMultiline{config: cf, matcher: regexp.MustCompile(cf.pattern), harvester: harvester, output: output, last_offset: harvester.Offset}
}

func (c *CodecMultiline) Teardown() int64 {
  return c.last_offset
}

func (c *CodecMultiline) Event(line uint64, text *string) {
  if c.config.what == CODECMULTILINE_PREVIOUS && !c.config.negate == c.matcher.MatchString(*text) {
    c.flush()
  }
  if len(c.buffer) == 0 {
    c.line = line
  }
  c.offset = c.harvester.Offset
  c.buffer = append(c.buffer, *text)
  if c.config.what == CODECMULTILINE_NEXT && !c.config.negate == c.matcher.MatchString(*text) {
    c.flush()
  }
}

func (c *CodecMultiline) flush() {
  if len(c.buffer) == 0 {
    return
  }
  text := strings.Join(c.buffer, "\n")

  event := &FileEvent{
    ProspectorInfo: c.harvester.ProspectorInfo,
    Source:         &c.harvester.Path, /* If the file rotates we still send the original name before rotation until restarted */
    Offset:         c.offset,
    Line:           c.line,
    Text:           &text,
    Fields:         &c.harvester.FileConfig.Fields,
  }

  c.output <- event // ship the new event downstream

  // Set last offset - this is returned in Teardown so if we're mid multiline and crash, we start this multiline again
  c.last_offset = c.offset
  c.buffer = nil
}
