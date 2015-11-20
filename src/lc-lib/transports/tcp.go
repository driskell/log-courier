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

	"github.com/driskell/log-courier/src/lc-lib/config"
	"github.com/driskell/log-courier/src/lc-lib/core"
)

import _ "crypto/sha256" // Support for newer SSL signature algorithms
import _ "crypto/sha512" // Support for newer SSL signature algorithms

const (
	// Essentially, this is how often we should check for disconnect/shutdown during socket reads
	socketIntervalSeconds = 1
)

// TransportTCPFactory holds the configuration from the configuration file
// It allows creation of TransportTCP instances that use this configuration
type TransportTCPFactory struct {
	transport string

	SSLCertificate string `config:"ssl certificate"`
	SSLKey         string `config:"ssl key"`
	SSLCA          string `config:"ssl ca"`

	hostportRegexp *regexp.Regexp
	tlsConfig      tls.Config
	netConfig      *config.Network
}

// TransportTCP implements a transport that sends over TCP
// It also can optionally introduce a TLS layer for security
type TransportTCP struct {
	config    *TransportTCPFactory
	socket    net.Conn
	tlssocket *tls.Conn

	controllerChan chan int
	endpoint       EndpointCallback
	failChan       chan error

	wait        sync.WaitGroup
	sendControl chan int
	recvControl chan int

	sendChan chan []byte

	// Use in receiver routine only
	pongPending bool
	pongTimer   *time.Timer
}

// NewTransportTCPFactory create a new TransportTCPFactory from the provided
// configuration data, reporting back any configuration errors it discovers.
func NewTransportTCPFactory(config *config.Config, configPath string, unused map[string]interface{}, name string) (interface{}, error) {
	var err error

	ret := &TransportTCPFactory{
		transport:      name,
		hostportRegexp: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
		netConfig:      &config.Network,
	}

	// Only allow SSL configurations if this is "tls"
	if name == "tls" {
		if err = config.PopulateConfig(ret, configPath, unused); err != nil {
			return nil, err
		}

		if len(ret.SSLCertificate) > 0 && len(ret.SSLKey) > 0 {
			cert, err := tls.LoadX509KeyPair(ret.SSLCertificate, ret.SSLKey)
			if err != nil {
				return nil, fmt.Errorf("Failed loading client ssl certificate: %s", err)
			}

			ret.tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if len(ret.SSLCA) > 0 {
			ret.tlsConfig.RootCAs = x509.NewCertPool()
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
					ret.tlsConfig.RootCAs.AddCert(cert)
					pemBlockNum++
				} else {
					break
				}
			}
		}
	} else {
		if err := config.ReportUnusedConfig(configPath, unused); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// NewTransport returns a new Transport interface using the settings from the
// TransportTCPFactory.
func (f *TransportTCPFactory) NewTransport(endpoint EndpointCallback) Transport {
	ret := &TransportTCP{
		config:         f,
		endpoint:       endpoint,
		controllerChan: make(chan int),
	}

	go ret.controller()

	return ret
}

// ReloadConfig returns true if the transport needs to be restarted in order
// for the new configuration to apply
func (t *TransportTCP) ReloadConfig(newNetConfig *config.Network) bool {
	newConfig := newNetConfig.Factory.(*TransportTCPFactory)

	// TODO: Check timestamps of underlying certificate files to detect changes
	if newConfig.SSLCertificate != t.config.SSLCertificate || newConfig.SSLKey != t.config.SSLKey || newConfig.SSLCA != t.config.SSLCA {
		return true
	}

	t.config.netConfig = newNetConfig

	return false
}

// controller is the master routine which handles connection and reconnection
// When reconnecting, the socket and all routines are torn down and restarted.
// It also
func (t *TransportTCP) controller() {
	defer func() {
		t.endpoint.Finished()
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
			case err = <-t.failChan:
				// Error occurred
				if err != nil {
					// If err is nil, it's a forced failure by publisher so we need not
					// call endpoint fail to let it know about it
					t.endpoint.Fail()
				}
			}
		}

		if err != nil {
			log.Error("[%s] Transport error, reconnecting: %s", t.endpoint.Pool().Server(), err)
		} else {
			log.Info("[%s] Transport reconnecting", t.endpoint.Pool().Server())
		}

		t.disconnect()

		// If this returns false, we are shutting down
		if !t.reconnectWait() {
			break
		}
	}
}

// reconnectWait waits the reconnect timeout before attempting to reconnect.
// It also monitors for shutdown and configuration reload events while waiting.
func (t *TransportTCP) reconnectWait() bool {
	now := time.Now()
	reconnectDue := now.Add(t.config.netConfig.Reconnect)

ReconnectWaitLoop:
	for {
		select {
		// TODO: Handle configuration reload
		case <-t.controllerChan:
			// Shutdown request
			return false
		case <-time.After(reconnectDue.Sub(now)):
			break ReconnectWaitLoop
		}

		now = time.Now()
		if now.After(reconnectDue) {
			break
		}
	}

	return true
}

