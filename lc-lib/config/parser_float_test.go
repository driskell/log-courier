package config

import "testing"

type TestParserPopulateFloatFixture struct {
	Value float64 `config:"value"`
}

func populateFloat(t *testing.T, input interface{}) (float64, error) {
	parser := NewParser(nil)

	item := &TestParserPopulateFloatFixture{}
	err := parser.Populate(item, map[string]interface{}{"value": input}, "/", false)
	return item.Value, err
}

func TestParserPopulateFloatFromFloat(t *testing.T) {
	value, err := populateFloat(t, 0.8)
	if err != nil {
		t.Fatalf("Parsing failed unexpectedly: %s", err)
	}
	if value != 0.8 {
		t.Fatalf("Unexpected value: %v", value)
	}
}

func TestParserPopulateFloatFromInt(t *testing.T) {
	value, err := populateFloat(t, 1)
	if err != nil {
		t.Fatalf("Parsing failed unexpectedly: %s", err)
	}
	if value != 1 {
		t.Fatalf("Unexpected value: %v", value)
	}
}

func TestParserPopulateFloatRejectsString(t *testing.T) {
	if _, err := populateFloat(t, "0.8"); err == nil {
		t.Fatal("Expected a string to be rejected")
	}
}
