package codecs

import (
  "lc-lib/core"
  "testing"
)

var gt *testing.T
var lines int

func createCodec(unused map[string]interface{}, callback core.CodecCallbackFunc, t *testing.T) core.Codec {
  config := core.NewConfig()
  config.General.MaxLineBytes = 1048576
  config.General.SpoolMaxBytes = 10485760

  factory, err := NewMultilineCodecFactory(config, "", unused, "multiline")
  if err != nil {
    t.Logf("Failed to create multiline codec: %s", err)
    t.FailNow()
  }

  return factory.NewCodec(callback, 0)
}

func checkMultiline(start_offset int64, end_offset int64, text string) {
  lines++

  if text != "DEBUG First line\nsecond line\nthird line" {
    gt.Logf("Event data incorrect [% X]", text)
    gt.FailNow()
  }

  if end_offset != 5 {
    gt.Logf("Event end offset is incorrect [%d]", end_offset)
    gt.FailNow()
  }
}

func TestMultiline(t *testing.T) {
  gt = t
  lines = 0

  codec := createCodec(map[string]interface{}{
    "pattern": "^DEBUG ",
    "negate": true,
  }, checkMultiline, t)

  // Send some data
  codec.Event(0, 1, "DEBUG First line")
  codec.Event(2, 3, "second line")
  codec.Event(4, 5, "third line")
  codec.Event(6, 7, "DEBUG Next line")

  if lines != 1 {
    gt.Logf("Wrong line count received")
    gt.FailNow()
  }
}

func checkMultilineMaxBytes(start_offset int64, end_offset int64, text string) {
  lines++

  if lines == 1 {
    if text != "DEBUG First line\nsecond line\nthi" {
      gt.Logf("Event data incorrect [% X]", text)
      gt.FailNow()
    }
    return
  }

  if text != "rd line" {
    gt.Logf("Second event data incorrect [% X]", text)
    gt.FailNow()
  }
}

func TestMultilineMaxBytes(t *testing.T) {
  gt = t
  lines = 0

  codec := createCodec(map[string]interface{}{
    "max multiline bytes": int64(32),
    "pattern": "^DEBUG ",
    "negate": true,
  }, checkMultilineMaxBytes, t)

  // Send some data
  codec.Event(0, 1, "DEBUG First line")
  codec.Event(2, 3, "second line")
  codec.Event(4, 5, "third line")
  codec.Event(6, 7, "DEBUG Next line")

  if lines != 2 {
    gt.Logf("Wrong line count received")
    gt.FailNow()
  }
}
