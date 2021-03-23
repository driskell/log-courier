package config

import (
	"strconv"
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
	// TODO: Fix - it seems the vField.CanSet is false if we pass a struct wrapped in interface, which makes sense as it's a value based pass - we should runtime error this
	t.Skip()

	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"keyed": 678,
	}

	item := TestParserPopulateStructFixture{}
	err := parser.Populate(item, input, "/", false)
	if err == nil {
		t.Errorf("Parsing with struct that cannot be set (no pointer) succeeded unexpectedly")
		t.FailNow()
	}
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

type TestParserPopulateStructSliceInStructFixture struct {
	Slice          []TestParserPopulateStructFixture  `config:"slice"`
	SliceOfPointer []*TestParserPopulateStructFixture `config:"slicep"`
	// TODO: See the test function body
	// SliceOfPointerPointer []**TestParserPopulateStructFixture `config:"slicepp"`
	// PointerSlice          *[]TestParserPopulateStructFixture  `config:"pslice"`
}

func TestParserPopulateStructSliceInStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"slice": []map[string]interface{}{
			{
				"keyed": 100,
			},
			{
				"keyed": 200,
			},
		},
		"slicep": []map[string]interface{}{
			{
				"keyed": 100,
			},
			{
				"keyed": 200,
			},
		},
		"slicepp": []map[string]interface{}{
			{
				"keyed": 100,
			},
			{
				"keyed": 200,
			},
			{
				"keyed": 300,
			},
			{
				"keyed": 400,
			},
			{
				"keyed": 500,
			},
			{
				"keyed": 600,
			},
		},
		"pslice": []map[string]interface{}{
			{
				"keyed": 100,
			},
			{
				"keyed": 200,
			},
		},
		"pslicep": []map[string]interface{}{
			{
				"keyed": 100,
			},
			{
				"keyed": 200,
			},
		},
	}

	item := &TestParserPopulateStructSliceInStructFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item.Slice) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.Slice))
	}
	for index := 0; index < len(item.Slice); index++ {
		if item.Slice[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in Slice property at location %d: %d", index, item.Slice[index].ValueWithKey)
		}
	}
	if len(item.SliceOfPointer) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointer))
	}
	for index := 0; index < len(item.SliceOfPointer); index++ {
		if item.SliceOfPointer[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in SliceOfPointer property at location %d: %d", index, item.SliceOfPointer[index].ValueWithKey)
		}
	}
	// TODO: This currently panics because we're not allocating the values the pointers are pointing to - it should refuse or allocate properly
	// if len(item.SliceOfPointerPointer) != 2 {
	// 	t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointerPointer))
	// }
	// for index := 0; index < len(item.SliceOfPointerPointer); index++ {
	// 	if (*item.SliceOfPointerPointer[index]).ValueWithKey != 100*(index+1) {
	// 		t.Errorf("Unexpected value in SliceOfPointerPointer property at location %d: %d", index, (*item.SliceOfPointerPointer[index]).ValueWithKey)
	// 	}
	// }
	// if len(*item.PointerSlice) != 2 {
	// 	t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSlice))
	// }
	// for index := 0; index < len(*item.PointerSlice); index++ {
	// 	if (*item.PointerSlice)[index].ValueWithKey != 100*(index+1) {
	// 		t.Errorf("Unexpected value in Slice property at location %d: %d", index, (*item.PointerSlice)[index].ValueWithKey)
	// 	}
	// }
}

type TestParserPopulateValueSliceInStructFixture struct {
	Slice []string `config:"slice"`
	// TODO: See the test function body
	// SliceOfPointer []*string `config:"slicep"`
	// TODO: See the test function body
	// SliceOfPointerPointer []**string `config:"slicepp"`
	// PointerSlice          *[]string  `config:"pslice"`
}

func TestParserPopulateValueSliceInStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := map[string]interface{}{
		"slice":   []interface{}{"100", "200"},
		"slicep":  []interface{}{"100", "200"},
		"slicepp": []interface{}{"100", "200", "300", "400", "500", "600"},
		"pslice":  []interface{}{"100", "200"},
		"pslicep": []interface{}{"100", "200"},
	}

	item := &TestParserPopulateValueSliceInStructFixture{}
	err := parser.Populate(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item.Slice) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.Slice))
	}
	for index := 0; index < 2; index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if item.Slice[index] != value {
			t.Errorf("Unexpected value in Slice property at location %d: %s", index, item.Slice[index])
		}
	}
	// TODO: This currently panics because it doesn't know about allocating string pointers
	// if len(item.SliceOfPointer) != 2 {
	// 	t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointer))
	// }
	// for index := 0; index < len(item.SliceOfPointer); index++ {
	// 	value := strconv.FormatInt((int64)(100*(index+1)), 10)
	// 	if *item.SliceOfPointer[index] != value {
	// 		t.Errorf("Unexpected value in SliceOfPointer property at location %d: %s", index, *item.SliceOfPointer[index])
	// 	}
	// }
	// TODO: This currently panics because we're not allocating the values the pointers are pointing to - it should refuse or allocate properly
	// if len(item.SliceOfPointerPointer) != 2 {
	// 	t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointerPointer))
	// }
	// for index := 0; index < len(item.SliceOfPointerPointer); index++ {
	// 	value := strconv.FormatInt((int64)(100*(index+1)), 10)
	// 	if **item.SliceOfPointerPointer[index] != value {
	// 		t.Errorf("Unexpected value in SliceOfPointerPointer property at location %d: %s", index, *(*item.SliceOfPointerPointer[index]))
	// 	}
	// }
	// if len(*item.PointerSlice) != 2 {
	// 	t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSlice))
	// }
	// for index := 0; index < len(*item.PointerSlice); index++ {
	// 	value := strconv.FormatInt((int64)(100*(index+1)), 10)
	// 	if (*item.PointerSlice)[index] != value {
	// 		t.Errorf("Unexpected value in Slice property at location %d: %s", index, (*item.PointerSlice)[index])
	// 	}
	// }
}

