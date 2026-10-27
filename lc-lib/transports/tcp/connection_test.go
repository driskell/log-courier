package tcp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/transports"
)

func newTestConnection(t *testing.T, source io.Reader) *connection {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	conn := &connection{
		ctx:                  ctx,
		shutdownFunc:         cancel,
		socket:               newBenchSocket(t),
		protocol:             &benchProtocol{},
		receiverShutdownChan: make(chan struct{}),
	}
	conn.rwBuffer.Reader = bufio.NewReader(source)
	return conn
}

func TestConnectionReadByteReadsInOrder(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader("abc"))
	for _, expected := range []byte("abc") {
		value, err := conn.ReadByte()
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if value != expected {
			t.Fatalf("Expected %q, got %q", expected, value)
		}
	}
}

func TestConnectionReadFillsAcrossBufferRefills(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader("abcdef"))
	first := make([]byte, 2)
	if n, err := conn.Read(first); err != nil || string(first[:n]) != "ab" {
		t.Fatalf("Expected \"ab\", got %q (error: %v)", first[:n], err)
	}
	rest := make([]byte, 4)
	if n, err := conn.Read(rest); err != nil || string(rest[:n]) != "cdef" {
		t.Fatalf("Expected \"cdef\", got %q (error: %v)", rest[:n], err)
	}
}

func TestConnectionReadStopsAtShutdownOnceBufferedDataIsRead(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader("abcd"))
	first := make([]byte, 2)
	if n, err := conn.Read(first); err != nil || string(first[:n]) != "ab" {
		t.Fatalf("Expected \"ab\", got %q (error: %v)", first[:n], err)
	}

	close(conn.receiverShutdownChan)
	buffered := make([]byte, 2)
	if n, err := conn.Read(buffered); err != nil || string(buffered[:n]) != "cd" {
		t.Fatalf("Expected buffered \"cd\" after shutdown, got %q (error: %v)", buffered[:n], err)
	}
	if _, err := conn.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("Expected io.EOF once buffered data was read after shutdown, got %v", err)
	}
}

func TestConnectionReadByteStopsAtShutdownOnceBufferedDataIsRead(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader("ab"))
	if value, err := conn.ReadByte(); err != nil || value != 'a' {
		t.Fatalf("Expected 'a', got %q (error: %v)", value, err)
	}

	close(conn.receiverShutdownChan)
	if value, err := conn.ReadByte(); err != nil || value != 'b' {
		t.Fatalf("Expected buffered 'b' after shutdown, got %q (error: %v)", value, err)
	}
	if _, err := conn.ReadByte(); err != io.EOF {
		t.Fatalf("Expected io.EOF once buffered data was read after shutdown, got %v", err)
	}
}

// testStreamMessage is a received message with preset events, where a nil event is one too large to hold, and whose
// event sizes are known before they are read
type testStreamMessage struct {
	ctx    context.Context
	nonce  string
	events [][]byte
}

func (m *testStreamMessage) Context() context.Context { return m.ctx }
func (m *testStreamMessage) Nonce() *string           { return &m.nonce }

func (m *testStreamMessage) Next(reserve func(int) error) ([]byte, map[string]interface{}, int, bool, error) {
	if len(m.events) == 0 {
		return nil, nil, 0, false, io.EOF
	}
	data := m.events[0]
	if data == nil {
		m.events = m.events[1:]
		return nil, nil, 0, true, nil
	}
	if err := reserve(len(data)); err != nil {
		return nil, nil, 0, false, err
	}
	m.events = m.events[1:]
	return data, nil, len(data), false, nil
}

// testProtocol returns preset received messages, then io.EOF
type testProtocol struct {
	benchProtocol
	messages []transports.Event
}

func (p *testProtocol) Read() (transports.Event, error) {
	if len(p.messages) == 0 {
		return nil, io.EOF
	}
	message := p.messages[0]
	p.messages = p.messages[1:]
	return message, nil
}

// newTestReceivingConnection returns a receiving connection with a receive window of the given size, and the channel
// its events for the receiver arrive on
func newTestReceivingConnection(t *testing.T, windowSize int) (*connection, chan transports.Event) {
	conn := newTestConnection(t, strings.NewReader(""))
	events := make(chan transports.Event, 10)
	conn.eventChan = events
	conn.window = transports.NewReceiveWindow(conn.ctx, windowSize, 1)
	return conn, events
}

