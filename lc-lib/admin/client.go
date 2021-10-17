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

package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/driskell/log-courier/lc-lib/admin/api"
)

var (
	// callMap is a list of commands known to be Call only, and the Client uses
	// this to automatically translate Request calls into Call calls to simplify
	// logic in clients
	callMap = map[string]interface{}{
		"reload": nil,
	}
)

// Client provides an interface for accessing the REST API with pretty responses
type Client struct {
	adminConnect  string
	transport     *http.Transport
	client        *http.Client
	remoteSummary map[string]interface{}
	remoteName    string
	remoteVersion string
	fakeHost      string
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
		return fmt.Errorf("unknown transport specified for admin bind: '%s'", bind[0])
	}

	dialerStruct, err := dialer(bind[0], bind[1])
	if err != nil {
		return err
	}

	c.fakeHost = dialerStruct.Host()

	c.transport = &http.Transport{
		Dial: func(network string, addr string) (net.Conn, error) {
			return dialerStruct.Dial(network, addr)
		},
	}

	c.client = &http.Client{
		Transport: c.transport,
	}

	c.remoteSummary = make(map[string]interface{})
	err = c.RequestJSONSummary("", &c.remoteSummary)
	if err != nil {
		return err
	}

	if name, ok := c.remoteSummary["name"].(string); ok {
		c.remoteName = name
	}
	if version, ok := c.remoteSummary["version"].(string); ok {
		c.remoteVersion = version
	}

	return nil
}

// RemoteVersion returns the version of the remotely connected Log Courier
func (c *Client) RemoteClient() (string, string) {
	return c.remoteName, c.remoteVersion
}

// RemoteSummary returns the summary data of the root node on the remote
// It allows detection of pipeline features
func (c *Client) RemoteSummary() map[string]interface{} {
	return c.remoteSummary
}

// RequestJSON performs a request and returns a JSON response
// It will defer to Call method if the path is a known Call path
func (c *Client) RequestJSON(path string, target interface{}) error {
	return c.rawRequestJSON(path, target, map[string]string{})
}

// RequestJSONSummary performs a request and returns a JSON response
// It will only return top level items from the requested node
// Any deeper nodes will be replaced with a structure containing their type
// It will defer to Call method if the path is a known Call path
func (c *Client) RequestJSONSummary(path string, target interface{}) error {
	return c.rawRequestJSON(path, target, map[string]string{"w": "summary"})
}

// RequestPretty performs a request and returns a human readable response
// It will defer to Call method if the path is a known Call path
func (c *Client) RequestPretty(path string) (string, error) {
	return c.rawRequest(path, map[string]string{"w": "pretty"})
}

func (c *Client) rawRequestJSON(path string, target interface{}, opts map[string]string) error {
	resp, err := c.rawRequest(path, opts)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(resp), target)
}

func (c *Client) rawRequest(path string, opts map[string]string) (string, error) {
	// Is this a Call request?
	if _, ok := callMap[path]; ok {
		return c.Call(path, url.Values{})
	}

	requestUrl := url.URL{}
	requestUrl.Scheme = "http"
	requestUrl.Host = c.fakeHost
	requestUrl.Path = path
	query := url.Values{}
	for key, value := range opts {
		query.Add(key, value)
	}
	requestUrl.RawQuery = query.Encode()

	resp, err := c.client.Get(requestUrl.String())
	if err != nil {
		return "", err
	}

	return c.handleResponse(resp)
}

// Call performs a remote action and returns the result
func (c *Client) Call(path string, values url.Values) (string, error) {
	resp, err := c.client.PostForm("http://"+c.fakeHost+"/"+path+"?w=pretty", values)
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
		return "", api.ErrNotFound
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if dataErr, ok := data["error"].(string); ok {
		return "", api.ErrUnknown(errors.New(dataErr))
	}

	return string(body), nil
}
