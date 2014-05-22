package main

import (
  "errors"
  "fmt"
)

type CodecPlainRegistrar struct {
}

type CodecPlainFactory struct {
}

type CodecPlain struct {
  harvester *Harvester
  output    chan *FileEvent
}

func (r *CodecPlainRegistrar) NewFactory(config map[string]interface{}) (CodecFactory, error) {
  for key := range config {
    if key == "name" {
    } else {
      return nil, errors.New(fmt.Sprintf("Invalid property for plain codec, '%s'.", key))
    }
  }
  return &CodecPlainFactory{}, nil
}

func (f *CodecPlainFactory) NewCodec(harvester *Harvester, output chan *FileEvent) Codec {
  return &CodecPlain{harvester: harvester, output: output}
}

func (c *CodecPlain) Teardown() int64 {
  return c.harvester.Offset
}

func (c *CodecPlain) Event(offset int64, line uint64, text *string) {
  event := &FileEvent{
    ProspectorInfo: c.harvester.ProspectorInfo,
    Offset:         c.harvester.Offset,
    Event:          NewEvent(c.harvester.FileConfig.Fields, &c.harvester.Path, offset, line, text),
  }

  c.output <- event // ship the new event downstream
}

func (c *CodecPlain) Flush() {
}

// Register the codec as default
func init() {
  RegisterCodec(&CodecPlainRegistrar{}, "")
  RegisterCodec(&CodecPlainRegistrar{}, "plain")
}
