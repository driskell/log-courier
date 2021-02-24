package codecs

import (
	"testing"

	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
)

var filterLines []string

func createFilterCodec(unused map[string]interface{}, callback codecs.CallbackFunc, t *testing.T) codecs.Codec {
	cfg := config.NewConfig()
	factory, err := NewFilterCodecFactory(config.NewParser(cfg), "", unused, "filter")
	if err != nil {
		t.Logf("Failed to create filter codec: %s", err)
		t.FailNow()
	}

	return codecs.NewCodec(factory, callback, 0)
}

func checkFilter(startOffset int64, endOffset int64, data map[string]interface{}) {
	if message, ok := data["message"].(string); ok {
		filterLines = append(filterLines, message)
	}
}

func codecEvent(codec codecs.Codec, startOffset int64, endOffset int64, data string) {
	codec.ProcessEvent(startOffset, endOffset, map[string]interface{}{"message": data})
}

func TestFilter(t *testing.T) {
	filterLines = make([]string, 0, 1)

	codec := createFilterCodec(map[string]interface{}{
		"patterns": []string{"^NEXT line$"},
	}, checkFilter, t)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

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
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

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
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

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
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line DEBUG another line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

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
