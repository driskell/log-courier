package processor

import (
	"context"
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/event"
)

func newTimestampGuardTestEvent(timestamp interface{}) *event.Event {
	return event.NewEvent(context.Background(), nil, map[string]interface{}{"@timestamp": timestamp})
}

func TestTimestampGuardInRange(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, time.Hour)

	evnt := newTimestampGuardTestEvent(now)
	if guard.Check(evnt, now) {
		t.Error("Event within bounds was unexpectedly dropped")
	}
	if evnt.Dropped() {
		t.Error("Event within bounds was unexpectedly marked as dropped")
	}
}

func TestTimestampGuardTooOld(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, time.Hour)

	evnt := newTimestampGuardTestEvent(now.Add(-2 * time.Hour))
	if !guard.Check(evnt, now) {
		t.Error("Event older than the maximum age was not dropped")
	}
	if !evnt.Dropped() {
		t.Error("Event older than the maximum age was not marked as dropped")
	}

	tooOld, future := guard.Dropped()
	if tooOld != 1 || future != 0 {
		t.Errorf("Unexpected drop counters: tooOld=%d future=%d", tooOld, future)
	}
}

func TestTimestampGuardTooFarFuture(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, time.Hour)

	evnt := newTimestampGuardTestEvent(now.Add(2 * time.Hour))
	if !guard.Check(evnt, now) {
		t.Error("Event further ahead than the maximum future age was not dropped")
	}
	if !evnt.Dropped() {
		t.Error("Event further ahead than the maximum future age was not marked as dropped")
	}

	tooOld, future := guard.Dropped()
	if tooOld != 0 || future != 1 {
		t.Errorf("Unexpected drop counters: tooOld=%d future=%d", tooOld, future)
	}
}

func TestTimestampGuardBoundaryExact(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, time.Hour)

	oldest := newTimestampGuardTestEvent(now.Add(-time.Hour))
	if guard.Check(oldest, now) {
		t.Error("Event exactly at the maximum age boundary was unexpectedly dropped")
	}

	newest := newTimestampGuardTestEvent(now.Add(time.Hour))
	if guard.Check(newest, now) {
		t.Error("Event exactly at the maximum future age boundary was unexpectedly dropped")
	}
}

func TestTimestampGuardDisabled(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(0, 0)

	ancient := newTimestampGuardTestEvent(now.Add(-90 * 24 * time.Hour))
	if guard.Check(ancient, now) {
		t.Error("Event was dropped despite maximum age being disabled")
	}

	distant := newTimestampGuardTestEvent(now.Add(90 * 24 * time.Hour))
	if guard.Check(distant, now) {
		t.Error("Event was dropped despite maximum future age being disabled")
	}
}

func TestTimestampGuardDisabledFutureOnly(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, 0)

	ancient := newTimestampGuardTestEvent(now.Add(-2 * time.Hour))
	if !guard.Check(ancient, now) {
		t.Error("Event older than the maximum age was not dropped")
	}

	distant := newTimestampGuardTestEvent(now.Add(90 * 24 * time.Hour))
	if guard.Check(distant, now) {
		t.Error("Event was dropped despite maximum future age being disabled")
	}
}

func TestTimestampGuardCounters(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Hour, time.Hour)

	for i := 0; i < 2; i++ {
		guard.Check(newTimestampGuardTestEvent(now.Add(-2*time.Hour)), now)
	}
	for i := 0; i < 3; i++ {
		guard.Check(newTimestampGuardTestEvent(now.Add(2*time.Hour)), now)
	}

	tooOld, future := guard.Dropped()
	if tooOld != 2 {
		t.Errorf("Unexpected too-old drop count: %d", tooOld)
	}
	if future != 3 {
		t.Errorf("Unexpected future drop count: %d", future)
	}
}

func TestTimestampGuardParseFailureKept(t *testing.T) {
	now := time.Now()
	guard := &timestampGuard{}
	guard.SetLimits(time.Minute, time.Minute)

	// An unparseable @timestamp falls back to the current time, so it should
	// never be dropped by the guard
	evnt := newTimestampGuardTestEvent("not a valid timestamp")
	if guard.Check(evnt, now) {
		t.Error("Event with a timestamp parse failure was unexpectedly dropped")
	}
	if evnt.Dropped() {
		t.Error("Event with a timestamp parse failure was unexpectedly marked as dropped")
	}
}
