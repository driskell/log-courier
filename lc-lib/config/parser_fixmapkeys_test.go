package config

import (
	"testing"
)

func TestFixMapKeysSimple(t *testing.T) {
	parser := NewParser(nil)

	input := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	err := parser.FixMapKeys("/", input)
	if err != nil {
		t.Errorf("FixMapKeys failed unexpectedly: %s", err)
	}

	if input["key1"] != "value1" {
		t.Errorf("Unexpected value for key1: %v", input["key1"])
	}
	if input["key2"] != 123 {
		t.Errorf("Unexpected value for key2: %v", input["key2"])
	}
}

func TestFixMapKeysInterfaceMap(t *testing.T) {
	parser := NewParser(nil)

	input := map[string]interface{}{
		"nested": map[interface{}]interface{}{
			"key1": "value1",
			"key2": 456,
		},
	}

	err := parser.FixMapKeys("/", input)
	if err != nil {
		t.Errorf("FixMapKeys failed unexpectedly: %s", err)
	}

	// Check that the interface{} map was converted to string keys
	nested, ok := input["nested"].(map[string]interface{})
	if !ok {
		t.Errorf("Nested map was not converted to map[string]interface{}")
		t.FailNow()
	}

	if nested["key1"] != "value1" {
		t.Errorf("Unexpected value for nested key1: %v", nested["key1"])
	}
	if nested["key2"] != 456 {
		t.Errorf("Unexpected value for nested key2: %v", nested["key2"])
	}
}

func TestFixMapKeysDeeplyNested(t *testing.T) {
	parser := NewParser(nil)

	input := map[string]interface{}{
		"level1": map[interface{}]interface{}{
			"level2": map[interface{}]interface{}{
				"key1": "value1",
				"key2": 789,
			},
		},
	}

	err := parser.FixMapKeys("/", input)
	if err != nil {
		t.Errorf("FixMapKeys failed unexpectedly: %s", err)
	}

	// Check deeply nested structure
	level1, ok := input["level1"].(map[string]interface{})
	if !ok {
		t.Errorf("Level 1 map was not converted")
		t.FailNow()
	}

	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Errorf("Level 2 map was not converted")
		t.FailNow()
	}

	if level2["key1"] != "value1" {
		t.Errorf("Unexpected value for deeply nested key1: %v", level2["key1"])
	}
}

func TestFixMapKeysSliceWithMap(t *testing.T) {
	parser := NewParser(nil)

	input := map[string]interface{}{
		"containers": []interface{}{
			map[interface{}]interface{}{
				"name": "container1",
				"metadata": map[interface{}]interface{}{
					"version": "1.0",
					"tags": []interface{}{
						map[interface{}]interface{}{
							"key":   "env",
							"value": "prod",
						},
					},
				},
			},
		},
	}

	err := parser.FixMapKeys("/", input)
	if err != nil {
		t.Errorf("FixMapKeys failed unexpectedly: %s", err)
	}

	// Check the structure was properly converted
	containers, ok := input["containers"].([]interface{})
	if !ok {
		t.Errorf("Containers is not a slice")
		t.FailNow()
	}

	container1, ok := containers[0].(map[string]interface{})
	if !ok {
		t.Errorf("Container was not converted")
		t.FailNow()
	}

	metadata, ok := container1["metadata"].(map[string]interface{})
	if !ok {
		t.Errorf("Metadata was not converted")
		t.FailNow()
	}

	tags, ok := metadata["tags"].([]interface{})
	if !ok {
		t.Errorf("Tags is not a slice: %T", metadata["tags"])
		t.FailNow()
	}

	tag, ok := tags[0].(map[string]interface{})
	if !ok {
		t.Errorf("Tag was not converted to map[string]interface{}, got %T", tags[0])
		t.FailNow()
	}

	if tag["key"] != "env" {
		t.Errorf("Unexpected tag key: %v", tag["key"])
	}
	if tag["value"] != "prod" {
		t.Errorf("Unexpected tag value: %v", tag["value"])
	}
}
