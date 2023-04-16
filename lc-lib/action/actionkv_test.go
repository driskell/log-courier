package action

import (
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

func processKVTestEvent(t *testing.T, arguments map[string]interface{}, data map[string]interface{}) *event.Event {
	t.Helper()
	return processTestEvent(t, "kv", arguments, data)
}

func TestKVAcceptsAllArguments(t *testing.T) {
	mustNewTestNode(t, "kv", map[string]interface{}{
		"field":        "message",
		"prefix":       "kv_",
		"separators":   " \x1D",
		"nested":       []string{"msg"},
		"unquoted_hex": []string{"proctitle", "cmd"},
	})
}

func TestKVSeparatorsDefaultToSpace(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": "a=1,b=2 c=3"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "a=1,b=2 c=3", "a": "1,b=2", "c": "3"})
}

func TestKVConfigRejectsUnknownKey(t *testing.T) {
	if _, err := newTestNode("kv", map[string]interface{}{"field": "message", "unknown": true}); err == nil {
		t.Error("Unknown key was not rejected")
	}
}

func TestKVConfigRejectsInvalidSeparators(t *testing.T) {
	for _, separators := range []string{"", "=", " \"", "'", "\\"} {
		if _, err := newTestNode("kv", map[string]interface{}{"field": "message", "separators": separators}); err == nil {
			t.Errorf("Separators %#v were not rejected", separators)
		}
	}
}

func assertKVTestData(t *testing.T, evnt *event.Event, expected map[string]interface{}) {
	t.Helper()
	data := make(map[string]interface{})
	for key, value := range evnt.Data() {
		if key != "@metadata" && key != "@timestamp" && key != "tags" {
			data[key] = value
		}
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Unexpected event data:\n got      %#v\n expected %#v", data, expected)
	}
}

func assertKVTestError(t *testing.T, evnt *event.Event, expected string) {
	t.Helper()
	if message := evnt.Data()["_kv_error"]; message != expected {
		t.Errorf("Unexpected _kv_error: got %#v, expected %#v", message, expected)
	}
}

func TestKVParsesUnquotedAndQuotedValues(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{
		"message": `number=500 double="value in quotes" single='other value' escaped="say \"hi\""`,
	})

	assertKVTestData(t, evnt, map[string]interface{}{
		"message": `number=500 double="value in quotes" single='other value' escaped="say \"hi\""`,
		"number":  "500",
		"double":  "value in quotes",
		"single":  "other value",
		"escaped": `say \"hi\"`,
	})
}

func TestKVQuotedValueKeepsOtherQuoteStyle(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": `a="it's" b='say "hi"'`})

	assertKVTestData(t, evnt, map[string]interface{}{"message": `a="it's" b='say "hi"'`, "a": "it's", "b": `say "hi"`})
}

func TestKVValueKeepsEqualsSigns(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": "query=a=b"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "query=a=b", "query": "a=b"})
}

func TestKVKeepsNamesLiterally(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "kv_"}, map[string]interface{}{"message": `nested[entry]=value back\slash=1`})

	assertKVTestData(t, evnt, map[string]interface{}{"message": `nested[entry]=value back\slash=1`, "kv_nested[entry]": "value", `kv_back\slash`: "1"})
}

func TestKVReadsNestedSourceField(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "source[line]"}, map[string]interface{}{
		"source": map[string]interface{}{"line": "a=1"},
	})

	assertKVTestData(t, evnt, map[string]interface{}{"source": map[string]interface{}{"line": "a=1"}, "a": "1"})
}

func TestKVPrefixesFieldsUsingPattern(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "audit_%{type}_"}, map[string]interface{}{
		"message": "pid=1",
		"type":    "SYSCALL",
	})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "pid=1", "type": "SYSCALL", "audit_SYSCALL_pid": "1"})
}

func TestKVSkipsRepeatedSeparators(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": "  a=1   b=\"2\"  "})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "  a=1   b=\"2\"  ", "a": "1", "b": "2"})
}

func TestKVStoresEmptyValues(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": `a= b=2 c="" d=`})

	assertKVTestData(t, evnt, map[string]interface{}{"message": `a= b=2 c="" d=`, "a": "", "b": "2", "c": "", "d": ""})
}

func TestKVEmptyInputAddsNothing(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": ""})

	assertKVTestData(t, evnt, map[string]interface{}{"message": ""})
}

