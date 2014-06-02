package main

// Implement io.Writer in the transport interface so we can use binary.Write
type Transport interface {
  Connect() error
  CanSend() <-chan int
  Write([]byte) (int, error)
  Flush() error
  Read() <-chan interface{}
  Disconnect()
}
