package main

type Codec interface {
  Teardown()
  Event(uint64, *string)
}

type CodecFactory interface {
  Create(*Harvester, chan *FileEvent) Codec
}
