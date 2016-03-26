package codecs

import (
	"testing"

	"github.com/driskell/log-courier/lc-lib/config"
)

var filterLines []string

func createFilterCodec(unused map[string]interface{}, callback CallbackFunc, t *testing.T) Codec {
	config := config.NewConfig()

	factory, err := NewFilterCodecFactory(config, "", unused, "filter")
	if err != nil {
		t.Logf("Failed to create filter codec: %s", err)
		t.FailNow()
	}

	return NewCodec(factory, callback, 0)
}

func checkFilter(startOffset int64, endOffset int64, text string) {
	filterLines = append(filterLines, text)
}

func TestFilter(t *testing.T) {
	filterLines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$"},
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filterLines) != 1 {
		t.Error("Wrong line count received")
	} else if filterLines[0] != "NEXT line" {
		t.Errorf("Wrong line[0] received: %s", filterLines[0])
	}

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestFilterNegate(t *testing.T) {
	filterLines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"!^NEXT line$"},
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filterLines) != 3 {
		t.Error("Wrong line count received")
	} else if filterLines[0] != "DEBUG First line" {
		t.Errorf("Wrong line[0] received: %s", filterLines[0])
	} else if filterLines[1] != "ANOTHER line" {
		t.Errorf("Wrong line[1] received: %s", filterLines[1])
	} else if filterLines[2] != "DEBUG Next line" {
		t.Errorf("Wrong line[2] received: %s", filterLines[2])
	}

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestFilterMultiple(t *testing.T) {
	filterLines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$", "=^DEBUG First line$"},
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filterLines) != 2 {
		t.Error("Wrong line count received")
	} else if filterLines[0] != "DEBUG First line" {
		t.Errorf("Wrong line[0] received: %s", filterLines[0])
	} else if filterLines[1] != "NEXT line" {
		t.Errorf("Wrong line[1] received: %s", filterLines[1])
	}

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestFilterMultipleAll(t *testing.T) {
	filterLines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line", "=DEBUG another line$"},
		"match":    "all",
	}, checkFilter, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line DEBUG another line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if len(filterLines) != 1 {
		t.Error("Wrong line count received")
	} else if filterLines[0] != "NEXT line DEBUG another line" {
		t.Errorf("Wrong line[0] received: %s", filterLines[0])
	}

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}
