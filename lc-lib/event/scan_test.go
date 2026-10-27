package event

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// decodedHasControl returns true if decoding the encoded event gives keys or values with control characters
func decodedHasControl(encoded []byte) (bool, bool) {
	var data map[string]interface{}
	if err := json.Unmarshal(encoded, &data); err != nil || data == nil {
		return false, false
	}
	_, found := EscapeControlValue(data)
	return found, true
}

func assertEncodedControlMatchesDecoded(t *testing.T, encoded []byte) {
	t.Helper()
	expected, valid := decodedHasControl(encoded)
	if !valid {
		return
	}
	if found := encodedHasControl(encoded); found != expected {
		t.Fatalf("Expected control characters found in %q to be %v, got %v", encoded, expected, found)
	}
}

func TestEncodedControlMatchesDecodedControl(t *testing.T) {
	for _, input := range []string{
		`{"message":"plain text"}`,
		`{"message":"tab\tand\nnewline"}`,
		`{"message":"tab\u0009and\u000anewline"}`,
		`{"message":"carriage\rreturn"}`,
		`{"message":"back\bspace"}`,
		`{"message":"form\ffeed"}`,
		`{"message":"escape \u001b[31m"}`,
		`{"message":"nul \u0000"}`,
		`{"message":"delete \u007F"}`,
		"{\"message\":\"delete raw \x7f\"}",
		`{"message":"c1 \u0085"}`,
		"{\"message\":\"c1 raw \xc2\x85\"}",
		`{"message":"after c1   é"}`,
		"{\"message\":\"after c1 raw \xc2\xa0\"}",
		`{"message":"escaped backslash then b \\b"}`,
		`{"message":"escaped backslash then u \\u001b"}`,
		`{"key with \u001b":"value"}`,
		`{"nested":{"list":["ok","bad \u0007"]}}`,
		"{ \r\n\t\"message\" : \"whitespace between tokens\" }",
	} {
		assertEncodedControlMatchesDecoded(t, []byte(input))
	}
}

func FuzzEncodedControlMatchesDecodedControl(f *testing.F) {
	for _, seed := range []string{`{"a":"\u001b"}`, `{"a":"\\b"}`, "{\"a\":\"\xc2\x85\"}", `{"a":["\u0085",{"\u007f":1}]}`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		assertEncodedControlMatchesDecoded(t, input)
	})
}

func TestEventTimestampMatchesDecodedTimestamp(t *testing.T) {
	for _, input := range []string{
		`{"@timestamp":"2026-10-27T09:15:02.123Z","message":"first"}`,
		`{"message":"later","@timestamp":"2026-10-27T09:15:02+05:30"}`,
		`{"message":"nested","fields":{"@timestamp":"2000-01-01T00:00:00Z"},"@timestamp":"2026-10-27T09:15:02Z"}`,
	} {
		expected := NewEventFromBytes(context.Background(), nil, []byte(input)).Data()["@timestamp"].(Timestamp)
		timestamp, ok := NewEventFromBytes(context.Background(), nil, []byte(input)).Timestamp()
		if !ok || !time.Time(timestamp).Equal(time.Time(expected)) {
			t.Fatalf("Expected @timestamp %v from %s, got %v (found %v)", expected, input, timestamp, ok)
		}
	}
}

func TestEventTimestampOfInvalidTimestampIsTheDecodedReplacement(t *testing.T) {
	for _, input := range []string{`{"message":"none"}`, `{"@timestamp":"yesterday"}`, `{"@timestamp":12}`} {
		event := NewEventFromBytes(context.Background(), nil, []byte(input))
		timestamp, ok := event.Timestamp()
		if !ok || !time.Time(timestamp).Equal(time.Time(event.Data()["@timestamp"].(Timestamp))) {
			t.Fatalf("Expected the decoded replacement @timestamp from %s, got %v (found %v)", input, timestamp, ok)
		}
	}
}

// decodedEncoding returns the event data validating an encoded event must produce, by decoding and encoding it
func decodedEncoding(input []byte) map[string]interface{} {
	event := NewEventFromBytes(context.Background(), nil, input)
	event.Data()
	event.ClearCache()
	var data map[string]interface{}
	json.Unmarshal(event.Bytes(), &data)
	return data
}

func TestEventValidateOfUnchangedEventEncodesAsDecoded(t *testing.T) {
	for _, input := range []string{
		`{"@timestamp":"2026-10-27T09:15:02.123Z","message":"normal","tags":["a"]}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","message":"no tags"}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":"single"}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":[],"@metadata":{"from":"client"}}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":[],"dup":1,"dup":2}`,
		"{\"@timestamp\":\"2026-10-27T09:15:02Z\",\"tags\":[],\"bad\":\"\xff\"}",
	} {
		event := NewEventFromBytes(context.Background(), nil, []byte(input))
		event.Validate()
		var data map[string]interface{}
		if err := json.Unmarshal(event.Bytes(), &data); err != nil {
			t.Fatalf("Unexpected error decoding %s: %s", event.Bytes(), err)
		}
		if expected := decodedEncoding([]byte(input)); !reflect.DeepEqual(data, expected) {
			t.Fatalf("Expected %s to encode as %v once validated, got %v", input, expected, data)
		}
	}
}

