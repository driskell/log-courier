package processor

import (
	"context"
	"reflect"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

func TestJSONKeepsKeyNamesLiterally(t *testing.T) {
	evnt := (&jsonAction{Field: "message"}).Process(event.NewEvent(context.Background(), nil, map[string]interface{}{
		"message": `{"a[b]":"c","nested":{"d":"e"}}`,
	}))

	if evnt.Data()["a[b]"] != "c" || !reflect.DeepEqual(evnt.Data()["nested"], map[string]interface{}{"d": "e"}) {
		t.Errorf("Unexpected decoded fields: %#v", evnt.Data())
	}
}
