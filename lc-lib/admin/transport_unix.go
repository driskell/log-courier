// +build !windows

/*
* Copyright 2012-2020 Jason Woods and contributors
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

package admin

import (
	"fmt"
	"net"
	"os"
)

func init() {
	registerTransport("unix", connectUnix, listenUnix)
}

type unixDialer struct {
	uaddr *net.UnixAddr
}

func (d *unixDialer) Dial(network string, addr string) (net.Conn, error) {
	return net.DialUnix("unix", nil, d.uaddr)
}

func (d *unixDialer) Host() string {
	return d.uaddr.Name
}

func connectUnix(transport, path string) (netDialer, error) {
	uaddr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("The connection address specified is not valid: %s", err)
	}

	// TODO: Change umask to 111 so all can write (need to move to _unix)
	// Permission will be controlled by folder permissions instead of file
	dialer := &unixDialer{
		uaddr: uaddr,
	}

	return dialer, nil
}

func listenUnix(transport, addr string) (netListener, error) {
	uaddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("The admin bind address specified is not valid: %s", err)
	}

	// Remove previous socket file if it's still there or we'll get address
	// already in use error
	if _, err = os.Stat(addr); err == nil || !os.IsNotExist(err) {
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("Failed to remove the existing socket file: %s", err)
		}
	}

	listener, err := net.ListenUnix("unix", uaddr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
