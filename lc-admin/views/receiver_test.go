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
