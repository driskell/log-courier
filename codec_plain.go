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
  path string
  fileconfig *FileConfig
  info *ProspectorInfo
  last_offset int64
  output chan<- *FileEvent
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

func (f *CodecPlainFactory) NewCodec(path string, fileconfig *FileConfig, info *ProspectorInfo, offset int64, output chan<- *FileEvent) Codec {
  return &CodecPlain{
    path: path,
    fileconfig: fileconfig,
    info: info,
    last_offset: offset,
    output: output,
  }
}

func (c *CodecPlain) Teardown() int64 {
  return c.last_offset
}

func (c *CodecPlain) Event(start_offset int64, end_offset int64, line uint64, text *string) {
  c.last_offset = end_offset

  // Ship downstream
  c.output <- &FileEvent{
    ProspectorInfo: c.info,
    Offset:         end_offset,
    Event:          NewEvent(c.fileconfig.Fields, &c.path, start_offset, line, text),
  }
}

// Register the codec as default
func init() {
  RegisterCodec(&CodecPlainRegistrar{}, "")
  RegisterCodec(&CodecPlainRegistrar{}, "plain")
}
