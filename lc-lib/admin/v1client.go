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
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/driskell/log-courier/lc-lib/core"
)

// V1Client is a client compatible with Log Courier 1.x
type V1Client struct {
	adminConnect string
	conn         net.Conn
	decoder      *gob.Decoder
}

// NewV1Client returns a new admin client compatible with Log Courier 1.x
func NewV1Client(adminConnect string) (*V1Client, error) {
	var err error

	ret := &V1Client{}

	if ret.conn, err = ret.connect(adminConnect); err != nil {
		return nil, err
	}

	ret.decoder = gob.NewDecoder(ret.conn)

	return ret, nil
}

func (c *V1Client) connect(adminConnect string) (net.Conn, error) {
	connect := splitAdminConnectString(adminConnect)

	if dialer, ok := registeredDialers[connect[0]]; ok {
		dialerStruct, err := dialer(connect[0], connect[1])
		if err != nil {
			return nil, err
		}

		return dialerStruct.Dial(connect[0], connect[1])
	}

	return nil, fmt.Errorf("Unknown transport specified in connection address: '%s'", connect[0])
}

func (c *V1Client) request(command string) (*Response, error) {
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}

	totalWritten := 0

	for {
		wrote, err := c.conn.Write([]byte(command[totalWritten:4]))
		if err != nil {
			return nil, err
		}

		totalWritten += wrote
		if totalWritten == 4 {
			break
		}
	}

	var response Response

	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}

	if err := c.decoder.Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *V1Client) resolveError(response *Response) error {
	ret, ok := response.Response.(*ErrorResponse)
	if ok {
		return ret
	}

	return &ErrorResponse{Message: fmt.Sprintf("Unrecognised response: %v\n", ret)}
}

// Ping sends a PING command to test the admin connection
func (c *V1Client) Ping() error {
	response, err := c.request("PING")
	if err != nil {
		return err
	}

	if _, ok := response.Response.(*PongResponse); ok {
		return nil
	}

	return c.resolveError(response)
}

// Reload requests the connected Log Courier to reload its configuration
func (c *V1Client) Reload() error {
	response, err := c.request("RELD")
	if err != nil {
		return err
	}

	if _, ok := response.Response.(*ReloadResponse); ok {
		return nil
	}

	return c.resolveError(response)
}

// FetchSnapshot requests a status snapshot from Log Courier
func (c *V1Client) FetchSnapshot() (*core.Snapshot, error) {
	response, err := c.request("SNAP")
	if err != nil {
		return nil, err
	}

	if ret, ok := response.Response.(*core.Snapshot); ok {
		return ret, nil
	}

	// Backwards compatibility
	if ret, ok := response.Response.([]*core.Snapshot); ok {
		snap := core.NewSnapshot("Log Courier")
		for _, sub := range ret {
			snap.AddSub(sub)
		}
		snap.Sort()
		return snap, nil
	}

	return nil, c.resolveError(response)
}
