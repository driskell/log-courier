package receiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// testConnection identifies a connection, with a field so each instance has a distinct address
type testConnection struct {
	name string
}

func newTestPool(maxQueueSize int64) (*Pool, *testReceiver, chan []*event.Event) {
	output := make(chan []*event.Event)
	eventChan := make(chan transports.Event)
	receiver := &testReceiver{eventChan: eventChan, acks: make(chan testAck, 64)}
	status := &poolReceiverStatus{config: &transports.ReceiverConfigEntry{Name: "test"}, listen: "test", active: true}
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
		done:              make(chan struct{}),
		closedRetention:   closedConnectionRetention,
		queue:             newFairQueue(),
		admission:         newAdmissionControl(),
	}
	return pool, receiver, output
}

// testPayload is a payload of events a test connection sends
type testPayload struct {
	ctx   context.Context
	nonce string
	count int
	size  int
}

func newTestPayload(ctx context.Context, nonce string, size int) *testPayload {
	return newTestPayloadOf(ctx, nonce, 1, size)
}

func newTestPayloadOf(ctx context.Context, nonce string, count int, size int) *testPayload {
	return &testPayload{ctx: ctx, nonce: nonce, count: count, size: size}
}

// hold adds the payload's events to a receive window, sharing its size between them, and optionally ends it
func (p *testPayload) hold(window *transports.ReceiveWindow, from int, to int, end bool) {
	for idx := from; idx < to; idx++ {
		size := p.size / p.count
		if idx == p.count-1 {
			size = p.size - size*(p.count-1)
		}
		window.Add(&p.nonce, map[string]interface{}{"message": "test"}, size)
	}
	if end {
		window.End(&p.nonce)
	}
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

	ctx := context.WithValue(context.WithValue(context.Background(), transports.ContextConnection, &testConnection{name: "test"}), transports.ContextReceiver, transports.Receiver(receiver))
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the connection")
	return pool, receiver, output, ctx, func() {
		sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test", nil), "Pool did not accept the disconnection")
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
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "first", 4, 4), "Pool did not receive the first payload")
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "second", 4, 4), "Pool did not receive the second payload")
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
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "payload", 6, 6), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)

	acknowledgeTestEvents(t, pool, events[3:])
	expectNoTestAck(t, receiver)
	acknowledgeTestEvents(t, pool, events[:3])
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 6})
	stop()
}

func TestPoolAcknowledgesPartialPrefix(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "payload", 6, 6), "Pool did not receive the payload")
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
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "first", 2, 2), "Pool did not receive the first payload")
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "second", 2, 2), "Pool did not receive the second payload")
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
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "payload", 2, 2), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test", nil), "Pool did not accept the disconnection")

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

func TestPoolAcknowledgementAfterExitReturns(t *testing.T) {
	pool, _, output, ctx, stop := startTestPool(t)
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "payload", 2, 2), "Pool did not receive the payload")
	events := receiveTestEvents(t, output)
	stop()

	acknowledged := make(chan struct{})
	go func() {
		pool.Acknowledge(events)
		close(acknowledged)
	}()
	select {
	case <-acknowledged:
	case <-time.After(5 * time.Second):
		t.Fatal("Acknowledgement blocked after the pool exited")
	}
}

func TestPoolDrainsLightPayloadAheadOfHeavyBacklog(t *testing.T) {
	pool, receiver, output, heavy, stop := startTestPool(t)
	light := context.WithValue(context.WithValue(context.Background(), transports.ContextConnection, &testConnection{name: "light"}), transports.ContextReceiver, transports.Receiver(receiver))
	sendTestEvent(t, pool, transports.NewConnectEvent(light, "light", "test"), "Pool did not accept the light connection")
	for i := 0; i < 3; i++ {
		sendTestPayload(t, pool, newTestPayloadOf(heavy, fmt.Sprintf("heavy%d", i), 600, 600*400), "Pool did not receive a heavy payload")
	}
	sendTestPayload(t, pool, newTestPayloadOf(light, "light", 10, 10*400), "Pool did not receive the light payload")

	drained := false
	for idx := 0; idx < 2 && !drained; idx++ {
		drained = len(receiveTestEvents(t, output)) == 10
	}
	if !drained {
		t.Fatal("Light payload was not drained within the first two slices, behind the heavy backlog")
	}
	sendTestEvent(t, pool, transports.NewDisconnectEvent(light, "light", "test", nil), "Pool did not accept the light disconnection")
	stop()
}

