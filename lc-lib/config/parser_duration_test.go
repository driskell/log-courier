package config

import (
	"testing"
	"time"
)

type TestParserPopulateDurationFixture struct {
	Value time.Duration `config:"value"`
}

func populateDuration(t *testing.T, input interface{}) (time.Duration, error) {
	parser := NewParser(nil)

	item := &TestParserPopulateDurationFixture{}
	err := parser.Populate(item, map[string]interface{}{"value": input}, "/", false)
	return item.Value, err
}

func TestParserPopulateDurationSecondsInt(t *testing.T) {
	value, err := populateDuration(t, 300)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 5*time.Minute {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationSecondsFloat(t *testing.T) {
	// JSON decodes numbers as float64, so this path matters as much as the int one
	value, err := populateDuration(t, float64(300))
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 5*time.Minute {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationSecondsFloatFractional(t *testing.T) {
	_, err := populateDuration(t, 1.5)
	if err == nil {
		t.Errorf("Parsing succeeded unexpectedly")
		t.FailNow()
	}
}

func TestParserPopulateDurationString(t *testing.T) {
	value, err := populateDuration(t, "720h")
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 720*time.Hour {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationDayUnit(t *testing.T) {
	value, err := populateDuration(t, "90d")
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 90*24*time.Hour {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationDayUnitCompound(t *testing.T) {
	value, err := populateDuration(t, "1d12h30m")
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 24*time.Hour+12*time.Hour+30*time.Minute {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationDayUnitFractional(t *testing.T) {
	value, err := populateDuration(t, "2.5d")
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if value != 60*time.Hour {
		t.Errorf("Unexpected duration: %v", value)
	}
}

func TestParserPopulateDurationInvalidUnit(t *testing.T) {
	_, err := populateDuration(t, "90x")
	if err == nil {
		t.Errorf("Parsing succeeded unexpectedly")
		t.FailNow()
	}
}

func TestParserPopulateDurationInvalidType(t *testing.T) {
	_, err := populateDuration(t, true)
	if err == nil {
		t.Errorf("Parsing succeeded unexpectedly")
		t.FailNow()
	}
}
