package main

import (
  "bytes"
  "encoding/binary"
  "errors"
  "fmt"
  "log"
  "net"
  "regexp"
  "runtime"
  "sync"
  "syscall"
  zmq "github.com/alecthomas/gozmq"
)

const (
  zmq_signal_output   = "O"
  zmq_signal_input    = "I"
  zmq_signal_shutdown = "S"
)

type TransportZmqFactory struct {
  CurveServerPubKey string   `json:"curve server pub key"`
  CurvePrivKey   string   `json:"curve priv key"`
  CurvePubKey    string   `json:"curve pub key"`

  hostport_re *regexp.Regexp
  context *zmq.Context
}

type TransportZmq struct {
  config *TransportZmqFactory
  net_config  *NetworkConfig
  dealer  *zmq.Socket

  wait sync.WaitGroup

  bridge_chan chan []byte

  send_chan chan *ZMQMessage
  recv_chan chan interface{}
  recv_bridge_chan chan interface{}

  can_send chan int
}

type ZMQMessage struct {
  part  []byte
  final bool
}

func NewTransportZmqFactory(config_path string, config map[string]interface{}) (TransportFactory, error) {
  var err error

  ret := &TransportZmqFactory{
    hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
  }

  if err = PopulateConfig(ret, config_path, config); err != nil {
    return nil, err
  }

  ret.context, err = zmq.NewContext()
  if err != nil {
    return nil, errors.New(fmt.Sprintf("Failed to create ZMQ context: %s", err))
  }

  return ret, nil
}

func (f *TransportZmqFactory) NewTransport(config *NetworkConfig) (Transport, error) {
  return &TransportZmq{config: f, net_config: config}, nil
}

func (t *TransportZmq) Connect() (err error) {
  endpoints := 0

  // Outbound dealer socket will fair-queue load balance amongst peers
  if t.dealer, err = t.config.context.NewSocket(zmq.DEALER); err != nil {
    return
  }

  // Connect endpoints
  for _, hostport := range t.net_config.Servers {
    submatch := t.config.hostport_re.FindSubmatch([]byte(hostport))
    if submatch == nil {
      log.Printf("Invalid host:port given: %s\n", hostport)
      continue
    }

    // Lookup the server in DNS (if this is IP it will implicitly return)
    host := string(submatch[1])
    port := string(submatch[2])
    addresses, err := net.LookupHost(host)
    if err != nil {
      log.Printf("DNS lookup failure \"%s\": %s\n", host, err)
      continue
    }

    // Connect to each address
    for _, address := range addresses {
      addressport := net.JoinHostPort(address, port)

      if err = t.dealer.Connect("tcp://" + addressport); err != nil {
        log.Printf("Failed to connect to %s (%s), skipping", addressport, host)
        continue
      }

      log.Printf("Connected with %s (%s) \n", addressport, host)
      endpoints++
    }
  }

  if endpoints == 0 {
    return errors.New("Failed to connect to any of the specified endpoints.")
  }

  // Control sockets to connect bridge to poller
  bridge_in, err := t.config.context.NewSocket(zmq.PUSH)
  if err != nil {
    t.dealer.Close()
    return
  }
  if err = bridge_in.Bind("inproc://notify"); err != nil {
    t.dealer.Close()
    bridge_in.Close()
    return
  }

  bridge_out, err := t.config.context.NewSocket(zmq.PULL)
  if err != nil {
    t.dealer.Close()
    bridge_in.Close()
    return err
  }
  if err = bridge_out.Connect("inproc://notify"); err != nil {
    t.dealer.Close()
    bridge_in.Close()
    bridge_out.Close()
    return
  }

  // Signal channels
  t.bridge_chan = make(chan []byte, 1)
  t.send_chan = make(chan *ZMQMessage, 2)
  t.recv_chan = make(chan interface{}, 1)
  t.recv_bridge_chan = make(chan interface{}, 1)
  t.can_send = make(chan int, 1)

  // Waiter we use to wait for shutdown
  t.wait.Add(2)

  // Bridge between channels and ZMQ
  go t.bridge(bridge_in)

  // The poller
  go t.poller(bridge_out)

  return nil
}

