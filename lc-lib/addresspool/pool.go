/*
 * Copyright 2012-2023 Jason Woods and contributors
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
	"time"
)

// Pool looks up server addresses and manages a pool of IPs
type Pool struct {
	server         string
	rfc2782        bool
	rfc2782Service string
	hostAddresses  []*hostAddress
	current        []*hostAddress
	ttl            time.Duration
}

// NewPool creates a new Pool instance for a server
func NewPool(server string) *Pool {
	return &Pool{
		server: server,
		ttl:    300 * time.Second,
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
	p.expireAndPopulate()

	return len(p.current) == 0
}

// Next returns the next available IP address from the pool
// Each time all IPs have been returned, the server is looked up again if
// necessary and the IP addresses are returned again in order.
func (p *Pool) Next() (*Address, error) {
	p.expireAndPopulate()

	if len(p.current) == 0 {
		if err := p.populateHostAddresses(); err != nil {
			return nil, err
		}
		p.current = p.hostAddresses
	}

	next := p.current[0]
	p.current = p.current[1:]
	return next.Next()
}

func (p *Pool) expireAndPopulate() {
	// Skip past any expired hosts
	// This ensures after long period of no activity, we refresh DNS
	for len(p.current) != 0 && p.current[0].Finished() {
		p.current = p.current[1:]
	}

	// Have we exhausted the address list we had? Look up the addresses again
	// Otherwise reuse the hosts we still have addresses for
	// We don't need to worry about primary hosts with single addresses dropping
	// off and secondaries getting lots of attempts due to more addresses as we
	// expire the address list fairly quickly, forcing a fresh lookup and adding
	// the primary again
	if len(p.current) == 0 {
		newHostAddresses := make([]*hostAddress, 0)
		for _, v := range p.hostAddresses {
			if !v.Finished() {
				newHostAddresses = append(newHostAddresses, v)
			}
		}
		p.hostAddresses = newHostAddresses
		p.current = newHostAddresses
	}
}

// Server returns the server configuration entry the address pool was associated with
func (p *Pool) Server() string {
	return p.server
}

// populateAddresses performs the lookups necessary to obtain the pool of
// addresses for the associated server
func (p *Pool) populateHostAddresses() error {
	// @hostname means SRV record where the host and port are in the record
	if len(p.server) > 0 && p.server[0] == '@' {
		return p.processSrv(p.server[1:])
	}

	p.hostAddresses = append(p.hostAddresses, &hostAddress{
		expire:   time.Now().Add(p.ttl),
		desc:     p.server,
		hostPort: p.server,
	})
	return nil
}

// processSrv looks up SRV records based on the SRV settings
func (p *Pool) processSrv(server string) error {
	var service, protocol string

	if p.rfc2782 {
		service, protocol = p.rfc2782Service, "tcp"
	} else {
		service, protocol = "", ""
	}

	_, srvs, err := net.LookupSRV(service, protocol, server)
	if err != nil {
		return fmt.Errorf("DNS SRV lookup failure \"%s\": %s", server, err)
	} else if len(srvs) == 0 {
		return fmt.Errorf("DNS SRV lookup failure \"%s\": No targets found", server)
	}

	for _, srv := range srvs {
		p.hostAddresses = append(p.hostAddresses, &hostAddress{
			expire:   time.Now().Add(p.ttl),
			desc:     fmt.Sprintf("%s (%s)", server, srv.Target),
			hostPort: fmt.Sprintf("%s:%d", srv.Target, srv.Port),
		})
	}

	return nil
}
