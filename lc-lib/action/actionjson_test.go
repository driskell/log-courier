package action

import (
	"reflect"
	"testing"
)

func TestJSONKeepsKeyNamesLiterally(t *testing.T) {
	evnt := processTestEvent(t, "json", map[string]interface{}{"field": "message"}, map[string]interface{}{
		"message": `{"a[b]":"c","nested":{"d":"e"}}`,
	})

	if evnt.Data()["a[b]"] != "c" || !reflect.DeepEqual(evnt.Data()["nested"], map[string]interface{}{"d": "e"}) {
		t.Errorf("Unexpected decoded fields: %#v", evnt.Data())
	}
}
