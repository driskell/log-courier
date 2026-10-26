package courier

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/transports"
	"github.com/driskell/log-courier/lc-lib/transports/tcp"
)

type newMessageFunc func(tcp.Connection, uint32) (tcp.ProtocolMessage, error)

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

func readTestMessage(t *testing.T, newMessage newMessageFunc, body []byte, bodyLength uint32) (tcp.ProtocolMessage, error) {
	t.Helper()
	reader := &benchConnection{}
	reader.reader.Reset(body)
	return newMessage(reader, bodyLength)
}

func assertTestEvents(t *testing.T, message tcp.ProtocolMessage, expected [][]byte) {
	t.Helper()
	if events := message.(transports.RawEventsEvent).RawEvents(); !reflect.DeepEqual(events, expected) {
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
		message, err := readTestMessage(t, newMessage, check.body, check.length)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		assertTestEvents(t, message, check.events)
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
	if _, err := readTestMessage(t, newProtocolEVNT, corrupt, length); err == nil {
		t.Fatal("Expected an error reading a corrupt message")
	}

	message, err := readTestMessage(t, newProtocolEVNT, body, length)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	assertTestEvents(t, message, events)
}
