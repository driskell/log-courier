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

package transports

import (
	"fmt"
	"net"
	"math/rand"
	"strconv"
	"time"
)

type AddressPool struct {
	servers        []string
	rfc2782        bool
	rfc2782Service string
	roundrobin     int
	host_is_ip     bool
	host           string
	addresses      []*net.TCPAddr
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewAddressPool(servers []string) *AddressPool {
	ret := &AddressPool{
		servers: servers,
	}

	// Randomise the initial host - after this it will round robin
	// Round robin after initial attempt ensures we don't retry same host twice,
	// and also ensures we try all hosts one by one
	ret.roundrobin = rand.Intn(len(servers))

	return ret
}

func (p *AddressPool) SetRfc2782(enabled bool, service string) {
	p.rfc2782 = enabled
	p.rfc2782Service = service
}

func (p *AddressPool) IsLast() bool {
	return p.addresses == nil && p.roundrobin%len(p.servers) == 0
}

func (p *AddressPool) Next() (*net.TCPAddr, string, error) {
	// Have we exhausted the address list we had?
	if p.addresses == nil {
		if err := p.populateAddresses(); err != nil {
			return nil, "", err
		}
	}

	next := p.addresses[0]
	if len(p.addresses) > 1 {
		p.addresses = p.addresses[1:]
	} else {
		p.addresses = nil
	}

	var desc string
	if p.host_is_ip {
		desc = fmt.Sprintf("%s", next)
	} else {
		desc = fmt.Sprintf("%s (%s)", next, p.host)
	}

	return next, desc, nil
}

func (p *AddressPool) Host() string {
	return p.host
}

func (p *AddressPool) populateAddresses() (err error) {
	// Round robin to the next server
	selected := p.servers[p.roundrobin%len(p.servers)]
	p.roundrobin++

	p.addresses = make([]*net.TCPAddr, 0)

	// @hostname means SRV record where the host and port are in the record
	if len(selected) > 0 && selected[0] == '@' {
		var srvs []*net.SRV
		var service, protocol string

		p.host = selected[1:]
		p.host_is_ip = false

		if p.rfc2782 {
			service, protocol = p.rfc2782Service, "tcp"
		} else {
			service, protocol = "", ""
		}

		_, srvs, err = net.LookupSRV(service, protocol, p.host)
		if err != nil {
			return fmt.Errorf("DNS SRV lookup failure \"%s\": %s", p.host, err)
		} else if len(srvs) == 0 {
			return fmt.Errorf("DNS SRV lookup failure \"%s\": No targets found", p.host)
		}

		for _, srv := range srvs {
			if _, err = p.populateLookup(srv.Target, int(srv.Port)); err != nil {
				return
			}
		}

		return
	}

	// Standard host:port declaration
	var port_str string
	var port uint64
	if p.host, port_str, err = net.SplitHostPort(selected); err != nil {
		return fmt.Errorf("Invalid hostport given: %s", selected)
	}

	if port, err = strconv.ParseUint(port_str, 10, 16); err != nil {
		return fmt.Errorf("Invalid port given: %s", port_str)
	}

	if p.host_is_ip, err = p.populateLookup(p.host, int(port)); err != nil {
		return
	}

	return nil
}

func (p *AddressPool) populateLookup(host string, port int) (bool, error) {
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
