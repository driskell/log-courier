package main

import (
  "bytes"
  "errors"
  "fmt"
  "log"
  "net"
  "regexp"
  "runtime"
  "sync"
  zmq "github.com/alecthomas/gozmq"
)

type TransportZmq struct {
  config  *NetworkConfig
  context *zmq.Context
  dealer  *zmq.Socket

  write_buffer *bytes.Buffer

  wait sync.WaitGroup

  notify_chan chan []byte

  send_chan chan *ZMQMessage
  recv_chan chan interface{}

  can_send chan int
  can_recv chan int
}

type ZMQMessage struct {
  part  []byte
  final bool
}

const (
  zmq_signal_output   = "O"
  zmq_signal_input    = "I"
  zmq_signal_shutdown = "S"
)

var TransportZmq_Context *zmq.Context

func CreateTransportZmq(config *NetworkConfig) (*TransportZmq, error) {
  if TransportZmq_Context == nil {
    var err error
    TransportZmq_Context, err = zmq.NewContext()
    if err != nil {
      return nil, errors.New(fmt.Sprintf("Failed to create ZMQ context: %s", err))
    }
  }
  return &TransportZmq{config: config, context: TransportZmq_Context}, nil
}

func (t *TransportZmq) Connect() error {
  t.write_buffer = new(bytes.Buffer)

  hostport_re := regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`)
  endpoints := 0

  // TODO: check error return
  t.dealer, _ = t.context.NewSocket(zmq.DEALER)

  // Control sockets that poller will wait on
  // TODO: check error return
  notify_in, _ := t.context.NewSocket(zmq.REQ)
  notify_out, _ := t.context.NewSocket(zmq.REP)

  // Connect endpoints
  for _, hostport := range t.config.Servers {
    submatch := hostport_re.FindSubmatch([]byte(hostport))
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

      log.Printf("Connecting to %s (%s) \n", addressport, host)
      t.dealer.Connect("tcp://" + addressport)
    }
  }

  if endpoints == 0 {
    return errors.New("Failed to connect to any of the specified endpoints.")
  }

  notify_in.Bind("inproc://notify")
  notify_out.Connect("inproc://notify")

  // Signal channels
  t.notify_chan = make(chan []byte, 1)
  t.send_chan = make(chan *ZMQMessage, 2)
  t.recv_chan = make(chan interface{}, 2)
  t.can_send = make(chan int, 1)
  t.can_recv = make(chan int, 1)

  // Waiter we use to wait for shutdown
  t.wait.Add(2)

  // Translator for turning chan receive into a ZeroMQ poll interrupt
  go t.translateChan(notify_in)

  // The poller
  go t.poller(notify_out)

  return nil
}

func (t *TransportZmq) translateChan(notify_in *zmq.Socket) {
  // Wait on channel, passing into socket
  // This keeps the socket in a single thread, otherwise we have to lock the entire publisher
  runtime.LockOSThread()

  for {
    notify := <- t.notify_chan

    notify_in.Send(notify, 0)

    // Shutdown?
    if string(notify) == zmq_signal_shutdown {
      break
    }
  }

  // We should linger by default to ensure shutdown is transmitted
  notify_in.Close()
  runtime.UnlockOSThread()
  t.wait.Done()
}

func (t *TransportZmq) poller(notify_out *zmq.Socket) {
  var pollitems []zmq.PollItem
  var send_stage *ZMQMessage
  var recv_stage []byte
  var recv_body bool

  runtime.LockOSThread()

  pollitems = make([]zmq.PollItem, 2)
  pollitems[0].Socket = notify_out
  pollitems[0].Events = zmq.POLLIN | zmq.POLLOUT
  pollitems[1].Socket = t.dealer

PollLoop:
  for {
    // Poll for events
    // TODO: check return code
    zmq.Poll(pollitems, -1)

    // Process control channel
    if pollitems[0].REvents & zmq.POLLIN != 0 {
      // TODO: check return code / try again or fail
      msg, _ := notify_out.Recv(0)

      switch string(msg) {
      case zmq_signal_output:
        // Start polling for send
        pollitems[1].Events = pollitems[1].Events | zmq.POLLOUT
      case zmq_signal_input:
        // If we staged a receive, process that
        if recv_stage != nil {
          select {
          case t.recv_chan <- recv_stage:
            recv_stage = nil
            // Start polling for receive
            pollitems[1].Events = pollitems[1].Events | zmq.POLLIN
          default:
            // Do nothing, we were asked for receive but channel is already full
          }
        }
      case zmq_signal_shutdown:
        // Shutdown
        break PollLoop
      }
    } /* if notify POLLING */

    // Process dealer send
  PollOut:
    for pollitems[1].Events & zmq.POLLOUT != 0 {
      sent_one := false

      // Something in the staging buffer?
      if send_stage != nil {
        var err error
        if send_stage.final {
          err = t.dealer.Send(send_stage.part, zmq.DONTWAIT)
        } else {
          err = t.dealer.Send(send_stage.part, zmq.DONTWAIT | zmq.SNDMORE)
        }
        // If send failed, skip and try again later, means poll lied
        if err != nil {
          // TODO: report an error? fail completely?
          break PollOut
        }
        sent_one = true
      }

      // Send messages from channel
      for {
        select {
        case msg := <-t.send_chan:
          var err error
          if msg.final {
            err = t.dealer.Send(msg.part, zmq.DONTWAIT)
          } else {
            err = t.dealer.Send(msg.part, zmq.DONTWAIT | zmq.SNDMORE)
          }
          // If send failed, buffer
          if err != nil {
            // TODO: report an error? fail completely if consistently happening?
            send_stage = msg
            break PollOut
          }
          sent_one = true
        default:
          break
        }
      }

      if sent_one {
        // We just sent something, check POLLOUT still active before signalling we can send more
        // TODO: Check why Events() is returning uint64 instead of PollEvents
        if events, _ := t.dealer.Events(); zmq.PollEvents(events) & zmq.POLLOUT != 0 {
          t.setChan(t.can_send)
        }
      } else {
        t.setChan(t.can_send)
      }

      pollitems[1].Events = pollitems[1].Events ^ zmq.POLLOUT
      break
    } /* for dealer POLLOUT */

    // Process dealer receive
    if pollitems[1].Events & zmq.POLLIN != 0 {
      for {
        // Bring in the messages
        // TODO: check error return
        msg, _ := t.dealer.Recv(0)
        more, _ := t.dealer.RcvMore()
        // Sanity check, and don't save until empty message
        if string(msg) == "" && more {
          // Empty message, start returning
          recv_body = true
          continue
        } else if more {
          // Ignore all but last message
        } else if recv_body {
          // Last message and receiving, pass to channel
          select {
          case t.recv_chan <- msg:
            recv_body = false
            t.setChan(t.can_recv)
          default:
            // We filled the channel, stop polling until we pull something off of it and stage the recv
            recv_stage = msg
            recv_body = false
            pollitems[1].Events = pollitems[1].Events ^ zmq.POLLIN
            break
          }
        } else {
          // Hmmm, just discard
          recv_body = false
        }

        // Finished receiving?
        // TODO: Check why Events() is returning uint64 instead of PollEvents
        if events, _ := t.dealer.Events(); zmq.PollEvents(events) & zmq.POLLIN == 0 {
          break
        }
      } /* loop forever */
    } /* if dealer POLLIN */
  }

  notify_out.Close()
  runtime.UnlockOSThread()
  t.wait.Done()
}

func (t *TransportZmq) setChan(set chan int) {
  select {
  case set <- 1:
  default:
  }
}

func (t *TransportZmq) CanRecv() chan int {
  return t.can_recv
}

func (t *TransportZmq) CanSend() chan int {
  return t.can_send
}

func (t *TransportZmq) Write(p []byte) (int, error) {
  return t.write_buffer.Write(p)
}

func (t *TransportZmq) Flush() error {
  t.send_chan <- &ZMQMessage{part: []byte(""), final: false}
  t.send_chan <- &ZMQMessage{part: t.write_buffer.Bytes(), final: true}
  t.write_buffer.Reset()

  // Ask for send to start
  t.notify_chan <- []byte(zmq_signal_output)
  return nil
}

func (t *TransportZmq) Read() ([]byte, error) {
  var msg interface{}
  select {
  case msg = <-t.recv_chan:
    // Ask for more
    t.notify_chan <- []byte(zmq_signal_input)
  default:
    t.notify_chan <- []byte(zmq_signal_input)
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

func (t *TransportZmq) Disconnect() {
  // Send shutdown request
  t.notify_chan <- []byte(zmq_signal_shutdown)
  t.wait.Wait()
  t.dealer.Close()
  t.write_buffer.Reset()
}
