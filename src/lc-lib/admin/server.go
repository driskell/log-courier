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
  "net"
	"sync"
)

type server struct {
  conn       *net.TCPConn
	conn_group *sync.WaitGroup
}

func newServer(conn_group *sync.WaitGroup, conn *net.TCPConn) *server {
  return &server{
    conn:       conn,
		conn_group: conn_group,
  }
}

func (s *server) Run() {
  s.conn.Close()

  s.conn_group.Done()
}
