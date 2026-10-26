package tcp

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
)

func newTestConnection(t *testing.T, source io.Reader) *connection {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	conn := &connection{
		ctx:                  ctx,
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
