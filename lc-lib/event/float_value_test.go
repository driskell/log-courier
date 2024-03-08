package event

import (
	"encoding/json"
	"testing"
)

func TestFloatValue64Round(t *testing.T) {
	result, err := json.Marshal(FloatValue64(1.0))
	if err != nil {
		t.Fatalf("Failed to encode: %s", err)
	}
	if string(result) != "1.0" {
		t.Fatalf("Unexpected result: %s", result)
	}
}

func TestFloatValue64Decimal(t *testing.T) {
	result, err := json.Marshal(FloatValue64(1.5))
	if err != nil {
		t.Fatalf("Failed to encode: %s", err)
	}
	if string(result) != "1.5" {
		t.Fatalf("Unexpected result: %s", result)
	}
}

func TestFloatValue32Round(t *testing.T) {
	result, err := json.Marshal(FloatValue32(1.0))
	if err != nil {
		t.Fatalf("Failed to encode: %s", err)
	}
	if string(result) != "1.0" {
		t.Fatalf("Unexpected result: %s", result)
	}
}

func TestFloatValue32Decimal(t *testing.T) {
	result, err := json.Marshal(FloatValue32(1.5))
	if err != nil {
		t.Fatalf("Failed to encode: %s", err)
	}
	if string(result) != "1.5" {
		t.Fatalf("Unexpected result: %s", result)
	}
}