func TestPoolAcknowledgesPayloadDrainedInSlices(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	sendTestPayload(t, pool, newTestPayloadOf(ctx, "payload", 600, 600), "Pool did not receive the payload")
	first := receiveTestEvents(t, output)
	second := receiveTestEvents(t, output)
	third := receiveTestEvents(t, output)
	if len(first) != 256 || len(second) != 256 || len(third) != 88 {
		t.Fatalf("Expected slices of 256, 256 and 88 events, got %d, %d and %d", len(first), len(second), len(third))
	}

	acknowledgeTestEvents(t, pool, third)
	expectNoTestAck(t, receiver)
	acknowledgeTestEvents(t, pool, second)
	expectNoTestAck(t, receiver)
	acknowledgeTestEvents(t, pool, first)
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 600})
	stop()
}

// requestTestTake asks the pool to take the events held in a receive window, as a connection does
func requestTestTake(t *testing.T, pool *Pool, window *transports.ReceiveWindow) {
	t.Helper()
	sendTestEvent(t, pool, transports.NewAdmissionEvent(window.Context(), window), "Pool did not receive the admission request")
}

func expectTestTaken(t *testing.T, window *transports.ReceiveWindow, message string) {
	t.Helper()
	select {
	case <-window.Taken():
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func expectTestNotTaken(t *testing.T, window *transports.ReceiveWindow, message string) {
	t.Helper()
	select {
	case <-window.Taken():
		t.Fatal(message)
	case <-time.After(100 * time.Millisecond):
	}
}

// holdTestPayload holds a whole payload in a new receive window and asks the pool to take it
func holdTestPayload(t *testing.T, pool *Pool, payload *testPayload) *transports.ReceiveWindow {
	t.Helper()
	window := transports.NewReceiveWindow(payload.ctx, 1<<30, 1)
	payload.hold(window, 0, payload.count, true)
	requestTestTake(t, pool, window)
	return window
}

// sendTestPayload sends a payload as a connection does, waiting until the pool takes it
func sendTestPayload(t *testing.T, pool *Pool, payload *testPayload, message string) {
	t.Helper()
	expectTestTaken(t, holdTestPayload(t, pool, payload), "Pool did not take the payload: "+message)
}

func newTestConnectionContext(receiver *testReceiver, name string) context.Context {
	return context.WithValue(context.WithValue(context.Background(), transports.ContextConnection, &testConnection{name: name}), transports.ContextReceiver, transports.Receiver(receiver))
}

// startTestPoolWithQueue runs a pool with the given queue budget, returning a function that shuts it down
func startTestPoolWithQueue(t *testing.T, maxQueueSize int64) (*Pool, *testReceiver, <-chan []*event.Event, func()) {
	t.Helper()
	pool, receiver, output := newTestPool(maxQueueSize)
	shutdownChan := make(chan struct{})
	pool.shutdownChan = shutdownChan
	done := make(chan struct{})
	go func() {
		pool.Run()
		close(done)
	}()
	return pool, receiver, output, func() {
		close(shutdownChan)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Pool did not shut down")
		}
	}
}

func TestPoolHoldsEventsWhileQueueFull(t *testing.T) {
	pool, receiver, output, stop := startTestPoolWithQueue(t, 100)
	ctx := newTestConnectionContext(receiver, "test")
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the connection")
	sendTestPayload(t, pool, newTestPayload(ctx, "first", 100), "Pool did not receive a payload while its queue had space")

	window := holdTestPayload(t, pool, newTestPayload(ctx, "second", 100))
	expectTestNotTaken(t, window, "Pool took events from a connection with events queued while its queue was full")
	receiveTestOutput(t, output)
	expectTestTaken(t, window, "Pool did not take the events once its queue drained")

	if closes := receiver.closes.Load(); closes != 0 {
		t.Fatalf("Pool closed %d connections instead of holding them", closes)
	}
	receiveTestOutput(t, output)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test", nil), "Pool did not accept the disconnection")
	stop()
}

func TestPoolTakesEventsFromConnectionWithNothingQueuedWhileQueueFull(t *testing.T) {
	pool, receiver, output, stop := startTestPoolWithQueue(t, 100)
	busy := newTestConnectionContext(receiver, "busy")
	idle := newTestConnectionContext(receiver, "idle")
	sendTestEvent(t, pool, transports.NewConnectEvent(busy, "busy", "test"), "Pool did not accept the busy connection")
	sendTestEvent(t, pool, transports.NewConnectEvent(idle, "idle", "test"), "Pool did not accept the idle connection")
	sendTestPayload(t, pool, newTestPayload(busy, "busy", 100), "Pool did not receive the busy payload")

	sendTestPayload(t, pool, newTestPayload(idle, "idle", 100), "Pool held back a connection with nothing queued while its queue was full")

	receiveTestOutput(t, output)
	receiveTestOutput(t, output)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(busy, "busy", "test", nil), "Pool did not accept the busy disconnection")
	sendTestEvent(t, pool, transports.NewDisconnectEvent(idle, "idle", "test", nil), "Pool did not accept the idle disconnection")
	stop()
}

