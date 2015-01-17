/*
 * Copyright 2014 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

package transports

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"github.com/driskell/log-courier/src/lc-lib/core"
	"io/ioutil"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// Support for newer SSL signature algorithms
import _ "crypto/sha256"
import _ "crypto/sha512"

const (
	// Essentially, this is how often we should check for disconnect/shutdown during socket reads
	socket_interval_seconds = 1
)

type TransportTcpRegistrar struct {
}

type TransportTcpFactory struct {
	transport string

	SSLCertificate string `config:"ssl certificate"`
	SSLKey         string `config:"ssl key"`
	SSLCA          string `config:"ssl ca"`

	hostport_re *regexp.Regexp
	tls_config  tls.Config
}

type TransportTcp struct {
	config     *TransportTcpFactory
	net_config *core.NetworkConfig
	socket     net.Conn
	tlssocket  *tls.Conn

	wait     sync.WaitGroup
	shutdown chan interface{}

	send_chan chan []byte
	recv_chan chan interface{}

	can_send chan int

	roundrobin int
	host_is_ip bool
	host       string
	addresses  []*net.TCPAddr
}

func NewTcpTransportFactory(config *core.Config, config_path string, unused map[string]interface{}, name string) (core.TransportFactory, error) {
	var err error

	ret := &TransportTcpFactory{
		transport:   name,
		hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
	}

	// Only allow SSL configurations if this is "tls"
	if name == "tls" {
		if err = config.PopulateConfig(ret, config_path, unused); err != nil {
			return nil, err
		}

		if len(ret.SSLCertificate) > 0 && len(ret.SSLKey) > 0 {
			cert, err := tls.LoadX509KeyPair(ret.SSLCertificate, ret.SSLKey)
			if err != nil {
				return nil, fmt.Errorf("Failed loading client ssl certificate: %s", err)
			}

			ret.tls_config.Certificates = []tls.Certificate{cert}
		}

		if len(ret.SSLCA) > 0 {
			ret.tls_config.RootCAs = x509.NewCertPool()
			pemdata, err := ioutil.ReadFile(ret.SSLCA)
			if err != nil {
				return nil, fmt.Errorf("Failure reading CA certificate: %s\n", err)
			}
			rest := pemdata
			var block *pem.Block
			var pemBlockNum = 1
			for {
				block, rest = pem.Decode(rest)
				if block != nil {
					if block.Type != "CERTIFICATE" {
						return nil, fmt.Errorf("Block %d does not contain a certificate: %s\n", pemBlockNum, ret.SSLCA)
					}
					cert, err := x509.ParseCertificate(block.Bytes)
					if err != nil {
						return nil, fmt.Errorf("Failed to parse CA certificate in block %d: %s\n", pemBlockNum, ret.SSLCA)
					}
					ret.tls_config.RootCAs.AddCert(cert)
					pemBlockNum += 1
				} else {
					break
				}
			}
		}
	} else {
		if err := config.ReportUnusedConfig(config_path, unused); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (f *TransportTcpFactory) NewTransport(config *core.NetworkConfig) (core.Transport, error) {
	ret := &TransportTcp{
		config:     f,
		net_config: config,
	}

	// Randomise the initial host - after this it will round robin
	// Round robin after initial attempt ensures we don't retry same host twice,
	// and also ensures we try all hosts one by one
	ret.roundrobin = rand.Intn(len(config.Servers))

	return ret, nil
}

func (t *TransportTcp) ReloadConfig(new_net_config *core.NetworkConfig) int {
	// Check we can grab new TCP config to compare, if not force transport reinit
	new_config, ok := new_net_config.TransportFactory.(*TransportTcpFactory)
	if !ok {
		return core.Reload_Transport
	}

	// TODO - This does not catch changes to the underlying certificate file!
	if new_config.SSLCertificate != t.config.SSLCertificate || new_config.SSLKey != t.config.SSLKey || new_config.SSLCA != t.config.SSLCA {
		return core.Reload_Transport
	}

	// Publisher handles changes to net_config, but ensure we store the latest in case it asks for a reconnect
	t.net_config = new_net_config

	return core.Reload_None
}

func (t *TransportTcp) Init() error {
	if t.shutdown != nil {
		t.disconnect()
	}

	// Have we exhausted the address list we had?
	if t.addresses == nil {
		if err := t.populateAddresses(); err != nil {
			return err
		}
	}

	// Try next address and drop it from our list
	addressport := t.addresses[0].String()
	if len(t.addresses) > 1 {
		t.addresses = t.addresses[1:]
	} else {
		t.addresses = nil
	}

	var desc string
	if t.host_is_ip {
		desc = fmt.Sprintf("%s", addressport)
	} else {
		desc = fmt.Sprintf("%s (%s)", addressport, t.host)
	}

	log.Info("Attempting to connect to %s", desc)

	tcpsocket, err := net.DialTimeout("tcp", addressport, t.net_config.Timeout)
	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Now wrap in TLS if this is the "tls" transport
	if t.config.transport == "tls" {
		// Disable SSLv3 (mitigate POODLE vulnerability)
		t.config.tls_config.MinVersion = tls.VersionTLS10

		// Set the tlsconfig server name for server validation (required since Go 1.3)
		t.config.tls_config.ServerName = t.host

		t.tlssocket = tls.Client(&transportTcpWrap{transport: t, tcpsocket: tcpsocket}, &t.config.tls_config)
		t.tlssocket.SetDeadline(time.Now().Add(t.net_config.Timeout))
		err = t.tlssocket.Handshake()
		if err != nil {
			t.tlssocket.Close()
			tcpsocket.Close()
			return fmt.Errorf("TLS Handshake failure with %s: %s", desc, err)
		}

		t.socket = t.tlssocket
	} else {
		t.socket = tcpsocket
	}

	log.Info("Connected to %s", desc)

	// Signal channels
	t.shutdown = make(chan interface{}, 1)
	t.send_chan = make(chan []byte, 1)
	// Buffer of two for recv_chan since both routines may send an error to it
	// First error we get back initiates disconnect, thus we must not block routines
	t.recv_chan = make(chan interface{}, 2)
	t.can_send = make(chan int, 1)

	// Start with a send
	t.can_send <- 1

	t.wait.Add(2)

	// Start separate sender and receiver so we can asynchronously send and receive for max performance
	// They have to be different routines too because we don't have cross-platform poll, so they will need to block
	// Of course, we'll time out and check shutdown on occasion
	go t.sender()
	go t.receiver()

	return nil
}

func (t *TransportTcp) populateAddresses() (err error) {
	// Round robin to the next server
	selected := t.net_config.Servers[t.roundrobin%len(t.net_config.Servers)]
	t.roundrobin++

	t.addresses = make([]*net.TCPAddr, 0)

	// @hostname means SRV record where the host and port are in the record
	if len(t.host) > 0 && t.host[0] == '@' {
		var srvs []*net.SRV
		var service, protocol string

		t.host_is_ip = false

		if t.net_config.Rfc2782Srv {
			service, protocol = t.net_config.Rfc2782Service, "tcp"
		} else {
			service, protocol = "", ""
		}

		_, srvs, err = net.LookupSRV(service, protocol, t.host[1:])
		if err != nil {
			return fmt.Errorf("DNS SRV lookup failure \"%s\": %s", t.host, err)
		} else if len(srvs) == 0 {
			return fmt.Errorf("DNS SRV lookup failure \"%s\": No targets found", t.host)
		}

		for _, srv := range srvs {
			if _, err = t.populateLookup(srv.Target, int(srv.Port)); err != nil {
				return
			}
		}

		return
	}

	// Standard host:port declaration
	var port_str string
	var port uint64
	if t.host, port_str, err = net.SplitHostPort(selected); err != nil {
		return fmt.Errorf("Invalid hostport given: %s", selected)
	}

	if port, err = strconv.ParseUint(port_str, 10, 16); err != nil {
		return fmt.Errorf("Invalid port given: %s", port_str)
	}

	if t.host_is_ip, err = t.populateLookup(t.host, int(port)); err != nil {
		return
	}

	return nil
}

func (t *TransportTcp) populateLookup(host string, port int) (bool, error) {
	if ip := net.ParseIP(host); ip != nil {
		// IP address
		t.addresses = append(t.addresses, &net.TCPAddr{
			IP:   ip,
			Port: port,
		})

		return true, nil
	}

	// Lookup the hostname in DNS
	ips, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("DNS lookup failure \"%s\": %s", host, err)
	} else if len(ips) == 0 {
		return false, fmt.Errorf("DNS lookup failure \"%s\": No addresses found", host)
	}

	for _, ip := range ips {
		t.addresses = append(t.addresses, &net.TCPAddr{
			IP:   ip,
			Port: port,
		})
	}

	return false, nil
}

func (t *TransportTcp) disconnect() {
	if t.shutdown == nil {
		return
	}

	// Send shutdown request
	close(t.shutdown)
	t.wait.Wait()
	t.shutdown = nil

	// If tls, shutdown tls socket first
	if t.config.transport == "tls" {
		t.tlssocket.Close()
	}

	t.socket.Close()
}

func (t *TransportTcp) sender() {
SendLoop:
	for {
		select {
		case <-t.shutdown:
			// Shutdown
			break SendLoop
		case msg := <-t.send_chan:
			// Ask for more while we send this
			t.setChan(t.can_send)
			// Write deadline is managed by our net.Conn wrapper that tls will call into
			_, err := t.socket.Write(msg)
			if err != nil {
				if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
					// Shutdown will have been received by the wrapper
					break SendLoop
				} else {
					// Pass the error back and abort
					t.recv_chan <- err
					break SendLoop
				}
			}
		}
	}

	t.wait.Done()
}

func (t *TransportTcp) receiver() {
	var err error
	var shutdown bool
	header := make([]byte, 8)

	for {
		if err, shutdown = t.receiverRead(header); err != nil || shutdown {
			break
		}

		// Grab length of message
		length := binary.BigEndian.Uint32(header[4:8])

		// Sanity
		if length > 1048576 {
			err = fmt.Errorf("Data too large (%d)", length)
			break
		}

		// Allocate for full message
		message := make([]byte, length)

		if err, shutdown = t.receiverRead(message); err != nil || shutdown {
			break
		}

		// Pass back the message
		select {
		case <-t.shutdown:
			break
		case t.recv_chan <- [][]byte{header[0:4], message}:
		}
	} /* loop until shutdown */

	if err != nil {
		// Pass the error back and abort
		select {
		case <-t.shutdown:
		case t.recv_chan <- err:
		}
	}

	t.wait.Done()
}

