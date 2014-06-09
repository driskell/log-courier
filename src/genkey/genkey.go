package main

import (
  "flag"
  "fmt"
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
  var single bool

  flag.BoolVar(&single, "single", false, "generate a single keypair")
  flag.Parse()

  if single {
    fmt.Println("Generating single keypair...")

    pub, priv, err := CurveKeyPair()
    if err != nil {
      fmt.Println("An error occurred: ", err)
      if err == syscall.ENOTSUP {
        fmt.Print("Please ensure that your zeromq installation was built with libsodium support")
      }
      return
    }

    fmt.Println("Public key:  ", pub)
    fmt.Println("Private key: ", priv)
    return
  }

  fmt.Println("Generating configuration keys...")
  fmt.Println("(Use 'genkey --single' to generate a single keypair.)")
  fmt.Println("")

  server_pub, server_priv, err := CurveKeyPair()
  if err != nil {
    fmt.Println("An error occurred: ", err)
    if err == syscall.ENOTSUP {
      fmt.Print("Please ensure that your zeromq installation was built with libsodium support")
    }
    return
  }

  client_pub, client_priv, err := CurveKeyPair()
  if err != nil {
    fmt.Println("An error occurred: ", err)
    if err == syscall.ENOTSUP {
      fmt.Println("Please ensure that your zeromq installation was built with libsodium support")
    }
    return
  }

  fmt.Println("Copy and paste the following into your fmtstash-forwarder configuration(s):")
  fmt.Println("    \"curve server key\": \"", server_pub, "\",")
  fmt.Println("    \"curve public key\": \"", client_pub, "\",")
  fmt.Println("    \"curve secret key\": \"", client_priv, "\",")
  fmt.Println("")
  fmt.Println("Copy and paste the following into your fmtstash configuration:")
  fmt.Println("    curve_secret_key => \"", server_priv, "\",")
}

// Because gozmq does not yet expose this for us, we have to expose it ourselves
func CurveKeyPair() (string, string, error) {
  var pub [41]C.char
  var priv [41]C.char

  // Because gozmq does not yet expose this for us, we have to expose it ourselves
  if rc, err := C.zmq_curve_keypair(&pub[0], &priv[0]); rc != 0 {
    return "", "", casterr(err)
  }

  return C.GoString(&pub[0]), C.GoString(&priv[0]), nil
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
