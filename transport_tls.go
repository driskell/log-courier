package main

import (
  "bytes"
  "crypto/tls"
  "crypto/x509"
  "encoding/pem"
  "errors"
  "fmt"
  "io/ioutil"
  "log"
  "math/rand"
  "net"
  "regexp"
  "time"
)

// Support for newer SSL signature algorithms
import _ "crypto/sha256"
import _ "crypto/sha512"

type TransportTls struct {
  config      *NetworkConfig
  tls_config  tls.Config
  socket      *tls.Conn
  hostport_re *regexp.Regexp

  write_buffer *bytes.Buffer

  can_send chan int
  can_recv chan int
}

func CreateTransportTls(config *NetworkConfig) (*TransportTls, error) {
  rand.Seed(time.Now().UnixNano())

  ret := &TransportTls{
    config: config,
    hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
    write_buffer: new(bytes.Buffer),
  }

  if len(config.SSLCertificate) > 0 && len(config.SSLKey) > 0 {
    log.Printf("Loading client ssl certificate: %s and %s\n", config.SSLCertificate, config.SSLKey)
    cert, err := tls.LoadX509KeyPair(config.SSLCertificate, config.SSLKey)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed loading client ssl certificate: %s", err))
    }
    ret.tls_config.Certificates = []tls.Certificate{cert}
  }

  if len(config.SSLCA) > 0 {
    log.Printf("Setting trusted CA from file: %s\n", config.SSLCA)
    ret.tls_config.RootCAs = x509.NewCertPool()

    pemdata, err := ioutil.ReadFile(config.SSLCA)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failure reading CA certificate: %s", err))
    }

    block, _ := pem.Decode(pemdata)
    if block == nil {
      return nil, errors.New("Failed to decode CA certificate data")
    }
    if block.Type != "CERTIFICATE" {
      return nil, errors.New(fmt.Sprintf("Specified CA certificate is not a certificate: %s", config.SSLCA))
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed to parse CA certificate: %s", err))
    }
    ret.tls_config.RootCAs.AddCert(cert)
  }

  return ret, nil
}

func (t *TransportTls) Connect() error {
Connect:
  for {
    for {
      // Pick a random server from the list.
      hostport := t.config.Servers[rand.Int()%len(t.config.Servers)]
      submatch := t.hostport_re.FindSubmatch([]byte(hostport))
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

      tcpsocket, err := net.DialTimeout("tcp", addressport, t.config.timeout)
      if err != nil {
        log.Printf("Failure connecting to %s: %s\n", address, err)
        break
      }

      t.socket = tls.Client(tcpsocket, &t.tls_config)
      t.socket.SetDeadline(time.Now().Add(t.config.timeout))
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

    time.Sleep(t.config.reconnect)
  } /* Connect: for */

  // Send/Recv signal channels
  t.can_send = make(chan int, 1)
  t.can_recv = make(chan int, 1)

  // Start with a send
  t.can_send <- 1

  return nil
}

func (t *TransportTls) setChan(set chan int) {
  select {
  case set <- 1:
  default:
  }
}

func (t *TransportTls) CanSend() chan int {
  return t.can_send
}

func (t *TransportTls) CanRecv() chan int {
  return t.can_recv
}

func (t *TransportTls) Write(p []byte) (int, error) {
  return t.write_buffer.Write(p)
}

func (t *TransportTls) Flush() error {
  // Expect to finish writing within the timeout period
  t.socket.SetWriteDeadline(time.Now().Add(t.config.timeout))

  _, err := t.write_buffer.WriteTo(t.socket)
  if err == nil {
    t.setChan(t.can_recv)
  }
  return err
}

func (t *TransportTls) Read() ([]byte, error) {
  // Expect to hear back within the timeout period
  t.socket.SetReadDeadline(time.Now().Add(t.config.timeout))

  // We only receive ACK at the moment, which is 6 bytes
  msg := make([]byte, 6)
  _, err := t.socket.Read(msg)
  if err == nil {
    t.setChan(t.can_send)
    return msg, nil
  }
  return nil, err
}

func (t *TransportTls) Disconnect() {
  t.socket.Close()
  t.write_buffer.Reset()
}
