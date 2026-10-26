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

// testReceiver records connections the pool closes and reports itself finished when shut down
type testReceiver struct {
	eventChan chan<- transports.Event
	closes    atomic.Int64
}

func (r *testReceiver) Acknowledge(context.Context, *string, uint32) error { return nil }
func (r *testReceiver) Pong(context.Context) error                         { return nil }
func (r *testReceiver) FailConnection(context.Context, error)              { r.closes.Add(1) }
func (r *testReceiver) ShutdownConnection(context.Context)                 {}
func (r *testReceiver) ShutdownConnectionRead(context.Context, error)      { r.closes.Add(1) }
func (r *testReceiver) Factory() transports.ReceiverFactory                { return nil }
func (r *testReceiver) SupportsAck() bool                                  { return true }

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
	receiver := &testReceiver{eventChan: eventChan}
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
	return transports.NewEventsEvent(ctx, &nonce, []map[string]interface{}{{"message": "test"}}, size)
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
