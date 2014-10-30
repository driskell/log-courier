// +build zmq

/*
 * Copyright 2014 Jason Woods.
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
  "encoding/binary"
  "errors"
  "fmt"
  zmq "github.com/alecthomas/gozmq"
  "lc-lib/core"
  "net"
  "regexp"
  "runtime"
  "sync"
  "syscall"
)

const (
  zmq_signal_output   = "O"
  zmq_signal_input    = "I"
  zmq_signal_shutdown = "S"
)

const (
  Monitor_Part_Header = iota
  Monitor_Part_Data
  Monitor_Part_Extraneous
)

type TransportZmqFactory struct {
  transport string

  CurveServerkey string `config:"curve server key"`
  CurvePublickey string `config:"curve public key"`
  CurveSecretkey string `config:"curve secret key"`

  hostport_re *regexp.Regexp
}

type TransportZmq struct {
  config     *TransportZmqFactory
  net_config *core.NetworkConfig
  context    *zmq.Context
  dealer     *zmq.Socket
  monitor    *zmq.Socket
  poll_items []zmq.PollItem
  send_buff  *ZMQMessage
  recv_buff  [][]byte
  recv_body  bool
  event      ZMQEvent
  ready      bool

  wait sync.WaitGroup

  bridge_chan chan []byte

  send_chan        chan *ZMQMessage
  recv_chan        chan interface{}
  recv_bridge_chan chan interface{}

  can_send chan int
}

type ZMQMessage struct {
  part  []byte
  final bool
}

type ZMQEvent struct {
  part  int
  event zmq.Event
  val   int32
  data  string
}

func (e *ZMQEvent) Log() {
  switch e.event {
  case zmq.EVENT_CONNECTED:
    if e.data == "" {
      log.Info("Connected")
    } else {
      log.Info("Connected to %s", e.data)
    }
  case zmq.EVENT_CONNECT_DELAYED:
    // Don't log anything for this
  case zmq.EVENT_CONNECT_RETRIED:
    if e.data == "" {
      log.Info("Attempting to connect")
    } else {
      log.Info("Attempting to connect to %s", e.data)
    }
  case zmq.EVENT_CLOSED:
    if e.data == "" {
      log.Error("Connection closed")
    } else {
      log.Error("Connection to %s closed", e.data)
    }
  case zmq.EVENT_DISCONNECTED:
    if e.data == "" {
      log.Error("Lost connection")
    } else {
      log.Error("Lost connection to %s", e.data)
    }
  default:
    log.Debug("Unknown monitor message (event:%d, val:%d, data:[% X])", e.event, e.val, e.data)
  }
}

type TransportZmqRegistrar struct {
}

func NewZmqTransportFactory(config *core.Config, config_path string, unused map[string]interface{}, name string) (core.TransportFactory, error) {
  var err error

  ret := &TransportZmqFactory{
    transport:   name,
    hostport_re: regexp.MustCompile(`^\[?([^]]+)\]?:([0-9]+)$`),
  }

  if name == "zmq" {
    if err = config.PopulateConfig(ret, config_path, unused); err != nil {
      return nil, err
    }

    if err := ret.processConfig(config_path); err != nil {
      return nil, err
    }
  } else {
    if err := config.ReportUnusedConfig(config_path, unused); err != nil {
      return nil, err
    }
  }

  return ret, nil
}

func (f *TransportZmqFactory) NewTransport(config *core.NetworkConfig) (core.Transport, error) {
  return &TransportZmq{config: f, net_config: config}, nil
}

func (t *TransportZmq) ReloadConfig(new_net_config *core.NetworkConfig) int {
  // Check we can grab new ZMQ config to compare, if not force transport reinit
  new_config, ok := new_net_config.TransportFactory.(*TransportZmqFactory)
  if !ok {
    return core.Reload_Transport
  }

  if new_config.CurveServerkey != t.config.CurveServerkey || new_config.CurvePublickey != t.config.CurvePublickey || new_config.CurveSecretkey != t.config.CurveSecretkey {
    return core.Reload_Transport
  }

  // Publisher handles changes to net_config, but ensure we store the latest in case it asks for a reconnect
  t.net_config = new_net_config

  return core.Reload_None
}

func (t *TransportZmq) Init() (err error) {
  // Initialise once for ZMQ
  if t.ready {
    // If already initialised, ask if we can send again
    t.bridge_chan <- []byte(zmq_signal_output)
    return nil
  }

  t.context, err = zmq.NewContext()
  if err != nil {
    return fmt.Errorf("Failed to create ZMQ context: %s", err)
  }
  defer func() {
    if err != nil {
      t.context.Close()
    }
  }()

  // Control sockets to connect bridge to poller
  bridge_in, err := t.context.NewSocket(zmq.PUSH)
  if err != nil {
    return fmt.Errorf("Failed to create internal ZMQ PUSH socket: %s", err)
  }
  defer func() {
    if err != nil {
      bridge_in.Close()
    }
  }()

  if err = bridge_in.Bind("inproc://notify"); err != nil {
    return fmt.Errorf("Failed to bind internal ZMQ PUSH socket: %s", err)
  }

  bridge_out, err := t.context.NewSocket(zmq.PULL)
  if err != nil {
    return fmt.Errorf("Failed to create internal ZMQ PULL socket: %s", err)
  }
  defer func() {
    if err != nil {
      bridge_out.Close()
    }
  }()

  if err = bridge_out.Connect("inproc://notify"); err != nil {
    return fmt.Errorf("Failed to connect internal ZMQ PULL socket: %s", err)
  }

  // Outbound dealer socket will fair-queue load balance amongst peers
  if t.dealer, err = t.context.NewSocket(zmq.DEALER); err != nil {
    return fmt.Errorf("Failed to create ZMQ DEALER socket: %s", err)
  }
  defer func() {
    if err != nil {
      t.dealer.Close()
    }
  }()

  if err = t.dealer.Monitor("inproc://monitor", zmq.EVENT_ALL); err != nil {
    return fmt.Errorf("Failed to bind DEALER socket to monitor: %s", err)
  }

  if err = t.configureSocket(); err != nil {
    return fmt.Errorf("Failed to configure DEALER socket: %s", err)
  }

  // Configure reconnect interval
  if err = t.dealer.SetReconnectIvlMax(t.net_config.Reconnect); err != nil {
    return fmt.Errorf("Failed to set ZMQ reconnect interval: %s", err)
  }

  // We should not LINGER. If we do, socket Close and also context Close will
  // block infinitely until the message queue is flushed. Set to 0 to discard
  // all messages immediately when we call Close
  if err = t.dealer.SetLinger(0); err != nil {
    return fmt.Errorf("Failed to set ZMQ linger period: %s", err)
  }

  // Monitor socket
  if t.monitor, err = t.context.NewSocket(zmq.PULL); err != nil {
    return fmt.Errorf("Failed to create monitor ZMQ PULL socket: %s", err)
  }
  defer func() {
    if err != nil {
      t.monitor.Close()
    }
  }()

  if err = t.monitor.Connect("inproc://monitor"); err != nil {
    return fmt.Errorf("Failed to connect monitor ZMQ PULL socket: %s", err)
  }

  // Register endpoints
  endpoints := 0
  for _, hostport := range t.net_config.Servers {
    submatch := t.config.hostport_re.FindSubmatch([]byte(hostport))
    if submatch == nil {
      log.Warning("Invalid host:port given: %s", hostport)
      continue
    }

    // Lookup the server in DNS (if this is IP it will implicitly return)
    host := string(submatch[1])
    port := string(submatch[2])
    addresses, err := net.LookupHost(host)
    if err != nil {
      log.Warning("DNS lookup failure \"%s\": %s", host, err)
      continue
    }

    // Register each address
    for _, address := range addresses {
      addressport := net.JoinHostPort(address, port)

      if err = t.dealer.Connect("tcp://" + addressport); err != nil {
        log.Warning("Failed to register %s (%s) with ZMQ, skipping", addressport, host)
        continue
      }

      log.Info("Registered %s (%s) with ZMQ", addressport, host)
      endpoints++
    }
  }

  if endpoints == 0 {
    return errors.New("Failed to register any of the specified endpoints.")
  }

  major, minor, patch := zmq.Version()
  log.Info("libzmq version %d.%d.%d", major, minor, patch)

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

  t.ready = true
  t.send_buff = nil
  t.recv_buff = nil
  t.recv_body = false

  return nil
}

func (t *TransportZmq) bridge(bridge_in *zmq.Socket) {
  var message interface{}

  // Wait on channel, passing into socket
  // This keeps the socket in a single thread, otherwise we have to lock the entire publisher
  runtime.LockOSThread()

BridgeLoop:
  for {
    select {
    case notify := <-t.bridge_chan:
      bridge_in.Send(notify, 0)

      // Shutdown?
      if string(notify) == zmq_signal_shutdown {
        break BridgeLoop
      }
    case message = <-t.recv_bridge_chan:
    case func() chan<- interface{} {
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
  // ZMQ sockets are not thread-safe, so we have to send/receive on same thread
  // Thus, we cannot use a sender/receiver thread pair like we can with TLS so we use a single threaded poller instead
  // In order to asynchronously send and receive we just poll and do necessary actions

  // When data is ready to send we'll get a channel ping, that is bridged to ZMQ so we can then send data
  // For receiving, we receive here and bridge it to the channels, then receive more once that's through
  runtime.LockOSThread()

  t.poll_items = make([]zmq.PollItem, 3)

  // Listen always on bridge
  t.poll_items[0].Socket = bridge_out
  t.poll_items[0].Events = zmq.POLLIN | zmq.POLLOUT

  // Always check for input on dealer - but also initially check for OUT so we can flag send is ready
  t.poll_items[1].Socket = t.dealer
  t.poll_items[1].Events = zmq.POLLIN | zmq.POLLOUT

  // Always listen for input on monitor
  t.poll_items[2].Socket = t.monitor
  t.poll_items[2].Events = zmq.POLLIN

  for {
    // Poll for events
    if _, err := zmq.Poll(t.poll_items, -1); err != nil {
      // Retry on EINTR
      if err == syscall.EINTR {
        continue
      }

      // Failure
      t.recv_chan <- fmt.Errorf("zmq.Poll failure %s", err)
      break
    }

    // Process control channel
    if t.poll_items[0].REvents&zmq.POLLIN != 0 {
      if !t.processControlIn(bridge_out) {
        break
      }
    }

    // Process dealer send
    if t.poll_items[1].REvents&zmq.POLLOUT != 0 {
      if !t.processDealerOut() {
        break
      }
    }

    // Process dealer receive
    if t.poll_items[1].REvents&zmq.POLLIN != 0 {
      if !t.processDealerIn() {
        break
      }
    }

    // Process monitor receive
    if t.poll_items[2].REvents&zmq.POLLIN != 0 {
      if !t.processMonitorIn() {
        break
      }
    }
  }

  bridge_out.Close()
  runtime.UnlockOSThread()
  t.wait.Done()
}

func (t *TransportZmq) processControlIn(bridge_out *zmq.Socket) (ok bool) {
  var err error

RetryControl:
  msg, err := bridge_out.Recv(zmq.DONTWAIT)
  if err != nil {
    switch err {
    case syscall.EINTR:
      // Try again
      goto RetryControl
    case syscall.EAGAIN:
      // No more messages
      return true
    }

    // Failure
    t.recv_chan <- fmt.Errorf("Pull zmq.Socket.Recv failure %s", err)
    return
  }

  switch string(msg) {
  case zmq_signal_output:
    // Start polling for send
    t.poll_items[1].Events = t.poll_items[1].Events | zmq.POLLOUT
  case zmq_signal_input:
    // If we staged a receive, process that
    if t.recv_buff != nil {
      select {
      case t.recv_bridge_chan <- t.recv_buff:
        t.recv_buff = nil

        // Start polling for receive
        t.poll_items[1].Events = t.poll_items[1].Events | zmq.POLLIN
      default:
        // Do nothing, we were asked for receive but channel is already full
      }
    } else {
      // Start polling for receive
      t.poll_items[1].Events = t.poll_items[1].Events | zmq.POLLIN
    }
  case zmq_signal_shutdown:
    // Shutdown
    return
  }

  ok = true
  return
}

func (t *TransportZmq) processDealerOut() (ok bool) {
  var sent_one bool

  // Something in the staging buffer?
  if t.send_buff != nil {
    sent, s_ok := t.dealerSend(t.send_buff)
    if !s_ok {
      return
    }
    if !sent {
      ok = true
      return
    }

    t.send_buff = nil
    sent_one = true
  }

  // Send messages from channel
LoopSend:
  for {
    select {
    case msg := <-t.send_chan:
      sent, s_ok := t.dealerSend(msg)
      if !s_ok {
        return
      }
      if !sent {
        t.send_buff = msg
        break
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
    /*if events, _ := t.dealer.Events(); zmq.PollEvents(events)&zmq.POLLOUT != 0 {
      t.poll_items[1].Events = t.poll_items[1].Events ^ zmq.POLLOUT
      t.setChan(t.can_send)
    }*/
  } else {
    t.poll_items[1].Events = t.poll_items[1].Events ^ zmq.POLLOUT
    t.setChan(t.can_send)
  }

  ok = true
  return
}

