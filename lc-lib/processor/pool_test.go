package processor

import (
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/event"
)

type recordingTestEntry struct {
	message interface{}
}

func (r *recordingTestEntry) Process(evnt *event.Event) *event.Event {
	r.message = evnt.Data()["message"]
	return evnt
}

func TestPoolEscapesControlCharactersAfterPipeline(t *testing.T) {
	recorder := &recordingTestEntry{}
	pool := &Pool{pipelines: &Config{AST: []ASTEntry{recorder}}, guard: &timestampGuard{}, escapeControl: true}

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "a\x1Db\x1b"}))

	if recorder.message != "a\x1Db\x1b" {
		t.Errorf("Pipeline did not receive the original message: %q", recorder.message)
	}
	if evnt.Data()["message"] != `a\x1Db\x1B` {
		t.Errorf("Message was not escaped after the pipeline: %q", evnt.Data()["message"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_message")
}

func TestPoolEscapesKVResultsAfterSplittingOnControlSeparators(t *testing.T) {
	kv := newKVTestAction(t, map[string]interface{}{"field": "message", "prefix": "audit_", "separators": " \x1D", "unquoted hex": []interface{}{"proctitle"}})
	pool := &Pool{pipelines: &Config{AST: []ASTEntry{kv}}, guard: &timestampGuard{}, escapeControl: true}

	evnt := pool.processEvent(newControlTestEvent(map[string]interface{}{"message": "proctitle=1B5B324A6C73\x1DUID=\"root\""}))

	if evnt.Data()["audit_proctitle"] != `\x1B[2Jls` || evnt.Data()["audit_UID"] != "root" {
		t.Errorf("Unexpected parsed fields: %#v", evnt.Data())
	}
	if evnt.Data()["message"] != `proctitle=1B5B324A6C73\x1DUID="root"` {
		t.Errorf("Message was not escaped after the pipeline: %q", evnt.Data()["message"])
	}
	assertControlTestTags(t, evnt, "_control_chars", "_control_chars_audit_proctitle", "_control_chars_message")
}

func TestPoolKeepsControlCharactersWhenDisabled(t *testing.T) {
	pool := &Pool{pipelines: &Config{}, guard: &timestampGuard{}}

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
