package action

import (
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
)

func TestDateAcceptsAllArguments(t *testing.T) {
	mustNewTestNode(t, "date", map[string]interface{}{"field": "epoch", "remove": true, "formats": []string{"UNIX"}})
}

func TestDateConfigRejectsUnknownKey(t *testing.T) {
	if _, err := newTestNode("date", map[string]interface{}{"field": "epoch", "formats": []string{"UNIX"}, "unknown": true}); err == nil {
		t.Error("Unknown key was not rejected")
	}
}

func processDateTestEvent(t *testing.T, arguments map[string]interface{}, data map[string]interface{}) *event.Event {
	t.Helper()
	return processTestEvent(t, "date", arguments, data)
}

func dateTestTimestamp(t *testing.T, evnt *event.Event) time.Time {
	t.Helper()
	timestamp, ok := evnt.MustResolve("@timestamp", nil).(event.Timestamp)
	if !ok {
		t.Fatalf("@timestamp is not an event.Timestamp: %#v", evnt.MustResolve("@timestamp", nil))
	}
	return time.Time(timestamp)
}

func assertDateTestError(t *testing.T, evnt *event.Event, expected string) {
	t.Helper()
	if message := evnt.Data()["_date_error"]; message != expected {
		t.Errorf("Unexpected _date_error: got %#v, expected %#v", message, expected)
	}
}

func TestDateUnixKeepsFraction(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "epoch", "formats": []string{"UNIX"}}, map[string]interface{}{"epoch": "1792914775.574"})

	expected := time.Unix(1792914775, 574*int64(time.Millisecond))
	if timestamp := dateTestTimestamp(t, evnt); !timestamp.Equal(expected) {
		t.Errorf("Unexpected @timestamp: got %s, expected %s", timestamp.UTC(), expected.UTC())
	}
}

func TestDateFallsBackToNextFormat(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "when", "formats": []string{"UNIX", time.RFC3339}}, map[string]interface{}{"when": "2026-10-25T07:52:55Z"})

	expected := time.Date(2026, 10, 25, 7, 52, 55, 0, time.UTC)
	if timestamp := dateTestTimestamp(t, evnt); !timestamp.Equal(expected) {
		t.Errorf("Unexpected @timestamp: got %s, expected %s", timestamp.UTC(), expected.UTC())
	}
}

func TestDateWithoutYearUsesCurrentYear(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "when", "formats": []string{time.Stamp}}, map[string]interface{}{"when": "Oct 25 07:52:55"})

	expected := time.Date(time.Now().Year(), 10, 25, 7, 52, 55, 0, time.UTC)
	if timestamp := dateTestTimestamp(t, evnt); !timestamp.Equal(expected) {
		t.Errorf("Unexpected @timestamp: got %s, expected %s", timestamp.UTC(), expected.UTC())
	}
}

func TestDateRemovesField(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "epoch", "remove": true, "formats": []string{"UNIX"}}, map[string]interface{}{"epoch": "1792914775"})

	if _, ok := evnt.Data()["epoch"]; ok {
		t.Error("Field was not removed after parsing")
	}
}

func TestDateNoFormatMatches(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "when", "formats": []string{"UNIX", time.RFC3339}}, map[string]interface{}{"when": "not a date"})

	assertDateTestError(t, evnt, "Field 'when' could not be parsed with any of the given formats")
}

func TestDateFieldNotString(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "when", "formats": []string{"UNIX"}}, map[string]interface{}{"when": 1792914775})

	assertDateTestError(t, evnt, "Field 'when' is not present or not a string")
}

func TestDateFieldUnresolvable(t *testing.T) {
	evnt := processDateTestEvent(t, map[string]interface{}{"field": "when]", "formats": []string{"UNIX"}}, map[string]interface{}{})

	assertDateTestError(t, evnt, "Field 'when]' could not be resolved: Invalid field: when]")
}
