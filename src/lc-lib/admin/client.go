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
  "strings"
  "time"
)

type Client struct {
  admin_connect string
  conn          net.Conn
  decoder       *gob.Decoder
}

func NewClient(admin_connect string) (*Client, error) {
  var err error

  ret := &Client{}

  if ret.conn, err = ret.connect(admin_connect); err != nil {
    return nil, err
  }

  ret.decoder = gob.NewDecoder(ret.conn)

  return ret, nil
}

func (c *Client) connect(admin_connect string) (net.Conn, error) {
  connect := strings.SplitN(admin_connect, ":", 2)
  if len(connect) == 1 {
    connect = append(connect, connect[0])
    connect[0] = "tcp"
  }

  switch connect[0] {
  case "tcp":
    return c.connectTCP("tcp", connect[1])
  case "tcp4":
    return c.connectTCP("tcp4", connect[1])
  case "tcp6":
    return c.connectTCP("tcp6", connect[1])
  case "unix":
    return c.connectUnix(connect[1])
  }

  return nil, fmt.Errorf("Unknown transport specified in connection address: '%s'", connect[0])
}

func (c *Client) connectTCP(transport, addr string) (net.Conn, error) {
  taddr, err := net.ResolveTCPAddr(transport, addr)
  if err != nil {
    return nil, fmt.Errorf("The connection address specified is not valid: %s", err)
  }

  conn, err := net.DialTCP(transport, nil, taddr)
  if err != nil {
    return nil, err
  }

  return conn, nil
}

func (c *Client) connectUnix(path string) (net.Conn, error) {
  uaddr, err := net.ResolveUnixAddr("unix", path)
  if err != nil {
    return nil, fmt.Errorf("The connection address specified is not valid: %s", err)
  }

  // TODO: Change umask to 111 so all can write (need to move to _unix)
  // Permission will be controlled by folder permissions instead of file
  conn, err := net.DialUnix("unix", nil, uaddr)
  if err != nil {
    return nil, err
  }

  return conn, nil
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
