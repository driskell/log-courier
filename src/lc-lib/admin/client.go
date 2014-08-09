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
  "lc-lib/core"
)

type Client struct {
  host string
  port int
}

func NewClient(host string, port int) (*Client, error) {
  ret := &Client{
    host: host,
    port: port,
  }

  if err := ret.Connect(); err != nil {
    return nil, err
  }

  return ret, nil
}

func (c *Client) Connect() error {
  return nil
}

func (c *Client) FetchSnapshot() []core.Snapshot {
  return []core.Snapshot{}
}
