package event

import (
	"context"
	"reflect"
	"testing"
)

func TestEscapeControlKeepsCleanValues(t *testing.T) {
	for _, value := range []string{"", "plain text", "tab\tand\nnewline", `back\slash`, "non\u00a0breaking ünïcödé ✓", "lone \xc2"} {
		escaped, found := EscapeControl(value)
		if found || escaped != value {
			t.Errorf("Clean value %q was changed: %q (found %v)", value, escaped, found)
		}
	}
}

func TestEscapeControlCleanValueDoesNotAllocate(t *testing.T) {
	value := "a reasonably long clean log line with ünïcödé, tabs\tand a newline\n"
	if allocs := testing.AllocsPerRun(100, func() { EscapeControl(value) }); allocs != 0 {
		t.Errorf("Unexpected allocations: %v", allocs)
	}
}

func TestEscapeControlEscapesControlCharacters(t *testing.T) {
	cases := map[string]string{
		"a\x1b[31mred":   `a\x1B[31mred`,
		"over\rwrite":    `over\x0Dwrite`,
		"bell\x07\x00":   `bell\x07\x00`,
		"del\x7f":        `del\x7F`,
		"c1 \u009b2J":    `c1 \xC2\x9B2J`,
		"\x1b":           `\x1B`,
		"tab\tkept\x1b":  "tab\tkept\\x1B",
		"ünï\x1bcödé":    `ünï\x1Bcödé`,
		"nbsp\u00a0\x1b": "nbsp\u00a0\\x1B",
	}
	for value, expected := range cases {
		escaped, found := EscapeControl(value)
		if !found || escaped != expected {
			t.Errorf("Unexpected escape of %q: got %q (found %v), expected %q", value, escaped, found, expected)
		}
	}
}

func TestEscapeControlEscapesBackslashesWhenEscaping(t *testing.T) {
	escaped, found := EscapeControl(`C:\x1B` + "\x1b")
	if !found || escaped != `C:\\x1B\x1B` {
		t.Errorf("Unexpected escape: %q (found %v)", escaped, found)
	}
}

func TestEscapeControlValueEscapesNestedValuesAndKeysInPlace(t *testing.T) {
	data := map[string]interface{}{
		"clean":      "value",
		"number":     5,
		"raw\x1bkey": "raw\x1bvalue",
		"nested": map[string]interface{}{
			"list": []interface{}{"ok", "bad\x07", 1, map[string]interface{}{"deep": "\x1b"}},
		},
	}

	result, found := EscapeControlValue(data)
	if !found {
		t.Fatal("Control characters were not found")
	}
	expected := map[string]interface{}{
		"clean":      "value",
		"number":     5,
		`raw\x1Bkey`: `raw\x1Bvalue`,
		"nested": map[string]interface{}{
			"list": []interface{}{"ok", `bad\x07`, 1, map[string]interface{}{"deep": `\x1B`}},
		},
	}
	if !reflect.DeepEqual(data, expected) || !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected escaped data:\n got      %#v\n expected %#v", data, expected)
	}
}

func TestEscapeControlValueKeepsCleanAndOtherValues(t *testing.T) {
	for _, value := range []interface{}{"clean", 5, true, nil, []byte("\x1b"), map[string]interface{}{"a": []interface{}{"b"}}} {
		result, found := EscapeControlValue(value)
		if found || !reflect.DeepEqual(result, value) {
			t.Errorf("Value %#v was changed: %#v (found %v)", value, result, found)
		}
	}
}

func TestEscapeControlValueCleanMapDoesNotAllocate(t *testing.T) {
	data := map[string]interface{}{"message": "clean line", "nested": map[string]interface{}{"list": []interface{}{"a", 1}}}
	if allocs := testing.AllocsPerRun(100, func() { EscapeControlValue(data) }); allocs != 0 {
		t.Errorf("Unexpected allocations: %v", allocs)
	}
}

func TestEscapeControlValueEscapesTagsAndKeepsThemSorted(t *testing.T) {
	evnt := NewEvent(context.Background(), nil, map[string]interface{}{"tags": []interface{}{"b", "z\x1b", "a"}})

	if _, found := EscapeControlValue(evnt.Data()); !found {
		t.Fatal("Control characters were not found")
	}
	if tags := evnt.Data()["tags"].(Tags); !reflect.DeepEqual(tags, Tags{"a", "b", `z\x1B`}) {
		t.Errorf("Unexpected tags: %q", tags)
	}

	clean := Tags{"a", "b"}
	if _, found := EscapeControlValue(clean); found {
		t.Error("Clean tags were reported as escaped")
	}
}

func TestEscapeControlValueSkipsMetadataAndTimestamp(t *testing.T) {
	evnt := NewEvent(context.Background(), nil, map[string]interface{}{"message": "a\x1b"})
	evnt.MustResolve("@metadata[raw]", "m\x1b")
	timestamp := evnt.Data()["@timestamp"]

	if _, found := EscapeControlValue(evnt.Data()); !found {
		t.Fatal("Control characters were not found")
	}
	if evnt.Data()["message"] != `a\x1B` {
		t.Errorf("Message was not escaped: %q", evnt.Data()["message"])
	}
	if metadata := evnt.MustResolve("@metadata[raw]", nil); metadata != "m\x1b" {
		t.Errorf("Metadata was changed: %q", metadata)
	}
	if evnt.Data()["@timestamp"] != timestamp {
		t.Error("Timestamp was changed")
	}
}
