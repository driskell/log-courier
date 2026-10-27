//go:build go1.27 && goexperiment.jsonv2

package event

import (
	"context"
	"strings"
	"testing"
)

func TestEventValidateKeepsEncodedFormThatDecodingWouldNotNormalise(t *testing.T) {
	input := `{"@timestamp":"2026-10-27T09:15:02.123Z","message":"normal","tags":["a"]}`
	event := NewEventFromBytes(context.Background(), nil, []byte(input))
	event.Validate()
	if string(event.Bytes()) != input || !event.FreeOfControl() {
		t.Fatalf("Expected the event kept undecoded with its encoded form, got %s", event.Bytes())
	}
}

func TestEventTimestampIsReadWithoutDecoding(t *testing.T) {
	event := NewEventFromBytes(context.Background(), nil, []byte(`{"@timestamp":"2026-10-27T09:15:02Z","message":"x"}`))
	if _, ok := event.Timestamp(); !ok || !event.FreeOfControl() {
		t.Fatal("Expected @timestamp read while the event stays undecoded")
	}
}

func TestEncodeDataWritesTimestampFirst(t *testing.T) {
	encoded, err := encodeData(map[string]interface{}{"0first": "sorts before", "@timestamp": "value", "message": "m"})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if !strings.HasPrefix(string(encoded), `{"@timestamp":"value",`) {
		t.Fatalf("Expected @timestamp encoded first, got %s", encoded)
	}
}

func TestEventValidateKeepsEncodedFormOfEventOnlyRead(t *testing.T) {
	input := `{"@timestamp":"2026-10-27T09:15:02.123Z","message":"read only","tags":["a"]}`
	event := NewEventFromBytes(context.Background(), nil, []byte(input))
	if event.MustResolve("message", nil) != "read only" {
		t.Fatal("Event was not readable")
	}
	event.Validate()
	if string(event.Bytes()) != input {
		t.Fatalf("Expected the event read but unchanged to keep its encoded form, got %s", event.Bytes())
	}
}
