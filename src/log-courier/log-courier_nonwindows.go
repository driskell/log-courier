// +build !windows

package main

import "syscall"

func init() {
	// *nix systems support SIGTERM so handle shutdown on that too
	RegisterShutdownSignal(syscall.SIGTERM)
}