func (t *TransportZmq) dealerSend(msg *ZMQMessage) (sent bool, ok bool) {
  var err error

RetrySend:
  if msg.final {
    err = t.dealer.Send(msg.part, zmq.DONTWAIT)
  } else {
    err = t.dealer.Send(msg.part, zmq.DONTWAIT|zmq.SNDMORE)
  }
  if err != nil {
    switch err {
    case syscall.EINTR:
      // Try again
      goto RetrySend
    case syscall.EAGAIN:
      // No more messages
      ok = true
      return
    }

    // Failure
    t.recv_chan <- fmt.Errorf("Dealer zmq.Socket.Send failure %s", err)
    return
  }

  sent = true
  ok = true
  return
}

func (t *TransportZmq) processDealerIn() (ok bool) {
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
        // No more messages
        ok = true
        return
      }

      // Failure
      t.recv_chan <- fmt.Errorf("Dealer zmq.Socket.Recv failure %s", err)
      return
    }

    more, err := t.dealer.RcvMore()
    if err != nil {
      // Failure
      t.recv_chan <- fmt.Errorf("Dealer zmq.Socket.RcvMore failure %s", err)
      return
    }

    // Sanity check, and don't save until empty message
    if len(data) == 0 && more {
      // Message separator, start returning
      t.recv_body = true
      continue
    } else if more || !t.recv_body {
      // Ignore all but last message
      continue
    }

    t.recv_body = false

    // Last message and receiving, validate it first
    if len(data) < 8 {
      log.Warning("Skipping invalid message: not enough data")
      continue
    }

    length := binary.BigEndian.Uint32(data[4:8])
    if length > 1048576 {
      log.Warning("Skipping invalid message: data too large (%d)", length)
      continue
    } else if length != uint32(len(data))-8 {
      log.Warning("Skipping invalid message: data has invalid length (%d != %d)", len(data)-8, length)
      continue
    }

    message := [][]byte{data[0:4], data[8:]}

    // Bridge to channels
    select {
    case t.recv_bridge_chan <- message:
    default:
      // We filled the channel, stop polling until we pull something off of it and stage the recv
      t.recv_buff = message
      t.poll_items[1].Events = t.poll_items[1].Events ^ zmq.POLLIN
      ok = true
      return
    }
  }
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

  t.send_chan <- &ZMQMessage{part: []byte(""), final: false}
  t.send_chan <- &ZMQMessage{part: write_buffer.Bytes(), final: true}

  // Ask for send to start
  t.bridge_chan <- []byte(zmq_signal_output)
  return nil
}

func (t *TransportZmq) Read() <-chan interface{} {
  return t.recv_chan
}

func (t *TransportZmq) Shutdown() {
  if t.ready {
    // Send shutdown request
    t.bridge_chan <- []byte(zmq_signal_shutdown)
    t.wait.Wait()
    t.dealer.Close()
    t.monitor.Close()
    t.context.Close()
    t.ready = false
  }
}

// Register the transport
func init() {
  core.RegisterTransport("plainzmq", NewZmqTransportFactory)
}
