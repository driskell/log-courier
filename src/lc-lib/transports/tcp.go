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
  "errors"
  "fmt"
  "io/ioutil"
  "lc-lib/core"
  "math/rand"
  "net"
  "regexp"
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
        return nil, fmt.Errorf("Failure reading CA certificate: %s", err)
      }

      block, _ := pem.Decode(pemdata)
      if block == nil {
        return nil, errors.New("Failed to decode CA certificate data")
      }
      if block.Type != "CERTIFICATE" {
        return nil, fmt.Errorf("Specified CA certificate is not a certificate: %s", ret.SSLCA)
      }

      cert, err := x509.ParseCertificate(block.Bytes)
      if err != nil {
        return nil, fmt.Errorf("Failed to parse CA certificate: %s", err)
      }

      ret.tls_config.RootCAs.AddCert(cert)
    }
  } else {
    if err := config.ReportUnusedConfig(config_path, unused); err != nil {
      return nil, err
    }
  }

  return ret, nil
}

func (f *TransportTcpFactory) NewTransport(config *core.NetworkConfig) (core.Transport, error) {
  return &TransportTcp{config: f, net_config: config}, nil
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

  // Pick a random server from the list.
  hostport := t.net_config.Servers[rand.Int()%len(t.net_config.Servers)]
  // TODO: Parse and lookup using net.ResolveTCPAddr
  submatch := t.config.hostport_re.FindSubmatch([]byte(hostport))
  if submatch == nil {
    return fmt.Errorf("Invalid host:port given: %s", hostport)
  }

  // Lookup the server in DNS (if this is IP it will implicitly return)
  host := string(submatch[1])
  port := string(submatch[2])
  addresses, err := net.LookupHost(host)
  if err != nil {
    return fmt.Errorf("DNS lookup failure \"%s\": %s", host, err)
  }

  // Select a random address from the pool of addresses provided by DNS
  address := addresses[rand.Int()%len(addresses)]
  addressport := net.JoinHostPort(address, port)

  log.Info("Attempting to connect to %s (%s)", addressport, host)

  tcpsocket, err := net.DialTimeout("tcp", addressport, t.net_config.Timeout)
  if err != nil {
    return fmt.Errorf("Failed to connect to %s: %s", address, err)
  }

  // Now wrap in TLS if this is the "tls" transport
  if t.config.transport == "tls" {
    // Set the tlsconfig server name for server validation (since Go 1.3)
    t.config.tls_config.ServerName = host

    t.tlssocket = tls.Client(&transportTcpWrap{transport: t, tcpsocket: tcpsocket}, &t.config.tls_config)
    t.tlssocket.SetDeadline(time.Now().Add(t.net_config.Timeout))
    err = t.tlssocket.Handshake()
    if err != nil {
      t.tlssocket.Close()
      tcpsocket.Close()
      return fmt.Errorf("TLS Handshake failure with %s: %s", address, err)
    }

    t.socket = t.tlssocket
  } else {
    t.socket = tcpsocket
  }

  log.Info("Connected to %s", address)

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
          // Pass error back
          t.recv_chan <- err
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
