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

package tcp

import (
	"net"
	"testing"

	proxyproto "github.com/pires/go-proxyproto"
)

func TestNewProxyPolicyRequiresHeaderWithNoTrustedSources(t *testing.T) {
	policy, err := newProxyPolicy(nil)
	if err != nil {
		t.Fatalf("newProxyPolicy failed: %s", err)
	}

	decision, err := policy(&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234})
	if err != nil {
		t.Fatalf("policy returned an error: %s", err)
	}
	if decision != proxyproto.REQUIRE {
		t.Errorf("decision was %v, expected REQUIRE for any peer when no trusted sources are configured", decision)
	}
}

func TestNewProxyPolicyRejectsSourcesOutsideTrustedList(t *testing.T) {
	policy, err := newProxyPolicy([]string{"10.0.2.0/24", "192.168.1.5"})
	if err != nil {
		t.Fatalf("newProxyPolicy failed: %s", err)
	}

	if _, err := policy(&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234}); err == nil {
		t.Error("expected an error for a peer outside the trusted sources")
	}
}

func TestNewProxyPolicyAllowsTrustedCIDR(t *testing.T) {
	policy, err := newProxyPolicy([]string{"10.0.2.0/24"})
	if err != nil {
		t.Fatalf("newProxyPolicy failed: %s", err)
	}

	decision, err := policy(&net.TCPAddr{IP: net.IPv4(10, 0, 2, 50), Port: 48392})
	if err != nil {
		t.Fatalf("policy returned an error for a trusted peer: %s", err)
	}
	if decision != proxyproto.REQUIRE {
		t.Errorf("decision was %v, expected REQUIRE for a trusted peer", decision)
	}
}

func TestNewProxyPolicyAllowsTrustedExactAddress(t *testing.T) {
	policy, err := newProxyPolicy([]string{"192.168.1.5"})
	if err != nil {
		t.Fatalf("newProxyPolicy failed: %s", err)
	}

	decision, err := policy(&net.TCPAddr{IP: net.IPv4(192, 168, 1, 5), Port: 48392})
	if err != nil {
		t.Fatalf("policy returned an error for a trusted peer: %s", err)
	}
	if decision != proxyproto.REQUIRE {
		t.Errorf("decision was %v, expected REQUIRE for a trusted peer", decision)
	}
}

func TestNewProxyPolicyRejectsInvalidTrustedSource(t *testing.T) {
	if _, err := newProxyPolicy([]string{"not-an-address"}); err == nil {
		t.Error("expected an error for an invalid trusted source entry")
	}
}