func TestKVKeepsMultibyteCharacters(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": `café=crème name="Ünïcödé ✓"`})

	assertKVTestData(t, evnt, map[string]interface{}{"message": `café=crème name="Ünïcödé ✓"`, "café": "crème", "name": "Ünïcödé ✓"})
}

func TestKVSplitsOnEveryConfiguredSeparator(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "separators": " \x1D"}, map[string]interface{}{
		"message": "res=success'\x1DUID=\"root\"\x1Dgid=0 AUID=\"root\"",
	})

	assertKVTestData(t, evnt, map[string]interface{}{
		"message": "res=success'\x1DUID=\"root\"\x1Dgid=0 AUID=\"root\"",
		"res":     "success'",
		"UID":     "root",
		"gid":     "0",
		"AUID":    "root",
	})
}

func TestKVSeparatorAfterQuotedValue(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "separators": " \x1D"}, map[string]interface{}{
		"message": "a=\"1\"\x1Db=2",
	})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "a=\"1\"\x1Db=2", "a": "1", "b": "2"})
}

func TestKVOnlySplitsOnConfiguredSeparators(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "separators": ","}, map[string]interface{}{"message": "a=one two,b=three"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "a=one two,b=three", "a": "one two", "b": "three"})
}

func TestKVParsesNestedValuesWithNamePrefix(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "p_", "nested": []string{"msg"}}, map[string]interface{}{
		"message": `pid=1 msg='pid=2 exe="/usr/bin/sudo"' res=ok`,
	})

	assertKVTestData(t, evnt, map[string]interface{}{
		"message":   `pid=1 msg='pid=2 exe="/usr/bin/sudo"' res=ok`,
		"p_pid":     "1",
		"p_msg_pid": "2",
		"p_msg_exe": "/usr/bin/sudo",
		"p_res":     "ok",
	})
}

func TestKVParsesUnquotedNestedValues(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "nested": []string{"inner"}}, map[string]interface{}{"message": "inner=a=1"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "inner=a=1", "inner_a": "1"})
}

func TestKVNestedNamesKeepBrackets(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "nested": []string{"m[s]g"}}, map[string]interface{}{"message": "m[s]g='a=1'"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "m[s]g='a=1'", "m[s]g_a": "1"})
}

func TestKVDecodesListedUnquotedHex(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"proctitle"}}, map[string]interface{}{
		"message": "proctitle=2F7573722F7362696E2F63726F6E64002D6E",
	})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "proctitle=2F7573722F7362696E2F63726F6E64002D6E", "proctitle": "/usr/sbin/crond -n"})
}

func TestKVKeepsQuotedValuesOfHexFields(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"proctitle", "comm"}}, map[string]interface{}{
		"message": `proctitle="(systemd)" comm="ABCD"`,
	})

	assertKVTestData(t, evnt, map[string]interface{}{"message": `proctitle="(systemd)" comm="ABCD"`, "proctitle": "(systemd)", "comm": "ABCD"})
}

func TestKVKeepsUnlistedHexLikeValues(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"proctitle"}}, map[string]interface{}{"message": "pid=3363 arch=C000003E"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "pid=3363 arch=C000003E", "pid": "3363", "arch": "C000003E"})
}

func TestKVKeepsHexFieldsThatAreNotUppercaseHex(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"a", "b", "c", "d"}}, map[string]interface{}{
		"message": "a=2f7573 b=2F757 c=(null) d=FF",
	})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "a=2f7573 b=2F757 c=(null) d=FF", "a": "2f7573", "b": "2F757", "c": "(null)", "d": "FF"})
}

func TestKVDecodesEmptyHexFieldAsEmpty(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"cmd"}}, map[string]interface{}{"message": "cmd= a=1"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "cmd= a=1", "cmd": "", "a": "1"})
}

