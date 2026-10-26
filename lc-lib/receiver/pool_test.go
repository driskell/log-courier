package receiver

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/scheduler"
	"github.com/driskell/log-courier/lc-lib/transports"
)

// testAck is an acknowledgement the pool sent to a client
type testAck struct {
	nonce    string
	sequence uint32
}

// testReceiver records acknowledgements and connections the pool closes, and reports itself finished when shut down
type testReceiver struct {
	eventChan chan<- transports.Event
	acks      chan testAck
	closes    atomic.Int64
	shutdowns atomic.Int64
}

func (r *testReceiver) Acknowledge(_ context.Context, nonce *string, sequence uint32) error {
	r.acks <- testAck{nonce: *nonce, sequence: sequence}
	return nil
}

func (r *testReceiver) Pong(context.Context) error                    { return nil }
func (r *testReceiver) FailConnection(context.Context, error)         { r.closes.Add(1) }
func (r *testReceiver) ShutdownConnection(context.Context)            { r.shutdowns.Add(1) }
func (r *testReceiver) ShutdownConnectionRead(context.Context, error) { r.closes.Add(1) }
func (r *testReceiver) Factory() transports.ReceiverFactory           { return nil }
func (r *testReceiver) SupportsAck() bool                             { return true }

func (r *testReceiver) Shutdown() {
	ctx := context.WithValue(context.Background(), transports.ContextReceiver, transports.Receiver(r))
	go func() {
		r.eventChan <- transports.NewStatusEvent(ctx, transports.Finished, nil)
	}()
}

type testConnection struct{}

func newTestPool(maxQueueSize int64) (*Pool, *testReceiver, chan []*event.Event) {
	output := make(chan []*event.Event)
	eventChan := make(chan transports.Event)
	receiver := &testReceiver{eventChan: eventChan, acks: make(chan testAck, 64)}
	status := &poolReceiverStatus{config: &transports.ReceiverConfigEntry{Name: "test", MaxPendingPayloads: 10}, listen: "test", active: true}
	pool := &Pool{
		output:            output,
		shutdownChan:      make(chan struct{}),
		configChan:        make(chan *config.Config),
		ackChan:           make(chan []*event.Event),
		eventChan:         eventChan,
		receivers:         map[transports.Receiver]*poolReceiverStatus{receiver: status},
		receiversByListen: map[string]transports.Receiver{"test": receiver},
		scheduler:         scheduler.NewScheduler(),
		connectionStatus:  make(map[interface{}]*poolConnectionStatus),
		generalConfig:     &config.General{MaxQueueSize: maxQueueSize},
	}
	return pool, receiver, output
}

func newTestPayload(ctx context.Context, nonce string, size int) transports.EventsEvent {
	return newTestPayloadOf(ctx, nonce, 1, size)
}

func newTestPayloadOf(ctx context.Context, nonce string, count int, size int) transports.EventsEvent {
	events := make([]map[string]interface{}, count)
	for idx := range events {
		events[idx] = map[string]interface{}{"message": "test"}
	}
	return transports.NewEventsEvent(ctx, &nonce, events, size)
}

// startTestPool runs a pool with one connected client, returning a function that disconnects it and shuts the pool down
func startTestPool(t *testing.T) (*Pool, *testReceiver, <-chan []*event.Event, context.Context, func()) {
	t.Helper()
	pool, receiver, output := newTestPool(1 << 20)
	shutdownChan := make(chan struct{})
	pool.shutdownChan = shutdownChan
	done := make(chan struct{})
	go func() {
		pool.Run()
		close(done)
	}()

	ctx := context.WithValue(context.WithValue(context.Background(), transports.ContextConnection, &testConnection{}), transports.ContextReceiver, transports.Receiver(receiver))
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the connection")
	return pool, receiver, output, ctx, func() {
		sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test"), "Pool did not accept the disconnection")
		close(shutdownChan)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Pool did not shut down")
		}
	}
}

func receiveTestEvents(t *testing.T, output <-chan []*event.Event) []*event.Event {
	t.Helper()
	select {
	case events := <-output:
		return events
	case <-time.After(5 * time.Second):
		t.Fatal("Pool did not output its queued payload")
	}
	return nil
}

func acknowledgeTestEvents(t *testing.T, pool *Pool, events []*event.Event) {
	t.Helper()
	select {
	case pool.ackChan <- events:
	case <-time.After(5 * time.Second):
		t.Fatal("Pool did not receive the acknowledgement")
	}
}

