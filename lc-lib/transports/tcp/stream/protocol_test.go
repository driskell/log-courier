package stream

import (
	"context"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

// testConnection reads from a preset stream of lines
type testConnection struct {
	reader     *strings.Reader
	windowSize int
}

func (c *testConnection) Context() context.Context                      { return context.Background() }
func (c *testConnection) Write(data []byte) (int, error)                { return len(data), nil }
func (c *testConnection) Flush() error                                  { return nil }
func (c *testConnection) Read(data []byte) (int, error)                 { return c.reader.Read(data) }
func (c *testConnection) SendMessage(message tcp.ProtocolMessage) error { return nil }
func (c *testConnection) ReceiveWindowSize() int                        { return c.windowSize }
func (c *testConnection) LocalAddr() net.Addr                           { return &net.TCPAddr{} }
func (c *testConnection) RemoteAddr() net.Addr                          { return &net.TCPAddr{} }

func TestProtocolReadsLinesReadyWithoutWaitingAsOneMessage(t *testing.T) {
	conn := &testConnection{reader: strings.NewReader("first\nsecond\n"), windowSize: 1024}
	protocol := (&protocolFactory{}).NewProtocol(conn)

	message, err := protocol.Read()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	events := message.(tcp.StreamMessage)
	var lines []interface{}
	for {
		_, decoded, _, _, err := events.Next(nil)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		lines = append(lines, decoded["message"])
	}
	if !reflect.DeepEqual(lines, []interface{}{"first", "second"}) {
		t.Fatalf("Expected both lines in one message, got %v", lines)
	}

	message, err = protocol.Read()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if _, _, _, _, err := message.(tcp.StreamMessage).Next(nil); err != io.EOF {
		t.Fatalf("Expected the next message to end at the end of the stream, got %v", err)
	}
	if _, err := protocol.Read(); err != io.EOF {
		t.Fatalf("Expected io.EOF once the stream ended, got %v", err)
	}
}

func TestProtocolSplitsAndTagsLineLongerThanReceiveWindow(t *testing.T) {
	conn := &testConnection{reader: strings.NewReader("abcdefghijkl\nxy\n"), windowSize: 8}
	protocol := (&protocolFactory{}).NewProtocol(conn)

	message, err := protocol.Read()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	events := message.(tcp.StreamMessage)
	for _, expected := range []struct {
		message string
		split   bool
	}{{"abcdefgh", true}, {"ijkl", true}, {"xy", false}} {
		_, decoded, _, _, err := events.Next(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		_, tagged := decoded["tags"]
		if decoded["message"] != expected.message || tagged != expected.split {
			t.Fatalf("Expected %q with split tag %v, got %v", expected.message, expected.split, decoded)
		}
		if tagged && !reflect.DeepEqual(decoded["tags"], event.Tags{"splitline"}) {
			t.Fatalf("Expected a split line to be tagged splitline, got %v", decoded["tags"])
		}
	}
}
