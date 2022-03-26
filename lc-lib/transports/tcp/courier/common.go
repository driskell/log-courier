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

package courier

import "github.com/driskell/log-courier/lc-lib/transports/tcp"

const (
	// TransportTCPTCP is the transport name for plain TCP
	TransportCourier = "tcp"
	// TransportTCPTLS is the transport name for encrypted TLS
	TransportCourierTLS = "tls"
)

var (
	// clientName holds the client identifier to send in VERS and HELO
	clientName string = "\x00\x00\x00\x00"

	// clientNameMapping holds mapping from short name to full name for HELO and VERS
	clientNameMapping map[string]string = map[string]string{
		"LCOR": "Log Courier",
		"LCVR": "Log Carver",
		"RYLC": "Ruby Log Courier",
	}
)

type eventsMessage interface {
	tcp.ProtocolMessage
	Nonce() *string
	Events() [][]byte
}

func SetClientName(client string) {
	clientName = client
}
