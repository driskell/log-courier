package ast

import (
	"bytes"
	"context"
	"testing"

	"github.com/driskell/log-courier/lc-lib/event"
)

func TestNoop(t *testing.T) {
	subject := event.NewEvent(context.Background(), nil, map[string]interface{}{"message": "test"})
	subjectData := subject.Bytes()
	subject.ClearCache()
	node := &noopNode{}
	result := node.Process(subject)
	if result != subject || !bytes.Equal(subjectData, result.Bytes()) {
		t.Fatal("noop did not return the subject unchanged")
	}
}
