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

/*
#cgo pkg-config: libzmq
#include <zmq.h>
#include <zmq_utils.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"syscall"
	"unsafe"
)

func (f *TransportZmqFactory) processConfig(config_path string) (err error) {
	if len(f.CurveServerkey) == 0 {
		return fmt.Errorf("Option %scurve server key is required", config_path)
	} else if len(f.CurveServerkey) != 40 || !z85Validate(f.CurveServerkey) {
		return fmt.Errorf("Option %scurve server key must be a valid 40 character Z85 encoded string", config_path)
	}
	if len(f.CurvePublickey) == 0 {
		return fmt.Errorf("Option %scurve public key is required", config_path)
	} else if len(f.CurvePublickey) != 40 || !z85Validate(f.CurvePublickey) {
		return fmt.Errorf("Option %scurve public key must be a valid 40 character Z85 encoded string", config_path)
	}
	if len(f.CurveSecretkey) == 0 {
		return fmt.Errorf("Option %scurve secret key is required", config_path)
	} else if len(f.CurveSecretkey) != 40 || !z85Validate(f.CurveSecretkey) {
		return fmt.Errorf("Option %scurve secret key must be a valid 40 character Z85 encoded string", config_path)
	}

	return nil
}

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

// Process ZMQ 4.0.x monitor messages
// http://api.zeromq.org/4-0:zmq-socket-monitor
func (t *TransportZmq) processMonitorIn() (ok bool) {
	for {
		// Bring in the messages
	RetryRecv:
		data, err := t.monitor.Recv(zmq.DONTWAIT)
		if err != nil {
			switch err {
			case syscall.EINTR:
				// Try again
				goto RetryRecv
			case syscall.EAGAIN:
				// No more messages
				ok = true
				return
			}

			// Failure
			t.recv_chan <- fmt.Errorf("Monitor zmq.Socket.Recv failure %s", err)
			return
		}

		switch t.event.part {
		case Monitor_Part_Header:
			t.event.event = zmq.Event(binary.LittleEndian.Uint16(data[0:2]))
			t.event.val = int32(binary.LittleEndian.Uint32(data[2:6]))
			t.event.data = ""
		case Monitor_Part_Data:
			t.event.data = string(data)
			t.event.Log()
		default:
			log.Debug("Extraneous data in monitor message. Silently discarding.")
			continue
		}

		more, err := t.monitor.RcvMore()
		if err != nil {
			// Failure
			t.recv_chan <- fmt.Errorf("Monitor zmq.Socket.RcvMore failure %s", err)
			return
		}

		if !more {
			if t.event.part < Monitor_Part_Data {
				t.event.Log()
				log.Debug("Unexpected end of monitor message. Skipping.")
			}

			t.event.part = Monitor_Part_Header
			continue
		}

		if t.event.part <= Monitor_Part_Data {
			t.event.part++
		}
	}
}

func z85Validate(z85 string) bool {
	var decoded []C.uint8_t

	if len(z85)%5 != 0 {
		return false
	} else {
		// Avoid literal floats
		decoded = make([]C.uint8_t, 8*len(z85)/10)
	}

	// Grab a CString of the z85 we need to decode
	c_z85 := C.CString(z85)
	defer C.free(unsafe.Pointer(c_z85))

	// Because gozmq does not yet expose this for us, we have to expose it ourselves
	if ret := C.zmq_z85_decode(&decoded[0], c_z85); ret == nil {
		return false
	}

	return true
}

// Register the transport
func init() {
	core.RegisterTransport("zmq", NewZmqTransportFactory)
}
