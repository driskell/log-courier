package views

import "testing"

func TestStyleCellColoursTextWithoutReadingBracketsAsStyling(t *testing.T) {
	if styled := styleCell("[::1]:1234", "red"); styled != "[(::1):1234](fg:red)" {
		t.Fatalf("Unexpected styled cell %q", styled)
	}
	if styled := styleCell("plain", ""); styled != "plain" {
		t.Fatalf("Expected text without a colour unchanged, got %q", styled)
	}
}

func TestReceiverStatusColourMarksConnectionsThatAreEnding(t *testing.T) {
	for status, expected := range map[string]string{"active": "", "closing": "yellow", "closed": "yellow", "failed": "red"} {
		if colour := receiverStatusColour(status); colour != expected {
			t.Errorf("Expected status %q shown in %q, got %q", status, expected, colour)
		}
	}
}

func TestReceiverLineCellsShowWhatEachConnectionReports(t *testing.T) {
	received := int64(1234)
	for name, check := range map[string]struct {
		received    *int64
		supportsAck bool
		acked       string
		receivedOut string
	}{
		"with acknowledgements":    {&received, true, "459", "1234"},
		"without acknowledgements": {&received, false, "-", "1234"},
		"older remote":             {nil, true, "459", "-"},
	} {
		acked, receivedOut := receiverLineCells(459, check.received, check.supportsAck)
		if acked != check.acked || receivedOut != check.receivedOut {
			t.Errorf("%s: expected %s / %s, got %s / %s", name, check.acked, check.receivedOut, acked, receivedOut)
		}
	}
}

func TestReceiveBufferLabelShowsAnythingHeld(t *testing.T) {
	if label := receiveBufferLabel(0, 10); label != "<1%" {
		t.Errorf("Expected <1%% for a little held, got %q", label)
	}
	if label := receiveBufferLabel(0, 0); label != "" {
		t.Errorf("Expected the gauge's own label when empty, got %q", label)
	}
	if label := receiveBufferLabel(45, 1000); label != "" {
		t.Errorf("Expected the gauge's own label from 1%%, got %q", label)
	}
}
