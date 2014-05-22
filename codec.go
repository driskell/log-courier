package main

type CodecRegistrar interface {
  NewFactory(map[string]interface{}) (CodecFactory, error)
}

type CodecFactory interface {
  NewCodec(*Harvester, chan *FileEvent) Codec
}

type Codec interface {
  Teardown() int64
  Event(int64, uint64, *string)
}

var codecRegistry map[string]CodecRegistrar = make(map[string]CodecRegistrar);

func RegisterCodec(registrar CodecRegistrar, name string) {
  codecRegistry[name] = registrar
}

func NewCodecFactory(name string, config map[string]interface{}) (CodecFactory, error) {
  if registrar, ok := codecRegistry[name]; ok {
    return registrar.NewFactory(config)
  }
  return nil, nil
}
