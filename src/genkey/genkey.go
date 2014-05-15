package main

import (
  "log"
  "syscall"
  zmq "github.com/alecthomas/gozmq"
)

/*
#cgo pkg-config: libzmq
#include <zmq.h>
#include <zmq_utils.h>
*/
import "C"

func main() {
  var priv [41]C.char
  var pub [41]C.char

  log.Printf("Generating key pair\n")

  // Because gozmq does not yet expose this for us, we have to expose it ourselves
  if rc, err := C.zmq_curve_keypair(&priv[0], &pub[0]); rc != 0 {
    if err == syscall.ENOTSUP {
      log.Printf("An error occurred: %s\n", casterr(err))
      log.Fatalf("Please ensure that your zeromq installation was built with libsodium support\n")
    } else {
      log.Fatalf("An error occurred: %s\n", casterr(err))
    }
  }

  log.Printf("Private: %s\n", C.GoString(&priv[0]))
  log.Printf("Public: %s\n", C.GoString(&pub[0]))
}

// The following is copy-pasted from gozmq's zmq.go
// If possible, convert a syscall.Errno to a zmqErrno.
type zmqErrno syscall.Errno

func (e zmqErrno) Error() string {
	return C.GoString(C.zmq_strerror(C.int(e)))
}

func casterr(fromcgo error) error {
	errno, ok := fromcgo.(syscall.Errno)
	if !ok {
		return fromcgo
	}
	zmqerrno := zmqErrno(errno)
	switch zmqerrno {
	case zmq.ENOTSOCK:
		return zmqerrno
	}
	if zmqerrno >= C.ZMQ_HAUSNUMERO {
		return zmqerrno
	}
	return errno
}
