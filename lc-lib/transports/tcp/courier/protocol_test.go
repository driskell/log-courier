package courier

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type newMessageFunc func(tcp.Connection, uint32) (receivedMessage, error)

// writeTestMessage returns the body of the message as it is sent, and the body length from its header
func writeTestMessage(t *testing.T, message tcp.ProtocolMessage) ([]byte, uint32) {
	t.Helper()
	writer := &benchConnection{}
	if err := message.Write(writer); err != nil {
		t.Fatal(err)
	}
	written := writer.written.Bytes()
	return written[8:], binary.BigEndian.Uint32(written[4:8])
}

func newTestReader(body []byte) *benchConnection {
	reader := &benchConnection{windowSize: 1 << 20}
	reader.reader.Reset(body)
	return reader
}

// readTestMessage reads every event of a message, returning the events and the number skipped
func readTestMessage(t *testing.T, newMessage newMessageFunc, conn tcp.Connection, bodyLength uint32) ([][]byte, int, error) {
	t.Helper()
	message, err := newMessage(conn, bodyLength)
	if err != nil {
		return nil, 0, err
	}
	var events [][]byte
	skipped := 0
	for {
		data, _, _, skip, err := message.(tcp.StreamMessage).Next(reserveTestEvent)
		if err == io.EOF {
			return events, skipped, nil
		}
		if err != nil {
			return nil, 0, err
		}
		if skip {
			skipped++
			continue
		}
		events = append(events, data)
	}
}

// reserveTestEvent allows every event to be held
func reserveTestEvent(int) error { return nil }

// readTestEvents reads every event of a message
func readTestEvents(message tcp.StreamMessage) (int, error) {
	count := 0
	for {
		_, _, _, _, err := message.Next(reserveTestEvent)
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		count++
	}
}

func assertTestEvents(t *testing.T, events [][]byte, expected [][]byte) {
	t.Helper()
	if !reflect.DeepEqual(events, expected) {
		t.Fatalf("Expected events %q, got %q", expected, events)
	}
}

func testProtocolReadsConsecutiveMessages(t *testing.T, newMessage newMessageFunc, build func(nonce string, events [][]byte) tcp.ProtocolMessage) {
	first := [][]byte{[]byte(`{"message":"first"}`), []byte(`{"message":"second"}`)}
	second := [][]byte{[]byte(`{"message":"third"}`)}
	firstBody, firstLength := writeTestMessage(t, build("0123456789abcdef", first))
	secondBody, secondLength := writeTestMessage(t, build("fedcba9876543210", second))

	for _, check := range []struct {
		body   []byte
		length uint32
		events [][]byte
	}{{firstBody, firstLength, first}, {secondBody, secondLength, second}, {firstBody, firstLength, first}} {
		events, _, err := readTestMessage(t, newMessage, newTestReader(check.body), check.length)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		assertTestEvents(t, events, check.events)
	}
}

func TestProtocolEVNTReadsConsecutiveMessages(t *testing.T) {
	testProtocolReadsConsecutiveMessages(t, newProtocolEVNT, func(nonce string, events [][]byte) tcp.ProtocolMessage {
		return &protocolEVNT{nonce: &nonce, events: events}
	})
}

func TestProtocolJDATReadsConsecutiveMessages(t *testing.T) {
	testProtocolReadsConsecutiveMessages(t, newProtocolJDAT, func(nonce string, events [][]byte) tcp.ProtocolMessage {
		return &protocolJDAT{nonce: &nonce, events: events}
	})
}

func TestProtocolEVNTReadsMessageAfterCorruptMessage(t *testing.T) {
	events := [][]byte{[]byte(`{"message":"valid"}`)}
	nonce := "0123456789abcdef"
	body, length := writeTestMessage(t, &protocolEVNT{nonce: &nonce, events: events})

	corrupt := append([]byte(nonce), []byte("not compressed data")...)
	if _, _, err := readTestMessage(t, newProtocolEVNT, newTestReader(corrupt), length); err == nil {
		t.Fatal("Expected an error reading a corrupt message")
	}

	read, _, err := readTestMessage(t, newProtocolEVNT, newTestReader(body), length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, read, events)
}

func testProtocolReadsMessagesFromOneStream(t *testing.T, newMessage newMessageFunc, build func(nonce string, events [][]byte) tcp.ProtocolMessage) {
	first := [][]byte{[]byte(`{"message":"first"}`), []byte(`{"message":"second"}`)}
	second := [][]byte{[]byte(`{"message":"third"}`)}
	firstBody, firstLength := writeTestMessage(t, build("0123456789abcdef", first))
	secondBody, secondLength := writeTestMessage(t, build("fedcba9876543210", second))
	secondHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(secondHeader[4:], secondLength)
	conn := newTestReader(append(append(append([]byte{}, firstBody...), secondHeader...), secondBody...))

	events, _, err := readTestMessage(t, newMessage, conn, firstLength)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, events, first)
	var header [8]byte
	if _, err := conn.Read(header[:]); err != nil {
		t.Fatalf("Unexpected error reading the next header: %s", err)
	}
	events, _, err = readTestMessage(t, newMessage, conn, binary.BigEndian.Uint32(header[4:]))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, events, second)
}

