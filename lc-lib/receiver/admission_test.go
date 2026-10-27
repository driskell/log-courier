package receiver

import (
	"context"
	"testing"

	"github.com/driskell/log-courier/lc-lib/transports"
)

// admissionTestState is what the admission control sees of the queue
type admissionTestState struct {
	queued map[interface{}]bool
	full   bool
}

func (s *admissionTestState) admit(admission *admissionControl) []interface{} {
	var granted []interface{}
	admission.admit(
		func(connection interface{}) bool { return s.queued[connection] },
		func() bool { return !s.full },
		func(request *admissionRequest) { granted = append(granted, request.connection) },
	)
	return granted
}

func requestAdmission(admission *admissionControl, connection interface{}) {
	admission.request(connection, transports.NewReceiveWindow(context.Background(), 1024, 1))
}

func assertGranted(t *testing.T, granted []interface{}, expected ...interface{}) {
	t.Helper()
	if len(granted) != len(expected) {
		t.Fatalf("Expected %v to be admitted, got %v", expected, granted)
	}
	for idx := range expected {
		if granted[idx] != expected[idx] {
			t.Fatalf("Expected %v to be admitted, got %v", expected, granted)
		}
	}
}

func TestAdmissionAdmitsConnectionWithNothingQueuedWhileQueueFull(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{}, full: true}
	requestAdmission(admission, "idle")
	assertGranted(t, state.admit(admission), "idle")
}

func TestAdmissionHoldsConnectionWithEventsQueuedUntilRoom(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{"busy": true}, full: true}
	requestAdmission(admission, "busy")
	assertGranted(t, state.admit(admission))

	state.full = false
	assertGranted(t, state.admit(admission), "busy")
}

func TestAdmissionAdmitsConnectionsWithNothingQueuedFirst(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{"busy": true}}
	requestAdmission(admission, "busy")
	requestAdmission(admission, "idle")
	assertGranted(t, state.admit(admission), "idle", "busy")
}

func TestAdmissionAdmitsSmallSenderBeforeLargeSender(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{"heavy": true, "light": true}}
	admission.charge("heavy", 1<<20)
	admission.charge("light", 4096)
	requestAdmission(admission, "heavy")
	requestAdmission(admission, "light")
	assertGranted(t, state.admit(admission), "light", "heavy")
}

func TestAdmissionForgetsConnectionThatHasGone(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{}, full: true}
	admission.charge("gone", 100)
	requestAdmission(admission, "gone")
	requestAdmission(admission, "next")

	admission.forget("gone")
	assertGranted(t, state.admit(admission), "next")
	if _, ok := admission.credit["gone"]; ok {
		t.Fatal("Forgotten connection kept its credit")
	}
}

func TestAdmissionAdmitsAfterLargeChargeWithoutSpinning(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{"huge": true}}
	admission.charge("huge", 1<<30)
	requestAdmission(admission, "huge")
	assertGranted(t, state.admit(admission), "huge")
}

func TestAdmissionForgivesDebtOfConnectionWithNothingQueued(t *testing.T) {
	admission := newAdmissionControl()
	state := &admissionTestState{queued: map[interface{}]bool{}}
	admission.charge("trickle", 1<<30)
	requestAdmission(admission, "trickle")
	assertGranted(t, state.admit(admission), "trickle")
	admission.charge("trickle", 100)

	state.queued = map[interface{}]bool{"trickle": true, "newcomer": true}
	admission.charge("newcomer", 100)
	requestAdmission(admission, "trickle")
	requestAdmission(admission, "newcomer")
	assertGranted(t, state.admit(admission), "trickle", "newcomer")
}