// connect connects the socket and starts the sender and receiver routines
func (t *TransportTCP) connect() error {
	if t.sendControl != nil {
		t.disconnect()
	}

	addr, err := t.endpoint.Pool().Next()
	if err != nil {
		return fmt.Errorf("Failed to select next address: %s", err)
	}

	desc := t.endpoint.Pool().Desc()

	log.Info("[%s] Attempting to connect to %s", t.endpoint.Pool().Server(), desc)

	tcpsocket, err := net.DialTimeout("tcp", addr.String(), t.config.netConfig.Timeout)
	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", desc, err)
	}

	// Now wrap in TLS if this is the "tls" transport
	if t.config.transport == "tls" {
		// Disable SSLv3 (mitigate POODLE vulnerability)
		t.config.tlsConfig.MinVersion = tls.VersionTLS10

		// Set the tlsconfig server name for server validation (required since Go 1.3)
		t.config.tlsConfig.ServerName = t.endpoint.Pool().Host()

		t.tlssocket = tls.Client(&transportTCPWrap{transport: t, tcpsocket: tcpsocket}, &t.config.tlsConfig)
		t.tlssocket.SetDeadline(time.Now().Add(t.config.netConfig.Timeout))
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

	log.Notice("[%s] Connected to %s", t.endpoint.Pool().Server(), desc)

	// Signal channels
	t.sendControl = make(chan int, 1)
	t.recvControl = make(chan int, 1)
	t.sendChan = make(chan []byte, 1)

	// Failure channel - ensure we can fit 2 errors here, one from sender and one
	// from receive - otherwise if both fail at the same time, disconnect blocks
	// NOTE: This may not be necessary anymore - both recv_chan pushes also select
	//       on the shutdown channel, which will close on the first error returned
	t.failChan = make(chan error, 2)

	// Send a recovery single - this implicitly means we're also ready
	t.endpoint.Recover()

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
func (t *TransportTCP) disconnect() {
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

	log.Notice("[%s] Disconnected from %s", t.endpoint.Pool().Server(), t.endpoint.Pool().Desc())
}

// sender handles socket writes
func (t *TransportTCP) sender() {
SenderLoop:
	for {
		select {
		case <-t.sendControl:
			// Shutdown
			break SenderLoop
		case msg := <-t.sendChan:
			// Ask for more while we send this
			t.endpoint.Ready()

			// Write deadline is managed by our net.Conn wrapper that tls will call into
			_, err := t.socket.Write(msg)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Shutdown will have been received by the wrapper
					break SenderLoop
				}
				// Fail the transport
				select {
				case <-t.sendControl:
				case t.failChan <- err:
				}
				break SenderLoop
			}
		}
	}

	t.wait.Done()
}

// receiver handles socket reads
func (t *TransportTCP) receiver() {
	var err error
	var shutdown bool
	var message []byte

	header := make([]byte, 8)

ReceiverLoop:
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
			if shutdown = t.sendResponse(&PongResponse{t.endpoint}); shutdown {
				break ReceiverLoop
			}
		case bytes.Compare(header[0:4], []byte("ACKN")) == 0:
			if len(message) != 20 {
				err = fmt.Errorf("Protocol error: Corrupt message (ACKN size %d != 20)", len(message))
				break ReceiverLoop
			}

			if shutdown = t.sendResponse(&AckResponse{t.endpoint, string(message[0:16]), binary.BigEndian.Uint32(message[16:20])}); shutdown {
				break ReceiverLoop
			}
		default:
			err = fmt.Errorf("Unexpected message code: %s", header[0:4])
			break ReceiverLoop
		}
	}

	if err != nil {
		// Pass the error back and abort
	FailLoop:
		for {
			select {
			case <-t.recvControl:
				// Shutdown
				break FailLoop
			case t.failChan <- err:
			}
		}
	}

	t.wait.Done()
}

// receiverRead will repeatedly read from the socket until the given byte array
// is filled.
func (t *TransportTCP) receiverRead(data []byte) (bool, error) {
	received := 0

ReceiverReadLoop:
	for {
		select {
		case <-t.recvControl:
			// Shutdown
			break ReceiverReadLoop
		default:
			// Timeout after socketIntervalSeconds, check for shutdown, and try again
			t.socket.SetReadDeadline(time.Now().Add(socketIntervalSeconds * time.Second))

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

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Keep trying
				continue
			}

			// Pass an error back
			return false, err
		}
	}

	return true, nil
}

// sendResponse ships a response to the Publisher whilst also monitoring for any
// shutdown signal. Returns true if shutdown was signalled
func (t *TransportTCP) sendResponse(response Response) bool {
	select {
	case <-t.recvControl:
		return true
	case t.endpoint.ResponseChan() <- response:
	}
	return false
}

// Write a message to the transport
func (t *TransportTCP) Write(nonce string, events []*core.EventDescriptor) error {
	var dataBuffer bytes.Buffer

	// Create the compressed data payload
	// 16-byte Nonce, followed by the compressed event data
	// The event data is each event, prefixed with a 4-byte uint32 length, one
	// after the other
	if _, err := dataBuffer.Write([]byte(nonce)); err != nil {
		return err
	}

	compressor, err := zlib.NewWriterLevel(&dataBuffer, 3)
	if err != nil {
		return err
	}

	for _, event := range events {
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

	t.sendChan <- messageBuffer.Bytes()
	return nil
}

// Ping the remote server
func (t *TransportTCP) Ping() error {
	// Encapsulate the ping into a message
	// 4-byte message header (PING)
	// 4-byte uint32 data length (0 length for PING)
	t.sendChan <- []byte{'P', 'I', 'N', 'G', 0, 0, 0, 0}
	return nil
}

// Fail the transport
func (t *TransportTCP) Fail() {
	t.failChan <- nil
}

// Shutdown the transport
func (t *TransportTCP) Shutdown() {
	close(t.controllerChan)
}

// Register the transports
func init() {
	config.RegisterTransport("tcp", NewTransportTCPFactory)
	config.RegisterTransport("tls", NewTransportTCPFactory)
}
