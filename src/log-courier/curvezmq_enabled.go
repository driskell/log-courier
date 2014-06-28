// +build zmq_4_x

package main

import zmq "github.com/alecthomas/gozmq"

func ZMQConfigureSocket(dealer *zmq.Socket, config *TransportZmqFactory) (err error) {
  if config.transport == "curvezmq" {
    // Configure CurveMQ security
    if err = dealer.SetCurveServerkey(config.CurveServerkey); err != nil {
      return
    }
    if err = dealer.SetCurvePublickey(config.CurvePublickey); err != nil {
      return
    }
    if err = dealer.SetCurveSecretkey(config.CurveSecretkey); err != nil {
      return
    }
  }
  return
}

// Register the transport
func init() {
  RegisterTransport(&TransportZmqRegistrar{}, "curvezmq")
}
