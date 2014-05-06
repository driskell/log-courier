package main

import (
  "errors"
  "fmt"
)

type CodecPlainFactory struct {
}

type CodecPlain struct {
  harvester   *Harvester
  output chan *FileEvent
}

func CreateCodecPlainFactory(config map[string]interface{}) (*CodecPlainFactory, error) {
  for key := range config {
    if key == "name" {
    } else {
      return nil, errors.New(fmt.Sprintf("Invalid property for plain codec, '%s'.", key))
    }
  }
  return &CodecPlainFactory{}, nil
}

func (cf *CodecPlainFactory) Create(harvester *Harvester, output chan *FileEvent) Codec {
  return &CodecPlain{harvester: harvester, output: output}
}


func (c *CodecPlain) Teardown() int64 {
  return c.harvester.Offset
}

func (c *CodecPlain) Event(line uint64, text *string) {
  event := &FileEvent{
    ProspectorInfo: c.harvester.ProspectorInfo,
    Source:         &c.harvester.Path, /* If the file rotates we still send the original name before rotation until restarted */
    Offset:         c.harvester.Offset,
    Line:           line,
    Text:           text,
    Fields:         &c.harvester.FileConfig.Fields,
  }

  c.output <- event // ship the new event downstream
}

func (c *CodecPlain) Flush() {
}
