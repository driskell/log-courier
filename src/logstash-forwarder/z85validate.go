// +build zmq

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
