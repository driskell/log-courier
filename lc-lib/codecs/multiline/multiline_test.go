package codecs

import (
	"errors"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/driskell/log-courier/lc-lib/codecs"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/harvester"
	"github.com/driskell/log-courier/lc-lib/spooler"
)

var (
	errMultilineTest = errors.New("ERROR")
)

func createMultilineCodec(unused map[string]interface{}, callback codecs.CallbackFunc, t *testing.T) codecs.Codec {
	cfg := config.NewConfig()
	cfg.GeneralPart("harvester").(*harvester.General).MaxLineBytes = 1048576
	cfg.GeneralPart("spooler").(*spooler.General).SpoolMaxBytes = 10485760

	factory, err := NewMultilineCodecFactory(config.NewParser(cfg), "", unused, "multiline")
	if err != nil {
		t.Errorf("Failed to create multiline codec: %s", err)
		t.FailNow()
	}

	return codecs.NewCodec(factory, callback, 0)
}

type checkMultilineExpect struct {
	start, end int64
	text       string
}

type checkMultiline struct {
	expect []checkMultilineExpect
	t      *testing.T

	mutex    sync.Mutex
	lines    int
	errAfter int
}

func (c *checkMultiline) formatPrintable(text string) string {
	runes := []rune(text)
	for i, char := range runes {
		if unicode.IsPrint(char) {
			runes[i] = char
		} else {
			runes[i] = '.'
		}
	}
	return string(runes)
}

func (c *checkMultiline) incorrectLineCount(lines int, message string) {
	c.t.Error(message)
	c.t.Errorf("Got:      %d", lines)
	c.t.Errorf("Expected: %d", len(c.expect))
}

func (c *checkMultiline) EventCallback(startOffset int64, endOffset int64, data map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	line := c.lines + 1

	if line > len(c.expect) {
		c.incorrectLineCount(line, "Too many lines received")
		c.t.FailNow()
	}

	if startOffset != c.expect[c.lines].start {
		c.t.Error("Start offset incorrect for line: ", line)
		c.t.Errorf("Got:      %d", startOffset)
		c.t.Errorf("Expected: %d", c.expect[c.lines].start)
	}

	if endOffset != c.expect[c.lines].end {
		c.t.Error("End offset incorrect for line: ", line)
		c.t.Errorf("Got:      %d", endOffset)
		c.t.Errorf("Expected: %d", c.expect[c.lines].end)
	}

	text, ok := data["message"].(string)
	if !ok {
		c.t.Error("Text missing for line: ", line)
		c.t.Errorf("Expected: [%s]", c.formatPrintable(c.expect[c.lines].text))
	} else if text != c.expect[c.lines].text {
		c.t.Error("Text incorrect for line: ", line)
		c.t.Errorf("Got:      [%s]", c.formatPrintable(text))
		c.t.Errorf("Expected: [%s]", c.formatPrintable(c.expect[c.lines].text))
	}

	c.lines = line
	return nil
}

func (c *checkMultiline) EventCallbackError(startOffset int64, endOffset int64, data map[string]interface{}) error {
	if c.errAfter == 0 {
		return errMultilineTest
	}
	c.errAfter -= 1
	return c.EventCallback(startOffset, endOffset, data)
}

func (c *checkMultiline) CheckCurrentCount(count int, message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.lines != count {
		c.incorrectLineCount(c.lines, message)
	}
}

func (c *checkMultiline) CheckFinalCount() {
	c.CheckCurrentCount(len(c.expect), "Incorrect line count received")
}

func codecEvent(codec codecs.Codec, startOffset int64, endOffset int64, data string) error {
	return codec.ProcessEvent(startOffset, endOffset, map[string]interface{}{"message": data})
}

