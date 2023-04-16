package processor

import (
	"fmt"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

type errorListener struct {
	*antlr.DefaultErrorListener
	errors []error
}

func (c *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	c.errors = append(c.errors, fmt.Errorf("line "+strconv.Itoa(line)+":"+strconv.Itoa(column)+" "+msg))
}
