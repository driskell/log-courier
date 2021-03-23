package config

import (
	"io"
	"reflect"
	"testing"
)

type TestParserPopulateStructFixture struct {
	Value        string
	ValueWithKey int `config:"keyed"`
}

func TestParserPopulateStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"keyed": 678,
	}

	item := &TestParserPopulateStructFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if item.Value != "" {
		t.Errorf("Unexpected parse of unkeyed Value property: %s", item.Value)
	}
	if item.ValueWithKey != 678 {
		t.Errorf("Unexpected value of ValueWithKey property: %d", item.ValueWithKey)
	}

	// Double pointer
	input = map[string]interface{}{
		"keyed": 678,
	}

	item = &TestParserPopulateStructFixture{}
	item2 := &item
	err = parser.Populate(item2, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if item.ValueWithKey != 678 {
		t.Errorf("Unexpected value of ValueWithKey property: %d", item.ValueWithKey)
	}
}

func TestParserPopulateReportUnused(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"value":        "testing",
		"unused":       123,
		"keyed":        678,
		"ValueWithKey": 987,
	}

	item := &TestParserPopulateStructFixture{}
	err := parser.Populate(item, input, "/", true)
	if err == nil {
		t.Errorf("Parsing with unused succeeded unexpectedly")
		t.FailNow()
	}

	if item.ValueWithKey != 678 {
		t.Errorf("Unexpected value of ValueWithKey property: %d", item.ValueWithKey)
	}

	// Verify that we exhausted from the map - this isn't a bug it's how we process values
	if _, ok := input["keyed"]; ok {
		t.Errorf("Parsing did not remove used value from map")
	}
	if _, ok := input["ValueWithKey"]; !ok {
		t.Errorf("Parsing removed unexpected value from map")
	}
	if _, ok := input["unused"]; !ok {
		t.Errorf("Parsing removed unexpected value from map")
	}
	if _, ok := input["value"]; !ok {
		t.Errorf("Parsing removed unexpected value from map")
	}
}

func TestParserPopulateStructNoPointer(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"keyed": 678,
	}

	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("Parsing with struct that cannot be set (no pointer) succeeded unexpectedly")
		}
	}()

	item := TestParserPopulateStructFixture{}
	parser.Populate(item, input, "/", false)
}

type TestParserPopulateEmbeddedStructFixture struct {
	Inner          *TestParserPopulateStructFixture `config:"inner"`
	InnerNoPointer TestParserPopulateStructFixture  `config:"innernp"`
}

func TestParserPopulateEmbeddedStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"inner": map[string]interface{}{
			"keyed": 678,
		},
		"innernp": map[string]interface{}{
			"keyed": 123,
		},
	}

	item := &TestParserPopulateEmbeddedStructFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if item.Inner.ValueWithKey != 678 {
		t.Errorf("Unexpected value of Inner.ValueWithKey property: %d", item.Inner.ValueWithKey)
	}
	if item.InnerNoPointer.ValueWithKey != 123 {
		t.Errorf("Unexpected value of InnerNoPointer.ValueWithKey property: %d", item.InnerNoPointer.ValueWithKey)
	}
}

var TestParserPopulateStructCallbacksCalled []string

type TestParserPopulateStructCallbacksFixture struct {
	Value        string
	ValueWithKey int `config:"keyed"`
}

func (f TestParserPopulateStructCallbacksFixture) Defaults() {
	TestParserPopulateStructCallbacksCalled = append(TestParserPopulateStructCallbacksCalled, "defaults")
}

func (f TestParserPopulateStructCallbacksFixture) Init(p *Parser, path string) error {
	TestParserPopulateStructCallbacksCalled = append(TestParserPopulateStructCallbacksCalled, "init")
	return nil
}

func (f TestParserPopulateStructCallbacksFixture) Validate(p *Parser, path string) error {
	TestParserPopulateStructCallbacksCalled = append(TestParserPopulateStructCallbacksCalled, "validate")
	return nil
}

func TestParserPopulateStructCallbacks(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	TestParserPopulateStructCallbacksCalled = make([]string, 0, 0)

	input := map[string]interface{}{
		"keyed": 678,
	}

	item := &TestParserPopulateStructCallbacksFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	err = parser.validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %s", err)
	}

	if !reflect.DeepEqual(TestParserPopulateStructCallbacksCalled, []string{"defaults", "init", "validate"}) {
		t.Errorf("Unexpected or missing callback; Expected: %v Received: %v", []string{"defaults", "init", "validate"}, TestParserPopulateStructCallbacksCalled)
	}
}

type TestParserPopulateStructCallbacksInitErrorFixture struct {
	Value        string
	ValueWithKey int `config:"keyed"`
}

func (f TestParserPopulateStructCallbacksInitErrorFixture) Init(p *Parser, path string) error {
	return io.EOF
}

func TestParserPopulateStructCallbacksInitError(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"keyed": 678,
	}

	item := &TestParserPopulateStructCallbacksInitErrorFixture{}
	err := parser.Populate(item, input, "/", false)
	if err == nil {
		t.Error("Parsing succeeded unexpectedly")
		t.FailNow()
	}
}

type TestParserPopulateStructCallbacksValidateErrorFixture struct {
	Value        string
	ValueWithKey int `config:"keyed"`
}

func (f TestParserPopulateStructCallbacksValidateErrorFixture) Validate(p *Parser, path string) error {
	return io.EOF
}

func TestParserPopulateStructCallbacksValidateError(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"keyed": 678,
	}

	item := &TestParserPopulateStructCallbacksValidateErrorFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	err = parser.validate()
	if err == nil {
		t.Errorf("Unexpected validation success")
	}
}
