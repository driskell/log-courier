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
	"net/url"
)

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
// It will defer to Call method if the path is a known Call path
func (c *Client) Request(path string) (string, error) {
	// Is this a Call request?
	if _, ok := callMap[path]; ok {
		return c.Call(path, url.Values{})
	}

	resp, err := c.client.Get("http://log-courier/" + path + "?w=pretty")
	if err != nil {
		return "", err
	}

	return c.handleResponse(resp)
}

// Call performs a remote action and returns the result
func (c *Client) Call(path string, values url.Values) (string, error) {
	resp, err := c.client.PostForm("http://log-courier/"+path+"?w=pretty", values)
	if err != nil {
		return "", err
	}

	return c.handleResponse(resp)
}

func (c *Client) handleResponse(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Ensure we close body so we don't leave hanging connections open
	if err := resp.Body.Close(); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp, body)
	}

	return string(body), nil
}

func (c *Client) handleError(resp *http.Response, body []byte) (string, error) {
	// Return friendly Not Found as Unknown command
	switch resp.StatusCode {
	case http.StatusNotFound:
		return "", ErrNotFound
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if dataErr, ok := data["error"].(string); ok {
		return "", ErrUnknown(errors.New(dataErr))
	}

	return string(body), nil
}