func TestPoolTakesFromLightSenderBeforeHeavySenderOfLargePayload(t *testing.T) {
	pool, receiver, output, stop := startTestPoolWithQueue(t, 1<<20)
	heavy := newTestConnectionContext(receiver, "heavy")
	light := newTestConnectionContext(receiver, "light")
	sendTestEvent(t, pool, transports.NewConnectEvent(heavy, "heavy", "test"), "Pool did not accept the heavy connection")
	sendTestEvent(t, pool, transports.NewConnectEvent(light, "light", "test"), "Pool did not accept the light connection")
	sendTestPayload(t, pool, newTestPayloadOf(heavy, "heavy", 1024, 1<<20), "Pool did not receive the heavy payload")
	sendTestPayload(t, pool, newTestPayloadOf(light, "light", 10, 4096), "Pool did not receive the light payload")

	heavyWindow := holdTestPayload(t, pool, newTestPayloadOf(heavy, "heavy2", 10, 4096))
	lightWindow := holdTestPayload(t, pool, newTestPayloadOf(light, "light2", 10, 4096))
	for {
		receiveTestEvents(t, output)
		// Checked in turn, as a select would choose at random once both are taken
		select {
		case <-lightWindow.Taken():
			sendTestEvent(t, pool, transports.NewDisconnectEvent(light, "light", "test", nil), "Pool did not accept the light disconnection")
			sendTestEvent(t, pool, transports.NewDisconnectEvent(heavy, "heavy", "test", nil), "Pool did not accept the heavy disconnection")
			stop()
			return
		default:
		}
		select {
		case <-heavyWindow.Taken():
			t.Fatal("Pool took from the heavy sender again before the light sender")
		default:
		}
	}
}

func TestPoolAcknowledgesPayloadReceivedInSegments(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	payload := newTestPayloadOf(ctx, "payload", 6, 6)
	window := transports.NewReceiveWindow(ctx, 1<<30, 1)
	payload.hold(window, 0, 3, false)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the first segment")
	first := receiveTestEvents(t, output)

	acknowledgeTestEvents(t, pool, first)
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 3})

	payload.hold(window, 3, 6, true)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the second segment")
	second := receiveTestEvents(t, output)
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("Expected segments of 3 and 3 events, got %d and %d", len(first), len(second))
	}

	acknowledgeTestEvents(t, pool, second)
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 6})
	sendTestEvent(t, pool, transports.NewPingEvent(ctx), "Pool did not receive the ping")
	if closes := receiver.closes.Load(); closes != 0 {
		t.Fatal("Pool failed the connection for a ping once every payload was acknowledged")
	}
	stop()
}

func TestPoolCompletesPayloadEndedAfterItsEventsAreAcknowledged(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	payload := newTestPayloadOf(ctx, "payload", 2, 2)
	window := transports.NewReceiveWindow(ctx, 1<<30, 1)
	payload.hold(window, 0, 2, false)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the events")
	acknowledgeTestEvents(t, pool, receiveTestEvents(t, output))
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 2})

	sendTestEvent(t, pool, transports.NewEndEvent(ctx), "Pool did not receive the end of the connection")
	if shutdowns := receiver.shutdowns.Load(); shutdowns != 0 {
		t.Fatal("Pool shut down the connection before its payload was received in full")
	}

	window.End(&payload.nonce)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the end of the payload")
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 2})
	if shutdowns := receiver.shutdowns.Load(); shutdowns != 1 {
		t.Fatalf("Pool shut down the connection %d times once its payload completed, expected once", shutdowns)
	}
	stop()
}

