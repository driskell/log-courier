package main

import (
  "errors"
  "fmt"
  "regexp"
  "strings"
)

type CodecMultilineConfig struct {
  Pattern string
  Negate  bool
}

func NewCodecMultilineConfig(config map[string]interface{}) (*CodecMultilineConfig, error) {
  var ok bool
  result := &CodecMultilineConfig{}
  for key, value := range config {
    if key == "name" {
    } else if key == "pattern" {
      result.Pattern, ok = value.(string)
      if !ok {
        return nil, errors.New("Invalid value for 'pattern'.")
      }
    } else if key == "negate" {
      result.Negate, ok = value.(bool)
      if !ok {
        return nil, errors.New("Invalid value for 'negate'.")
      }
    } else {
      return nil, errors.New(fmt.Sprintf("Invalid property for multiline codec, '%s'.", key))
    }
  }
  if result.Pattern == "" {
    return nil, errors.New("Multiline codec pattern must be specified.")
  }
  return result, nil
}

type CodecMultiline struct {
  h       *Harvester
  output  chan *FileEvent

  config  *CodecMultilineConfig
  offset  int64
  line    uint64
  matcher *regexp.Regexp
  buffer  []string
}

func (codec *CodecMultiline) Init() {
  codec.config = codec.h.FileConfig.codec.(*CodecMultilineConfig)
  codec.matcher = regexp.MustCompile(codec.config.Pattern)
}

func (codec *CodecMultiline) Teardown() {
  codec.Flush()
}

func (codec *CodecMultiline) Event(line uint64, text *string) {
  if !codec.config.Negate == codec.matcher.MatchString(*text) {
    if len(codec.buffer) != 0 {
      codec.Flush()
      codec.buffer = nil
    }
  }
  if len(codec.buffer) == 0 {
    codec.line = line
  }
  codec.offset = codec.h.Offset
  codec.buffer = append(codec.buffer, *text)
}

func (codec *CodecMultiline) Flush() {
  text := strings.Join(codec.buffer, "\n")

  event := &FileEvent{
	Source:   &codec.h.Path,
	Offset:   codec.offset,
	Line:     codec.line,
	Text:     &text,
	Fields:   &codec.h.FileConfig.Fields,
	fileinfo: &codec.h.Info,
  }

  codec.output <- event // ship the new event downstream
}
