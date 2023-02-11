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

package addresspool

import (
	"testing"
	"time"
)

func TestPoolIP(t *testing.T) {
	pool := NewPool("127.0.0.1:1234")
	addr, err := pool.Next()

	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if pool.Server() != "127.0.0.1:1234" {
		t.Errorf("Address pool did not return correct server: %s", pool.Server())
	} else if addr.Host() != "127.0.0.1" {
		t.Errorf("Address pool did not return correct host: %s", addr.Host())
	} else if addr.Desc() != "127.0.0.1:1234" {
		t.Errorf("Address pool did not return correct desc: %s", addr.Desc())
	} else if addr.Addr().String() != "127.0.0.1:1234" {
		t.Errorf("Address pool did not return correct addr: %s", addr.Addr().String())
	} else if !pool.IsLast() {
		t.Error("Address pool has more addresses than expexected")
	}
}

func TestPoolHost(t *testing.T) {
	pool := NewPool("host.test.woods.dev:555")
	addr, err := pool.Next()

	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if pool.Server() != "host.test.woods.dev:555" {
		t.Errorf("Address pool did not return correct server: %s", pool.Server())
	} else if addr.Host() != "host.test.woods.dev" {
		t.Errorf("Address pool did not return correct host: %s", addr.Host())
	}

	secondAddr, err := pool.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if !pool.IsLast() {
		t.Error("Address pool has more addresses than expexected")
	}
	if addr.Addr().String() != "8.8.8.8:555" {
		tmp := addr
		addr = secondAddr
		secondAddr = tmp
	}

	if addr.Desc() != "host.test.woods.dev:555 (8.8.8.8:555)" {
		t.Errorf("Address pool did not return correct desc: %s", addr.Desc())
	} else if addr.Addr().String() != "8.8.8.8:555" {
		t.Errorf("Address pool did not return correct addr: %s", addr.Addr().String())
	}
	if secondAddr.Desc() != "host.test.woods.dev:555 (2001:4860:4860::8888:555)" {
		t.Errorf("Address pool did not return correct desc: %s", secondAddr.Desc())
	} else if secondAddr.Addr().String() != "[2001:4860:4860::8888]:555" {
		t.Errorf("Address pool did not return correct addr: %s", secondAddr.Addr().String())
	}
}

func TestPoolHostMultipleAndReuse(t *testing.T) {
	pool := NewPool("multihost.test.woods.dev:555")

	lastCount := 0
	addresses := map[string]int{
		"8.8.8.8:555": 3,
		"5.5.5.5:555": 3,
		"2.2.2.2:555": 3,
		"1.1.1.1:555": 3,
	}

	for {
		addr, err := pool.Next()
		if err != nil {
			t.Fatalf("Address pool did not parse Host correctly: %s", err)
		} else if addr == nil {
			t.Fatal("Address pool returned nil addr")
		}

		if _, ok := addresses[addr.Addr().String()]; !ok {
			t.Errorf("Address pool returned unexpected addr: %s", addr.Addr().String())
			break
		} else {
			addresses[addr.Addr().String()] -= 1
			if addresses[addr.Addr().String()] == 0 {
				delete(addresses, addr.Addr().String())
			}
		}

		if pool.IsLast() {
			lastCount += 1
			if lastCount == 3 {
				break
			}
		}
	}

	for k := range addresses {
		t.Errorf("Address pool did not return expected addr: %s", k)
	}
}

