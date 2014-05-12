package main

type Codec interface {
  Teardown() int64
  Event(int64, uint64, *string)
}

type CodecFactory interface {
  Create(*Harvester, chan *FileEvent) Codec
}
