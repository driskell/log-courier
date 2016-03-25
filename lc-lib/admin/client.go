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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

// ClientError represents a successful request where Log Courier returned an
// error, as opposed to an error processing a request
type ClientError error

// Client provides an interface for accessing the REST API with pretty responses
type Client struct {
	adminConnect  string
	transport     *http.Transport
	client        *http.Client
	remoteVersion string
}

// NewClient returns a new Client interface for the given endpoint
func NewClient(adminConnect string) (*Client, error) {
	ret := &Client{
		adminConnect: adminConnect,
	}

	if err := ret.initClient(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) initClient() error {
	bind := splitAdminConnectString(c.adminConnect)

	dialer, ok := registeredDialers[bind[0]]
	if !ok {
		return fmt.Errorf("Unknown transport specified for admin bind: '%s'", bind[0])
	}

	dialerStruct, err := dialer(bind[0], bind[1])
	if err != nil {
		return err
	}

	c.transport = &http.Transport{
		Dial: func(network string, addr string) (net.Conn, error) {
			return dialerStruct.Dial(network, addr)
		},
	}

	c.client = &http.Client{
		Transport: c.transport,
	}

	remoteVersion, err := c.Request("version")
	if err != nil {
		return err
	}

	c.remoteVersion = remoteVersion

	return nil
}

// RemoteVersion returns the version of the remotely connected Log Courier
func (c *Client) RemoteVersion() string {
	return c.remoteVersion
}

// Request performs a request and returns a pretty response
func (c *Client) Request(path string) (string, error) {
	resp, err := c.client.Get("http://log-courier/" + path + "?w=pretty")
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		data := make(map[string]interface{})
		if err := json.Unmarshal(body, &data); err != nil {
			return "", err
		}

		if dataErr, ok := data["error"].(string); ok {
			return "", ClientError(errors.New(dataErr))
		}
	}

	return string(body), nil
}
