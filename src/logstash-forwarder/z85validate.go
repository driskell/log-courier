// +build zmq

package main

import (
  "unsafe"
)

/*
#cgo pkg-config: libzmq
#include <zmq.h>
#include <zmq_utils.h>
*/
import "C"

func Z85Validate(z85 string) bool {
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
