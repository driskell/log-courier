package endpoint

import (
	"testing"
	"time"
)

func TestLatencyMillisecondsKeepsFractions(t *testing.T) {
	if latency := LatencyMilliseconds(250 * time.Microsecond); latency != 0.25 {
		t.Fatalf("Expected 0.25 milliseconds, got %v", latency)
	}
}
