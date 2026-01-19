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

func TestGeneratePoolSrv(t *testing.T) {
	pool, err := GeneratePool([]string{"@_test._tcp.test.woods.dev"}, false, "", time.Minute)
	if err != nil {
		t.Fatalf("Address pool did not parse SRV correctly: %s", err)
	}

	if len(pool) != 1 {
		t.Fatalf("Address pool contains incorrect number of items: %d", len(pool))
	} else if pool[0].HostPort != "host.test.woods.dev:1234" {
		t.Fatalf("Address pool item host unexpected: %s", pool[0].HostPort)
	} else if pool[0].Server != "@_test._tcp.test.woods.dev" {
		t.Fatalf("Address pool item server unexpected: %s", pool[0].Server)
	} else if pool[0].Desc != "@_test._tcp.test.woods.dev" {
		t.Fatalf("Address pool item desc unexpected: %s", pool[0].Desc)
	}
}

func TestPoolSrvRfc(t *testing.T) {
	pool, err := GeneratePool([]string{"@test.woods.dev"}, true, "test", time.Minute)
	if err != nil {
		t.Fatalf("Address pool did not parse SRV correctly: %s", err)
	}

	if len(pool) != 1 {
		t.Fatalf("Address pool contains incorrect number of items: %d", len(pool))
	} else if pool[0].HostPort != "host.test.woods.dev:1234" {
		t.Fatalf("Address pool item host unexpected: %s", pool[0].HostPort)
	} else if pool[0].Server != "@test.woods.dev" {
		t.Fatalf("Address pool item server unexpected: %s", pool[0].Server)
	} else if pool[0].Desc != "@test.woods.dev" {
		t.Fatalf("Address pool item desc unexpected: %s", pool[0].Desc)
	}
}

func TestPoolSrvRfcMultiple(t *testing.T) {
	pool, err := GeneratePool([]string{"@multitest.woods.dev"}, true, "test", time.Minute)
	if err != nil {
		t.Fatalf("Address pool did not parse SRV correctly: %s", err)
	}

	if len(pool) != 2 {
		t.Fatalf("Address pool contains incorrect number of items: %d", len(pool))
	} else if pool[0].HostPort != "host.test.woods.dev:1234" {
		t.Fatalf("Address pool item host unexpected: %s", pool[0].HostPort)
	} else if pool[0].Server != "@multitest.woods.dev" {
		t.Fatalf("Address pool item server unexpected: %s", pool[0].Server)
	} else if pool[0].Desc != "@multitest.woods.dev" {
		t.Fatalf("Address pool item desc unexpected: %s", pool[0].Desc)
	} else if pool[1].HostPort != "multihost.test.woods.dev:1235" {
		t.Fatalf("Address pool item host unexpected: %s", pool[1].HostPort)
	} else if pool[1].Server != "@multitest.woods.dev" {
		t.Fatalf("Address pool item server unexpected: %s", pool[1].Server)
	} else if pool[1].Desc != "@multitest.woods.dev" {
		t.Fatalf("Address pool item desc unexpected: %s", pool[1].Desc)
	}
}

func TestPoolInvalidRfcSrv(t *testing.T) {
	pool, err := GeneratePool([]string{"@woods.dev"}, false, "", time.Minute)
	if err == nil || len(pool) != 0 {
		t.Errorf("Address pool did not return failure correctly")
	}
}

func TestPoolInvalidNonRfcSrv(t *testing.T) {
	pool, err := GeneratePool([]string{"@_missing._tcp.woods.dev"}, false, "", time.Minute)
	if err == nil || len(pool) != 0 {
		t.Errorf("Address pool did not return failure correctly")
	}
}
