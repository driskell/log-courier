package config

import (
	"os"
	"testing"
)

func TestIntEnvVar(t *testing.T) {
	p := &Parser{}

	// Case 1: Missing env var
	os.Unsetenv("TEST_INT_ENV")
	if val, err := p.intEnvVar("%INTENV(TEST_INT_ENV)%"); val != 0 || err != nil {
		t.Errorf("Expected 0 for missing env, got %d (error: %v)", val, err)
	}

	// Case 2: Valid integer env var
	os.Setenv("TEST_INT_ENV", "42")
	if val, err := p.intEnvVar("%INTENV(TEST_INT_ENV)%"); val != 42 || err != nil {
		t.Errorf("Expected 42 for valid env, got %d (error: %v)", val, err)
	}

	// Case 3: Mixed number text (should error and return 0)
	os.Setenv("TEST_INT_ENV", "42abc")
	if val, err := p.intEnvVar("%INTENV(TEST_INT_ENV)%"); val != 0 || err == nil {
		t.Errorf("Expected 0 for mixed env, got %d (error: %v)", val, err)
	}

	os.Unsetenv("TEST_INT_ENV")
}
