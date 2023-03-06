/*
 * Copyright 2012-2023 Jason Woods and contributors
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

func TestPoolEntryIP(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "127.0.0.1:1234",
	}
	addr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Host() != "127.0.0.1" {
		t.Errorf("Address pool did not return correct host: %s", addr.Host())
	} else if addr.Desc() != "desc" {
		t.Errorf("Address pool did not return correct desc: %s", addr.Desc())
	} else if addr.Addr().String() != "127.0.0.1:1234" {
		t.Errorf("Address pool did not return correct addr: %s", addr.Addr().String())
	}
}

func TestPoolEntryHost(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "host.test.woods.dev:555",
	}
	addr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Host() != "host.test.woods.dev" {
		t.Errorf("Address pool did not return correct host: %s", addr.Host())
	}

	secondAddr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if secondAddr == nil {
		t.Fatal("Address pool returned nil addr")
	}

	if addr.Addr().String() != "8.8.8.8:555" {
		tmp := addr
		addr = secondAddr
		secondAddr = tmp
	}

	if addr.Desc() != "8.8.8.8:555 - desc" {
		t.Errorf("Address pool did not return correct desc: %s", addr.Desc())
	} else if addr.Addr().String() != "8.8.8.8:555" {
		t.Errorf("Address pool did not return correct addr: %s", addr.Addr().String())
	}
	if secondAddr.Desc() != "[2001:4860:4860::8888]:555 - desc" {
		t.Errorf("Address pool did not return correct desc: %s", secondAddr.Desc())
	} else if secondAddr.Addr().String() != "[2001:4860:4860::8888]:555" {
		t.Errorf("Address pool did not return correct addr: %s", secondAddr.Addr().String())
	}
}

func TestPoolEntryHostMultipleAndReuse(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "multihost.test.woods.dev:555",
	}

	addresses := map[string]int{
		"8.8.8.8:555": 3,
		"5.5.5.5:555": 3,
		"2.2.2.2:555": 3,
		"1.1.1.1:555": 3,
	}

	for {
		addr, err := poolEntry.Next()
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

		if len(addresses) == 0 {
			break
		}
	}

	for k := range addresses {
		t.Errorf("Address pool did not return expected addr: %s", k)
	}
}

func TestPoolEntryInvalid(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "[127.0..0:1234",
	}
	_, err := poolEntry.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolEntryInvalidPort(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "127.0.0.1:65539",
	}
	_, err := poolEntry.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolEntryHostFailure(t *testing.T) {
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "missing.test.woods.dev:1234",
	}
	_, err := poolEntry.Next()

	// Should have failed
	if err == nil {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestHostLooping(t *testing.T) {
	// Host address should lookup continuously - this should return the 2 entries, relookup, and return again
	poolEntry := &PoolEntry{
		Expire:   time.Minute,
		Server:   "server",
		Desc:     "desc",
		HostPort: "host.test.woods.dev:1234",
	}
	addr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Desc() != "8.8.8.8:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}
	addr, err = poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Desc() != "[2001:4860:4860::8888]:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}
	addr, err = poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Desc() != "8.8.8.8:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}
}

func TestHostLoopingRefresh(t *testing.T) {
	// Host address should lookup continuously and refresh at expiry
	// We should return first entry, refresh, then first and second
	poolEntry := &PoolEntry{
		Expire:   time.Second,
		Server:   "server",
		Desc:     "desc",
		HostPort: "host.test.woods.dev:1234",
	}
	addr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	} else if addr.Desc() != "8.8.8.8:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}

	time.Sleep(1100 * time.Millisecond)

	addr, err = poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if addr == nil {
		t.Fatal("Address pool returned nil addr")
	}

	secondAddr, err := poolEntry.Next()
	if err != nil {
		t.Fatalf("Address pool did not parse IP correctly: %s", err)
	} else if secondAddr == nil {
		t.Fatal("Address pool returned nil addr")
	}

	if addr.Addr().String() != "8.8.8.8:555" {
		tmp := addr
		addr = secondAddr
		secondAddr = tmp
	}

	if addr.Desc() != "8.8.8.8:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}
	if secondAddr.Desc() != "[2001:4860:4860::8888]:1234 - desc" {
		t.Errorf("Address pool returned incorrect desc: %s", addr.Desc())
	}
}