func (t *TransportZmq) bridge(bridge_in *zmq.Socket) {
  var message interface{}

  // Wait on channel, passing into socket
  // This keeps the socket in a single thread, otherwise we have to lock the entire publisher
  runtime.LockOSThread()

  for {
    select {
    case notify := <- t.bridge_chan:
      bridge_in.Send(notify, 0)

      // Shutdown?
      if string(notify) == zmq_signal_shutdown {
        break
      }
    case message = <-t.recv_bridge_chan:
    case func () chan<- interface{} {
      if message != nil {
        return t.recv_chan
      }
      return nil
    }() <- message:
      // The reason we flush recv through the bridge and not directly to recv_chan is so that if
      // the poller was quick and had to cache a receive as the channel was full, it will stop
      // polling - flushing through bridge allows us to signal poller to start polling again
      // It is not the publisher's responsibility to do this, and TLS wouldn't need it
      bridge_in.Send([]byte(zmq_signal_input), 0)
      message = nil
    }
  }

  // We should linger by default to ensure shutdown is transmitted
  bridge_in.Close()
  runtime.UnlockOSThread()
  t.wait.Done()
}

func (t *TransportZmq) poller(bridge_out *zmq.Socket) {
  var pollitems []zmq.PollItem
  var send_stage *ZMQMessage
  var recv_stage [][]byte
  var recv_body bool

  // ZMQ sockets are not thread-safe, so we have to send/receive on same thread
  // Thus, we cannot use a sender/receiver thread pair like we can with TLS so we use a single threaded poller instead
  // In order to asynchronously send and receive we just poll and do necessary actions

  // When data is ready to send we'll get a channel ping, that is bridged to ZMQ so we can then send data
  // For receiving, we receive here and bridge it to the channels, then receive more once that's through
  runtime.LockOSThread()

  pollitems = make([]zmq.PollItem, 2)
  pollitems[0].Socket = bridge_out
  pollitems[0].Events = zmq.POLLIN | zmq.POLLOUT
  pollitems[1].Socket = t.dealer
  pollitems[1].Events = zmq.POLLIN | zmq.POLLOUT

PollLoop:
  for {
    // Poll for events
    if _, err := zmq.Poll(pollitems, -1); err != nil {
      // Retry on EINTR
      if err == syscall.EINTR {
        continue
      }

      // Failure
      t.recv_chan <- errors.New(fmt.Sprintf("zmq.Poll failure %s", err))
      break
    }

    // Process control channel
    if pollitems[0].REvents & zmq.POLLIN != 0 {
    RetryControl:
      msg, err := bridge_out.Recv(zmq.DONTWAIT)
      if err != nil {
        switch err {
        case syscall.EINTR:
          // Try again
          goto RetryControl
        case syscall.EAGAIN:
          // Poll lied, poll again
          continue
        }

        // Failure
        t.recv_chan <- errors.New(fmt.Sprintf("Pull zmq.Socket.Recv failure %s", err))
        break
      }

      switch string(msg) {
      case zmq_signal_output:
        // Start polling for send
        pollitems[1].Events = pollitems[1].Events | zmq.POLLOUT
      case zmq_signal_input:
        // If we staged a receive, process that
        if recv_stage != nil {
          select {
          case t.recv_bridge_chan <- recv_stage:
            recv_stage = nil

            // Start polling for receive
            pollitems[1].Events = pollitems[1].Events | zmq.POLLIN
          default:
            // Do nothing, we were asked for receive but channel is already full
          }
        } else {
          // Start polling for receive
          pollitems[1].Events = pollitems[1].Events | zmq.POLLIN
        }
      case zmq_signal_shutdown:
        // Shutdown
        break PollLoop
      }
    }

    // Process dealer send
    if pollitems[1].REvents & zmq.POLLOUT != 0 {
      sent_one := false

      // Something in the staging buffer?
      if send_stage != nil {
        var err error
      RetrySendStage:
        if send_stage.final {
          err = t.dealer.Send(send_stage.part, zmq.DONTWAIT)
        } else {
          err = t.dealer.Send(send_stage.part, zmq.DONTWAIT | zmq.SNDMORE)
        }
        if err != nil {
          switch err {
          case syscall.EINTR:
            // Try again
            goto RetrySendStage
          case syscall.EAGAIN:
            // Poll lied, poll again
            continue
          }

          // Failure
          t.recv_chan <- errors.New(fmt.Sprintf("Dealer zmq.Socket.Send failure %s", err))
          break
        }

        sent_one = true
      }

      // Send messages from channel
    LoopSend:
      for {
        select {
        case msg := <-t.send_chan:
          var err error
        RetrySend:
          if msg.final {
            err = t.dealer.Send(msg.part, zmq.DONTWAIT)
          } else {
            err = t.dealer.Send(msg.part, zmq.DONTWAIT | zmq.SNDMORE)
          }
          if err != nil {
            switch err {
            case syscall.EINTR:
              // Try again
              goto RetrySend
            case syscall.EAGAIN:
              // Poll lied, poll again after we check others
              send_stage = msg
              goto PollRecv
            }

            // Failure
            t.recv_chan <- errors.New(fmt.Sprintf("Dealer zmq.Socket.Send failure %s", err))
            break PollLoop
          }

          sent_one = true
        default:
          break LoopSend
        }
      }

      if sent_one {
        // We just sent something, check POLLOUT still active before signalling we can send more
        // TODO: Check why Events() is returning uint64 instead of PollEvents
        // TODO: This is broken and actually returns an error
        if events, _ := t.dealer.Events(); zmq.PollEvents(events) & zmq.POLLOUT != 0 {
          t.setChan(t.can_send)
        }
      } else {
        t.setChan(t.can_send)
      }

      pollitems[1].Events = pollitems[1].Events ^ zmq.POLLOUT
    }

    // Process dealer receive
  PollRecv:
    if pollitems[1].REvents & zmq.POLLIN != 0 {
    LoopRecv:
      for {
        // Bring in the messages
      RetryRecv:
        data, err := t.dealer.Recv(zmq.DONTWAIT)
        if err != nil {
          switch err {
          case syscall.EINTR:
            // Try again
            goto RetryRecv
          case syscall.EAGAIN:
            // Poll lied, poll again
            continue PollLoop
          }

          // Failure
          t.recv_chan <- errors.New(fmt.Sprintf("Dealer zmq.Socket.Recv failure %s", err))
          break PollLoop
        }

        more, err := t.dealer.RcvMore()
        if err != nil {
          // Failure
          t.recv_chan <- errors.New(fmt.Sprintf("Dealer zmq.Socket.RcvMore failure %s", err))
          break PollLoop
        }

        // Sanity check, and don't save until empty message
        if len(data) == 0 && more {
          // Message separator, start returning
          recv_body = true
          continue
        } else if more {
          // Ignore all but last message
        } else if recv_body {
          recv_body = false

          // Last message and receiving, validate it first
          if len(data) < 8 {
            log.Printf("Skipping invalid message: not enough data")
          } else {
            length := binary.BigEndian.Uint32(data[4:8])
            if length > 1048576 {
              log.Printf("Skipping invalid message: data too large (%d)", length)
            } else if length != uint32(len(data)) - 8 {
              log.Printf("Skipping invalid message: data has invalid length (%d != %d)", len(data) - 8, length)
            } else {
              message := [][]byte{data[0:4], data[8:]}

              // Bridge to channels
              select {
              case t.recv_bridge_chan <- message:
              default:
                // We filled the channel, stop polling until we pull something off of it and stage the recv
                recv_stage = message
                pollitems[1].Events = pollitems[1].Events ^ zmq.POLLIN
                break LoopRecv
              }
            }
          }
        } // Hmmm, discard anything else
      }
    }
  }
  // TODO: Can we tidy the nesting above? :/

  bridge_out.Close()
  runtime.UnlockOSThread()
  t.wait.Done()
}

func (t *TransportZmq) setChan(set chan int) {
  select {
  case set <- 1:
  default:
  }
}

func (t *TransportZmq) CanSend() <-chan int {
  return t.can_send
}

func (t *TransportZmq) Write(signature string, message []byte) (err error) {
  var write_buffer *bytes.Buffer
  write_buffer = bytes.NewBuffer(make([]byte, 0, len(signature) + 4 + len(message)))

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

  t.send_chan <- &ZMQMessage{part: []byte(""), final: false}
  t.send_chan <- &ZMQMessage{part: write_buffer.Bytes(), final: true}

  // Ask for send to start
  t.bridge_chan <- []byte(zmq_signal_output)
  return nil
}

func (t *TransportZmq) Read() <-chan interface{} {
  return t.recv_chan
}

func (t *TransportZmq) Disconnect() {
  // Send shutdown request
  t.bridge_chan <- []byte(zmq_signal_shutdown)
  t.wait.Wait()
  t.dealer.Close()
}
