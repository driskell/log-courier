package main

import (
	"testing"
	"time"
)

func TestFormatUptimeLeavesOutLeadingZeroUnits(t *testing.T) {
	for duration, expected := range map[time.Duration]string{
		5 * time.Second:                                      "5s",
		3*time.Minute + 5*time.Second:                        "3m 5s",
		2*time.Hour + 5*time.Second:                          "2h 0m 5s",
		50*time.Hour + 3*time.Minute + 1500*time.Millisecond: "2d 2h 3m 1s",
	} {
		if formatted := formatUptime(duration); formatted != expected {
			t.Errorf("Expected %v to format as %q, got %q", duration, expected, formatted)
		}
	}
}
