package config

import (
	"io"
	"reflect"
	"strconv"
	"testing"
)

type TestParserPopulateStructSliceInStructFixture struct {
	Slice                 []TestParserPopulateStructFixture   `config:"slice"`
	SliceOfPointer        []*TestParserPopulateStructFixture  `config:"slicep"`
	SliceOfPointerPointer []**TestParserPopulateStructFixture `config:"slicepp"`
	PointerSlice          *[]TestParserPopulateStructFixture  `config:"pslice"`
	PointerSliceOfPointer *[]*TestParserPopulateStructFixture `config:"pslicep"`
}

func TestParserPopulateStructSliceInStruct(t *testing.T) {
	parser := NewParser(nil)

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
	if len(item.SliceOfPointerPointer) != 6 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointerPointer))
	}
	for index := 0; index < len(item.SliceOfPointerPointer); index++ {
		if (*item.SliceOfPointerPointer[index]).ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in SliceOfPointerPointer property at location %d: %d", index, (*item.SliceOfPointerPointer[index]).ValueWithKey)
		}
	}
	if len(*item.PointerSlice) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSlice))
	}
	for index := 0; index < len(*item.PointerSlice); index++ {
		if (*item.PointerSlice)[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in Slice property at location %d: %d", index, (*item.PointerSlice)[index].ValueWithKey)
		}
	}
	if len(*item.PointerSliceOfPointer) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSliceOfPointer))
	}
	for index := 0; index < len(*item.PointerSliceOfPointer); index++ {
		if (*item.PointerSliceOfPointer)[index].ValueWithKey != 100*(index+1) {
			t.Errorf("Unexpected value in Slice property at location %d: %d", index, (*item.PointerSliceOfPointer)[index].ValueWithKey)
		}
	}
}

type TestParserPopulateValueSliceInStructFixture struct {
	Slice                 []string   `config:"slice"`
	SliceOfPointer        []*string  `config:"slicep"`
	SliceOfPointerPointer []**string `config:"slicepp"`
	PointerSlice          *[]string  `config:"pslice"`
	PointerSliceOfPointer *[]*string `config:"pslicep"`
}

func TestParserPopulateValueSliceInStruct(t *testing.T) {
	parser := NewParser(nil)

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
	if len(item.SliceOfPointer) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointer))
	}
	for index := 0; index < len(item.SliceOfPointer); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *item.SliceOfPointer[index] != value {
			t.Errorf("Unexpected value in SliceOfPointer property at location %d: %s", index, *item.SliceOfPointer[index])
		}
	}
	if len(item.SliceOfPointerPointer) != 6 {
		t.Errorf("Unexpected size of Slice property: %d", len(item.SliceOfPointerPointer))
	}
	for index := 0; index < len(item.SliceOfPointerPointer); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if **item.SliceOfPointerPointer[index] != value {
			t.Errorf("Unexpected value in SliceOfPointerPointer property at location %d: %s", index, *(*item.SliceOfPointerPointer[index]))
		}
	}
	if len(*item.PointerSlice) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSlice))
	}
	for index := 0; index < len(*item.PointerSlice); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if (*item.PointerSlice)[index] != value {
			t.Errorf("Unexpected value in Slice property at location %d: %s", index, (*item.PointerSlice)[index])
		}
	}
	if len(*item.PointerSliceOfPointer) != 2 {
		t.Errorf("Unexpected size of Slice property: %d", len(*item.PointerSliceOfPointer))
	}
	for index := 0; index < len(*item.PointerSliceOfPointer); index++ {
		value := strconv.FormatInt((int64)(100*(index+1)), 10)
		if *(*item.PointerSliceOfPointer)[index] != value {
			t.Errorf("Unexpected value in Slice property at location %d: %s", index, *(*item.PointerSliceOfPointer)[index])
		}
	}
}

type TestParserPopulateSliceStructFixture []TestParserPopulateStructFixture

func TestParserPopulateSliceStruct(t *testing.T) {
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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
	parser := NewParser(nil)

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

var TestParserPopulateSliceCallbacksCalled []string

type TestParserPopulateSliceCallbacksFixture []string

func (f TestParserPopulateSliceCallbacksFixture) Defaults() {
	TestParserPopulateSliceCallbacksCalled = append(TestParserPopulateSliceCallbacksCalled, "defaults")
}

func (f TestParserPopulateSliceCallbacksFixture) Init(p *Parser, path string) error {
	TestParserPopulateSliceCallbacksCalled = append(TestParserPopulateSliceCallbacksCalled, "init")
	return nil
}

func (f TestParserPopulateSliceCallbacksFixture) Validate(p *Parser, path string) error {
	TestParserPopulateSliceCallbacksCalled = append(TestParserPopulateSliceCallbacksCalled, "validate")
	return nil
}

func TestParserPopulateSliceCallbacks(t *testing.T) {
	parser := NewParser(nil)

	TestParserPopulateSliceCallbacksCalled = make([]string, 0, 0)

	input := []interface{}{"100", "200"}

	item := TestParserPopulateSliceCallbacksFixture{}
	retItem, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}
	item = retItem.(TestParserPopulateSliceCallbacksFixture)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	err = parser.validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %s", err)
	}

	if !reflect.DeepEqual(TestParserPopulateSliceCallbacksCalled, []string{"init", "validate"}) {
		t.Errorf("Unexpected or missing callback; Expected: %v Received: %v", []string{"init", "validate"}, TestParserPopulateSliceCallbacksCalled)
	}
}

type TestParserPopulateSliceCallbacksInitErrorFixture []string

func (f TestParserPopulateSliceCallbacksInitErrorFixture) Init(p *Parser, path string) error {
	return io.EOF
}

func TestParserPopulateSliceCallbacksInitError(t *testing.T) {
	parser := NewParser(nil)

	input := []interface{}{"100", "200"}

	item := TestParserPopulateSliceCallbacksInitErrorFixture{}
	_, err := parser.PopulateSlice(item, input, "/", false)
	if err == nil {
		t.Error("Parsing succeeded unexpectedly")
		t.FailNow()
	}
}

type TestParserPopulateSliceCallbacksValidateErrorFixture []string

func (f TestParserPopulateSliceCallbacksValidateErrorFixture) Validate(p *Parser, path string) error {
	return io.EOF
}

func TestParserPopulateSliceCallbacksValidateError(t *testing.T) {
	parser := NewParser(nil)

	input := []interface{}{"100", "200"}

	item := TestParserPopulateSliceCallbacksValidateErrorFixture{}
	_, err := parser.PopulateSlice(item, input, "/", false)
	if err != nil {
		t.Errorf("Parsing failed unexpectedly: %s", err)
		t.FailNow()
	}

	err = parser.validate()
	if err == nil {
		t.Errorf("Unexpected validation success")
	}
}
