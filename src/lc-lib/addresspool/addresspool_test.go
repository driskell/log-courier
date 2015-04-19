// +build ignore
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

package addresspool

import (
  "testing"
)

func TestPoolIP(t *testing.T) {
  pool := NewPool("127.0.0.1:1234")
  addr, err := pool.Next()

  if err != nil {
    t.Error("Address pool did not parse IP correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool returned nil addr")
  } else if pool.Server() != "127.0.0.1:1234" {
    t.Error("Address pool did not return correct server: ", pool.Server())
  } else if pool.Host() != "127.0.0.1" {
    t.Error("Address pool did not return correct host: ", pool.Host())
  } else if pool.Desc() != "127.0.0.1:1234" {
    t.Error("Address pool did not return correct desc: ", pool.Desc())
  } else if addr.String() != "127.0.0.1:1234" {
    t.Error("Address pool did not return correct addr: ", addr.String())
  }
}

func TestPoolHost(t *testing.T) {
  pool := NewPool("google-public-dns-a.google.com:555")
  addr, err := pool.Next()

  if err != nil {
    t.Error("Address pool did not parse IP correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool returned nil addr")
  } else if pool.Server() != "google-public-dns-a.google.com:555" {
    t.Error("Address pool did not return correct server: ", pool.Server())
  } else if pool.Host() != "google-public-dns-a.google.com" {
    t.Error("Address pool did not return correct host: ", pool.Host())
  } else if pool.Desc() != "8.8.8.8:555 (google-public-dns-a.google.com)" && pool.Desc() != "[2001:4860:4860::8888]:555 (google-public-dns-a.google.com)" {
    t.Error("Address pool did not return correct desc: ", pool.Desc())
  } else if addr.String() != "8.8.8.8:555" && addr.String() != "[2001:4860:4860::8888]:555" {
    t.Error("Address pool did not return correct addr: ", addr.String())
  }
}

func TestPoolHostMultiple(t *testing.T) {
  pool := NewPool("google.com:555")

  for i := 0; i < 2; i++ {
    addr, err := pool.Next()

    // Should have succeeeded
    if err != nil {
      t.Error("Address pool did not parse Host correctly: ", err)
    } else if addr == nil {
      t.Error("Address pool returned nil addr")
    }

    if i == 0 {
      if pool.IsLast() {
        t.Error("Address pool did not return multiple addresses")
      }
    }
  }
}

func TestPoolSrv(t *testing.T) {
  pool := NewPool("@_xmpp-server._tcp.google.com")
  addr, err := pool.Next()

  // Should have succeeeded
  if err != nil {
    t.Error("Address pool did not parse SRV correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool returned nil addr")
  }
}

func TestPoolSrvRfc(t *testing.T) {
  pool := NewPool("@google.com")
  pool.SetRfc2782(true, "xmpp-server")
  addr, err := pool.Next()

  // Should have succeeeded
  if err != nil {
    t.Error("Address pool did not parse RFC SRV correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool did not returned nil addr")
  }
}

func TestPoolInvalid(t *testing.T) {
  pool := NewPool("127.0..0:1234")
  _, err := pool.Next()

  // Should have failed
  if err == nil {
    t.Logf("Address pool did not return failure correctly")
    t.FailNow()
  }
}

func TestPoolHostFailure(t *testing.T) {
  pool := NewPool("google-public-dns-not-exist.google.com:1234")
  _, err := pool.Next()

  // Should have failed
  if err == nil {
    t.Logf("Address pool did not return failure correctly")
    t.FailNow()
  }
}

func TestPoolIsLast(t *testing.T) {
  pool := NewPool("outlook.com:1234")

  // Should report as last
  if !pool.IsLast() {
    t.Error("Address pool IsLast did not return correctly")
  }

  for i := 0; i <= 42; i++ {
    _, err := pool.Next()

    // Should succeed
    if err != nil {
      t.Error("Address pool did not parse Host correctly")
    }

    if i <= 1 {
      // Should not report as last
      if pool.IsLast() {
        t.Error("Address pool IsLast did not return correctly")
      }

      continue
    }

    // Wait until last
    if pool.IsLast() {
      return
    }
  }

  // Hit 42 servers without hitting last
  t.Error("Address pool IsLast did not return correctly")
}
