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

func TestAddressPoolIP(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"127.0.0.1:1234"})

  addr, desc, err := pool.Next()

  // Should have succeeeded
  if err != nil {
    t.Error("Address pool did not parse IP correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool did not returned nil addr")
  } else if desc != "127.0.0.1:1234" {
    t.Error("Address pool did not return correct desc: ", desc)
  } else if addr.String() != "127.0.0.1:1234" {
    t.Error("Address pool did not return correct addr: ", addr.String())
  }
}

func TestAddressPoolHost(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"google-public-dns-a.google.com:555"})

  addr, desc, err := pool.Next()

  if err != nil {
    t.Error("Address pool did not parse Host correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool did not returned nil addr")
  } else if desc != "8.8.8.8:555 (google-public-dns-a.google.com)" && desc != "[2001:4860:4860::8888]:555 (google-public-dns-a.google.com)" {
    t.Error("Address pool did not return correct desc: ", desc)
  } else if addr.String() != "8.8.8.8:555" && addr.String() != "[2001:4860:4860::8888]:555" {
    t.Error("Address pool did not return correct addr: ", addr.String())
  }
}

func TestAddressPoolHostMultiple(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"google.com:555"})

  for i := 0; i < 2; i++ {
    addr, _, err := pool.Next()

    // Should have succeeeded
    if err != nil {
      t.Error("Address pool did not parse Host correctly: ", err)
    } else if addr == nil {
      t.Error("Address pool did not returned nil addr")
    }

    if i == 0 {
      if pool.IsLast() {
        t.Error("Address pool did not return multiple addresses")
      }
    }
  }
}

func TestAddressPoolSrv(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"@_xmpp-server._tcp.google.com"})

  addr, _, err := pool.Next()

  // Should have succeeeded
  if err != nil {
    t.Error("Address pool did not parse SRV correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool did not returned nil addr")
  }
}

func TestAddressPoolSrvRfc(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"@google.com"})
  pool.SetRfc2782(true, "xmpp-server")

  addr, _, err := pool.Next()

  // Should have succeeeded
  if err != nil {
    t.Error("Address pool did not parse RFC SRV correctly: ", err)
  } else if addr == nil {
    t.Error("Address pool did not returned nil addr")
  }
}

func TestAddressPoolInvalid(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"127.0..0:1234"})

  _, _, err := pool.Next()

  // Should have failed
  if err == nil {
    t.Logf("Address pool did not return failure correctly")
    t.FailNow()
  }
}

func TestAddressPoolHostFailure(t *testing.T) {
  // Test failures when parsing
  pool := NewAddressPool([]string{"google-public-dns-not-exist.google.com:1234"})

  _, _, err := pool.Next()

  // Should have failed
  if err == nil {
    t.Logf("Address pool did not return failure correctly")
    t.FailNow()
  }
}

func TestAddressPoolIsLast(t *testing.T) {
  // Test that IsLastServer works correctly
  pool := NewAddressPool([]string{"outlook.com:1234"})

  // Should report as last
  if !pool.IsLast() {
    t.Error("Address pool IsLast did not return correctly")
  }

  for i := 0; i <= 42; i++ {
    _, _, err := pool.Next()

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

func TestAddressPoolIsLastServer(t *testing.T) {
  // Test that IsLastServer works correctly
  pool := NewAddressPool([]string{"127.0.0.1:1234", "127.0.0.1:1234", "127.0.0.1:1234"})

  // Should report as last server
  if !pool.IsLastServer() {
    t.Error("Address pool IsLastServer did not return correctly")
  }

  for i := 0; i < 3; i++ {
    _, _, err := pool.Next()

    // Should succeed
    if err != nil {
      t.Error("Address pool did not parse IP correctly")
    }

    if i < 2 {
      // Should not report as last server
      if pool.IsLastServer() {
        t.Error("Address pool IsLastServer did not return correctly")
      }

      continue
    }
  }

  // Should report as last server
  if !pool.IsLastServer() {
    t.Error("Address pool IsLastServer did not return correctly")
  }
}

func TestAddressPoolNextServer(t *testing.T) {
  // Test that IsLastServer works correctly
  pool := NewAddressPool([]string{"google.com:1234", "google.com:1234"})

  cnt := 0
  for i := 0; i < 42; i++ {
    addr, err := pool.NextServer()

    // Should succeed
    if err != nil {
      t.Error("Address pool did not parse IP correctly")
    } else if addr != "google.com:1234" {
      t.Error("Address pool returned incorrect address: ", addr)
    }

    cnt++

    // Break on last server
    if pool.IsLastServer() {
      break
    }
  }

  // Should have stopped at 2 servers
  if cnt != 2 {
    t.Error("Address pool NextServer failed")
  }
}
