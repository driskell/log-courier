/*
 * Copyright 2012-2023 Jason Woods and contributors
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

package addresspool

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type PoolEntry struct {
	Expire   time.Duration
	Server   string
	Desc     string
	HostPort string

	addresses []*Address
	refreshAt time.Time
}

func (e *PoolEntry) populateAddresses() error {
	// Standard host:port declaration
	var host string
	var portStr string
	var port uint64
	var err error
	if host, portStr, err = net.SplitHostPort(e.HostPort); err != nil {
		return fmt.Errorf("invalid hostport given: %s", e.HostPort)
	}

	if port, err = strconv.ParseUint(portStr, 10, 16); err != nil {
		return fmt.Errorf("invalid port given: %s", portStr)
	}

	if ip := net.ParseIP(host); ip != nil {
		// IP address
		e.addresses = append(e.addresses, &Address{
			host: host,
			desc: e.Desc,
			addr: &net.TCPAddr{
				IP:   ip,
				Port: int(port),
			},
		})
		return nil
	}

	// Lookup the hostname in DNS
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failure \"%s\": %s", host, err)
	} else if len(ips) == 0 {
		return fmt.Errorf("DNS lookup failure \"%s\": No addresses found", host)
	}

	for _, ip := range ips {
		addr := &net.TCPAddr{
			IP:   ip,
			Port: int(port),
		}
		e.addresses = append(e.addresses, &Address{
			host: host,
			desc: fmt.Sprintf("%s - %s", addr.String(), e.Desc),
			addr: addr,
		})
	}

	return nil
}

func (e *PoolEntry) Next() (*Address, error) {
	if len(e.addresses) == 0 {
		e.refreshAt = time.Now().Add(e.Expire)
		e.addresses = nil
		if err := e.populateAddresses(); err != nil {
			return nil, err
		}
	}

	address := e.addresses[0]
	e.addresses = e.addresses[1:]
	return address, nil
}
