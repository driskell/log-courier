package event

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// assertMarshalMatches checks encoding a value gives the same JSON as its MarshalJSON method
func assertMarshalMatches(t *testing.T, value json.Marshaler) {
	t.Helper()
	expected, expectedErr := value.MarshalJSON()
	encoded, err := json.Marshal(value)
	if (err == nil) != (expectedErr == nil) {
		t.Fatalf("Expected error %v encoding %#v, got %v", expectedErr, value, err)
	}
	if err == nil && string(encoded) != string(expected) {
		t.Fatalf("Expected %#v to encode as %s, got %s", value, expected, encoded)
	}
}

func TestTimestampEncodesAsItsMarshalJSON(t *testing.T) {
	zone := time.FixedZone("test", 5*3600+30*60)
	for _, value := range []time.Time{
		time.Date(2026, 10, 27, 9, 15, 2, 123456789, time.UTC),
		time.Date(2026, 10, 27, 9, 15, 2, 0, zone),
		time.Date(2026, 10, 27, 9, 15, 2, 120000000, zone),
		time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		assertMarshalMatches(t, Timestamp(value))
	}
}

func TestFloatValuesEncodeAsTheirMarshalJSON(t *testing.T) {
	for _, value := range []float64{0, 1, -2.5, 1234567.875, 1e21, math.SmallestNonzeroFloat64} {
		assertMarshalMatches(t, FloatValue64(value))
		assertMarshalMatches(t, FloatValue32(value))
	}
}
