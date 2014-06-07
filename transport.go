package main

type TransportFactory interface {
  NewTransport(*NetworkConfig) (Transport, error)
}

type Transport interface {
  Connect() error
  CanSend() <-chan int
  Write(string, []byte) error
  Read() <-chan interface{}
  Disconnect()
}