func TestEventFreeOfControlOnlyWhenUndecodedAndClean(t *testing.T) {
	clean := []byte(`{"message":"clean"}`)
	if !NewEventFromBytes(context.Background(), nil, clean).FreeOfControl() {
		t.Fatal("Expected an undecoded event without control characters to be free of them")
	}
	if NewEventFromBytes(context.Background(), nil, []byte(`{"message":"\u001b"}`)).FreeOfControl() {
		t.Fatal("Expected an undecoded event with control characters not to be free of them")
	}
	decoded := NewEventFromBytes(context.Background(), nil, clean)
	decoded.Data()
	if decoded.FreeOfControl() {
		t.Fatal("Expected a decoded event not to report being free of control characters")
	}
}

func FuzzEventValidateOfUnchangedEventEncodesAsDecoded(f *testing.F) {
	for _, seed := range []string{
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":["a"],"message":"m"}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":[1],"message":"m"}`,
		`{"@timestamp":"2026-10-27T09:15:02Z","tags":[],"@metadata":{}}`,
		`{"tags":[],"@timestamp":"2026-10-27T09:15:02+01:00","n":1e400}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		var raw map[string]interface{}
		if json.Unmarshal(input, &raw) != nil || raw == nil {
			return
		}
		// A missing or invalid @timestamp is replaced with the time it was decoded, which differs between the two
		if value, ok := raw["@timestamp"].(string); !ok || !validTimestamp(value) {
			return
		}
		event := NewEventFromBytes(context.Background(), nil, input)
		event.Validate()
		var data map[string]interface{}
		if err := json.Unmarshal(event.Bytes(), &data); err != nil {
			t.Fatalf("Unexpected error decoding %q: %s", event.Bytes(), err)
		}
		if expected := decodedEncoding(input); !reflect.DeepEqual(data, expected) {
			t.Fatalf("Expected %q to encode as %v once validated, got %v", input, expected, data)
		}
	})
}

func TestEventClearCacheLeavesUndecodedEventAsReceived(t *testing.T) {
	input := `{"message":"no tags or timestamp"}`
	event := NewEventFromBytes(context.Background(), nil, []byte(input))
	event.ClearCache()
	if string(event.Bytes()) != input || !event.FreeOfControl() {
		t.Fatalf("Expected an unchanged event kept as received, got %s", event.Bytes())
	}
}

func TestEventValidateEncodesEventChangedThroughItAgain(t *testing.T) {
	event := NewEventFromBytes(context.Background(), nil, []byte(`{"@timestamp":"2026-10-27T09:15:02Z","tags":[],"message":"old"}`))
	event.MustResolve("message", "changed")
	event.Validate()
	if !strings.Contains(string(event.Bytes()), `"message":"changed"`) {
		t.Fatalf("Expected the changed event encoded again, got %s", event.Bytes())
	}
}

func TestEventValidateAfterDecodingValidatesAgain(t *testing.T) {
	event := NewEventFromBytes(context.Background(), nil, []byte(`{"@timestamp":"2026-10-27T09:15:02Z","tags":[],"message":"old"}`))
	event.Validate()
	event.MustResolve("message", "changed")
	event.Validate()
	if !strings.Contains(string(event.Bytes()), `"message":"changed"`) {
		t.Fatalf("Expected the event changed after validating encoded again, got %s", event.Bytes())
	}
}

func TestEventChangesThroughItAreEncoded(t *testing.T) {
	input := []byte(`{"@timestamp":"2026-10-27T09:15:02Z","tags":["old"],"message":"old","remove":"me","control":"a\u001b"}`)
	for name, change := range map[string]struct {
		apply    func(*Event)
		expected string
	}{
		"set":            {func(e *Event) { e.MustResolve("message", "new") }, `"message":"new"`},
		"set nested":     {func(e *Event) { e.MustResolve("fields[env]", "prod") }, `"fields":{"env":"prod"}`},
		"unset":          {func(e *Event) { e.MustResolve("remove", ResolveParamUnset) }, `"message":"old","tags"`},
		"add tag":        {func(e *Event) { e.AddTag("new") }, `"tags":["new","old"]`},
		"remove tag":     {func(e *Event) { e.RemoveTag("old") }, `"tags":[]`},
		"add error":      {func(e *Event) { e.AddError("test", "failed") }, `"_test_error":"failed"`},
		"escape control": {func(e *Event) { e.EscapeControl() }, `"control":"a\\x1B"`},
	} {
		t.Run(name, func(t *testing.T) {
			event := NewEventFromBytes(context.Background(), nil, input)
			event.Data()
			event.Bytes()
			change.apply(event)
			event.Validate()
			if !strings.Contains(string(event.Bytes()), change.expected) {
				t.Fatalf("Expected the change encoded as %s, got %s", change.expected, event.Bytes())
			}
		})
	}
}