func TestKVDecodesHexInsideNestedValues(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{
		"field":        "message",
		"prefix":       "audit_%{audit_type}_",
		"separators":   " \x1D",
		"nested":       []string{"msg"},
		"unquoted_hex": []string{"cmd", "proctitle"},
	}, map[string]interface{}{
		"audit_type": "USER_CMD",
		"message":    "pid=3363 uid=0 auid=0 ses=15 msg='cwd=\"/root\" cmd=2D62617368202D2D6C6F67696E exe=\"/usr/bin/sudo\" terminal=? res=success'\x1DUID=\"root\" AUID=\"root\"",
	})

	assertKVTestData(t, evnt, map[string]interface{}{
		"audit_type":                  "USER_CMD",
		"message":                     "pid=3363 uid=0 auid=0 ses=15 msg='cwd=\"/root\" cmd=2D62617368202D2D6C6F67696E exe=\"/usr/bin/sudo\" terminal=? res=success'\x1DUID=\"root\" AUID=\"root\"",
		"audit_USER_CMD_pid":          "3363",
		"audit_USER_CMD_uid":          "0",
		"audit_USER_CMD_auid":         "0",
		"audit_USER_CMD_ses":          "15",
		"audit_USER_CMD_msg_cwd":      "/root",
		"audit_USER_CMD_msg_cmd":      "-bash --login",
		"audit_USER_CMD_msg_exe":      "/usr/bin/sudo",
		"audit_USER_CMD_msg_terminal": "?",
		"audit_USER_CMD_msg_res":      "success",
		"audit_USER_CMD_UID":          "root",
		"audit_USER_CMD_AUID":         "root",
	})
}

func TestKVReportsUnresolvableField(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "bad]"}, map[string]interface{}{})

	assertKVTestError(t, evnt, "Field 'bad]' could not be resolved: Invalid field: bad]")
}

func TestKVReportsMissingField(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{})

	assertKVTestError(t, evnt, "Field 'message' is not present or not a string")
}

func TestKVReportsNonStringField(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": 5})

	assertKVTestError(t, evnt, "Field 'message' is not present or not a string")
}

func TestKVReportsPrefixFormatFailure(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "%{bad]}_"}, map[string]interface{}{"message": "a=1"})

	assertKVTestError(t, evnt, "Failed to format prefix from event: %{bad]}_")
	if _, ok := evnt.Data()["a"]; ok {
		t.Error("Field was set despite prefix failure")
	}
}

func TestKVReportsFieldSetFailureAndStops(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "bad]"}, map[string]interface{}{"message": "a=1 b=2"})

	assertKVTestError(t, evnt, "Failed to set field 'bad]a': Invalid field: bad]a")
	if _, ok := evnt.Data()["bad]b"]; ok {
		t.Error("Parsing continued after field set failure")
	}
}

func TestKVReportsFieldSetFailureForEveryValueKind(t *testing.T) {
	for _, message := range []string{"a=", "a= b=1", `a="1"`, "a=1"} {
		evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "prefix": "bad]"}, map[string]interface{}{"message": message})
		assertKVTestError(t, evnt, "Failed to set field 'bad]a': Invalid field: bad]a")
	}
}

func TestKVReportsKeyWithNoName(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": "a=1 =2 c=3"})

	assertKVTestError(t, evnt, "Parsing interrupted, encountered key with no name")
	if evnt.Data()["a"] != "1" {
		t.Error("Field before the failure was not kept")
	}
	if _, ok := evnt.Data()["c"]; ok {
		t.Error("Parsing continued after key with no name")
	}
}

func TestKVReportsTextAfterQuotedValue(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": `a="1"b=2`})

	assertKVTestError(t, evnt, "Parsing interrupted, unexpected text after end of quoted value")
	if evnt.Data()["a"] != "1" {
		t.Error("Quoted value before the failure was not kept")
	}
}

func TestKVReportsUnexpectedEnd(t *testing.T) {
	for _, message := range []string{"a=1 flag", `a="unterminated`, `a="escape\`} {
		evnt := processKVTestEvent(t, map[string]interface{}{"field": "message"}, map[string]interface{}{"message": message})
		assertKVTestError(t, evnt, "Parsing interrupted, unexpected end of field")
	}
}

func TestKVNestedFailureStopsOuterParsing(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "nested": []string{"msg"}}, map[string]interface{}{"message": "msg='=1' b=2"})

	assertKVTestError(t, evnt, "Parsing interrupted, encountered key with no name")
	if _, ok := evnt.Data()["b"]; ok {
		t.Error("Outer parsing continued after nested failure")
	}
}

func TestKVNestedFailureAtEndOfField(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "nested": []string{"msg"}}, map[string]interface{}{"message": "msg=flag"})

	assertKVTestError(t, evnt, "Parsing interrupted, unexpected end of field")
}

func TestKVInvalidUTF8HexIsKept(t *testing.T) {
	evnt := processKVTestEvent(t, map[string]interface{}{"field": "message", "unquoted_hex": []string{"name"}}, map[string]interface{}{"message": "name=C328"})

	assertKVTestData(t, evnt, map[string]interface{}{"message": "name=C328", "name": "C328"})
}
