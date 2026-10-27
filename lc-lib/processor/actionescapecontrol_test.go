package processor

import (
	"context"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

func newControlTestEvent(data map[string]interface{}) *event.Event {
	return event.NewEvent(context.Background(), nil, data)
}

func controlTestTags(evnt *event.Event) event.Tags {
	return evnt.Data()["tags"].(event.Tags)
}

func assertControlTestTags(t *testing.T, evnt *event.Event, expected ...string) {
	t.Helper()
	if expected == nil {
		expected = []string{}
	}
	if tags := controlTestTags(evnt); !reflect.DeepEqual([]string(tags), expected) {
		t.Errorf("Unexpected tags: got %q, expected %q", tags, expected)
	}
}

func TestEscapeControlFromConfig(t *testing.T) {
	entry, err := newEscapeControlAction(config.NewParser(nil), "/escape_control/", map[string]interface{}{"fields": []interface{}{"message", "nested[field]"}}, "escape_control")
	if err != nil {
		t.Fatalf("Parsing failed unexpectedly: %s", err)
	}
	if action := entry.(*escapeControlAction); !reflect.DeepEqual(action.Fields, []string{"message", "nested[field]"}) {
		t.Errorf("Unexpected action from config: %#v", action)
	}
}

func TestEscapeControlConfigRejectsUnknownKey(t *testing.T) {
	if _, err := newEscapeControlAction(config.NewParser(nil), "/escape_control/", map[string]interface{}{"unknown": true}, "escape_control"); err == nil {
		t.Error("Unknown key was not rejected")
	}
}

func TestEscapeControlActionEscapesWholeEvent(t *testing.T) {
	evnt := (&escapeControlAction{}).Process(newControlTestEvent(map[string]interface{}{
		"message": "a\x1b[2J",
		"nested":  map[string]interface{}{"field": "b\x07"},
	}))

	if evnt.Data()["message"] != `a\x1B[2J` || evnt.MustResolve("nested[field]", nil) != `b\x07` {
		t.Errorf("Event was not escaped: %#v", evnt.Data())
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message", "_control_chars_nested")
}

func TestEscapeControlActionEscapesTopLevelKeysAndTagsThem(t *testing.T) {
	evnt := (&escapeControlAction{}).Process(newControlTestEvent(map[string]interface{}{"key\x07": "clean"}))

	if evnt.Data()[`key\x07`] != "clean" {
		t.Errorf("Key was not escaped: %#v", evnt.Data())
	}
	if _, ok := evnt.Data()["key\x07"]; ok {
		t.Error("Original key was kept")
	}
	assertControlTestTags(t, evnt, "_control_chars", `_control_chars_key\x07`)
}

func TestEscapeControlActionLeavesCleanEventUntagged(t *testing.T) {
	evnt := (&escapeControlAction{}).Process(newControlTestEvent(map[string]interface{}{"message": "clean"}))

	assertControlTestTags(t, evnt)
}

func TestEscapeControlActionEscapesOnlySelectedFields(t *testing.T) {
	evnt := (&escapeControlAction{Fields: []string{"message", "nested", "missing"}}).Process(newControlTestEvent(map[string]interface{}{
		"message": "a\x1b",
		"other":   "b\x1b",
		"nested":  map[string]interface{}{"field": "c\x1b"},
	}))

	if evnt.Data()["message"] != `a\x1B` || evnt.MustResolve("nested[field]", nil) != `c\x1B` {
		t.Errorf("Selected fields were not escaped: %#v", evnt.Data())
	}
	if evnt.Data()["other"] != "b\x1b" {
		t.Errorf("Unselected field was escaped: %q", evnt.Data()["other"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message", "_control_chars_nested")
}

func TestEscapeControlActionTagsSelectedFieldNamesEscaped(t *testing.T) {
	evnt := (&escapeControlAction{Fields: []string{"a\x07"}}).Process(newControlTestEvent(map[string]interface{}{"a\x07": "\x1b"}))

	assertControlTestTags(t, evnt, "_control_chars", `_control_chars_a\x07`)
}

func TestEscapeControlActionLeavesCleanSelectedFieldsUntagged(t *testing.T) {
	evnt := (&escapeControlAction{Fields: []string{"message"}}).Process(newControlTestEvent(map[string]interface{}{"message": "clean", "other": "b\x1b"}))

	assertControlTestTags(t, evnt)
}

func TestEscapeControlActionReportsUnresolvableField(t *testing.T) {
	evnt := (&escapeControlAction{Fields: []string{"bad]", "message"}}).Process(newControlTestEvent(map[string]interface{}{"message": "a\x1b"}))

	if message := evnt.Data()["_escape_control_error"]; message != "Field 'bad]' could not be resolved: Invalid field: bad]" {
		t.Errorf("Unexpected _escape_control_error: %#v", message)
	}
	if evnt.Data()["message"] != `a\x1B` {
		t.Error("Remaining fields were not escaped after a failure")
	}
}

func TestEscapeControlActionLeavesCleanReceivedEventUndecoded(t *testing.T) {
	evnt := (&escapeControlAction{}).Process(event.NewEventFromBytes(context.Background(), nil, []byte(`{"message":"clean"}`)))

	if !evnt.FreeOfControl() {
		t.Error("Clean received event was decoded to escape it")
	}
}

func TestEscapeControlActionEscapesReceivedEvent(t *testing.T) {
	evnt := (&escapeControlAction{}).Process(event.NewEventFromBytes(context.Background(), nil, []byte(`{"message":"a\u001b[2J"}`)))

	if evnt.Data()["message"] != `a\x1B[2J` {
		t.Errorf("Received event was not escaped: %#v", evnt.Data())
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message")
}
