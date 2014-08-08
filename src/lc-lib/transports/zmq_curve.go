// +build zmq_4_x

/*
 * Copyright 2014 Jason Woods.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transports

import (
  "fmt"
  "lc-lib/core"
)

func (t *TransportZmq) configureSocket() (err error) {
  if t.config.transport == "zmq" {
    // Configure CurveMQ security
    if err = t.dealer.SetCurveServerkey(t.config.CurveServerkey); err != nil {
      return fmt.Errorf("Failed to set ZMQ curve server key: %s", err)
    }
    if err = t.dealer.SetCurvePublickey(t.config.CurvePublickey); err != nil {
      return fmt.Errorf("Failed to set ZMQ curve public key: %s", err)
    }
    if err = t.dealer.SetCurveSecretkey(t.config.CurveSecretkey); err != nil {
      return fmt.Errorf("Failed to set ZMQ curve secret key: %s", err)
    }
  }
  return
}

// Register the transport
func init() {
  core.RegisterTransport("zmq", NewZmqTransportFactory)
}
