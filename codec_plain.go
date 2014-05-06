package main

import (
  "errors"
  "fmt"
)

type CodecPlainConfig struct {
}

func NewCodecPlainConfig(config map[string]interface{}) (*CodecPlainConfig, error) {
  for key := range config {
    if key == "name" {
    } else {
      return nil, errors.New(fmt.Sprintf("Invalid property for plain codec, '%s'.", key))
    }
  }
  return &CodecPlainConfig{}, nil
}

type CodecPlain struct {
  h      *Harvester
  output chan *FileEvent
}

func (codec *CodecPlain) Init() {
}

func (codec *CodecPlain) Teardown() {
}

func (codec *CodecPlain) Event(line uint64, text *string) {
  event := &FileEvent{
    ProspectorInfo: codec.h.ProspectorInfo,
    Source:         &codec.h.Path, /* If the file rotates we still send the original name before rotation until restarted */
    Offset:         codec.h.Offset,
    Line:           line,
    Text:           text,
    Fields:         &codec.h.FileConfig.Fields,
  }

  codec.output <- event // ship the new event downstream
}

func (codec *CodecPlain) Flush() {
}
