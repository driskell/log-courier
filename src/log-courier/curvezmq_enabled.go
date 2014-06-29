// +build zmq_4_x

package main

func (t *TransportZmq) configureSocket() (err error) {
  if t.config.transport == "curvezmq" {
    // Configure CurveMQ security
    if err = t.dealer.SetCurveServerkey(t.config.CurveServerkey); err != nil {
      return
    }
    if err = t.dealer.SetCurvePublickey(t.config.CurvePublickey); err != nil {
      return
    }
    if err = t.dealer.SetCurveSecretkey(t.config.CurveSecretkey); err != nil {
      return
    }
  }
  return
}

// Register the transport
func init() {
  RegisterTransport(&TransportZmqRegistrar{}, "curvezmq")
}
