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
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// GeneratePool parses a given list of servers and returns pool entries
// It also expands SRV records into multiple entries
func GeneratePool(servers []string, rfc2782 bool, service string, expire time.Duration) ([]*PoolEntry, error) {
	poolEntries := make([]*PoolEntry, 0, len(servers))

	var errs []string
	for _, server := range servers {
		// @hostname means SRV record where the host and port are in the record
		if len(server) > 0 && server[0] == '@' {
			var err error
			if poolEntries, err = processSrv(poolEntries, server, rfc2782, service, expire); err != nil {
				errs = append(errs, err.Error())
			}
			continue
		}

		poolEntries = append(poolEntries, &PoolEntry{
			Expire:   expire,
			Server:   server,
			Desc:     server,
			HostPort: server,
		})
	}

	if len(errs) != 0 {
		return poolEntries, errors.New(strings.Join(errs, "; "))
	}
	return poolEntries, nil
}

// processSrv looks up SRV records based on the SRV settings
func processSrv(poolEntries []*PoolEntry, server string, rfc2782 bool, rfc2782Service string, expire time.Duration) ([]*PoolEntry, error) {
	var service, protocol string

	if rfc2782 {
		service, protocol = rfc2782Service, "tcp"
	} else {
		service, protocol = "", ""
	}

	_, srvs, err := net.LookupSRV(service, protocol, server[1:])
	if err != nil {
		return poolEntries, fmt.Errorf("DNS SRV lookup failure \"%s\": %s", server[1:], err)
	} else if len(srvs) == 0 {
		return poolEntries, fmt.Errorf("DNS SRV lookup failure \"%s\": No targets found", server[1:])
	}

	for _, srv := range srvs {
		target := strings.TrimRight(srv.Target, ".")

		poolEntries = append(poolEntries, &PoolEntry{
			Expire:   expire,
			Server:   server,
			Desc:     fmt.Sprintf("%s:%d - %s", target, srv.Port, server),
			HostPort: fmt.Sprintf("%s:%d", target, srv.Port),
		})
	}

	return poolEntries, nil
}
