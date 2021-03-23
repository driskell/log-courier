package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func resetSections() {
	registeredSectionCreators = make(map[string]SectionCreator)
}

func loadTest(source string, reportUnused bool) (*Config, error) {
	rawConfig := make(map[string]interface{})

	decoder := json.NewDecoder(strings.NewReader(source))
	err := decoder.Decode(&rawConfig)
	if err != nil {
		return nil, err
	}

	// Populate configuration - reporting errors on spelling mistakes etc.
	config := NewConfig()
	if err = parseConfiguration(config, rawConfig, reportUnused); err != nil {
		return nil, err
	}
	return config, nil
}

type TestConfigMultipleSectionsFirst struct {
	Value string `config:"value"`
}

type TestConfigMultipleSectionsSecond struct {
	Value int `config:"value"`
}

type TestConfigMultipleSectionsThird struct {
	Value bool `config:"value"`
}

func TestConfigMultipleSections(t *testing.T) {
	resetSections()
	RegisterSection("first", func() interface{} {
		return &TestConfigMultipleSectionsFirst{}
	})
	RegisterSection("second", func() interface{} {
		return &TestConfigMultipleSectionsSecond{}
	})
	RegisterSection("third", func() interface{} {
		return &TestConfigMultipleSectionsThird{}
	})

	config, err := loadTest(
		"{\"first\":{\"value\":\"testing\"},\"third\":{\"value\":true}}",
		false,
	)
	if err != nil {
		t.Errorf("Failed to parse configuration: %s", err)
		t.FailNow()
	}

	valueFirst := config.Section("first").(*TestConfigMultipleSectionsFirst).Value
	if valueFirst != "testing" {
		t.Errorf("Expected 'testing' received: %s", valueFirst)
	}

	valueSecond := config.Section("second").(*TestConfigMultipleSectionsSecond).Value
	if valueSecond != 0 {
		t.Errorf("Expected 'testing' received: %d", valueSecond)
	}

	valueThird := config.Section("third").(*TestConfigMultipleSectionsThird).Value
	if valueThird != true {
		t.Errorf("Expected 'testing' received: %v", valueThird)
	}
}
