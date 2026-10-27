package processor

import (
	"encoding/json"
	"testing"
	"time"
)

func assertRoutinesJSON(t *testing.T, routines *apiRoutines, expected string) {
	t.Helper()
	if err := routines.Update(); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	encoded, err := json.Marshal(routines)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if string(encoded) != expected {
		t.Fatalf("Expected %s, got %s", expected, encoded)
	}
}

func TestAPIRoutinesListsEventsProcessedByEachRunningRoutine(t *testing.T) {
	pool := &Pool{}
	pool.routines.Store(2)
	pool.processed[0].Store(5)
	pool.processed[1].Store(3)
	routines := &apiRoutines{p: pool}
	assertRoutinesJSON(t, routines, `[{"id":0,"processedEvents":5,"speed":0},{"id":1,"processedEvents":3,"speed":0}]`)

	pool.routines.Store(1)
	assertRoutinesJSON(t, routines, `[{"id":0,"processedEvents":5,"speed":0}]`)
}

func TestAPIRoutinesReportsEachRoutineSpeed(t *testing.T) {
	pool := &Pool{}
	pool.routines.Store(1)
	start := time.Now()
	pool.takeMeasurements(start)
	pool.processed[0].Store(50)
	pool.takeMeasurements(start.Add(time.Second))
	assertRoutinesJSON(t, &apiRoutines{p: pool}, `[{"id":0,"processedEvents":50,"speed":50}]`)
}
