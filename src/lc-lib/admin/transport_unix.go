// +build !windows

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

package admin

import (
	"fmt"
	"net"
)

func init() {
	registerTransport("unix", connectUnix, listenUnix)
}

func connectUnix(transport, path string) (net.Conn, error) {
	uaddr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("The connection address specified is not valid: %s", err)
	}

	// TODO: Change umask to 111 so all can write (need to move to _unix)
	// Permission will be controlled by folder permissions instead of file
	conn, err := net.DialUnix("unix", nil, uaddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func listenUnix(transport, addr string) (NetListener, error) {
	uaddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("The admin bind address specified is not valid: %s", err)
	}

	listener, err := net.ListenUnix("unix", uaddr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
