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
	"context"
	"net"
	"testing"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

// newTestProxySocket wires up a connectionSocketProxy over one end of a net.Pipe, with a
// plain TCP inner socket, mirroring how receivertcp.go wires the real chain
func newTestProxySocket(server net.Conn, policy proxyproto.Policy) (*connectionSocketProxy, *proxyproto.Conn) {
	proxyConn := proxyproto.NewConn(server, proxyproto.WithPolicy(policy), proxyproto.SetReadHeaderTimeout(time.Second))
	underlying := &proxyConnCloseWriter{Conn: proxyConn}
	return newConnectionSocketProxy(proxyConn, newConnectionSocketTCP(underlying)), proxyConn
}

// stubSocket is a minimal connectionSocket used to isolate Desc() combination logic
// from the real inner socket types
type stubSocket struct {
	net.Conn
	desc string
}

func (s *stubSocket) Setup(context.Context) error { return nil }
func (s *stubSocket) Desc() string                { return s.desc }
func (s *stubSocket) CloseWrite() error           { return nil }

func TestConnectionSocketProxyReportsOriginalAddresses(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	header := proxyproto.HeaderProxyFromAddrs(2,
		&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234},
		&net.TCPAddr{IP: net.IPv4(10, 0, 2, 50), Port: 5000})

	go header.WriteTo(client)

	socket, _ := newTestProxySocket(server, proxyproto.REQUIRE)
	if err := socket.Setup(context.Background()); err != nil {
		t.Fatalf("Setup failed: %s", err)
	}

	if socket.RemoteAddr().String() != "203.0.113.7:51234" {
		t.Errorf("RemoteAddr was %q, expected the original client address", socket.RemoteAddr().String())
	}
	if socket.LocalAddr().String() != "10.0.2.50:5000" {
		t.Errorf("LocalAddr was %q, expected the original destination address", socket.LocalAddr().String())
	}
}

func TestConnectionSocketProxyLocalCommandUsesRealPeer(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	header := &proxyproto.Header{Version: 2, Command: proxyproto.LOCAL, TransportProtocol: proxyproto.UNSPEC}
	go header.WriteTo(client)

	socket, _ := newTestProxySocket(server, proxyproto.REQUIRE)
	if err := socket.Setup(context.Background()); err != nil {
		t.Fatalf("Setup failed: %s", err)
	}

	if socket.RemoteAddr().String() != server.RemoteAddr().String() {
		t.Errorf("RemoteAddr was %q, expected the real peer address for a LOCAL command", socket.RemoteAddr().String())
	}
	if socket.LocalAddr().String() != server.LocalAddr().String() {
		t.Errorf("LocalAddr was %q, expected the real local address for a LOCAL command", socket.LocalAddr().String())
	}
}

func TestConnectionSocketProxyPreservesDataAfterHeader(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	header := proxyproto.HeaderProxyFromAddrs(2,
		&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234},
		&net.TCPAddr{IP: net.IPv4(10, 0, 2, 50), Port: 5000})
	headerBytes, err := header.Format()
	if err != nil {
		t.Fatalf("Failed to format header: %s", err)
	}

	payload := []byte("hello world")
	go client.Write(append(headerBytes, payload...))

	socket, _ := newTestProxySocket(server, proxyproto.REQUIRE)
	if err := socket.Setup(context.Background()); err != nil {
		t.Fatalf("Setup failed: %s", err)
	}

	received := make([]byte, len(payload))
	if _, err := socket.Read(received); err != nil {
		t.Fatalf("Read failed: %s", err)
	}
	if string(received) != string(payload) {
		t.Errorf("Payload was %q, expected %q - header and data must have been coalesced in one segment", received, payload)
	}
}

func TestConnectionSocketProxyDescCombinesInnerAndProxy(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	header := proxyproto.HeaderProxyFromAddrs(2,
		&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234},
		&net.TCPAddr{IP: net.IPv4(10, 0, 2, 50), Port: 5000})
	go header.WriteTo(client)

	proxyConn := proxyproto.NewConn(server, proxyproto.WithPolicy(proxyproto.REQUIRE), proxyproto.SetReadHeaderTimeout(time.Second))
	inner := &stubSocket{Conn: &proxyConnCloseWriter{Conn: proxyConn}, desc: "client.example.com"}
	socket := newConnectionSocketProxy(proxyConn, inner)

	if err := socket.Setup(context.Background()); err != nil {
		t.Fatalf("Setup failed: %s", err)
	}

	desc := socket.Desc()
	if desc != "client.example.com via PROXY from pipe" {
		t.Errorf("Desc was %q, expected the inner description combined with the relaying proxy", desc)
	}
}

func TestConnectionSocketProxyDescWithPlainInner(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	header := proxyproto.HeaderProxyFromAddrs(2,
		&net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 51234},
		&net.TCPAddr{IP: net.IPv4(10, 0, 2, 50), Port: 5000})
	go header.WriteTo(client)

	socket, _ := newTestProxySocket(server, proxyproto.REQUIRE)
	if err := socket.Setup(context.Background()); err != nil {
		t.Fatalf("Setup failed: %s", err)
	}

	if desc := socket.Desc(); desc != "via PROXY from pipe" {
		t.Errorf("Desc was %q, expected just the relaying proxy since the plain TCP inner has none", desc)
	}
}

func TestConnectionSocketProxyRejectsMissingHeader(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go client.Write([]byte("not a proxy header"))

	socket, _ := newTestProxySocket(server, proxyproto.REQUIRE)
	if err := socket.Setup(context.Background()); err == nil {
		t.Error("Setup succeeded, expected an error for a connection with no PROXY header")
	}
}

func TestConnectionSocketProxyRejectsMalformedHeader(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// A truncated v2 signature followed by an incomplete rest-of-header
	go client.Write(append([]byte("\x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A"), 0x21, 0x11))

	proxyConn := proxyproto.NewConn(server, proxyproto.WithPolicy(proxyproto.REQUIRE), proxyproto.SetReadHeaderTimeout(50*time.Millisecond))
	underlying := &proxyConnCloseWriter{Conn: proxyConn}
	socket := newConnectionSocketProxy(proxyConn, newConnectionSocketTCP(underlying))
	if err := socket.Setup(context.Background()); err == nil {
		t.Error("Setup succeeded, expected an error for a malformed PROXY header")
	}
}

func TestConnectionSocketProxySetupDoesNotBlockForever(t *testing.T) {
	_, server := net.Pipe()
	defer server.Close()

	proxyConn := proxyproto.NewConn(server, proxyproto.WithPolicy(proxyproto.REQUIRE), proxyproto.SetReadHeaderTimeout(50*time.Millisecond))
	underlying := &proxyConnCloseWriter{Conn: proxyConn}
	socket := newConnectionSocketProxy(proxyConn, newConnectionSocketTCP(underlying))

	done := make(chan error, 1)
	go func() {
		done <- socket.Setup(context.Background())
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("Setup succeeded, expected a timeout error when the peer sends nothing")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Setup did not return within the header timeout")
	}
}
