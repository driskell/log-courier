package main

type CodecRegistrar interface {
  NewFactory(string, map[string]interface{}) (CodecFactory, error)
}

type CodecFactory interface {
  NewCodec(string, *FileConfig, *ProspectorInfo, int64, chan<- *FileEvent) Codec
}

type Codec interface {
  Teardown() int64
  Event(int64, int64, uint64, *string)
}

var codecRegistry map[string]CodecRegistrar = make(map[string]CodecRegistrar)

func RegisterCodec(registrar CodecRegistrar, name string) {
  codecRegistry[name] = registrar
}

func NewCodecFactory(config_path string, name string, config map[string]interface{}) (CodecFactory, error) {
  if registrar, ok := codecRegistry[name]; ok {
    return registrar.NewFactory(config_path, config)
  }
  return nil, nil
}
