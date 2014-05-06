package main

import (
  "errors"
  "fmt"
  "regexp"
  "strings"
)

type CodecMultilineFactory struct {
  pattern string
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
        return nil, errors.New("Invalid value for 'pattern'.")
      }
    } else if key == "negate" {
      result.negate, ok = value.(bool)
      if !ok {
        return nil, errors.New("Invalid value for 'negate'.")
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
  if !c.config.negate == c.matcher.MatchString(*text) {
    if len(c.buffer) != 0 {
      c.flush()
      c.buffer = nil
    }
  }
  if len(c.buffer) == 0 {
    c.line = line
  }
  c.offset = c.harvester.Offset
  c.buffer = append(c.buffer, *text)
}

func (c *CodecMultiline) flush() {
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
}
