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
  socket      *tls.Conn
  hostport_re *regexp.Regexp

  write_buffer *bytes.Buffer
}

func CreateTransportTls(config *NetworkConfig) *TransportTls {
  rand.Seed(time.Now().UnixNano())
  return &TransportTls{config: config, hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`)}
}

func (t *TransportTls) Connect() error {
  t.write_buffer = new(bytes.Buffer)

  var tlsconfig tls.Config

  if len(t.config.SSLCertificate) > 0 && len(t.config.SSLKey) > 0 {
    log.Printf("Loading client ssl certificate: %s and %s\n", t.config.SSLCertificate, t.config.SSLKey)
    cert, err := tls.LoadX509KeyPair(t.config.SSLCertificate, t.config.SSLKey)
    if err != nil {
      return errors.New(fmt.Sprintf("Failed loading client ssl certificate: %s", err))
    }
    tlsconfig.Certificates = []tls.Certificate{cert}
  }

  if len(t.config.SSLCA) > 0 {
    log.Printf("Setting trusted CA from file: %s\n", t.config.SSLCA)
    tlsconfig.RootCAs = x509.NewCertPool()

    pemdata, err := ioutil.ReadFile(t.config.SSLCA)
    if err != nil {
      return errors.New(fmt.Sprintf("Failure reading CA certificate: %s", err))
    }

    block, _ := pem.Decode(pemdata)
    if block == nil {
      return errors.New("Failed to decode CA certificate data")
    }
    if block.Type != "CERTIFICATE" {
      return errors.New(fmt.Sprintf("Specified CA certificate is not a certificate: %s", t.config.SSLCA))
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
      return errors.New(fmt.Sprintf("Failed to parse CA certificate: %s", err))
    }
    tlsconfig.RootCAs.AddCert(cert)
  }

  for {
    // Pick a random server from the list.
    hostport := t.config.Servers[rand.Int()%len(t.config.Servers)]
    submatch := t.hostport_re.FindSubmatch([]byte(hostport))
    if submatch == nil {
      log.Printf("Invalid host:port given: %s\n", hostport)
      goto TryNextServer
    }

    {
      // Lookup the server in DNS (if this is IP it will implicitly return)
      host := string(submatch[1])
      port := string(submatch[2])
      addresses, err := net.LookupHost(host)
      if err != nil {
        log.Printf("DNS lookup failure \"%s\": %s\n", host, err)
        goto TryNextServer
      }

      {
        // Select a random address from the pool of addresses provided by DNS
        address := addresses[rand.Int()%len(addresses)]
        addressport := net.JoinHostPort(address, port)

        log.Printf("Connecting to %s (%s) \n", addressport, host)

        tcpsocket, err := net.DialTimeout("tcp", addressport, t.config.timeout)
        if err != nil {
          log.Printf("Failure connecting to %s: %s\n", address, err)
          goto TryNextServer
        }

        t.socket = tls.Client(tcpsocket, &tlsconfig)
        t.socket.SetDeadline(time.Now().Add(t.config.timeout))
        err = t.socket.Handshake()
        if err != nil {
          log.Printf("Handshake failure with %s: Failed to TLS handshake: %s\n", address, err)
          goto TryNextServer
        }

        log.Printf("Connected with %s\n", address)

        // Connected, let's rock and roll.
        return nil
      }
    }

  TryNextServer:
    time.Sleep(t.config.reconnect)
    t.socket.Close()
    continue
  } /* Loop forever */
}

func (t *TransportTls) Write(p []byte) (int, error) {
  return t.write_buffer.Write(p)
}

func (t *TransportTls) Flush() (int64, error) {
  t.socket.SetDeadline(time.Now().Add(t.config.timeout))

  return t.write_buffer.WriteTo(t.socket)
}

func (t *TransportTls) Read(p []byte) (int, error) {
  t.socket.SetDeadline(time.Now().Add(t.config.timeout))

  return t.socket.Read(p)
}

func (t *TransportTls) Disconnect() {
  t.socket.Close()
}
