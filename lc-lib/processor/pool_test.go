package processor

import (
	"context"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

func newControlTestEvent(data map[string]interface{}) *event.Event {
	return event.NewEvent(context.Background(), nil, data)
}

func assertControlTestTags(t *testing.T, evnt *event.Event, expected ...string) {
	t.Helper()
	if expected == nil {
		expected = []string{}
	}
	if tags := evnt.Data()["tags"].(event.Tags); !reflect.DeepEqual([]string(tags), expected) {
		t.Errorf("Unexpected tags: got %q, expected %q", tags, expected)
	}
}

type recordingTestEntry struct {
	message interface{}
}

func (r *recordingTestEntry) Process(evnt *event.Event) *event.Event {
	r.message = evnt.Data()["message"]
	return evnt
}

type settingTestEntry struct {
	field string
	value string
}

func (s *settingTestEntry) Process(evnt *event.Event) *event.Event {
	evnt.MustResolve(s.field, s.value)
	return evnt
}

func TestPoolEscapesControlCharactersAfterPipeline(t *testing.T) {
	recorder := &recordingTestEntry{}
	pool := &Pool{ast: []ast.ProcessNode{recorder}, guard: &timestampGuard{}, escapeControl: true}

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "a\x1Db\x1b"}))

	if recorder.message != "a\x1Db\x1b" {
		t.Errorf("Pipeline did not receive the original message: %q", recorder.message)
	}
	if evnt.Data()["message"] != `a\x1Db\x1B` {
		t.Errorf("Message was not escaped after the pipeline: %q", evnt.Data()["message"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message")
}

func TestPoolEscapesFieldsSetByPipeline(t *testing.T) {
	setter := &settingTestEntry{field: "audit_proctitle", value: "\x1b[2Jls"}
	pool := &Pool{ast: []ast.ProcessNode{setter}, guard: &timestampGuard{}, escapeControl: true}

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "proctitle=1B5B324A6C73\x1DUID=\"root\""}))

	if evnt.Data()["audit_proctitle"] != `\x1B[2Jls` {
		t.Errorf("Unexpected set field: %#v", evnt.Data())
	}
	if evnt.Data()["message"] != `proctitle=1B5B324A6C73\x1DUID="root"` {
		t.Errorf("Message was not escaped after the pipeline: %q", evnt.Data()["message"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_audit_proctitle", "_control_chars_message")
}

func TestPoolKeepsControlCharactersWhenDisabled(t *testing.T) {
	pool := &Pool{guard: &timestampGuard{}}

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "a\x1b"}))

	if evnt.Data()["message"] != "a\x1b" {
		t.Errorf("Message was escaped: %q", evnt.Data()["message"])
	}
	assertControlTestTags(t, evnt)
}

func TestPoolEscapesControlCharactersByDefault(t *testing.T) {
	pool := &Pool{guard: &timestampGuard{}}
	pool.applyConfig(config.NewConfig())

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "a\x1b"}))

	if evnt.Data()["message"] != `a\x1B` {
		t.Errorf("Message was not escaped: %q", evnt.Data()["message"])
	}
}
