package config

import "testing"

func TestConfigParsePopulatesSectionsFromRawValues(t *testing.T) {
	cfg := NewConfig()
	err := cfg.Parse(map[string]interface{}{
		"general": map[string]interface{}{"persist directory": t.TempDir(), "max queue size": 4096},
	}, false)
	if err != nil {
		t.Fatalf("Parsing failed unexpectedly: %s", err)
	}
	if cfg.General().MaxQueueSize != 4096 {
		t.Fatalf("Expected max queue size 4096, got %d", cfg.General().MaxQueueSize)
	}
}

func TestConfigParseValidates(t *testing.T) {
	cfg := NewConfig()
	err := cfg.Parse(map[string]interface{}{
		"general": map[string]interface{}{"persist directory": t.TempDir(), "receive window fill": 1.5},
	}, false)
	if err == nil {
		t.Fatal("Expected an invalid receive window fill to be rejected")
	}
}
