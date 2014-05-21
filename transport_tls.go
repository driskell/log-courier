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
  "sync"
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

  wait sync.WaitGroup

  notify_send chan int
  notify_recv chan int

  send_chan chan []byte
  recv_chan chan interface{}

  can_send chan int
  can_recv chan int
}

const (
  tls_signal_process  = 1
  tls_signal_shutdown = 2
)

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
  t.write_buffer = new(bytes.Buffer)

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

  // Signal channels
  t.notify_send = make(chan int, 1)
  t.notify_recv = make(chan int, 1)
  t.send_chan = make(chan []byte, 1)
  t.recv_chan = make(chan interface{}, 1)
  t.can_send = make(chan int, 1)
  t.can_recv = make(chan int, 1)

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
SignalLoop:
  for {
    select {
      case msg := <-t.notify_send:
        // Control channel message - maybe something to send or need to shutdown
        switch msg {
        case tls_signal_process:
          // Loop until we sent everything
        SendLoop:
          for {
            select {
              case msg := <-t.send_chan:
                // Expect to finish writing within the timeout period
                // TODO: Loop with this so we don't have to wait config.timeout to process shutdown
                // TODO: Don't fail if write takes longer than config.timeout - it would be normal - we'll detect timeout in receive
                t.socket.SetWriteDeadline(time.Now().Add(t.config.timeout))
                log.Printf("sender() - sending %d bytes", len(msg))
                _, err := t.socket.Write(msg)
                if err == nil {
                  log.Printf("sender() - sent")
                  t.setChan(t.can_send)
                } else {
                  log.Printf("sender() - error: %s", err)
                  // Pass error back
                  t.recv_chan <- err
                }

                // Data has gone out, so we should expect some back (or an error)
                t.setChan(t.can_recv)
              default:
                // Nothing left to send
                break SendLoop
            }
          }
        case tls_signal_shutdown:
          log.Printf("sender() - shutdown")
          // Shutdown
          break SignalLoop
        }
    } /* select */
  } /* loop until shutdown */

  t.wait.Done()
}

func (t *TransportTls) receiver() {
SignalLoop:
  for {
    select {
      case signal := <-t.notify_recv:
        // Control channel message - maybe something to receive or need to shutdown
        switch signal {
        case tls_signal_process:
          // Expect to hear back within the timeout period
          // TODO: Loop with this so we don't have to wait config.timeout to process shutdown
          t.socket.SetReadDeadline(time.Now().Add(t.config.timeout))
          log.Printf("receiver() - receiving 6 bytes")
          // We only receive ACK at the moment, which is 6 bytes
          msg := make([]byte, 6)
          _, err := t.socket.Read(msg)
          if err == nil {
            log.Printf("receiver() - received")
            // Pass the message back
            t.recv_chan <- msg
          } else {
            log.Printf("receiver() - error: %s", err)
            // Pass an error back
            t.recv_chan <- err
          }

        case tls_signal_shutdown:
          log.Printf("receiver() - shutdown")
          // Shutdown
          break SignalLoop
        }
    } /* select */
  } /* loop until shutdown */

  t.wait.Done()
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
  t.send_chan <- t.write_buffer.Bytes()
  t.write_buffer.Reset()

  // Ask for send to start
  t.notify_send <- tls_signal_process
  return nil
}

func (t *TransportTls) Read() ([]byte, error) {
  var msg interface{}
  select {
  case msg = <-t.recv_chan:
    // Ask for more
    t.notify_recv <- tls_signal_process
  default:
    t.notify_recv <- tls_signal_process
    msg = <-t.recv_chan
  }

  // Error? Or data?
  switch msg.(type) {
    case error:
      return nil, msg.(error)
    default:
      return msg.([]byte), nil
  }
}

func (t *TransportTls) Disconnect() {
  // Send shutdown request
  t.notify_send <- tls_signal_shutdown
  t.notify_recv <- tls_signal_shutdown
  t.wait.Wait()
  t.socket.Close()
  t.write_buffer.Reset()
}
