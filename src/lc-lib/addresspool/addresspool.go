/*
 * Copyright 2014 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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
)

// Pool looks up server addresses and manages a pool of IPs
type Pool struct {
	server         string
	rfc2782        bool
	rfc2782Service string
	hostIsIP       bool
	host           string
	desc           string
	addresses      []*net.TCPAddr
}

// NewPool creates a new Pool instance for a server
func NewPool(server string) *Pool {
	return &Pool{
		server: server,
	}
}

// SetRfc2782 enables RFC compliant handling of SRV server entries using the
// given service name
func (p *Pool) SetRfc2782(enabled bool, service string) {
	p.rfc2782 = enabled
	p.rfc2782Service = service
}

// IsLast returns true if the next call to Next will return the first address
// in the pool. In other words, if the last call to Next returned the last entry
// or has never been called
func (p *Pool) IsLast() bool {
	return p.addresses == nil
}

// Next returns the next available IP address from the pool
// Each time all IPs have been returned, the server is looked up again if
// necessary and the IP addresses are returned again in order.
func (p *Pool) Next() (*net.TCPAddr, error) {
	// Have we exhausted the address list we had? Look up the addresses again
	// TODO: Should we expire the list to ensure old entries are not reused after
	// many many hours/days?
	if p.addresses == nil {
		p.addresses = make([]*net.TCPAddr, 0)
		if err := p.populateAddresses(); err != nil {
			p.addresses = nil
			return nil, err
		}
	}

	next := p.addresses[0]
	if len(p.addresses) > 1 {
		p.addresses = p.addresses[1:]
	} else {
		p.addresses = nil
	}

	if p.hostIsIP {
		p.desc = fmt.Sprintf("%s", next)
	} else {
		p.desc = fmt.Sprintf("%s (%s)", next, p.host)
	}

	return next, nil
}

// Server returns the server configuration entry the address pool was associated
// with
func (p *Pool) Server() string {
	return p.server
}

// Host returns the DNS hostname for the last returned address. This can be used
// for server name verification such as with TLS
func (p *Pool) Host() string {
	return p.host
}

// Desc returns a friendly description of the last returned server.
// Example for an IP: 127.0.0.1
//                Hostname: localhost (127.0.0.1)
// TODO: Improve Desc result for SRV records to include the SRV record
func (p *Pool) Desc() string {
	return p.host
}

// populateAddresses performs the lookups necessary to obtain the pool of IP
// addresses for the associated server
func (p *Pool) populateAddresses() error {
	// @hostname means SRV record where the host and port are in the record
	if len(p.server) > 0 && p.server[0] == '@' {
		srvs, err := p.processSrv(p.server[1:])
		if err != nil {
			return err
		}

		for _, srv := range srvs {
			if _, err := p.populateLookup(srv.Target, int(srv.Port)); err != nil {
				return err
			}
		}

		return nil
	}

	// Standard host:port declaration
	var portStr string
	var port uint64
	var err error
	if p.host, portStr, err = net.SplitHostPort(p.server); err != nil {
		return fmt.Errorf("Invalid hostport given: %s", p.server)
	}

	if port, err = strconv.ParseUint(portStr, 10, 16); err != nil {
		return fmt.Errorf("Invalid port given: %s", portStr)
	}

	if p.hostIsIP, err = p.populateLookup(p.host, int(port)); err != nil {
		return err
	}

	return nil
}

// processSrv looks up SRV records based on the SRV settings
// TODO: processSrv sets Host() to the SRV record name, not the target hostname,
//       which would potentially break certificate name verification
func (p *Pool) processSrv(server string) ([]*net.SRV, error) {
	var service, protocol string

	p.host = server
	p.hostIsIP = false

	if p.rfc2782 {
		service, protocol = p.rfc2782Service, "tcp"
	} else {
		service, protocol = "", ""
	}

	_, srvs, err := net.LookupSRV(service, protocol, p.host)
	if err != nil {
		return nil, fmt.Errorf("DNS SRV lookup failure \"%s\": %s", p.host, err)
	} else if len(srvs) == 0 {
		return nil, fmt.Errorf("DNS SRV lookup failure \"%s\": No targets found", p.host)
	}

	return srvs, nil
}

// populateLookup detects IP addresses and looks up DNS A records
func (p *Pool) populateLookup(host string, port int) (bool, error) {
	if ip := net.ParseIP(host); ip != nil {
		// IP address
		p.addresses = append(p.addresses, &net.TCPAddr{
			IP:   ip,
			Port: port,
		})

		return true, nil
	}

	// Lookup the hostname in DNS
	ips, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("DNS lookup failure \"%s\": %s", host, err)
	} else if len(ips) == 0 {
		return false, fmt.Errorf("DNS lookup failure \"%s\": No addresses found", host)
	}

	for _, ip := range ips {
		p.addresses = append(p.addresses, &net.TCPAddr{
			IP:   ip,
			Port: port,
		})
	}

	return false, nil
}
