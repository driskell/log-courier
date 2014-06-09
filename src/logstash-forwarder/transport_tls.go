package main

import (
  "bytes"
  "crypto/tls"
  "crypto/x509"
  "encoding/binary"
  "encoding/pem"
  "errors"
  "fmt"
  "io/ioutil"
  "log"
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
  socket_interval_seconds = 5
)

type TransportTlsFactory struct {
  SSLCertificate string `json:"ssl certificate"`
  SSLKey         string `json:"ssl key"`
  SSLCA          string `json:"ssl ca"`

  hostport_re *regexp.Regexp
  tls_config  tls.Config
}

type TransportTls struct {
  config     *TransportTlsFactory
  net_config *NetworkConfig
  socket     *tls.Conn

  wait     sync.WaitGroup
  shutdown chan int

  send_chan chan []byte
  recv_chan chan interface{}

  can_send chan int
}

// If tls.Conn.Write ever times out it will permanently break, so we cannot use SetWriteDeadline with it directly
// So we wrap the given tcpsocket and handle the SetWriteDeadline there and check shutdown signal and loop
// Inside tls.Conn the Write blocks until it finishes and everyone is happy
type TransportTlsWrap struct {
  transport *TransportTls
  tcpsocket net.Conn

  net.Conn
}

func init() {
  rand.Seed(time.Now().UnixNano())
}

func NewTransportTlsFactory(config_path string, config map[string]interface{}) (TransportFactory, error) {
  var err error

  ret := &TransportTlsFactory{
    hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
  }

  if err = PopulateConfig(ret, config_path, config); err != nil {
    return nil, err
  }

  if len(ret.SSLCertificate) > 0 && len(ret.SSLKey) > 0 {
    log.Printf("Loading client ssl certificate: %s and %s\n", ret.SSLCertificate, ret.SSLKey)
    cert, err := tls.LoadX509KeyPair(ret.SSLCertificate, ret.SSLKey)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed loading client ssl certificate: %s", err))
    }
    ret.tls_config.Certificates = []tls.Certificate{cert}
  }

  if len(ret.SSLCA) > 0 {
    log.Printf("Setting trusted CA from file: %s\n", ret.SSLCA)
    ret.tls_config.RootCAs = x509.NewCertPool()

    pemdata, err := ioutil.ReadFile(ret.SSLCA)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failure reading CA certificate: %s", err))
    }

    block, _ := pem.Decode(pemdata)
    if block == nil {
      return nil, errors.New("Failed to decode CA certificate data")
    }
    if block.Type != "CERTIFICATE" {
      return nil, errors.New(fmt.Sprintf("Specified CA certificate is not a certificate: %s", ret.SSLCA))
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed to parse CA certificate: %s", err))
    }
    ret.tls_config.RootCAs.AddCert(cert)
  }

  return ret, nil
}

func (f *TransportTlsFactory) NewTransport(config *NetworkConfig) (Transport, error) {
  return &TransportTls{config: f, net_config: config}, nil
}

func (t *TransportTls) Connect() error {
Connect:
  for {
    for {
      // Pick a random server from the list.
      hostport := t.net_config.Servers[rand.Int()%len(t.net_config.Servers)]
      submatch := t.config.hostport_re.FindSubmatch([]byte(hostport))
      if submatch == nil {
        log.Printf("Invalid host:port given: %s\n", hostport)
        break
      }

      // Lookup the server in DNS (if this is IP it will implicitly return)
      host := string(submatch[1])
      port := string(submatch[2])
      addresses, err := net.LookupHost(host)
      if err != nil {
        log.Printf("DNS lookup failure \"%s\": %s\n", host, err)
        break
      }

      // Select a random address from the pool of addresses provided by DNS
      address := addresses[rand.Int()%len(addresses)]
      addressport := net.JoinHostPort(address, port)

      log.Printf("Connecting to %s (%s) \n", addressport, host)

      tcpsocket, err := net.DialTimeout("tcp", addressport, t.net_config.Timeout)
      if err != nil {
        log.Printf("Failure connecting to %s: %s\n", address, err)
        break
      }

      t.socket = tls.Client(&TransportTlsWrap{transport: t, tcpsocket: tcpsocket}, &t.config.tls_config)
      t.socket.SetDeadline(time.Now().Add(t.net_config.Timeout))
      err = t.socket.Handshake()
      if err != nil {
        t.socket.Close()
        log.Printf("TLS Handshake failure with %s: %s\n", address, err)
        break
      }

      log.Printf("Connected with %s\n", address)

      // Connected, let's rock and roll.
      break Connect

    } /* for, break for sleep */

    time.Sleep(t.net_config.Reconnect)
  } /* Connect: for */

  // Signal channels
  t.shutdown = make(chan int, 1)
  t.send_chan = make(chan []byte, 1)
  t.recv_chan = make(chan interface{}, 1)
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

func (t *TransportTls) sender() {
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

func (t *TransportTls) receiver() {
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
      err = errors.New(fmt.Sprintf("Data too large (%d)", length))
      break
    }

    // Allocate for full message
    message := make([]byte, length)

    if err, shutdown = t.receiverRead(message); err != nil || shutdown {
      break
    }

    // Pass back the message
    t.recv_chan <- [][]byte{header[0:4], message}
  } /* loop until shutdown */

  if err != nil {
    // Pass the error back and abort
    t.recv_chan <- err
  }

  t.wait.Done()
}

func (t *TransportTls) receiverRead(data []byte) (error, bool) {
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

func (t *TransportTls) setChan(set chan<- int) {
  select {
  case set <- 1:
  default:
  }
}

func (t *TransportTls) CanSend() <-chan int {
  return t.can_send
}

func (t *TransportTls) Write(signature string, message []byte) (err error) {
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

func (t *TransportTls) Read() <-chan interface{} {
  return t.recv_chan
}

func (t *TransportTls) Disconnect() {
  // Send shutdown request
  close(t.shutdown)
  t.wait.Wait()
  t.socket.Close()
}

func (w *TransportTlsWrap) Read(b []byte) (int, error) {
  return w.tcpsocket.Read(b)
}

func (w *TransportTlsWrap) Write(b []byte) (n int, err error) {
  length := 0

RetrySend:
  for {
    // Timeout after socket_interval_seconds, check for shutdown, and try again
    w.tcpsocket.SetWriteDeadline(time.Now().Add(socket_interval_seconds * time.Second))

    n, err = w.tcpsocket.Write(b[length:])
    length += n
    if err == nil {
      return length, err
    } else if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
      // Check for shutdown, then try again
      select {
      case <-w.transport.shutdown:
        // Shutdown
        return length, err
      default:
        goto RetrySend
      }
    } else {
      return length, err
    }
  } /* loop forever */
}

func (w *TransportTlsWrap) Close() error {
  return w.tcpsocket.Close()
}

func (w *TransportTlsWrap) LocalAddr() net.Addr {
  return w.tcpsocket.LocalAddr()
}

func (w *TransportTlsWrap) RemoteAddr() net.Addr {
  return w.tcpsocket.RemoteAddr()
}

func (w *TransportTlsWrap) SetDeadline(t time.Time) error {
  return w.tcpsocket.SetDeadline(t)
}

func (w *TransportTlsWrap) SetReadDeadline(t time.Time) error {
  return w.tcpsocket.SetReadDeadline(t)
}

func (w *TransportTlsWrap) SetWriteDeadline(t time.Time) error {
  return w.tcpsocket.SetWriteDeadline(t)
}
