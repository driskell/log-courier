package processor

import (
	"errors"
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
	"github.com/driskell/log-courier/lc-lib/processor/gen"
)

func compileScript(cfg *config.Config, source string) ([]ast.ProcessNode, error) {
	inputStream := antlr.NewInputStream(source)
	lexer := gen.NewLogCarverLexer(inputStream)
	tokenStream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	errorListenerImpl := &errorListener{}
	parser := gen.NewLogCarverParser(tokenStream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListenerImpl)
	program := parser.Program()
	visitor := ast.NewVisitor(cfg, errorListenerImpl)
	ast := visitor.Visit(program).([]ast.ProcessNode)
	if len(errorListenerImpl.errors) != 0 {
		return nil, fmt.Errorf("failed to parse processor pipeline script:\n%s", errors.Join(errorListenerImpl.errors...))
	}
	return ast, nil
}
