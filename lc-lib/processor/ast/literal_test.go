package ast

import (
	"testing"

	"github.com/google/cel-go/common/types"
)

func TestLiteral(t *testing.T) {
	value := types.String("test")
	node := &literalNode{value}
	if node.Value(nil) != value {
		t.Fatalf("expected literal to return stored literal")
	}
}