func TestPoolAcknowledgesSkippedEventsWithTheirPayload(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	nonce := "payload"
	window := transports.NewReceiveWindow(ctx, 1<<30, 1)
	window.Skip(&nonce)
	window.Add(&nonce, map[string]interface{}{"message": "second"}, 1)
	window.Skip(&nonce)
	window.Add(&nonce, map[string]interface{}{"message": "fourth"}, 1)
	window.End(&nonce)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the payload")
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 1})

	second := receiveTestEvents(t, output)
	fourth := receiveTestEvents(t, output)
	if len(second) != 1 || len(fourth) != 1 {
		t.Fatalf("Expected the two events that were not skipped, got %d and %d", len(second), len(fourth))
	}
	acknowledgeTestEvents(t, pool, second)
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 3})
	acknowledgeTestEvents(t, pool, fourth)
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 4})
	stop()
}

func TestPoolAcknowledgesPayloadWhoseEventsWereAllSkipped(t *testing.T) {
	pool, receiver, _, ctx, stop := startTestPool(t)
	nonce := "payload"
	window := transports.NewReceiveWindow(ctx, 1<<30, 1)
	window.Skip(&nonce)
	window.Skip(&nonce)
	window.End(&nonce)
	requestTestTake(t, pool, window)
	expectTestTaken(t, window, "Pool did not take the payload")
	expectTestAck(t, receiver, testAck{nonce: "payload", sequence: 2})
	stop()
}

func TestPoolKeepsConnectionWithManyPayloadsAwaitingAcknowledgement(t *testing.T) {
	pool, receiver, output, ctx, stop := startTestPool(t)
	for idx := 0; idx < 20; idx++ {
		sendTestPayload(t, pool, newTestPayloadOf(ctx, fmt.Sprintf("payload%d", idx), 1, 1), "Pool did not receive a payload")
		receiveTestOutput(t, output)
	}
	if closes := receiver.closes.Load(); closes != 0 {
		t.Fatalf("Pool closed the connection %d times for having payloads awaiting acknowledgement", closes)
	}
	stop()
}

// testAPIConnections returns the connections listed in the pool's API
func testAPIConnections(t *testing.T, pool *Pool) []map[string]interface{} {
	t.Helper()
	if err := pool.apiConnections.Update(); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	encoded, err := json.Marshal(&pool.apiConnections)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	var connections []map[string]interface{}
	if err := json.Unmarshal(encoded, &connections); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	return connections
}

// awaitTestAPIConnections waits until the connections listed in the pool's API satisfy expected
func awaitTestAPIConnections(t *testing.T, pool *Pool, message string, expected func([]map[string]interface{}) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		connections := testAPIConnections(t, pool)
		if expected(connections) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s, got %v", message, connections)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPoolListsFailedConnectionUntilRetentionPasses(t *testing.T) {
	pool, receiver, _ := newTestPool(1 << 20)
	pool.closedRetention = 200 * time.Millisecond
	shutdownChan := make(chan struct{})
	pool.shutdownChan = shutdownChan
	done := make(chan struct{})
	go func() {
		pool.Run()
		close(done)
	}()
	ctx := newTestConnectionContext(receiver, "test")
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the connection")
	awaitTestAPIConnections(t, pool, "Expected one active connection", func(connections []map[string]interface{}) bool {
		return len(connections) == 1 && connections[0]["status"] == "active"
	})

	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test", errors.New("connection reset")), "Pool did not accept the disconnection")
	awaitTestAPIConnections(t, pool, "Expected the failed connection listed with its error", func(connections []map[string]interface{}) bool {
		return len(connections) == 1 && connections[0]["status"] == "failed" && connections[0]["error"] == "connection reset"
	})
	awaitTestAPIConnections(t, pool, "Expected the failed connection no longer listed once its retention passed", func(connections []map[string]interface{}) bool {
		return len(connections) == 0
	})
	close(shutdownChan)
	<-done
}

func TestPoolListsCleanlyClosedConnectionAsClosed(t *testing.T) {
	pool, _, _, ctx, stop := startTestPool(t)
	sendTestEvent(t, pool, transports.NewDisconnectEvent(ctx, "remote", "test", nil), "Pool did not accept the disconnection")
	awaitTestAPIConnections(t, pool, "Expected the closed connection listed as closed", func(connections []map[string]interface{}) bool {
		return len(connections) == 1 && connections[0]["status"] == "closed"
	})
	sendTestEvent(t, pool, transports.NewConnectEvent(ctx, "remote", "test"), "Pool did not accept the reconnection")
	stop()
}
