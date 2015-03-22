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
	"compress/zlib"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/driskell/log-courier/src/lc-lib/core"
	"github.com/driskell/log-courier/src/lc-lib/publisher"
)

// Support for newer SSL signature algorithms
import _ "crypto/sha256"
import _ "crypto/sha512"

const (
	// Essentially, this is how often we should check for disconnect/shutdown during socket reads
	socket_interval_seconds = 1
	// TODO(driskell): Make the idle timeout configurable like the network timeout is?
	keepalive_timeout = 15
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
	net_config  *core.NetworkConfig
}

type TransportTcp struct {
	config    *TransportTcpFactory
	socket    net.Conn
	tlssocket *tls.Conn

	controllerChan chan int
	controllerWait sync.WaitGroup
	endpoint       *publisher.EndpointRemote
	fail_chan      chan error

	wait        sync.WaitGroup
	sendControl chan int
	recvControl chan int

	send_chan chan []byte

	// Use in receiver routine only
	pong_pending bool
	pong_timer   *time.Timer
}

// NewTransportTcpFactory create a new TransportTcpFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTcpFactory(config *core.Config, config_path string, unused map[string]interface{}, name string) (core.TransportFactory, error) {
	var err error

	ret := &TransportTcpFactory{
		transport:   name,
		hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
		net_config:  &config.Network,
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

// NewTransport returns a new Transport interface using the settings from the
// TransportTcpFactory.
func (f *TransportTcpFactory) NewTransport(iendpoint interface{}) core.Transport {
	// TODO: Remove hack for stopping EndpointRemote needing to be in core
	endpoint := iendpoint.(*publisher.EndpointRemote)

	ret := &TransportTcp{
		config:         f,
		endpoint:       endpoint,
		controllerChan: make(chan int),
	}

	ret.controllerWait.Add(1)

	go ret.controller()

	return ret
}

// TODO: This is not called anymore
/*func (t *TransportTcp) ReloadConfig(new_net_config *core.NetworkConfig) int {
	// Check we can grab new TCP config to compare, if not force transport reinit
	new_config, ok := new_net_config.TransportFactory.(*TransportTcpFactory)
	if !ok {
		return core.Reload_Transport
	}

	// TODO: Check timestamps of underlying certificate files to detect changes
	if new_config.SSLCertificate != t.config.SSLCertificate || new_config.SSLKey != t.config.SSLKey || new_config.SSLCA != t.config.SSLCA {
		return core.Reload_Transport
	}

	// Publisher handles changes to net_config, but ensure we store the latest in case it asks for a reconnect
	t.config.net_config = new_net_config

	return core.Reload_None
}*/

// controller is the master routine which handles connection and reconnection
// When reconnecting, the socket and all routines are torn down and restarted.
// It also
func (t *TransportTcp) controller() {
	defer func() {
		t.controllerWait.Done()
	}()

	// Main connect loop
	for {
		var err error

		err = t.connect()
		if err == nil {
			// Connected - sit and wait for shutdown or error
			select {
			// TODO: Handle configuration reload
			case <-t.controllerChan:
				// Shutdown request
				t.disconnect()
				return
			case err = <-t.fail_chan:
			}
		}

		// Error occurred
		log.Error("Transport error, will try again: %s", err)

		t.disconnect()

		// If this returns false, we are shutting down
		if !t.reconnectWait() {
			break
		}
	}
}

// reconnectWait waits the reconnect timeout before attempting to reconnect.
// It also monitors for shutdown and configuration reload events while waiting.
func (t *TransportTcp) reconnectWait() bool {
	now := time.Now()
	reconnect_due := now.Add(t.config.net_config.Reconnect)

	for {
		select {
		// TODO: Handle configuration reload
		case <-t.controllerChan:
			// Shutdown request
			return false
		case <-time.After(reconnect_due.Sub(now)):
			break
		}

		now = time.Now()
		if now.After(reconnect_due) {
			break
		}
	}

	return true
}

// connect connects the socket and starts the sender and receiver routines
func (t *TransportTcp) connect() error {
	if t.sendControl != nil {
		t.disconnect()
	}

	addr, err := t.endpoint.AddressPool.Next()
	if err != nil {
		return fmt.Errorf("Failed to select next address for %s: %s", t.endpoint.AddressPool.Server(), err)
	}

	desc := t.endpoint.AddressPool.Desc()

	log.Info("Attempting to connect to %s", desc)

	tcpsocket, err := net.DialTimeout("tcp", addr.String(), t.config.net_config.Timeout)
	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Now wrap in TLS if this is the "tls" transport
	if t.config.transport == "tls" {
		// Disable SSLv3 (mitigate POODLE vulnerability)
		t.config.tls_config.MinVersion = tls.VersionTLS10

		// Set the tlsconfig server name for server validation (required since Go 1.3)
		t.config.tls_config.ServerName = t.endpoint.AddressPool.Host()

		t.tlssocket = tls.Client(&transportTcpWrap{transport: t, tcpsocket: tcpsocket}, &t.config.tls_config)
		t.tlssocket.SetDeadline(time.Now().Add(t.config.net_config.Timeout))
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
	t.sendControl = make(chan int, 1)
	t.recvControl = make(chan int, 1)
	t.send_chan = make(chan []byte, 1)

	// Failure channel - ensure we can fit 2 errors here, one from sender and one
	// from receive - otherwise if both fail at the same time, disconnect blocks
	// NOTE: This may not be necessary anymore - both recv_chan pushes also select
	//       on the shutdown channel, which will close on the first error returned
	t.fail_chan = make(chan error, 2)

	// Start with a send
	t.endpoint.Ready()

	t.wait.Add(2)

	// Start separate sender and receiver so we can asynchronously send and
	// receive for max performance. They have to be different routines too because
	// we don't have cross-platform poll, so they will need to block. Of course,
	// we'll time out and check shutdown on occasion
	go t.sender()
	go t.receiver()

	return nil
}

// disconnect shuts down the sender and receiver routines and disconnects the
// socket
func (t *TransportTcp) disconnect() {
	if t.sendControl == nil {
		return
	}

	// Send shutdown request
	close(t.sendControl)
	close(t.recvControl)
	t.wait.Wait()
	t.sendControl = nil
	t.recvControl = nil

	// If tls, shutdown tls socket first
	if t.config.transport == "tls" {
		t.tlssocket.Close()
	}

	t.socket.Close()
}

// sender handles socket writes
func (t *TransportTcp) sender() {
	ping_timer := time.NewTimer(keepalive_timeout * time.Second)

SendLoop:
	for {
		select {
		case <-t.sendControl:
			// Shutdown
			break SendLoop
		case <-ping_timer.C:
		case msg := <-t.send_chan:
			// Ask for more while we send this
			t.endpoint.Ready()

			// Write deadline is managed by our net.Conn wrapper that tls will call into
			_, err := t.socket.Write(msg)
			if err != nil {
				if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
					// Shutdown will have been received by the wrapper
					break SendLoop
				}
				// Fail the transport
				select {
				case <-t.sendControl:
				case t.fail_chan <- err:
				}
				break SendLoop
			}
		}
	}

	t.wait.Done()
}

// receiver handles socket reads
func (t *TransportTcp) receiver() {
	var err error
	var shutdown bool
	var message []byte

	header := make([]byte, 8)

	for {
		if shutdown, err = t.receiverRead(header); shutdown || err != nil {
			break
		}

		// Grab length of message
		length := binary.BigEndian.Uint32(header[4:8])

		// Sanity
		if length > 1048576 {
			err = fmt.Errorf("Data too large (%d)", length)
			break
		}

		if length > 0 {
			// Allocate for full message
			message = make([]byte, length)
			if shutdown, err = t.receiverRead(message); shutdown || err != nil {
				break
			}
		} else {
			message = []byte("")
		}

		switch {
		case bytes.Compare(header[0:4], []byte("PONG")) == 0:
			if shutdown = t.sendResponse(&publisher.PongResponse{}); shutdown {
				break
			}
		case bytes.Compare(header[0:4], []byte("ACKN")) == 0:
			if shutdown, err = t.processAckn(message); shutdown || err != nil {
				break
			}
		default:
			err = fmt.Errorf("Unexpected message code: %s", header[0:4])
			break
		}
	}

	if err != nil {
		// Pass the error back and abort
		for {
			select {
			case <-t.recvControl:
				// Shutdown
				break
			case t.fail_chan <- err:
			}
		}
	}

	t.wait.Done()
}

// receiverRead will repeatedly read from the socket until the given byte array
// is filled.
func (t *TransportTcp) receiverRead(data []byte) (bool, error) {
	received := 0

RecvLoop:
	for {
		select {
		case <-t.recvControl:
			// Shutdown
			break RecvLoop
		default:
			// Timeout after socket_interval_seconds, check for shutdown, and try again
			t.socket.SetReadDeadline(time.Now().Add(socket_interval_seconds * time.Second))

			length, err := t.socket.Read(data[received:])
			received += length

			if received >= len(data) {
				// Success
				return false, nil
			}

			if err == nil {
				// Keep trying
				continue
			}

			if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
				// Keep trying
				continue
			}

			// Pass an error back
			return false, err
		}
	}

	return true, nil
}

