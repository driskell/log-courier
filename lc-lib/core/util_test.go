package core

import (
	"testing"
	"time"
)

func TestSpeedMeterMeasuresIncreasePerSecond(t *testing.T) {
	meter := &SpeedMeter{}
	start := time.Now()
	if speed := meter.Measure(100, start); speed != 0 {
		t.Fatalf("Expected no speed from a single measurement, got %v", speed)
	}
	if speed := meter.Measure(300, start.Add(2*time.Second)); speed != 100 {
		t.Fatalf("Expected 100 per second, got %v", speed)
	}
}

func TestSpeedMeterFallsToZeroWhenCounterStops(t *testing.T) {
	meter := &SpeedMeter{}
	start := time.Now()
	meter.Measure(0, start)
	meter.Measure(100, start.Add(time.Second))
	for second := 2; second <= 6; second++ {
		meter.Measure(100, start.Add(time.Duration(second)*time.Second))
	}
	if speed := meter.Speed(); speed != 0 {
		t.Fatalf("Expected speed to reach 0 once the counter stopped, got %v", speed)
	}
}