func TestProtocolEVNTReadsMessagesFromOneStream(t *testing.T) {
	testProtocolReadsMessagesFromOneStream(t, newProtocolEVNT, func(nonce string, events [][]byte) tcp.ProtocolMessage {
		return &protocolEVNT{nonce: &nonce, events: events}
	})
}

func TestProtocolJDATReadsMessagesFromOneStream(t *testing.T) {
	testProtocolReadsMessagesFromOneStream(t, newProtocolJDAT, func(nonce string, events [][]byte) tcp.ProtocolMessage {
		return &protocolJDAT{nonce: &nonce, events: events}
	})
}

func TestProtocolEVNTSkipsEventLargerThanReceiveWindow(t *testing.T) {
	events := [][]byte{[]byte(`{"message":"small"}`), []byte(`{"message":"far too large"}`), []byte(`{"message":"next"}`)}
	nonce := "0123456789abcdef"
	body, length := writeTestMessage(t, &protocolEVNT{nonce: &nonce, events: events})
	conn := newTestReader(body)
	conn.windowSize = 20

	read, skipped, err := readTestMessage(t, newProtocolEVNT, conn, length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if skipped != 1 {
		t.Fatalf("Expected one event skipped, got %d", skipped)
	}
	assertTestEvents(t, read, [][]byte{events[0], events[2]})
}

// writeTestJDAT returns the body of a JDAT message whose compressed events are followed by trailing bytes
func writeTestJDAT(t *testing.T, nonce string, events [][]byte, trailing []byte) ([]byte, uint32) {
	t.Helper()
	body, _ := writeTestMessage(t, &protocolJDAT{nonce: &nonce, events: events})
	body = append(append([]byte{}, body...), trailing...)
	return body, uint32(len(body))
}

func TestProtocolJDATReadsNextMessageAfterTrailingBodyBytes(t *testing.T) {
	first := [][]byte{[]byte(`{"message":"first"}`)}
	second := [][]byte{[]byte(`{"message":"second"}`)}
	firstBody, firstLength := writeTestJDAT(t, "0123456789abcdef", first, []byte("trailing"))
	secondBody, secondLength := writeTestJDAT(t, "fedcba9876543210", second, nil)
	conn := newTestReader(append(append([]byte{}, firstBody...), secondBody...))

	events, _, err := readTestMessage(t, newProtocolJDAT, conn, firstLength)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, events, first)
	events, _, err = readTestMessage(t, newProtocolJDAT, conn, secondLength)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, events, second)
}

func TestProtocolJDATSkipsEventLargerThanReceiveWindow(t *testing.T) {
	events := [][]byte{[]byte(`{"message":"far too large"}`), []byte(`{"message":"next"}`)}
	body, length := writeTestJDAT(t, "0123456789abcdef", events, nil)
	conn := newTestReader(body)
	conn.windowSize = 20

	read, skipped, err := readTestMessage(t, newProtocolJDAT, conn, length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if skipped != 1 {
		t.Fatalf("Expected one event skipped, got %d", skipped)
	}
	assertTestEvents(t, read, events[1:])
}

func TestProtocolEVNTReservesRoomBeforeReadingEvent(t *testing.T) {
	events := [][]byte{[]byte(`{"message":"first"}`)}
	nonce := "0123456789abcdef"
	body, length := writeTestMessage(t, &protocolEVNT{nonce: &nonce, events: events})
	message, err := newProtocolEVNT(newTestReader(body), length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	refused := errors.New("no room")
	var reserved int
	_, _, _, _, err = message.(tcp.StreamMessage).Next(func(size int) error {
		reserved = size
		return refused
	})
	if err != refused || reserved != len(events[0]) {
		t.Fatalf("Expected room reserved for %d bytes before reading, got %d bytes reserved and error %v", len(events[0]), reserved, err)
	}
}

func TestProtocolEVNTDoesNotVerifyChecksum(t *testing.T) {
	events := [][]byte{[]byte(`{"message":"first"}`)}
	nonce := "0123456789abcdef"
	body, length := writeTestMessage(t, &protocolEVNT{nonce: &nonce, events: events})
	body = append([]byte{}, body...)
	body[len(body)-1] ^= 0xff
	conn := newTestReader(append(body, []byte("next")...))

	read, _, err := readTestMessage(t, newProtocolEVNT, conn, length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, read, events)
	var next [4]byte
	if _, err := conn.Read(next[:]); err != nil || string(next[:]) != "next" {
		t.Fatalf("Expected the data after the message to be read next, got %q (error: %v)", next, err)
	}
}
