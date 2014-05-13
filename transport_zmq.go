package main

import (
  "bytes"
  "errors"
  "fmt"
  "log"
  "math/rand"
  "net"
  "regexp"
  "time"
  zmq "github.com/alecthomas/gozmq"
)

type TransportZmq struct {
  config      *NetworkConfig
  context     *zmq.Context
  dealer      *zmq.Socket
  hostport_re *regexp.Regexp

  write_buffer *bytes.Buffer
}

var TransportZmq_Context *zmq.Context

func CreateTransportZmq(config *NetworkConfig) (*TransportZmq, error) {
  if TransportZmq_Context == nil {
    var err error
    TransportZmq_Context, err = zmq.NewContext()
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed to create ZMQ context: %s", err))
    }
  }
  return &TransportZmq{config: config, context: TransportZmq_Context, hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`)}, nil
}

func (t *TransportZmq) Connect() error {
  t.write_buffer = new(bytes.Buffer)

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

  	    var err error

        t.dealer, err = t.context.NewSocket(zmq.DEALER)
        if err != nil {
          log.Printf("Failed to create a new ZMQ DEALER socket: %s\n", err)
          goto TryNextServer
        }

        err = t.dealer.Connect("tcp://" + addressport)
        if err != nil {
          log.Printf("Failure connecting to %s: %s\n", address, err)
          goto TryNextServer
        }

        err = t.dealer.SetReconnectIvl(t.config.reconnect)
        if err != nil {
          log.Printf("Failure setting ZMQ dealer socket option: %s\n", address, err)
          t.dealer.Close()
          goto TryNextServer
        }

        err = t.dealer.SetLinger(0)
        if err != nil {
          log.Printf("Failure setting ZMQ dealer socket option: %s\n", address, err)
          t.dealer.Close()
          goto TryNextServer
        }

        err = t.dealer.SetRcvTimeout(t.config.timeout)
        if err != nil {
          log.Printf("Failure setting ZMQ dealer socket option: %s\n", address, err)
          t.dealer.Close()
          goto TryNextServer
        }
        err = t.dealer.SetSndTimeout(t.config.timeout)
        if err != nil {
          log.Printf("Failure setting ZMQ dealer socket option: %s\n", address, err)
          t.dealer.Close()
          goto TryNextServer
        }

        log.Printf("Connected with %s\n", address)

        // Connected, let's rock and roll.
        return nil
      }
    }

  TryNextServer:
    time.Sleep(t.config.reconnect)
    continue
  } /* Loop forever */
}

func (t *TransportZmq) Write(p []byte) (int, error) {
  return t.write_buffer.Write(p)
}

func (t *TransportZmq) Flush() (int64, error) {
  t.dealer.Send([]byte(""), zmq.SNDMORE)
  return int64(t.write_buffer.Len()), t.dealer.Send(t.write_buffer.Bytes(), 0)
}

func (t *TransportZmq) Read(p []byte) (int, error) {
  for {
    msg, err := t.dealer.Recv(0)
    if err != nil {
      return 0, err
    }
    more, err := t.dealer.RcvMore()
    if err != nil {
      return 0, err
    }
    if more {
      continue
    }
    copy(msg, p)
    return len(msg), nil
  } /* loop forever */
}

func (t *TransportZmq) Disconnect() {
  t.dealer.Close()
}