func (t *TransportTcp) receiverRead(data []byte) (error, bool) {
	received := 0

RecvLoop:
	for {
		select {
		case <-t.shutdown:
			// Shutdown
			break RecvLoop
		default:
			// Timeout after socket_interval_seconds, check for shutdown, and try again
			t.socket.SetReadDeadline(time.Now().Add(socket_interval_seconds * time.Second))

			length, err := t.socket.Read(data[received:])
			received += length
			if err == nil || received >= len(data) {
				// Success
				return nil, false
			} else if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
				// Keep trying
				continue
			} else {
				// Pass an error back
				return err, false
			}
		} /* select */
	} /* loop until required amount receive or shutdown */

	return nil, true
}

func (t *TransportTcp) setChan(set chan<- int) {
	select {
	case set <- 1:
	default:
	}
}

func (t *TransportTcp) CanSend() <-chan int {
	return t.can_send
}

func (t *TransportTcp) Write(signature string, message []byte) (err error) {
	var write_buffer *bytes.Buffer
	write_buffer = bytes.NewBuffer(make([]byte, 0, len(signature)+4+len(message)))

	if _, err = write_buffer.Write([]byte(signature)); err != nil {
		return
	}
	if err = binary.Write(write_buffer, binary.BigEndian, uint32(len(message))); err != nil {
		return
	}
	if len(message) != 0 {
		if _, err = write_buffer.Write(message); err != nil {
			return
		}
	}

	t.send_chan <- write_buffer.Bytes()
	return nil
}

func (t *TransportTcp) Read() <-chan interface{} {
	return t.recv_chan
}

func (t *TransportTcp) Shutdown() {
	t.disconnect()
}

// Register the transports
func init() {
	rand.Seed(time.Now().UnixNano())

	core.RegisterTransport("tcp", NewTcpTransportFactory)
	core.RegisterTransport("tls", NewTcpTransportFactory)
}
