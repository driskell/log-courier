package main

type Codec interface {
  Teardown() int64
  Event(uint64, *string)
}

type CodecFactory interface {
  Create(*Harvester, chan *FileEvent) Codec
}
