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
