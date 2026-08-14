package processor

import (
	"testing"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
)

func TestGeneralDefaultEventBounds(t *testing.T) {
	general := config.NewConfig().GeneralPart("processor").(*General)

	if general.MaximumEventAge != 90*24*time.Hour {
		t.Errorf("Unexpected default maximum event age: %v", general.MaximumEventAge)
	}
	if general.MaximumFutureEventAge != 24*time.Hour {
		t.Errorf("Unexpected default maximum future event age: %v", general.MaximumFutureEventAge)
	}
}

func populateGeneralEventBounds(t *testing.T, input map[string]interface{}) (*General, error) {
	parser := config.NewParser(nil)

	item := &General{}
	err := parser.Populate(item, input, "/general/", false)
	return item, err
}

func TestGeneralEventBoundsFromSeconds(t *testing.T) {
	general, err := populateGeneralEventBounds(t, map[string]interface{}{
		"maximum event age":        300,
		"maximum future event age": 60,
	})
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if general.MaximumEventAge != 5*time.Minute {
		t.Errorf("Unexpected maximum event age: %v", general.MaximumEventAge)
	}
	if general.MaximumFutureEventAge != time.Minute {
		t.Errorf("Unexpected maximum future event age: %v", general.MaximumFutureEventAge)
	}
}

func TestGeneralEventBoundsFromDayUnit(t *testing.T) {
	general, err := populateGeneralEventBounds(t, map[string]interface{}{
		"maximum event age":        "90d",
		"maximum future event age": "1d",
	})
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if general.MaximumEventAge != 90*24*time.Hour {
		t.Errorf("Unexpected maximum event age: %v", general.MaximumEventAge)
	}
	if general.MaximumFutureEventAge != 24*time.Hour {
		t.Errorf("Unexpected maximum future event age: %v", general.MaximumFutureEventAge)
	}
}

func TestGeneralValidateEventBoundsNegative(t *testing.T) {
	general := &General{MaximumEventAge: -time.Second}
	if err := general.Validate(nil, "/general/"); err == nil {
		t.Errorf("Validation succeeded unexpectedly")
	}

	general = &General{MaximumFutureEventAge: -time.Second}
	if err := general.Validate(nil, "/general/"); err == nil {
		t.Errorf("Validation succeeded unexpectedly")
	}
}

func TestGeneralValidateEventBoundsDisabled(t *testing.T) {
	general := &General{MaximumEventAge: 0, MaximumFutureEventAge: 0}
	if err := general.Validate(nil, "/general/"); err != nil {
		t.Errorf("Validation failed unexpectedly: %s", err)
	}
}
