package event

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

// assertEncodesAsJSONMarshal checks event data encodes exactly as encoding/json encodes it without @metadata
func assertEncodesAsJSONMarshal(t *testing.T, data map[string]interface{}) {
	t.Helper()
	pruned := make(map[string]interface{}, len(data))
	for key, value := range data {
		if key != MetadataKey {
			pruned[key] = value
		}
	}
	expected, expectedErr := json.Marshal(pruned)
	encoded, err := encodeData(data)
	if (err == nil) != (expectedErr == nil) {
		t.Fatalf("Expected error %v encoding %#v, got %v", expectedErr, data, err)
	}
	if err != nil {
		return
	}
	// @timestamp may be written ahead of keys that sort before it, so only the decoded data can be compared then
	if _, hasTimestamp := pruned["@timestamp"]; hasTimestamp && sortsBeforeTimestamp(pruned) {
		var expectedData, encodedData interface{}
		json.Unmarshal(expected, &expectedData)
		json.Unmarshal(encoded, &encodedData)
		if !reflect.DeepEqual(encodedData, expectedData) {
			t.Fatalf("Expected %#v to encode as %s, got %s", data, expected, encoded)
		}
		return
	}
	if string(encoded) != string(expected) {
		t.Fatalf("Expected %#v to encode as %s, got %s", data, expected, encoded)
	}
}

// sortsBeforeTimestamp returns true if any key sorts before @timestamp
func sortsBeforeTimestamp(data map[string]interface{}) bool {
	for key := range data {
		if key < "@timestamp" {
			return true
		}
	}
	return false
}

func TestEncodeDataMatchesJSONMarshal(t *testing.T) {
	for name, data := range map[string]map[string]interface{}{
		"strings":        {"plain": "text", "html": "<a href=\"x\">&</a>", "separators": "line para ", "invalid": "bad\xffbyte", "control": "tab\tnew\nnul\x00", "empty": ""},
		"numbers":        {"zero": 0.0, "negative zero": math.Copysign(0, -1), "fraction": 1.5, "large": 1e21, "small": 1e-7, "integer": 123456789.0},
		"literals":       {"true": true, "false": false, "null": nil},
		"nested":         {"fields": map[string]interface{}{"b": "2", "a": []interface{}{"x", 1.0, map[string]interface{}{"z": nil}}}},
		"nil containers": {"map": map[string]interface{}(nil), "slice": []interface{}(nil), "tags": Tags(nil)},
		"event types": {
			"@timestamp": Timestamp(time.Date(2026, 10, 27, 9, 15, 2, 123456789, time.FixedZone("test", 3600))),
			"tags":       Tags{"b", "a"},
			"float64":    FloatValue64(2),
			"float32":    FloatValue32(2.5),
		},
		"other types":         {"number": json.Number("12.50"), "int": int64(42), "strings": []string{"a"}, "map": map[string]string{"k": "v"}},
		"metadata":            {"message": "kept", MetadataKey: Metadata{"receiver": "left out"}},
		"nested metadata key": {"fields": map[string]interface{}{MetadataKey: "kept when nested"}},
		"unencodable float":   {"nan": math.NaN()},
		"unencodable year":    {"@timestamp": Timestamp(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC))},
	} {
		t.Run(name, func(t *testing.T) {
			assertEncodesAsJSONMarshal(t, data)
		})
	}
}

func FuzzEncodeDataMatchesJSONMarshal(f *testing.F) {
	for _, seed := range []string{
		`{}`,
		`{"message":"hello","tags":["a","b"],"fields":{"n":1.5,"ok":true,"none":null}}`,
		`{"html":"<&>","unicode":" é","nested":[[1,{"x":[]}],{}]}`,
		`{"@metadata":{"left":"out"},"big":1e300,"small":-1e-300}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		var data map[string]interface{}
		if json.Unmarshal(input, &data) != nil || data == nil {
			return
		}
		assertEncodesAsJSONMarshal(t, data)
	})
}
