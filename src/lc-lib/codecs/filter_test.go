package codecs

import (
	"github.com/driskell/log-courier/src/lc-lib/core"
	"testing"
)

var filter_lines []string

func createFilterCodec(unused map[string]interface{}, callback core.CodecCallbackFunc, t *testing.T) core.Codec {
	config := core.NewConfig()

	factory, err := NewFilterCodecFactory(config, "", unused, "filter")
	if err != nil {
		t.Logf("Failed to create filter codec: %s", err)
		t.FailNow()
	}

	return factory.NewCodec(callback, 0)
}

func checkFilter(start_offset int64, end_offset int64, text string) {
	filter_lines = append(filter_lines, text)
}

func TestFilter(t *testing.T) {
	filter_lines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$"},
		"negate":   false,
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filter_lines) != 1 {
		t.Logf("Wrong line count received")
		t.FailNow()
	} else if filter_lines[0] != "NEXT line" {
		t.Logf("Wrong line[0] received: %s", filter_lines[0])
		t.FailNow()
	}
}

func TestFilterNegate(t *testing.T) {
	filter_lines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$"},
		"negate":   true,
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filter_lines) != 3 {
		t.Logf("Wrong line count received")
		t.FailNow()
	} else if filter_lines[0] != "DEBUG First line" {
		t.Logf("Wrong line[0] received: %s", filter_lines[0])
		t.FailNow()
	} else if filter_lines[1] != "ANOTHER line" {
		t.Logf("Wrong line[1] received: %s", filter_lines[1])
		t.FailNow()
	} else if filter_lines[2] != "DEBUG Next line" {
		t.Logf("Wrong line[2] received: %s", filter_lines[2])
		t.FailNow()
	}
}

func TestFilterMultiple(t *testing.T) {
	filter_lines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$", "^DEBUG First line$"},
		"negate":   false,
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filter_lines) != 2 {
		t.Logf("Wrong line count received")
		t.FailNow()
	} else if filter_lines[0] != "DEBUG First line" {
		t.Logf("Wrong line[0] received: %s", filter_lines[0])
		t.FailNow()
	} else if filter_lines[1] != "NEXT line" {
		t.Logf("Wrong line[1] received: %s", filter_lines[1])
		t.FailNow()
	}
}
