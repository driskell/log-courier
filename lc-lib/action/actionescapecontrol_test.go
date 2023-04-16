package action

import (
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

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

func processControlTestEvent(t *testing.T, arguments map[string]interface{}, data map[string]interface{}) *event.Event {
	t.Helper()
	return processTestEvent(t, "escape_control", arguments, data)
}

func TestEscapeControlAcceptsFields(t *testing.T) {
	mustNewTestNode(t, "escape_control", map[string]interface{}{"fields": []string{"message", "nested[field]"}})
}

func TestEscapeControlConfigRejectsUnknownKey(t *testing.T) {
	if _, err := newTestNode("escape_control", map[string]interface{}{"unknown": true}); err == nil {
		t.Error("Unknown key was not rejected")
	}
}

func TestEscapeControlActionEscapesWholeEvent(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{}, map[string]interface{}{
		"message": "a\x1b[2J",
		"nested":  map[string]interface{}{"field": "b\x07"},
	})

	if evnt.Data()["message"] != `a\x1B[2J` || evnt.MustResolve("nested[field]", nil) != `b\x07` {
		t.Errorf("Event was not escaped: %#v", evnt.Data())
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message", "_control_chars_nested")
}

func TestEscapeControlActionEscapesTopLevelKeysAndTagsThem(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{}, map[string]interface{}{"key\x07": "clean"})

	if evnt.Data()[`key\x07`] != "clean" {
		t.Errorf("Key was not escaped: %#v", evnt.Data())
	}
	if _, ok := evnt.Data()["key\x07"]; ok {
		t.Error("Original key was kept")
	}
	assertControlTestTags(t, evnt, "_control_chars", `_control_chars_key\x07`)
}

func TestEscapeControlActionLeavesCleanEventUntagged(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{}, map[string]interface{}{"message": "clean"})

	assertControlTestTags(t, evnt)
}

func TestEscapeControlActionEscapesOnlySelectedFields(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{"fields": []string{"message", "nested", "missing"}}, map[string]interface{}{
		"message": "a\x1b",
		"other":   "b\x1b",
		"nested":  map[string]interface{}{"field": "c\x1b"},
	})

	if evnt.Data()["message"] != `a\x1B` || evnt.MustResolve("nested[field]", nil) != `c\x1B` {
		t.Errorf("Selected fields were not escaped: %#v", evnt.Data())
	}
	if evnt.Data()["other"] != "b\x1b" {
		t.Errorf("Unselected field was escaped: %q", evnt.Data()["other"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message", "_control_chars_nested")
}

func TestEscapeControlActionTagsSelectedFieldNamesEscaped(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{"fields": []string{"a\x07"}}, map[string]interface{}{"a\x07": "\x1b"})

	assertControlTestTags(t, evnt, "_control_chars", `_control_chars_a\x07`)
}

func TestEscapeControlActionLeavesCleanSelectedFieldsUntagged(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{"fields": []string{"message"}}, map[string]interface{}{"message": "clean", "other": "b\x1b"})

	assertControlTestTags(t, evnt)
}

func TestEscapeControlActionReportsUnresolvableField(t *testing.T) {
	evnt := processControlTestEvent(t, map[string]interface{}{"fields": []string{"bad]", "message"}}, map[string]interface{}{"message": "a\x1b"})

	if message := evnt.Data()["_escape_control_error"]; message != "Field 'bad]' could not be resolved: Invalid field: bad]" {
		t.Errorf("Unexpected _escape_control_error: %#v", message)
	}
	if evnt.Data()["message"] != `a\x1B` {
		t.Error("Remaining fields were not escaped after a failure")
	}
}