// processAckn parses an acknowledgement message and passes the information to
// the Publisher for processing
func (t *TransportTcp) processAckn(data []byte) (bool, error) {
	if len(data) != 20 {
		return false, fmt.Errorf("Protocol error: Corrupt message (ACKN size %d != 20)", len(data))
	}

	return t.sendResponse(&publisher.AckResponse{string(data[0:16]), binary.BigEndian.Uint32(data[16:20])}), nil
}

// sendResponse ships a response to the Publisher whilst also monitoring for any
// shutdown signal. Returns true if shutdown was signalled
func (t *TransportTcp) sendResponse(response interface{}) bool {
	select {
	case <-t.recvControl:
		return true
	case t.endpoint.ResponseChan() <- t.endpoint.NewResponse(response):
	}
	return false
}

// Write a message to the transport
func (t *TransportTcp) Write(ipayload interface{}) error {
	// TODO: This is a hack to prevent us having to put PendingPayload in core
	payload := ipayload.(*publisher.PendingPayload)

	var dataBuffer bytes.Buffer

	// Create the compressed data payload
	// 16-byte Nonce, followed by the compressed event data
	// The event data is each event, prefixed with a 4-byte uint32 length, one
	// after the other
	if _, err := dataBuffer.Write([]byte(payload.Nonce)); err != nil {
		return err
	}

	compressor, err := zlib.NewWriterLevel(&dataBuffer, 3)
	if err != nil {
		return err
	}

	for _, event := range payload.Events() {
		if err := binary.Write(compressor, binary.BigEndian, uint32(len(event.Event))); err != nil {
			return err
		}

		if _, err := compressor.Write(event.Event); err != nil {
			return err
		}
	}

	compressor.Close()

	// Encapsulate the data into the message
	// 4-byte message header (JDAT = JSON Data, Compressed)
	// 4-byte uint32 data length
	// Then the data
	messageBuffer := bytes.NewBuffer(make([]byte, 0, 4+4+dataBuffer.Len()))

	if _, err := messageBuffer.Write([]byte("JDAT")); err != nil {
		return err
	}

	if err := binary.Write(messageBuffer, binary.BigEndian, uint32(dataBuffer.Len())); err != nil {
		return err
	}

	if _, err := messageBuffer.ReadFrom(&dataBuffer); err != nil {
		return err
	}

	t.send_chan <- messageBuffer.Bytes()
	return nil
}

// Ping the remote server
func (t *TransportTcp) Ping() error {
	// Encapsulate the ping into a message
	// 4-byte message header (PING)
	// 4-byte uint32 data length (0 length for PING)
	t.send_chan <- []byte{'P','I','N','G',0,0,0,0}
	return nil
}

// Shutdown the transport
func (t *TransportTcp) Shutdown() {
	close(t.controllerChan)
}

// Wait until close finishes
func (t *TransportTcp) Wait() {
	t.controllerWait.Wait()
}

// Register the transports
func init() {
	core.RegisterTransport("tcp", NewTransportTcpFactory)
	core.RegisterTransport("tls", NewTransportTcpFactory)
}