func TestPoolSrv(t *testing.T) {
	pool := NewPool("@_test._tcp.test.woods.dev")
	addr, err := pool.Next()

	// Should have succeeeded
	if err != nil {
		t.Fatalf("Address pool did not parse SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}
}

func TestPoolSrvRfc(t *testing.T) {
	pool := NewPool("@test.woods.dev")
	pool.SetRfc2782(true, "test")
	addr, err := pool.Next()

	// Should have succeeeded
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}
}

func TestPoolSrvRfcMultiple(t *testing.T) {
	pool := NewPool("@multitest.woods.dev")
	pool.SetRfc2782(true, "test")
	addr, err := pool.Next()

	// Should have succeeeded and should return the single host priority 0 first
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	}

	var expectSecondAddr string
	if addr.Addr().String() == "8.8.8.8:1234" {
		expectSecondAddr = "[2001:4860:4860::8888]:1234"
	} else if addr.Addr().String() == "[2001:4860:4860::8888]:1234" {
		expectSecondAddr = "8.8.8.8:1234"
	} else {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}

	// Now should return one multihost, and then the IPv6 from single test,
	// and then the rest from the multihost
	addresses := map[string]int{
		"8.8.8.8:1235": 0,
		"5.5.5.5:1235": 0,
		"2.2.2.2:1235": 0,
		"1.1.1.1:1235": 0,
	}
	for {
		addr, err = pool.Next()
		if err != nil {
			t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
		}
		if addr.Addr().String() == expectSecondAddr {
			if len(addresses) != 3 {
				t.Errorf("Address pool returned first host second IP in wrong sequence: %s (expected at 1 but saw at %d)", addr.Addr().String(), 4-len(addresses))
				break
			}
			continue
		}
		if _, ok := addresses[addr.Addr().String()]; !ok {
			t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
			break
		} else {
			delete(addresses, addr.Addr().String())
		}
		if pool.IsLast() {
			break
		}
	}

	for k := range addresses {
		t.Errorf("Address pool did not return expected addr: %s", k)
	}

	// Verify we then pick up the primary again, as now we exhausted, it should lookup
	// and again priority be the first single host
	addr, err = pool.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" && addr.Addr().String() != "[2001:4860:4860::8888]:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}
}

func TestPoolSrvRfcMultipleTtl(t *testing.T) {
	pool := &Pool{
		server: "@multitest.woods.dev",
		ttl:    time.Second,
	}
	pool.SetRfc2782(true, "test")
	addr, err := pool.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" && addr.Addr().String() != "[2001:4860:4860::8888]:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}

	// Now we should wait 1 second, and the second multi-host should expire, and we'll restart again
	// This is in contract to the nonTtl test that would return from multi-host first
	time.Sleep(1100 * time.Millisecond)

	addr, err = pool.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" && addr.Addr().String() != "[2001:4860:4860::8888]:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}
}

func TestPoolSrvNoneRfc(t *testing.T) {
	pool := NewPool("@_test._tcp.multitest.woods.dev")
	addr, err := pool.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse RFC SRV correctly: %s", err)
	} else if addr.Addr().String() != "8.8.8.8:1234" && addr.Addr().String() != "[2001:4860:4860::8888]:1234" {
		t.Errorf("Address pool returned incorrect addr: %s", addr.Addr().String())
	}
}

func TestPoolInvalid(t *testing.T) {
	pool := NewPool("[127.0..0:1234")
	_, err := pool.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolInvalidPort(t *testing.T) {
	pool := NewPool("127.0.0.1:65539")
	_, err := pool.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolInvalidRfcSrv(t *testing.T) {
	pool := NewPool("@woods.dev")
	pool.SetRfc2782(true, "missing")
	_, err := pool.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolInvalidNonRfcSrv(t *testing.T) {
	pool := NewPool("@_missing._tcp.woods.dev")
	_, err := pool.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolHostFailure(t *testing.T) {
	pool := NewPool("missing.test.woods.dev:1234")
	_, err := pool.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestHostExhaustion(t *testing.T) {
	// Host address should lookup only once, and return nil after it finishes
	hostAddress := &hostAddress{hostPort: "host.test.woods.dev:1234", expire: time.Now().Add(time.Minute)}
	addr, err := hostAddress.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	}
	addr, err = hostAddress.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	}
	addr, err = hostAddress.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr != nil {
		t.Fatal("Address pool returned non-nil addr after exhausting addresses")
	}
}
