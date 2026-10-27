package transports

import (
	"context"
	"testing"
)

func assertSegment(t *testing.T, segment *EventsSegment, offset uint32, skipped uint32, count uint32, size int, final bool) {
	t.Helper()
	if segment.Offset != offset || segment.Skipped != skipped || segment.Count() != count || segment.Size != size || segment.Final != final {
		t.Fatalf("Expected segment at offset %d with %d skipped, %d events of %d bytes and final %v, got offset %d with %d skipped, %d events of %d bytes and final %v",
			offset, skipped, count, size, final, segment.Offset, segment.Skipped, segment.Count(), segment.Size, segment.Final)
	}
}

func TestReceiveWindowHoldsPayloadAsOneSegment(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	window.AddRaw(&nonce, []byte("first"))
	window.AddRaw(&nonce, []byte("second"))
	window.End(&nonce)

	segments := window.Take()
	if len(segments) != 1 {
		t.Fatalf("Expected one segment, got %d", len(segments))
	}
	assertSegment(t, segments[0], 0, 0, 2, 11, true)
}

func TestReceiveWindowContinuesPayloadAfterTake(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	window.AddRaw(&nonce, []byte("first"))
	first := window.Take()
	window.AddRaw(&nonce, []byte("second"))
	window.End(&nonce)
	second := window.Take()

	assertSegment(t, first[0], 0, 0, 1, 5, false)
	assertSegment(t, second[0], 1, 0, 1, 6, true)
}

func TestReceiveWindowEndsPayloadWhoseEventsWereTaken(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	window.AddRaw(&nonce, []byte("first"))
	window.Take()
	window.End(&nonce)

	segments := window.Take()
	assertSegment(t, segments[0], 1, 0, 0, 0, true)
}

func TestReceiveWindowStartsNextPayloadAtFirstEvent(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	first, second := "first", "second"
	window.AddRaw(&first, []byte("a"))
	window.End(&first)
	window.AddRaw(&second, []byte("b"))

	segments := window.Take()
	if len(segments) != 2 || *segments[1].Nonce != "second" {
		t.Fatalf("Expected a segment for each payload, got %d", len(segments))
	}
	assertSegment(t, segments[1], 0, 0, 1, 1, false)
}

func TestReceiveWindowRecordsSkippedEventsBeforeTheEventsThatFollow(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	window.Skip(&nonce)
	window.AddRaw(&nonce, []byte("second"))
	window.Skip(&nonce)
	window.Skip(&nonce)
	window.AddRaw(&nonce, []byte("fifth"))
	window.End(&nonce)

	segments := window.Take()
	if len(segments) != 2 {
		t.Fatalf("Expected two segments, got %d", len(segments))
	}
	assertSegment(t, segments[0], 0, 1, 1, 6, false)
	assertSegment(t, segments[1], 2, 2, 1, 5, true)
}

func TestReceiveWindowAsksToBeTakenOnceFilledToItsFraction(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 0.1)
	nonce := "payload"
	if window.AddRaw(&nonce, []byte("first")) {
		t.Fatal("Window asked to be taken before it was filled to its fraction")
	}
	if !window.AddRaw(&nonce, []byte("second")) {
		t.Fatal("Window did not ask to be taken once filled to its fraction")
	}
	if window.AddRaw(&nonce, []byte("third")) || window.End(&nonce) {
		t.Fatal("Window asked to be taken again before it was taken")
	}
}

func TestReceiveWindowAsksToBeTakenAtEndOfPayload(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	if window.AddRaw(&nonce, []byte("first")) {
		t.Fatal("Window asked to be taken before it was filled to its fraction")
	}
	if !window.End(&nonce) {
		t.Fatal("Window did not ask to be taken at the end of the payload")
	}
}

func TestReceiveWindowAsksToBeTakenOnRequestOnlyWhenHolding(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 100, 1)
	nonce := "payload"
	if window.Request() {
		t.Fatal("An empty window asked to be taken")
	}
	window.AddRaw(&nonce, []byte("first"))
	if !window.Request() {
		t.Fatal("Window holding events did not ask to be taken on request")
	}
	if window.Request() {
		t.Fatal("Window asked to be taken again before it was taken")
	}
	window.Take()
	window.AddRaw(&nonce, []byte("second"))
	if !window.Request() {
		t.Fatal("Window did not ask to be taken again once taken")
	}
}

func TestReceiveWindowFitsEventsUpToItsSize(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 10, 1)
	nonce := "payload"
	if !window.Fits(10) {
		t.Fatal("An empty window did not fit an event of its size")
	}
	window.AddRaw(&nonce, []byte("123456"))
	if !window.Fits(4) || window.Fits(5) {
		t.Fatal("Window did not fit exactly the space it had left")
	}
	window.Take()
	if !window.Fits(10) {
		t.Fatal("Window did not regain its space once taken")
	}
}

func TestReceiveWindowSignalsTake(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 10, 1)
	nonce := "payload"
	window.AddRaw(&nonce, []byte("event"))
	window.Take()
	select {
	case <-window.Taken():
	default:
		t.Fatal("Window did not signal it was taken")
	}
	if !window.Empty() {
		t.Fatal("Window held events after they were taken")
	}
}

func TestReceiveWindowReportsSizeHeld(t *testing.T) {
	window := NewReceiveWindow(context.Background(), 10, 1)
	nonce := "payload"
	window.AddRaw(&nonce, []byte("123456"))
	if held := window.Held(); held != 6 {
		t.Fatalf("Expected 6 bytes held, got %d", held)
	}
	window.Take()
	if held := window.Held(); held != 0 {
		t.Fatalf("Expected nothing held once taken, got %d", held)
	}
}