func expectTestAck(t *testing.T, receiver *testReceiver, expected testAck) {
	t.Helper()
	select {
	case ack := <-receiver.acks:
		if ack != expected {
			t.Fatalf("Expected acknowledgement %+v, got %+v", expected, ack)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Pool did not send acknowledgement %+v", expected)
	}
}

func expectNoTestAck(t *testing.T, receiver *testReceiver) {
	t.Helper()
	select {
	case ack := <-receiver.acks:
		t.Fatalf("Pool sent unexpected acknowledgement %+v", ack)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPoolAcknowledgesPayloadsInAnyOrder(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "first", 4, 4), "Pool did not receive the first payload")
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "second", 4, 4), "Pool did not receive the second payload")
	first := receiveTestEvents(t, output)
	second := receiveTestEvents(t, output)

	acknowledgeTestEvents(t, pool, second)
	expectTestAck(t, receiver, testAck{nonce: "second", sequence: 4})
	acknowledgeTestEvents(t, pool, first)
	expectTestAck(t, receiver, testAck{nonce: "first", sequence: 4})
	stop()
}

func TestPoolHoldsAcknowledgementUntilEarlierEventsAcknowledged(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "payload", 6, 6), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)

	acknowledgeTestEvents(t, pool, events[3:])
	expectNoTestAck(t, receiver)
	acknowledgeTestEvents(t, pool, events[:3])
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 6})
	stop()
}

func TestPoolAcknowledgesPartialPrefix(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "payload", 6, 6), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)

	acknowledgeTestEvents(t, pool, events[:2])
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 2})
	acknowledgeTestEvents(t, pool, events[:2])
	expectNoTestAck(t, receiver)
	acknowledgeTestEvents(t, pool, events[2:])
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 6})
	stop()
}

func TestPoolClosesFinishedConnectionOnceAcknowledged(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "first", 2, 2), "Pool did not receive the first payload")
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "second", 2, 2), "Pool did not receive the second payload")
	first := receiveTestEvents(t, output)
	second := receiveTestEvents(t, output)
	sendTestEvent(t, pool, transports.NewEndEvent(ctx), "Pool did not receive the end of the connection")

	acknowledgeTestEvents(t, pool, second)
	expectTestAck(t, receiver, testAck{nonce: "second", sequence: 2})
	if shutdowns := receiver.shutdowns.Load(); shutdowns != 0 {
		t.Fatal("Pool shut down the connection with a payload still unacknowledged")
	}
	acknowledgeTestEvents(t, pool, first)
	expectTestAck(t, receiver, testAck{nonce: "first", sequence: 2})
	if shutdowns := receiver.shutdowns.Load(); shutdowns != 1 {
		t.Fatalf("Pool shut down the connection %d times once fully acknowledged, expected once", shutdowns)
	}
	stop()
}

func TestPoolIgnoresAcknowledgementAfterDisconnect(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, newTestPayloadOf(ctx, "payload", 2, 2), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test"), "Pool did not accept the disconnection")

	acknowledgeTestEvents(t, pool, events)
	expectNoTestAck(t, receiver)
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the reconnection")
	stop()
}

func sendTestEvent(t *testing.T, pool *Pool, evnt transports.Event, message string) {
	t.Helper()
	select {
	case pool.eventChan <- evnt:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func receiveTestOutput(t *testing.T, output <-chan []*event.Event) {
	t.Helper()
	select {
	case <-output:
	case <-time.After(5 * time.Second):
		t.Fatal("Pool did not output its queued payload")
	}
}

func TestPoolWaitsWhenQueueFull(t *testing.T) {
	pool, receiver, output := newTestPool(100)
	shutdownChan := make(chan struct{})
	pool.shutdownChan = shutdownChan
	done := make(chan struct{})
	go func() {
		pool.Run()
		close(done)
	}()

	ctx := context.WithValue(context.WithValue(context.Background(), transports.ContextConnection, &testConnection{}), transports.ContextReceiver, transports.Receiver(receiver))
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the connection")
	sendTestEvent(t, pool, newTestPayload(ctx, "first", 100), "Pool did not receive a payload while its queue had space")

	select {
	case pool.eventChan <- newTestPayload(ctx, "second", 100):
		t.Fatal("Pool received a payload while its queue was full")
	case <-time.After(100 * time.Millisecond):
	}

	receiveTestOutput(t, output)
	sendTestEvent(t, pool, newTestPayload(ctx, "second", 100), "Pool did not resume receiving after its queue drained")

	if closes := receiver.closes.Load(); closes != 0 {
		t.Fatalf("Pool closed %d connections instead of waiting", closes)
	}

	receiveTestOutput(t, output)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test"), "Pool did not accept the disconnection")
	close(shutdownChan)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Pool did not shut down")
	}
}
