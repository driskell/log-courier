/*
* Copyright 2014-2015 Jason Woods.
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
	registerTransport("tcp", connectTCP, listenTCP)
	registerTransport("tcp4", connectTCP, listenTCP)
	registerTransport("tcp6", connectTCP, listenTCP)
}

func connectTCP(transport, addr string) (net.Conn, error) {
	taddr, err := net.ResolveTCPAddr(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("The connection address specified is not valid: %s", err)
	}

	conn, err := net.DialTCP(transport, nil, taddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func listenTCP(transport, addr string) (netListener, error) {
	taddr, err := net.ResolveTCPAddr(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("The admin bind address specified is not valid: %s", err)
	}

	listener, err := net.ListenTCP(transport, taddr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
