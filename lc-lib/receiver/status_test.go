package receiver

import (
	"testing"
)

func newTestProgress(count int) *poolEventProgress {
	return &poolEventProgress{count: uint32(count), final: true}
}

func assertProgress(t *testing.T, progress *poolEventProgress, advanced bool, expectedAdvanced bool, expectedAckedThrough uint32) {
	t.Helper()
	if advanced != expectedAdvanced {
		t.Fatalf("Expected advanced to be %v, got %v", expectedAdvanced, advanced)
	}
	if progress.ackedThrough != expectedAckedThrough {
		t.Fatalf("Expected acknowledged through %d, got %d (pending %v)", expectedAckedThrough, progress.ackedThrough, progress.pending)
	}
}

func TestProgressAcknowledgesInOrder(t *testing.T) {
	progress := newTestProgress(10)
	assertProgress(t, progress, progress.acknowledge(1, 4), true, 4)
	assertProgress(t, progress, progress.acknowledge(5, 10), true, 10)
	if !progress.complete() {
		t.Fatal("Progress was not complete after acknowledging every event")
	}
}

func TestProgressHoldsRangesBeyondAGap(t *testing.T) {
	progress := newTestProgress(10)
	assertProgress(t, progress, progress.acknowledge(6, 10), false, 0)
	assertProgress(t, progress, progress.acknowledge(3, 5), false, 0)
	if progress.complete() {
		t.Fatal("Progress was complete while events were unacknowledged")
	}
	assertProgress(t, progress, progress.acknowledge(1, 2), true, 10)
	if len(progress.pending) != 0 {
		t.Fatalf("Pending ranges remained after the gap filled: %v", progress.pending)
	}
}

func TestProgressAdvancesOnlyToTheNextGap(t *testing.T) {
	progress := newTestProgress(10)
	assertProgress(t, progress, progress.acknowledge(8, 10), false, 0)
	assertProgress(t, progress, progress.acknowledge(1, 5), true, 5)
	assertProgress(t, progress, progress.acknowledge(6, 7), true, 10)
}

func TestProgressIgnoresRepeatedAcknowledgements(t *testing.T) {
	progress := newTestProgress(10)
	assertProgress(t, progress, progress.acknowledge(1, 5), true, 5)
	assertProgress(t, progress, progress.acknowledge(1, 5), false, 5)
	assertProgress(t, progress, progress.acknowledge(2, 3), false, 5)
	assertProgress(t, progress, progress.acknowledge(8, 9), false, 5)
	assertProgress(t, progress, progress.acknowledge(8, 9), false, 5)
	if len(progress.pending) != 1 {
		t.Fatalf("Repeated range was held more than once: %v", progress.pending)
	}
}

func TestProgressMergesOverlappingRanges(t *testing.T) {
	progress := newTestProgress(10)
	assertProgress(t, progress, progress.acknowledge(1, 3), true, 3)
	assertProgress(t, progress, progress.acknowledge(2, 6), true, 6)
	assertProgress(t, progress, progress.acknowledge(9, 10), false, 6)
	assertProgress(t, progress, progress.acknowledge(7, 9), true, 10)
}

func TestProgressIsIncompleteUntilEveryEventIsReceived(t *testing.T) {
	progress := &poolEventProgress{count: 4}
	assertProgress(t, progress, progress.acknowledge(1, 4), true, 4)
	if progress.complete() {
		t.Fatal("Progress was complete before the payload was received in full")
	}
	progress.final = true
	if !progress.complete() {
		t.Fatal("Progress was not complete once the payload was received in full and acknowledged")
	}
}