func expectTestTakeRequest(t *testing.T, conn *connection, events <-chan transports.Event) {
	t.Helper()
	select {
	case requested := <-events:
		if admission, ok := requested.(*transports.AdmissionEvent); !ok || admission.Window() != conn.window {
			t.Fatalf("Expected a request to take the receive window, got %T", requested)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connection did not ask for its receive window to be taken")
	}
}

func expectNoTestEvent(t *testing.T, events <-chan transports.Event) {
	t.Helper()
	select {
	case unexpected := <-events:
		t.Fatalf("Connection sent unexpected %T", unexpected)
	default:
	}
}

func TestConnectionHoldsEventsWithoutWaitingWhileTheyFit(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 100)
	message := &testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{[]byte("first"), []byte("second"), []byte("third")}}
	if err := conn.receiveEvents(message); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	expectTestTakeRequest(t, conn, events)
	expectNoTestEvent(t, events)

	segments := conn.window.Take()
	if len(segments) != 1 || segments[0].Count() != 3 || !segments[0].Final {
		t.Fatalf("Expected the whole payload held as one segment, got %d segments", len(segments))
	}
}

func TestConnectionWaitsForRoomInReceiveWindow(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 10)
	message := &testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{[]byte("first!"), []byte("second")}}
	result := make(chan error, 1)
	go func() {
		result <- conn.receiveEvents(message)
	}()
	expectTestTakeRequest(t, conn, events)
	select {
	case err := <-result:
		t.Fatalf("Connection held more than its receive window (error: %v)", err)
	case <-time.After(100 * time.Millisecond):
	}

	if segments := conn.window.Take(); len(segments) != 1 || segments[0].Count() != 1 {
		t.Fatal("Expected only the first event held while the window was full")
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connection did not continue once its receive window was taken")
	}
	expectTestTakeRequest(t, conn, events)
}

func TestConnectionRecordsEventTooLargeToRead(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 100)
	message := &testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{nil, []byte("second")}}
	if err := conn.receiveEvents(message); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	expectTestTakeRequest(t, conn, events)

	segments := conn.window.Take()
	if len(segments) != 1 || segments[0].Skipped != 1 || segments[0].Count() != 1 {
		t.Fatal("Expected the skipped event recorded ahead of the event that followed")
	}
}

func TestConnectionStopsWaitingForRoomAtTeardown(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 10)
	message := &testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{[]byte("first!"), []byte("second")}}
	result := make(chan error, 1)
	go func() {
		result <- conn.receiveEvents(message)
	}()
	expectTestTakeRequest(t, conn, events)
	conn.Teardown()
	select {
	case err := <-result:
		if err != ErrHardCloseRequested {
			t.Fatalf("Expected ErrHardCloseRequested when torn down while waiting for room, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connection waiting for room did not stop at teardown")
	}
}

func TestConnectionReservesRoomBeforeReadingEventOfKnownSize(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 10)
	message := &testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{[]byte("first!"), []byte("second")}}
	result := make(chan error, 1)
	go func() {
		result <- conn.receiveEvents(message)
	}()
	expectTestTakeRequest(t, conn, events)
	if len(message.events) != 1 {
		t.Fatal("Connection read an event before the receive window had room for it")
	}
	conn.window.Take()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connection did not continue once its receive window was taken")
	}
}

func TestConnectionEndsOnlyOnceReceiveWindowIsTaken(t *testing.T) {
	conn, events := newTestReceivingConnection(t, 100)
	conn.protocol = &testProtocol{messages: []transports.Event{&testStreamMessage{ctx: conn.ctx, nonce: "payload", events: [][]byte{[]byte("event")}}}}
	result := make(chan error, 1)
	go func() {
		result <- conn.receiver(nil)
	}()
	expectTestTakeRequest(t, conn, events)
	select {
	case unexpected := <-events:
		t.Fatalf("Connection sent %T before its receive window was taken", unexpected)
	case <-time.After(100 * time.Millisecond):
	}

	conn.window.Take()
	select {
	case ended := <-events:
		if _, ok := ended.(*transports.EndEvent); !ok {
			t.Fatalf("Expected the end of the connection, got %T", ended)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connection did not end once its receive window was taken")
	}
	if err := <-result; err != io.EOF {
		t.Fatalf("Expected io.EOF, got %v", err)
	}
}

func TestConnectionRecordsFirstErrorAsReasonItEnded(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader(""))
	first, second := errors.New("first"), errors.New("second")
	conn.recordError(first)
	conn.recordError(second)
	conn.recordError(io.EOF)
	if conn.err != first {
		t.Fatalf("Expected the first error kept, got %v", conn.err)
	}
}

func TestConnectionFailureReplacesGracefulEnd(t *testing.T) {
	conn := newTestConnection(t, strings.NewReader(""))
	failure := errors.New("acknowledgement failed")
	conn.recordError(io.EOF)
	conn.Fail(failure)
	if conn.err != failure {
		t.Fatalf("Expected the failure to replace the graceful end, got %v", conn.err)
	}
}