type TestParserPopulateSliceStructFixture []TestParserPopulateStructFixture

func TestParserPopulateSliceStruct(t *testing.T) {
	// TODO: This entire thing panics
	t.SkipNow()

	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{
		map[string]interface{}{"keyed": 100},
		map[string]interface{}{"keyed": 200},
	}
	input2 := []interface{}{
		map[string]interface{}{"keyed": 300},
		map[string]interface{}{"keyed": 400},
	}

	item := TestParserPopulateSliceStructFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 2 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 4 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}
}

type TestParserPopulateSliceOfPointerStructFixture []*TestParserPopulateStructFixture

func TestParserPopulateSliceOfPointerStruct(t *testing.T) {
	// TODO: This entire thing panics
	t.SkipNow()

	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{
		map[string]interface{}{"keyed": 100},
		map[string]interface{}{"keyed": 200},
	}
	input2 := []interface{}{
		map[string]interface{}{"keyed": 300},
		map[string]interface{}{"keyed": 400},
	}

	item := TestParserPopulateSliceOfPointerStructFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 2 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 4 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}
}

type TestParserPopulateSliceOfPointerPointerStructFixture []**TestParserPopulateStructFixture

func TestParserPopulateSliceOfPointerPointerStruct(t *testing.T) {
	// TODO: This entire thing panics
	t.SkipNow()

	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{
		map[string]interface{}{"keyed": 100},
		map[string]interface{}{"keyed": 200},
	}
	input2 := []interface{}{
		map[string]interface{}{"keyed": 300},
		map[string]interface{}{"keyed": 400},
	}

	item := TestParserPopulateSliceOfPointerPointerStructFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerPointerStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 2 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if (*item[index]).ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, (*item[index]).ValueWithKey)
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerPointerStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 4 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if (*item[index]).ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, (*item[index]).ValueWithKey)
		}
	}
}

func TestParserPopulatePointerSliceStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{
		map[string]interface{}{"keyed": 100},
		map[string]interface{}{"keyed": 200},
	}
	input2 := []interface{}{
		map[string]interface{}{"keyed": 300},
		map[string]interface{}{"keyed": 400},
	}

	item := TestParserPopulateSliceStructFixture{}
	item2 := &item
	retItem, err := parser.PopulateSlice(item2, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 2 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item2, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 4 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}
}

func TestParserPopulatePointerSliceOfPointerStruct(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{
		map[string]interface{}{"keyed": 100},
		map[string]interface{}{"keyed": 200},
	}
	input2 := []interface{}{
		map[string]interface{}{"keyed": 300},
		map[string]interface{}{"keyed": 400},
	}

	item := TestParserPopulateSliceOfPointerStructFixture{}
	item2 := &item
	retItem, err := parser.PopulateSlice(item2, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceOfPointerStructFixture)

	if len(item) != 2 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item2, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceOfPointerStructFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 4 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		if item[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in slice at location %d: %d", index, item[index].ValueWithKey)
		}
	}
}

type TestParserPopulateSliceFixture []string

func TestParserPopulateSlice(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{"100", "200", "300"}
	input2 := []interface{}{"400", "500", "600"}

	item := TestParserPopulateSliceFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceFixture)

	if len(item) != 3 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, item[index])
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceFixture)

	if len(item) != 6 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, item[index])
		}
	}
}

type TestParserPopulateSliceOfPointerFixture []*string

func TestParserPopulateSliceOfPointer(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{"100", "200", "300"}
	input2 := []interface{}{"400", "500", "600"}

	item := TestParserPopulateSliceOfPointerFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 3 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, *item[index])
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 6 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, *item[index])
		}
	}
}

type TestParserPopulateSliceOfPointerPointerFixture []**string

func TestParserPopulateSliceOfPointerPointer(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{"100", "200", "300"}
	input2 := []interface{}{"400", "500", "600"}

	item := TestParserPopulateSliceOfPointerPointerFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 3 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if **item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, **item[index])
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceOfPointerPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 6 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if **item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, **item[index])
		}
	}
}

func TestParserPopulatePointerSlice(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{"100", "200", "300"}
	input2 := []interface{}{"400", "500", "600"}

	item := TestParserPopulateSliceFixture{}
	item2 := &item
	retItem, err := parser.PopulateSlice(item2, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 3 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, item[index])
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item2, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 6 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, item[index])
		}
	}
}

func TestParserPopulatePointerSliceOfPointer(t *testing.T) {
	config := NewConfig()
	parser := NewParser(config)

	input := []interface{}{"100", "200", "300"}
	input2 := []interface{}{"400", "500", "600"}

	item := TestParserPopulateSliceOfPointerFixture{}
	item2 := &item
	retItem, err := parser.PopulateSlice(item2, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceOfPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 3 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, *item[index])
		}
	}

	// Test appending to existing
	retItem, err = parser.PopulateSlice(item2, input2, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = *retItem.(*TestParserPopulateSliceOfPointerFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	if len(item) != 6 {
		t.Errorf("Unexpected size of slice: %d", len(item))
	}
	for index := 0; index < len(item); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *item[index] != value {
			t.Errorf("Unexpected value in slice at location %d: %s", index, *item[index])
		}
	}
}
