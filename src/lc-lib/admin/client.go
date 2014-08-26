/*
* Copyright 2014 Jason Woods.
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
  "lc-lib/core"
  "net"
  "time"
)

type Client struct {
  addr    net.TCPAddr
  conn    *net.TCPConn
  decoder *gob.Decoder
}

func NewClient(host string, port int) (*Client, error) {
  ret := &Client{}

  ret.addr.IP = net.ParseIP(host)
  if ret.addr.IP == nil {
    return nil, fmt.Errorf("Invalid admin connect address")
  }

  ret.addr.Port = port

  if err := ret.connect(); err != nil {
    return nil, err
  }

  return ret, nil
}

func (c *Client) connect() (err error) {
  if c.conn, err = net.DialTCP("tcp", nil, &c.addr); err != nil {
    return
  }

  c.decoder = gob.NewDecoder(c.conn)

  return nil
}

func (c *Client) request(command string) (*Response, error) {
  if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
    return nil, err
  }

  total_written := 0

  for {
    wrote, err := c.conn.Write([]byte(command[total_written:4]))
    if err != nil {
      return nil, err
    }

    total_written += wrote
    if total_written == 4 {
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

func (c *Client) resolveError(response *Response) error {
  ret, ok := response.Response.(*ErrorResponse)
  if ok {
    return ret
  }

  return &ErrorResponse{Message: fmt.Sprintf("Unrecognised response: %v\n", ret)}
}

func (c *Client) Ping() error {
  response, err := c.request("PING")
  if err != nil {
    return err
  }

  if _, ok := response.Response.(*PongResponse); ok {
    return nil
  }

  return c.resolveError(response)
}

func (c *Client) Reload() error {
  response, err := c.request("RELD")
  if err != nil {
    return err
  }

  if _, ok := response.Response.(*ReloadResponse); ok {
    return nil
  }

  return c.resolveError(response)
}

func (c *Client) FetchSnapshot() ([]*core.Snapshot, error) {
  response, err := c.request("SNAP")
  if err != nil {
    return nil, err
  }

  if ret, ok := response.Response.([]*core.Snapshot); ok {
    return ret, nil
  }

  return nil, c.resolveError(response)
}
