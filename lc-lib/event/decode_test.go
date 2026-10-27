package event

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// expectedDecode decodes event data as encoding/json does, then normalises it as an event does
func expectedDecode(input []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errNotObject
	}
	(&Event{data: data}).convertData()
	return data, nil
}

// assertDecodesAsJSONUnmarshal checks event data decodes and normalises exactly as through encoding/json
func assertDecodesAsJSONUnmarshal(t *testing.T, input []byte) {
	t.Helper()
	expected, expectedErr := expectedDecode(input)
	decoded, err := decodeData(input)
	if (err == nil) != (expectedErr == nil) {
		t.Fatalf("Expected error %v decoding %q, got %v", expectedErr, input, err)
	}
	if err != nil {
		return
	}
	(&Event{data: decoded}).convertData()

	// A missing or invalid @timestamp is replaced with the time it was normalised, which differs between the two
	var raw map[string]interface{}
	json.Unmarshal(input, &raw)
	if value, ok := raw["@timestamp"].(string); !ok || !validTimestamp(value) {
		delete(expected, "@timestamp")
		delete(decoded, "@timestamp")
	}
	if !reflect.DeepEqual(decoded, expected) {
		t.Fatalf("Expected %q to decode as %#v, got %#v", input, expected, decoded)
	}
}

func validTimestamp(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func TestDecodeDataMatchesJSONUnmarshal(t *testing.T) {
	for name, input := range map[string]string{
		"event":                   string(benchEventJSON),
		"empty":                   `{}`,
		"tags not all strings":    `{"tags":["a",1,"b"]}`,
		"tags string":             `{"tags":"single"}`,
		"tags empty":              `{"tags":[]}`,
		"tags object":             `{"tags":{"a":1}}`,
		"timestamp invalid":       `{"@timestamp":"yesterday"}`,
		"timestamp number":        `{"@timestamp":12}`,
		"timestamp with offset":   `{"@timestamp":"2026-10-27T09:15:02.5+05:30"}`,
		"nested":                  `{"a":{"b":[1,[2,{"c":null}],true,false,"s"]},"d":[]}`,
		"duplicate keys":          `{"a":1,"a":2,"tags":["x"],"tags":["y"]}`,
		"escapes":                 `{"s":"quote \" slash \\ unicode é 😀"}`,
		"invalid utf8":            "{\"s\":\"bad\xffbyte\"}",
		"numbers":                 `{"a":0,"b":-1.5e10,"c":1e-300,"d":123456789012345678901234567890}`,
		"number out of range":     `{"a":1e400}`,
		"whitespace":              " \n{ \"a\" : 1 }\n ",
		"trailing value":          `{} {}`,
		"trailing garbage":        `{}x`,
		"array":                   `[1,2]`,
		"string":                  `"text"`,
		"null":                    `null`,
		"invalid":                 `{"a":}`,
		"unterminated":            `{"a":1`,
		"metadata from the wire":  `{"@metadata":{"x":1}}`,
		"non string tag in front": `{"tags":[1,"a"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			assertDecodesAsJSONUnmarshal(t, []byte(input))
		})
	}
}

func TestEventDataOfJSONNullIsAnUnmarshalFailure(t *testing.T) {
	data := NewEventFromBytes(context.Background(), nil, []byte(`null`)).Data()
	if tags, ok := data["tags"].(Tags); !ok || len(tags) != 1 || tags[0] != "_unmarshal_failure" {
		t.Fatalf("Expected an unmarshal failure, got %#v", data)
	}
}

func FuzzDecodeDataMatchesJSONUnmarshal(f *testing.F) {
	for _, seed := range []string{
		`{}`,
		string(benchEventJSON),
		`{"tags":["a",{"b":[1,null]}],"@timestamp":"2026-10-27T09:15:02Z","x":{"y":[true,false,-0.5]}}`,
		`{"a":" <&>\ud800","b":1e308}`,
		`null`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		assertDecodesAsJSONUnmarshal(t, input)
	})
}
