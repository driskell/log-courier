package main

// Implement io.Writer in the transport interface so we can use binary.Read/Write etc
// Write will buffer until we flush to keep the message small as possible
type Transport interface {
  Connect() error
  Write([]byte) (int, error)
  Flush() (int64, error)
  Read([]byte) (int, error)
  Disconnect()
}