func TestMultilinePrevious(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"^(ANOTHER|NEXT) "},
			"what":     "previous",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	if offset := codec.Teardown(); offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilinePreviousNegate(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"!^DEBUG "},
			"what":     "previous",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	if offset := codec.Teardown(); offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilinePreviousTimeout(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
			{6, 7, "DEBUG Next line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns":         []string{"^(ANOTHER|NEXT) "},
			"what":             "previous",
			"previous timeout": "3s",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	// Allow a second
	time.Sleep(time.Second)

	check.CheckCurrentCount(1, "Timeout triggered too early")

	// Allow 5 seconds
	time.Sleep(5 * time.Second)

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineNext(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"^(DEBUG|NEXT) "},
			"what":     "next",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineNextNegate(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"!^ANOTHER "},
			"what":     "next",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineMaxBytes(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 32, "DEBUG First line\nsecond line\nthi"},
			{32, 39, "rd line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"max multiline bytes": int64(32),
			"patterns":            []string{"!^DEBUG "},
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 16, "DEBUG First line")
	codecEvent(codec, 17, 28, "second line")
	codecEvent(codec, 29, 39, "third line")
	codecEvent(codec, 40, 55, "DEBUG Next line")

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 39 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineMaxBytesOverflow(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 10, "START67890"},
			{10, 20, "abcdefg\n12"},
			{20, 30, "34567890ab"},
			{30, 39, "\nc1234567"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"max multiline bytes": int64(10),
			"patterns":            []string{"!^START"},
		},
		check.EventCallback,
		t,
	)

	// Ensure we reset state after each split (issue #188)
	// Also ensure we can split a single long line multiple times (issue #188)
	// Lastly, ensure we flush immediately if we receive max multiline bytes
	// rather than carrying over a full buffer and then crashing (issue #118)
	codecEvent(codec, 0, 17, "START67890abcdefg")
	codecEvent(codec, 18, 30, "1234567890ab")
	codecEvent(codec, 31, 39, "c1234567")
	codecEvent(codec, 40, 45, "START")

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 39 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineReset(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{4, 7, "DEBUG Next line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"^(ANOTHER|NEXT) "},
			"what":     "previous",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codec.Reset()
	codecEvent(codec, 4, 5, "DEBUG Next line")
	codecEvent(codec, 6, 7, "ANOTHER line")
	codecEvent(codec, 8, 9, "DEBUG Last line")

	check.CheckFinalCount()

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineMultiplePattern(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"=^ANOTHER ", "^NEXT "},
			"what":     "previous",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	if offset := codec.Teardown(); offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineMultiplePatternAll(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 5, "DEBUG First line\nNEXT line\nANOTHER line"},
		},
		t: t,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"!^DEBUG Next ", "!^DEBUG First "},
			"match":    "all",
			"what":     "previous",
		},
		check.EventCallback,
		t,
	)

	// Send some data
	codecEvent(codec, 0, 1, "DEBUG First line")
	codecEvent(codec, 2, 3, "NEXT line")
	codecEvent(codec, 4, 5, "ANOTHER line")
	codecEvent(codec, 6, 7, "DEBUG Next line")

	check.CheckFinalCount()

	if offset := codec.Teardown(); offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineError(t *testing.T) {
	check := &checkMultiline{
		t:        t,
		errAfter: 0,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"^(ANOTHER|NEXT) "},
			"what":     "previous",
		},
		check.EventCallbackError,
		t,
	)

	// Send some data
	if err := codecEvent(codec, 0, 1, "DEBUG last line of previous"); err != nil {
		t.Error("Expected codec to succeed")
	}
	if err := codecEvent(codec, 1, 2, "DEBUG First line"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}
	if err := codecEvent(codec, 3, 4, "NEXT line"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}
	if err := codecEvent(codec, 5, 6, "DEBUG Next line"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}

	if offset := codec.Teardown(); offset != 0 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineErrorOverflow(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 9, "DEBUG last"},
		},
		t:        t,
		errAfter: 1,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"max multiline bytes": int64(10),
			"patterns":            []string{"^(ANOTHER|NEXT) "},
			"what":                "previous",
		},
		check.EventCallbackError,
		t,
	)

	// Send some data
	if err := codecEvent(codec, 0, 9, "DEBUG last"); err != nil {
		t.Error("Expected codec to succeed")
	}
	if err := codecEvent(codec, 9, 37, "DEBUG First line overflowing"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}
	if err := codecEvent(codec, 37, 52, "DEBUG next line"); err != errMultilineTest {
		t.Error("Expected code to propogate the error")
	}

	if offset := codec.Teardown(); offset != 9 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineErrorOverflowMultiple(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 9, "DEBUG las"},
			{9, 19, "DEBUG Firs"},
		},
		t:        t,
		errAfter: 2,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"max multiline bytes": int64(10),
			"patterns":            []string{"^(ANOTHER|NEXT) "},
			"what":                "previous",
		},
		check.EventCallbackError,
		t,
	)

	// Send some data
	if err := codecEvent(codec, 0, 9, "DEBUG las"); err != nil {
		t.Error("Expected codec to succeed")
	}
	if err := codecEvent(codec, 9, 37, "DEBUG First line overflowing"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}
	if err := codecEvent(codec, 37, 53, "DEBUG next line"); err != errMultilineTest {
		t.Error("Expected code to propogate the error")
	}

	if offset := codec.Teardown(); offset != 19 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineErrorNext(t *testing.T) {
	check := &checkMultiline{
		t:        t,
		errAfter: 0,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns": []string{"^(DEBUG|NEXT) "},
			"what":     "next",
		},
		check.EventCallbackError,
		t,
	)

	// Send some data
	if err := codecEvent(codec, 0, 1, "DEBUG last line"); err != nil {
		t.Error("Expected codec to succeed")
	}
	if err := codecEvent(codec, 1, 2, "SEPARATE line"); err != errMultilineTest {
		t.Error("Expected codec to propogate the error")
	}
	if err := codecEvent(codec, 3, 4, "FINAL next line"); err != errMultilineTest {
		t.Error("Expected code to propogate the error")
	}

	if offset := codec.Teardown(); offset != 0 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineErrorTimeout(t *testing.T) {
	check := &checkMultiline{
		expect: []checkMultilineExpect{
			{0, 1, "DEBUG First line"},
		},
		t:        t,
		errAfter: 1,
	}

	codec := createMultilineCodec(
		map[string]interface{}{
			"patterns":         []string{"^(ANOTHER|NEXT) "},
			"what":             "previous",
			"previous timeout": "3s",
		},
		check.EventCallbackError,
		t,
	)

	// Send some data
	if err := codecEvent(codec, 0, 1, "DEBUG First line"); err != nil {
		t.Error("Expected codec to succeed")
	}
	if err := codecEvent(codec, 1, 2, "DEBUG Next line"); err != nil {
		t.Error("Expected codec to succeed")
	}

	// Allow a second
	time.Sleep(time.Second)

	check.CheckCurrentCount(1, "Timeout triggered too early")

	// Allow 5 seconds
	time.Sleep(5 * time.Second)

	check.CheckFinalCount()

	// Should now return error immediately
	if err := codecEvent(codec, 2, 3, "DEBUG Final line"); err != errMultilineTest {
		t.Error("Expected codec to propogate error")
	}

	offset := codec.Teardown()
	if offset != 1 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}
