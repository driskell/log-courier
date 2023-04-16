// Code generated from LogCarver.c4 by ANTLR 4.12.0. DO NOT EDIT.

package gen // LogCarver
import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type LogCarverParser struct {
	*antlr.BaseParser
}

var logcarverParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	literalNames           []string
	symbolicNames          []string
	ruleNames              []string
	predictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func logcarverParserInit() {
	staticData := &logcarverParserStaticData
	staticData.literalNames = []string{
		"", "'='", "'=='", "'!='", "'in'", "'<'", "'<='", "'>='", "'>'", "'&&'",
		"'||'", "'['", "']'", "'{'", "'}'", "'('", "')'", "'.'", "','", "'-'",
		"'!'", "'?'", "':'", "'+'", "'*'", "'/'", "'%'", "'true'", "'false'",
		"'null'", "", "", "", "", "", "", "", "'else if'", "'if'", "'else'",
		"'set'", "'unset'", "';'",
	}
	staticData.symbolicNames = []string{
		"", "", "EQUALS", "NOT_EQUALS", "IN", "LESS", "LESS_EQUALS", "GREATER_EQUALS",
		"GREATER", "LOGICAL_AND", "LOGICAL_OR", "LBRACKET", "RPRACKET", "LBRACE",
		"RBRACE", "LPAREN", "RPAREN", "DOT", "COMMA", "MINUS", "EXCLAM", "QUESTIONMARK",
		"COLON", "PLUS", "STAR", "SLASH", "PERCENT", "CEL_TRUE", "CEL_FALSE",
		"NUL", "WHITESPACE", "COMMENT", "NUM_FLOAT", "NUM_INT", "NUM_UINT",
		"STRING", "BYTES", "ELSEIF", "IF", "ELSE", "SET", "UNSET", "EOL", "IDENTIFIER",
	}
	staticData.ruleNames = []string{
		"program", "lines", "line", "statement", "ifexpr", "elseifexpr", "elseexpr",
		"condition", "block", "set", "unset", "resolver", "action", "argument",
		"expr", "conditionalOr", "conditionalAnd", "relation", "calc", "unary",
		"member", "primary", "exprList", "listInit", "fieldInitializerList",
		"optField", "mapInitializerList", "optExpr", "literal",
	}
	staticData.predictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 43, 356, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 1, 0, 1, 0, 1, 0, 1, 1, 4, 1, 63, 8,
		1, 11, 1, 12, 1, 64, 1, 2, 1, 2, 3, 2, 69, 8, 2, 1, 3, 1, 3, 1, 3, 3, 3,
		74, 8, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 5, 4, 82, 8, 4, 10, 4, 12,
		4, 85, 9, 4, 1, 4, 3, 4, 88, 8, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6,
		1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9,
		1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1,
		11, 1, 11, 1, 11, 5, 11, 121, 8, 11, 10, 11, 12, 11, 124, 9, 11, 1, 12,
		1, 12, 1, 12, 1, 12, 5, 12, 130, 8, 12, 10, 12, 12, 12, 133, 9, 12, 3,
		12, 135, 8, 12, 1, 13, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 14,
		1, 14, 1, 14, 3, 14, 147, 8, 14, 1, 15, 1, 15, 1, 15, 5, 15, 152, 8, 15,
		10, 15, 12, 15, 155, 9, 15, 1, 16, 1, 16, 1, 16, 5, 16, 160, 8, 16, 10,
		16, 12, 16, 163, 9, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1, 17, 1, 17, 5, 17,
		171, 8, 17, 10, 17, 12, 17, 174, 9, 17, 1, 18, 1, 18, 1, 18, 1, 18, 1,
		18, 1, 18, 1, 18, 1, 18, 1, 18, 5, 18, 185, 8, 18, 10, 18, 12, 18, 188,
		9, 18, 1, 19, 1, 19, 4, 19, 192, 8, 19, 11, 19, 12, 19, 193, 1, 19, 1,
		19, 4, 19, 198, 8, 19, 11, 19, 12, 19, 199, 1, 19, 3, 19, 203, 8, 19, 1,
		20, 1, 20, 1, 20, 1, 20, 1, 20, 1, 20, 3, 20, 211, 8, 20, 1, 20, 1, 20,
		1, 20, 1, 20, 1, 20, 1, 20, 3, 20, 219, 8, 20, 1, 20, 1, 20, 1, 20, 1,
		20, 3, 20, 225, 8, 20, 1, 20, 1, 20, 1, 20, 5, 20, 230, 8, 20, 10, 20,
		12, 20, 233, 9, 20, 1, 21, 3, 21, 236, 8, 21, 1, 21, 1, 21, 1, 21, 3, 21,
		241, 8, 21, 1, 21, 3, 21, 244, 8, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21,
		1, 21, 3, 21, 252, 8, 21, 1, 21, 3, 21, 255, 8, 21, 1, 21, 1, 21, 1, 21,
		3, 21, 260, 8, 21, 1, 21, 3, 21, 263, 8, 21, 1, 21, 1, 21, 3, 21, 267,
		8, 21, 1, 21, 1, 21, 1, 21, 5, 21, 272, 8, 21, 10, 21, 12, 21, 275, 9,
		21, 1, 21, 1, 21, 3, 21, 279, 8, 21, 1, 21, 3, 21, 282, 8, 21, 1, 21, 1,
		21, 3, 21, 286, 8, 21, 1, 22, 1, 22, 1, 22, 5, 22, 291, 8, 22, 10, 22,
		12, 22, 294, 9, 22, 1, 23, 1, 23, 1, 23, 5, 23, 299, 8, 23, 10, 23, 12,
		23, 302, 9, 23, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24,
		5, 24, 312, 8, 24, 10, 24, 12, 24, 315, 9, 24, 1, 25, 3, 25, 318, 8, 25,
		1, 25, 1, 25, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 5,
		26, 330, 8, 26, 10, 26, 12, 26, 333, 9, 26, 1, 27, 3, 27, 336, 8, 27, 1,
		27, 1, 27, 1, 28, 3, 28, 341, 8, 28, 1, 28, 1, 28, 1, 28, 3, 28, 346, 8,
		28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 3, 28, 354, 8, 28, 1, 28,
		0, 3, 34, 36, 40, 29, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26,
		28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 0, 4, 2, 0,
		35, 35, 43, 43, 1, 0, 2, 8, 1, 0, 24, 26, 2, 0, 19, 19, 23, 23, 383, 0,
		58, 1, 0, 0, 0, 2, 62, 1, 0, 0, 0, 4, 68, 1, 0, 0, 0, 6, 73, 1, 0, 0, 0,
		8, 77, 1, 0, 0, 0, 10, 89, 1, 0, 0, 0, 12, 93, 1, 0, 0, 0, 14, 96, 1, 0,
		0, 0, 16, 100, 1, 0, 0, 0, 18, 104, 1, 0, 0, 0, 20, 109, 1, 0, 0, 0, 22,
		114, 1, 0, 0, 0, 24, 125, 1, 0, 0, 0, 26, 136, 1, 0, 0, 0, 28, 140, 1,
		0, 0, 0, 30, 148, 1, 0, 0, 0, 32, 156, 1, 0, 0, 0, 34, 164, 1, 0, 0, 0,
		36, 175, 1, 0, 0, 0, 38, 202, 1, 0, 0, 0, 40, 204, 1, 0, 0, 0, 42, 285,
		1, 0, 0, 0, 44, 287, 1, 0, 0, 0, 46, 295, 1, 0, 0, 0, 48, 303, 1, 0, 0,
		0, 50, 317, 1, 0, 0, 0, 52, 321, 1, 0, 0, 0, 54, 335, 1, 0, 0, 0, 56, 353,
		1, 0, 0, 0, 58, 59, 3, 2, 1, 0, 59, 60, 5, 0, 0, 1, 60, 1, 1, 0, 0, 0,
		61, 63, 3, 4, 2, 0, 62, 61, 1, 0, 0, 0, 63, 64, 1, 0, 0, 0, 64, 62, 1,
		0, 0, 0, 64, 65, 1, 0, 0, 0, 65, 3, 1, 0, 0, 0, 66, 69, 3, 8, 4, 0, 67,
		69, 3, 6, 3, 0, 68, 66, 1, 0, 0, 0, 68, 67, 1, 0, 0, 0, 69, 5, 1, 0, 0,
		0, 70, 74, 3, 18, 9, 0, 71, 74, 3, 20, 10, 0, 72, 74, 3, 24, 12, 0, 73,
		70, 1, 0, 0, 0, 73, 71, 1, 0, 0, 0, 73, 72, 1, 0, 0, 0, 74, 75, 1, 0, 0,
		0, 75, 76, 5, 42, 0, 0, 76, 7, 1, 0, 0, 0, 77, 78, 5, 38, 0, 0, 78, 79,
		3, 14, 7, 0, 79, 83, 3, 16, 8, 0, 80, 82, 3, 10, 5, 0, 81, 80, 1, 0, 0,
		0, 82, 85, 1, 0, 0, 0, 83, 81, 1, 0, 0, 0, 83, 84, 1, 0, 0, 0, 84, 87,
		1, 0, 0, 0, 85, 83, 1, 0, 0, 0, 86, 88, 3, 12, 6, 0, 87, 86, 1, 0, 0, 0,
		87, 88, 1, 0, 0, 0, 88, 9, 1, 0, 0, 0, 89, 90, 5, 37, 0, 0, 90, 91, 3,
		14, 7, 0, 91, 92, 3, 16, 8, 0, 92, 11, 1, 0, 0, 0, 93, 94, 5, 39, 0, 0,
		94, 95, 3, 16, 8, 0, 95, 13, 1, 0, 0, 0, 96, 97, 5, 15, 0, 0, 97, 98, 3,
		28, 14, 0, 98, 99, 5, 16, 0, 0, 99, 15, 1, 0, 0, 0, 100, 101, 5, 13, 0,
		0, 101, 102, 3, 2, 1, 0, 102, 103, 5, 14, 0, 0, 103, 17, 1, 0, 0, 0, 104,
		105, 5, 40, 0, 0, 105, 106, 3, 22, 11, 0, 106, 107, 5, 1, 0, 0, 107, 108,
		3, 28, 14, 0, 108, 19, 1, 0, 0, 0, 109, 110, 5, 41, 0, 0, 110, 111, 3,
		22, 11, 0, 111, 112, 5, 1, 0, 0, 112, 113, 3, 28, 14, 0, 113, 21, 1, 0,
		0, 0, 114, 122, 5, 43, 0, 0, 115, 116, 5, 11, 0, 0, 116, 117, 7, 0, 0,
		0, 117, 121, 5, 12, 0, 0, 118, 119, 5, 17, 0, 0, 119, 121, 5, 43, 0, 0,
		120, 115, 1, 0, 0, 0, 120, 118, 1, 0, 0, 0, 121, 124, 1, 0, 0, 0, 122,
		120, 1, 0, 0, 0, 122, 123, 1, 0, 0, 0, 123, 23, 1, 0, 0, 0, 124, 122, 1,
		0, 0, 0, 125, 134, 5, 43, 0, 0, 126, 131, 3, 26, 13, 0, 127, 128, 5, 18,
		0, 0, 128, 130, 3, 26, 13, 0, 129, 127, 1, 0, 0, 0, 130, 133, 1, 0, 0,
		0, 131, 129, 1, 0, 0, 0, 131, 132, 1, 0, 0, 0, 132, 135, 1, 0, 0, 0, 133,
		131, 1, 0, 0, 0, 134, 126, 1, 0, 0, 0, 134, 135, 1, 0, 0, 0, 135, 25, 1,
		0, 0, 0, 136, 137, 5, 43, 0, 0, 137, 138, 5, 1, 0, 0, 138, 139, 3, 28,
		14, 0, 139, 27, 1, 0, 0, 0, 140, 146, 3, 30, 15, 0, 141, 142, 5, 21, 0,
		0, 142, 143, 3, 30, 15, 0, 143, 144, 5, 22, 0, 0, 144, 145, 3, 28, 14,
		0, 145, 147, 1, 0, 0, 0, 146, 141, 1, 0, 0, 0, 146, 147, 1, 0, 0, 0, 147,
		29, 1, 0, 0, 0, 148, 153, 3, 32, 16, 0, 149, 150, 5, 10, 0, 0, 150, 152,
		3, 32, 16, 0, 151, 149, 1, 0, 0, 0, 152, 155, 1, 0, 0, 0, 153, 151, 1,
		0, 0, 0, 153, 154, 1, 0, 0, 0, 154, 31, 1, 0, 0, 0, 155, 153, 1, 0, 0,
		0, 156, 161, 3, 34, 17, 0, 157, 158, 5, 9, 0, 0, 158, 160, 3, 34, 17, 0,
		159, 157, 1, 0, 0, 0, 160, 163, 1, 0, 0, 0, 161, 159, 1, 0, 0, 0, 161,
		162, 1, 0, 0, 0, 162, 33, 1, 0, 0, 0, 163, 161, 1, 0, 0, 0, 164, 165, 6,
		17, -1, 0, 165, 166, 3, 36, 18, 0, 166, 172, 1, 0, 0, 0, 167, 168, 10,
		1, 0, 0, 168, 169, 7, 1, 0, 0, 169, 171, 3, 34, 17, 2, 170, 167, 1, 0,
		0, 0, 171, 174, 1, 0, 0, 0, 172, 170, 1, 0, 0, 0, 172, 173, 1, 0, 0, 0,
		173, 35, 1, 0, 0, 0, 174, 172, 1, 0, 0, 0, 175, 176, 6, 18, -1, 0, 176,
		177, 3, 38, 19, 0, 177, 186, 1, 0, 0, 0, 178, 179, 10, 2, 0, 0, 179, 180,
		7, 2, 0, 0, 180, 185, 3, 36, 18, 3, 181, 182, 10, 1, 0, 0, 182, 183, 7,
		3, 0, 0, 183, 185, 3, 36, 18, 2, 184, 178, 1, 0, 0, 0, 184, 181, 1, 0,
		0, 0, 185, 188, 1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 186, 187, 1, 0, 0, 0,
		187, 37, 1, 0, 0, 0, 188, 186, 1, 0, 0, 0, 189, 203, 3, 40, 20, 0, 190,
		192, 5, 20, 0, 0, 191, 190, 1, 0, 0, 0, 192, 193, 1, 0, 0, 0, 193, 191,
		1, 0, 0, 0, 193, 194, 1, 0, 0, 0, 194, 195, 1, 0, 0, 0, 195, 203, 3, 40,
		20, 0, 196, 198, 5, 19, 0, 0, 197, 196, 1, 0, 0, 0, 198, 199, 1, 0, 0,
		0, 199, 197, 1, 0, 0, 0, 199, 200, 1, 0, 0, 0, 200, 201, 1, 0, 0, 0, 201,
		203, 3, 40, 20, 0, 202, 189, 1, 0, 0, 0, 202, 191, 1, 0, 0, 0, 202, 197,
		1, 0, 0, 0, 203, 39, 1, 0, 0, 0, 204, 205, 6, 20, -1, 0, 205, 206, 3, 42,
		21, 0, 206, 231, 1, 0, 0, 0, 207, 208, 10, 3, 0, 0, 208, 210, 5, 17, 0,
		0, 209, 211, 5, 21, 0, 0, 210, 209, 1, 0, 0, 0, 210, 211, 1, 0, 0, 0, 211,
		212, 1, 0, 0, 0, 212, 230, 5, 43, 0, 0, 213, 214, 10, 2, 0, 0, 214, 215,
		5, 17, 0, 0, 215, 216, 5, 43, 0, 0, 216, 218, 5, 15, 0, 0, 217, 219, 3,
		44, 22, 0, 218, 217, 1, 0, 0, 0, 218, 219, 1, 0, 0, 0, 219, 220, 1, 0,
		0, 0, 220, 230, 5, 16, 0, 0, 221, 222, 10, 1, 0, 0, 222, 224, 5, 11, 0,
		0, 223, 225, 5, 21, 0, 0, 224, 223, 1, 0, 0, 0, 224, 225, 1, 0, 0, 0, 225,
		226, 1, 0, 0, 0, 226, 227, 3, 28, 14, 0, 227, 228, 5, 12, 0, 0, 228, 230,
		1, 0, 0, 0, 229, 207, 1, 0, 0, 0, 229, 213, 1, 0, 0, 0, 229, 221, 1, 0,
		0, 0, 230, 233, 1, 0, 0, 0, 231, 229, 1, 0, 0, 0, 231, 232, 1, 0, 0, 0,
		232, 41, 1, 0, 0, 0, 233, 231, 1, 0, 0, 0, 234, 236, 5, 17, 0, 0, 235,
		234, 1, 0, 0, 0, 235, 236, 1, 0, 0, 0, 236, 237, 1, 0, 0, 0, 237, 243,
		5, 43, 0, 0, 238, 240, 5, 15, 0, 0, 239, 241, 3, 44, 22, 0, 240, 239, 1,
		0, 0, 0, 240, 241, 1, 0, 0, 0, 241, 242, 1, 0, 0, 0, 242, 244, 5, 16, 0,
		0, 243, 238, 1, 0, 0, 0, 243, 244, 1, 0, 0, 0, 244, 286, 1, 0, 0, 0, 245,
		246, 5, 15, 0, 0, 246, 247, 3, 28, 14, 0, 247, 248, 5, 16, 0, 0, 248, 286,
		1, 0, 0, 0, 249, 251, 5, 11, 0, 0, 250, 252, 3, 46, 23, 0, 251, 250, 1,
		0, 0, 0, 251, 252, 1, 0, 0, 0, 252, 254, 1, 0, 0, 0, 253, 255, 5, 18, 0,
		0, 254, 253, 1, 0, 0, 0, 254, 255, 1, 0, 0, 0, 255, 256, 1, 0, 0, 0, 256,
		286, 5, 12, 0, 0, 257, 259, 5, 13, 0, 0, 258, 260, 3, 52, 26, 0, 259, 258,
		1, 0, 0, 0, 259, 260, 1, 0, 0, 0, 260, 262, 1, 0, 0, 0, 261, 263, 5, 18,
		0, 0, 262, 261, 1, 0, 0, 0, 262, 263, 1, 0, 0, 0, 263, 264, 1, 0, 0, 0,
		264, 286, 5, 14, 0, 0, 265, 267, 5, 17, 0, 0, 266, 265, 1, 0, 0, 0, 266,
		267, 1, 0, 0, 0, 267, 268, 1, 0, 0, 0, 268, 273, 5, 43, 0, 0, 269, 270,
		5, 17, 0, 0, 270, 272, 5, 43, 0, 0, 271, 269, 1, 0, 0, 0, 272, 275, 1,
		0, 0, 0, 273, 271, 1, 0, 0, 0, 273, 274, 1, 0, 0, 0, 274, 276, 1, 0, 0,
		0, 275, 273, 1, 0, 0, 0, 276, 278, 5, 13, 0, 0, 277, 279, 3, 48, 24, 0,
		278, 277, 1, 0, 0, 0, 278, 279, 1, 0, 0, 0, 279, 281, 1, 0, 0, 0, 280,
		282, 5, 18, 0, 0, 281, 280, 1, 0, 0, 0, 281, 282, 1, 0, 0, 0, 282, 283,
		1, 0, 0, 0, 283, 286, 5, 14, 0, 0, 284, 286, 3, 56, 28, 0, 285, 235, 1,
		0, 0, 0, 285, 245, 1, 0, 0, 0, 285, 249, 1, 0, 0, 0, 285, 257, 1, 0, 0,
		0, 285, 266, 1, 0, 0, 0, 285, 284, 1, 0, 0, 0, 286, 43, 1, 0, 0, 0, 287,
		292, 3, 28, 14, 0, 288, 289, 5, 18, 0, 0, 289, 291, 3, 28, 14, 0, 290,
		288, 1, 0, 0, 0, 291, 294, 1, 0, 0, 0, 292, 290, 1, 0, 0, 0, 292, 293,
		1, 0, 0, 0, 293, 45, 1, 0, 0, 0, 294, 292, 1, 0, 0, 0, 295, 300, 3, 54,
		27, 0, 296, 297, 5, 18, 0, 0, 297, 299, 3, 54, 27, 0, 298, 296, 1, 0, 0,
		0, 299, 302, 1, 0, 0, 0, 300, 298, 1, 0, 0, 0, 300, 301, 1, 0, 0, 0, 301,
		47, 1, 0, 0, 0, 302, 300, 1, 0, 0, 0, 303, 304, 3, 50, 25, 0, 304, 305,
		5, 22, 0, 0, 305, 313, 3, 28, 14, 0, 306, 307, 5, 18, 0, 0, 307, 308, 3,
		50, 25, 0, 308, 309, 5, 22, 0, 0, 309, 310, 3, 28, 14, 0, 310, 312, 1,
		0, 0, 0, 311, 306, 1, 0, 0, 0, 312, 315, 1, 0, 0, 0, 313, 311, 1, 0, 0,
		0, 313, 314, 1, 0, 0, 0, 314, 49, 1, 0, 0, 0, 315, 313, 1, 0, 0, 0, 316,
		318, 5, 21, 0, 0, 317, 316, 1, 0, 0, 0, 317, 318, 1, 0, 0, 0, 318, 319,
		1, 0, 0, 0, 319, 320, 5, 43, 0, 0, 320, 51, 1, 0, 0, 0, 321, 322, 3, 54,
		27, 0, 322, 323, 5, 22, 0, 0, 323, 331, 3, 28, 14, 0, 324, 325, 5, 18,
		0, 0, 325, 326, 3, 54, 27, 0, 326, 327, 5, 22, 0, 0, 327, 328, 3, 28, 14,
		0, 328, 330, 1, 0, 0, 0, 329, 324, 1, 0, 0, 0, 330, 333, 1, 0, 0, 0, 331,
		329, 1, 0, 0, 0, 331, 332, 1, 0, 0, 0, 332, 53, 1, 0, 0, 0, 333, 331, 1,
		0, 0, 0, 334, 336, 5, 21, 0, 0, 335, 334, 1, 0, 0, 0, 335, 336, 1, 0, 0,
		0, 336, 337, 1, 0, 0, 0, 337, 338, 3, 28, 14, 0, 338, 55, 1, 0, 0, 0, 339,
		341, 5, 19, 0, 0, 340, 339, 1, 0, 0, 0, 340, 341, 1, 0, 0, 0, 341, 342,
		1, 0, 0, 0, 342, 354, 5, 33, 0, 0, 343, 354, 5, 34, 0, 0, 344, 346, 5,
		19, 0, 0, 345, 344, 1, 0, 0, 0, 345, 346, 1, 0, 0, 0, 346, 347, 1, 0, 0,
		0, 347, 354, 5, 32, 0, 0, 348, 354, 5, 35, 0, 0, 349, 354, 5, 36, 0, 0,
		350, 354, 5, 27, 0, 0, 351, 354, 5, 28, 0, 0, 352, 354, 5, 29, 0, 0, 353,
		340, 1, 0, 0, 0, 353, 343, 1, 0, 0, 0, 353, 345, 1, 0, 0, 0, 353, 348,
		1, 0, 0, 0, 353, 349, 1, 0, 0, 0, 353, 350, 1, 0, 0, 0, 353, 351, 1, 0,
		0, 0, 353, 352, 1, 0, 0, 0, 354, 57, 1, 0, 0, 0, 44, 64, 68, 73, 83, 87,
		120, 122, 131, 134, 146, 153, 161, 172, 184, 186, 193, 199, 202, 210, 218,
		224, 229, 231, 235, 240, 243, 251, 254, 259, 262, 266, 273, 278, 281, 285,
		292, 300, 313, 317, 331, 335, 340, 345, 353,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// LogCarverParserInit initializes any static state used to implement LogCarverParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewLogCarverParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func LogCarverParserInit() {
	staticData := &logcarverParserStaticData
	staticData.once.Do(logcarverParserInit)
}

// NewLogCarverParser produces a new parser instance for the optional input antlr.TokenStream.
func NewLogCarverParser(input antlr.TokenStream) *LogCarverParser {
	LogCarverParserInit()
	this := new(LogCarverParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &logcarverParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.predictionContextCache)
	this.RuleNames = staticData.ruleNames
	this.LiteralNames = staticData.literalNames
	this.SymbolicNames = staticData.symbolicNames
	this.GrammarFileName = "LogCarver.c4"

	return this
}

// LogCarverParser tokens.
const (
	LogCarverParserEOF            = antlr.TokenEOF
	LogCarverParserT__0           = 1
	LogCarverParserEQUALS         = 2
	LogCarverParserNOT_EQUALS     = 3
	LogCarverParserIN             = 4
	LogCarverParserLESS           = 5
	LogCarverParserLESS_EQUALS    = 6
	LogCarverParserGREATER_EQUALS = 7
	LogCarverParserGREATER        = 8
	LogCarverParserLOGICAL_AND    = 9
	LogCarverParserLOGICAL_OR     = 10
	LogCarverParserLBRACKET       = 11
	LogCarverParserRPRACKET       = 12
	LogCarverParserLBRACE         = 13
	LogCarverParserRBRACE         = 14
	LogCarverParserLPAREN         = 15
	LogCarverParserRPAREN         = 16
	LogCarverParserDOT            = 17
	LogCarverParserCOMMA          = 18
	LogCarverParserMINUS          = 19
	LogCarverParserEXCLAM         = 20
	LogCarverParserQUESTIONMARK   = 21
	LogCarverParserCOLON          = 22
	LogCarverParserPLUS           = 23
	LogCarverParserSTAR           = 24
	LogCarverParserSLASH          = 25
	LogCarverParserPERCENT        = 26
	LogCarverParserCEL_TRUE       = 27
	LogCarverParserCEL_FALSE      = 28
	LogCarverParserNUL            = 29
	LogCarverParserWHITESPACE     = 30
	LogCarverParserCOMMENT        = 31
	LogCarverParserNUM_FLOAT      = 32
	LogCarverParserNUM_INT        = 33
	LogCarverParserNUM_UINT       = 34
	LogCarverParserSTRING         = 35
	LogCarverParserBYTES          = 36
	LogCarverParserELSEIF         = 37
	LogCarverParserIF             = 38
	LogCarverParserELSE           = 39
	LogCarverParserSET            = 40
	LogCarverParserUNSET          = 41
	LogCarverParserEOL            = 42
	LogCarverParserIDENTIFIER     = 43
)

// LogCarverParser rules.
const (
	LogCarverParserRULE_program              = 0
	LogCarverParserRULE_lines                = 1
	LogCarverParserRULE_line                 = 2
	LogCarverParserRULE_statement            = 3
	LogCarverParserRULE_ifexpr               = 4
	LogCarverParserRULE_elseifexpr           = 5
	LogCarverParserRULE_elseexpr             = 6
	LogCarverParserRULE_condition            = 7
	LogCarverParserRULE_block                = 8
	LogCarverParserRULE_set                  = 9
	LogCarverParserRULE_unset                = 10
	LogCarverParserRULE_resolver             = 11
	LogCarverParserRULE_action               = 12
	LogCarverParserRULE_argument             = 13
	LogCarverParserRULE_expr                 = 14
	LogCarverParserRULE_conditionalOr        = 15
	LogCarverParserRULE_conditionalAnd       = 16
	LogCarverParserRULE_relation             = 17
	LogCarverParserRULE_calc                 = 18
	LogCarverParserRULE_unary                = 19
	LogCarverParserRULE_member               = 20
	LogCarverParserRULE_primary              = 21
	LogCarverParserRULE_exprList             = 22
	LogCarverParserRULE_listInit             = 23
	LogCarverParserRULE_fieldInitializerList = 24
	LogCarverParserRULE_optField             = 25
	LogCarverParserRULE_mapInitializerList   = 26
	LogCarverParserRULE_optExpr              = 27
	LogCarverParserRULE_literal              = 28
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Lines() ILinesContext
	EOF() antlr.TerminalNode

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_program
	return p
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) Lines() ILinesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILinesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILinesContext)
}

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(LogCarverParserEOF, 0)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitProgram(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Program() (localctx IProgramContext) {
	this := p
	_ = this

	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, LogCarverParserRULE_program)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(58)
		p.Lines()
	}
	{
		p.SetState(59)
		p.Match(LogCarverParserEOF)
	}

	return localctx
}

// ILinesContext is an interface to support dynamic dispatch.
type ILinesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllLine() []ILineContext
	Line(i int) ILineContext

	// IsLinesContext differentiates from other interfaces.
	IsLinesContext()
}

type LinesContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLinesContext() *LinesContext {
	var p = new(LinesContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_lines
	return p
}

func (*LinesContext) IsLinesContext() {}

func NewLinesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LinesContext {
	var p = new(LinesContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_lines

	return p
}

func (s *LinesContext) GetParser() antlr.Parser { return s.parser }

func (s *LinesContext) AllLine() []ILineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILineContext); ok {
			len++
		}
	}

	tst := make([]ILineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILineContext); ok {
			tst[i] = t.(ILineContext)
			i++
		}
	}

	return tst
}

func (s *LinesContext) Line(i int) ILineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILineContext)
}

func (s *LinesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LinesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LinesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitLines(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Lines() (localctx ILinesContext) {
	this := p
	_ = this

	localctx = NewLinesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, LogCarverParserRULE_lines)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	p.SetState(62)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = ((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&12369505812480) != 0) {
		{
			p.SetState(61)
			p.Line()
		}

		p.SetState(64)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// ILineContext is an interface to support dynamic dispatch.
type ILineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Ifexpr() IIfexprContext
	Statement() IStatementContext

	// IsLineContext differentiates from other interfaces.
	IsLineContext()
}

type LineContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLineContext() *LineContext {
	var p = new(LineContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_line
	return p
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_line

	return p
}

func (s *LineContext) GetParser() antlr.Parser { return s.parser }

func (s *LineContext) Ifexpr() IIfexprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIfexprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIfexprContext)
}

func (s *LineContext) Statement() IStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatementContext)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitLine(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Line() (localctx ILineContext) {
	this := p
	_ = this

	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, LogCarverParserRULE_line)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(68)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case LogCarverParserIF:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(66)
			p.Ifexpr()
		}

	case LogCarverParserSET, LogCarverParserUNSET, LogCarverParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(67)
			p.Statement()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}

	return localctx
}

// IStatementContext is an interface to support dynamic dispatch.
type IStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOL() antlr.TerminalNode
	Set() ISetContext
	Unset() IUnsetContext
	Action_() IActionContext

	// IsStatementContext differentiates from other interfaces.
	IsStatementContext()
}

type StatementContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStatementContext() *StatementContext {
	var p = new(StatementContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_statement
	return p
}

func (*StatementContext) IsStatementContext() {}

func NewStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StatementContext {
	var p = new(StatementContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_statement

	return p
}

func (s *StatementContext) GetParser() antlr.Parser { return s.parser }

func (s *StatementContext) EOL() antlr.TerminalNode {
	return s.GetToken(LogCarverParserEOL, 0)
}

func (s *StatementContext) Set() ISetContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISetContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISetContext)
}

func (s *StatementContext) Unset() IUnsetContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnsetContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnsetContext)
}

func (s *StatementContext) Action_() IActionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IActionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IActionContext)
}

func (s *StatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Statement() (localctx IStatementContext) {
	this := p
	_ = this

	localctx = NewStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, LogCarverParserRULE_statement)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	p.SetState(73)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case LogCarverParserSET:
		{
			p.SetState(70)
			p.Set()
		}

	case LogCarverParserUNSET:
		{
			p.SetState(71)
			p.Unset()
		}

	case LogCarverParserIDENTIFIER:
		{
			p.SetState(72)
			p.Action_()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}
	{
		p.SetState(75)
		p.Match(LogCarverParserEOL)
	}

	return localctx
}

// IIfexprContext is an interface to support dynamic dispatch.
type IIfexprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IF() antlr.TerminalNode
	Condition() IConditionContext
	Block() IBlockContext
	AllElseifexpr() []IElseifexprContext
	Elseifexpr(i int) IElseifexprContext
	Elseexpr() IElseexprContext

	// IsIfexprContext differentiates from other interfaces.
	IsIfexprContext()
}

type IfexprContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIfexprContext() *IfexprContext {
	var p = new(IfexprContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_ifexpr
	return p
}

func (*IfexprContext) IsIfexprContext() {}

func NewIfexprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IfexprContext {
	var p = new(IfexprContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_ifexpr

	return p
}

func (s *IfexprContext) GetParser() antlr.Parser { return s.parser }

func (s *IfexprContext) IF() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIF, 0)
}

func (s *IfexprContext) Condition() IConditionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConditionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConditionContext)
}

func (s *IfexprContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *IfexprContext) AllElseifexpr() []IElseifexprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IElseifexprContext); ok {
			len++
		}
	}

	tst := make([]IElseifexprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IElseifexprContext); ok {
			tst[i] = t.(IElseifexprContext)
			i++
		}
	}

	return tst
}

func (s *IfexprContext) Elseifexpr(i int) IElseifexprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IElseifexprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IElseifexprContext)
}

func (s *IfexprContext) Elseexpr() IElseexprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IElseexprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IElseexprContext)
}

func (s *IfexprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IfexprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IfexprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitIfexpr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Ifexpr() (localctx IIfexprContext) {
	this := p
	_ = this

	localctx = NewIfexprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, LogCarverParserRULE_ifexpr)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(77)
		p.Match(LogCarverParserIF)
	}
	{
		p.SetState(78)
		p.Condition()
	}
	{
		p.SetState(79)
		p.Block()
	}
	p.SetState(83)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == LogCarverParserELSEIF {
		{
			p.SetState(80)
			p.Elseifexpr()
		}

		p.SetState(85)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(87)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == LogCarverParserELSE {
		{
			p.SetState(86)
			p.Elseexpr()
		}

	}

	return localctx
}

// IElseifexprContext is an interface to support dynamic dispatch.
type IElseifexprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ELSEIF() antlr.TerminalNode
	Condition() IConditionContext
	Block() IBlockContext

	// IsElseifexprContext differentiates from other interfaces.
	IsElseifexprContext()
}

type ElseifexprContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyElseifexprContext() *ElseifexprContext {
	var p = new(ElseifexprContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_elseifexpr
	return p
}

func (*ElseifexprContext) IsElseifexprContext() {}

func NewElseifexprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ElseifexprContext {
	var p = new(ElseifexprContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_elseifexpr

	return p
}

func (s *ElseifexprContext) GetParser() antlr.Parser { return s.parser }

func (s *ElseifexprContext) ELSEIF() antlr.TerminalNode {
	return s.GetToken(LogCarverParserELSEIF, 0)
}

func (s *ElseifexprContext) Condition() IConditionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConditionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConditionContext)
}

func (s *ElseifexprContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ElseifexprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ElseifexprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ElseifexprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitElseifexpr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Elseifexpr() (localctx IElseifexprContext) {
	this := p
	_ = this

	localctx = NewElseifexprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, LogCarverParserRULE_elseifexpr)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(89)
		p.Match(LogCarverParserELSEIF)
	}
	{
		p.SetState(90)
		p.Condition()
	}
	{
		p.SetState(91)
		p.Block()
	}

	return localctx
}

// IElseexprContext is an interface to support dynamic dispatch.
type IElseexprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ELSE() antlr.TerminalNode
	Block() IBlockContext

	// IsElseexprContext differentiates from other interfaces.
	IsElseexprContext()
}

type ElseexprContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyElseexprContext() *ElseexprContext {
	var p = new(ElseexprContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_elseexpr
	return p
}

func (*ElseexprContext) IsElseexprContext() {}

func NewElseexprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ElseexprContext {
	var p = new(ElseexprContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_elseexpr

	return p
}

func (s *ElseexprContext) GetParser() antlr.Parser { return s.parser }

func (s *ElseexprContext) ELSE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserELSE, 0)
}

func (s *ElseexprContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ElseexprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ElseexprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ElseexprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitElseexpr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Elseexpr() (localctx IElseexprContext) {
	this := p
	_ = this

	localctx = NewElseexprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, LogCarverParserRULE_elseexpr)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(93)
		p.Match(LogCarverParserELSE)
	}
	{
		p.SetState(94)
		p.Block()
	}

	return localctx
}

// IConditionContext is an interface to support dynamic dispatch.
type IConditionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	Expr() IExprContext
	RPAREN() antlr.TerminalNode

	// IsConditionContext differentiates from other interfaces.
	IsConditionContext()
}

type ConditionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConditionContext() *ConditionContext {
	var p = new(ConditionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_condition
	return p
}

func (*ConditionContext) IsConditionContext() {}

func NewConditionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConditionContext {
	var p = new(ConditionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_condition

	return p
}

func (s *ConditionContext) GetParser() antlr.Parser { return s.parser }

func (s *ConditionContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLPAREN, 0)
}

func (s *ConditionContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *ConditionContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPAREN, 0)
}

func (s *ConditionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConditionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConditionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitCondition(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Condition() (localctx IConditionContext) {
	this := p
	_ = this

	localctx = NewConditionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, LogCarverParserRULE_condition)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(96)
		p.Match(LogCarverParserLPAREN)
	}
	{
		p.SetState(97)
		p.Expr()
	}
	{
		p.SetState(98)
		p.Match(LogCarverParserRPAREN)
	}

	return localctx
}

// IBlockContext is an interface to support dynamic dispatch.
type IBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	Lines() ILinesContext
	RBRACE() antlr.TerminalNode

	// IsBlockContext differentiates from other interfaces.
	IsBlockContext()
}

type BlockContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockContext() *BlockContext {
	var p = new(BlockContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_block
	return p
}

func (*BlockContext) IsBlockContext() {}

func NewBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockContext {
	var p = new(BlockContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_block

	return p
}

func (s *BlockContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACE, 0)
}

func (s *BlockContext) Lines() ILinesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILinesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILinesContext)
}

func (s *BlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRBRACE, 0)
}

func (s *BlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Block() (localctx IBlockContext) {
	this := p
	_ = this

	localctx = NewBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, LogCarverParserRULE_block)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(100)
		p.Match(LogCarverParserLBRACE)
	}
	{
		p.SetState(101)
		p.Lines()
	}
	{
		p.SetState(102)
		p.Match(LogCarverParserRBRACE)
	}

	return localctx
}

// ISetContext is an interface to support dynamic dispatch.
type ISetContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SET() antlr.TerminalNode
	Resolver() IResolverContext
	Expr() IExprContext

	// IsSetContext differentiates from other interfaces.
	IsSetContext()
}

type SetContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySetContext() *SetContext {
	var p = new(SetContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_set
	return p
}

func (*SetContext) IsSetContext() {}

func NewSetContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SetContext {
	var p = new(SetContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_set

	return p
}

func (s *SetContext) GetParser() antlr.Parser { return s.parser }

func (s *SetContext) SET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserSET, 0)
}

func (s *SetContext) Resolver() IResolverContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IResolverContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IResolverContext)
}

func (s *SetContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *SetContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SetContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SetContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitSet(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Set() (localctx ISetContext) {
	this := p
	_ = this

	localctx = NewSetContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, LogCarverParserRULE_set)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(104)
		p.Match(LogCarverParserSET)
	}
	{
		p.SetState(105)
		p.Resolver()
	}
	{
		p.SetState(106)
		p.Match(LogCarverParserT__0)
	}
	{
		p.SetState(107)
		p.Expr()
	}

	return localctx
}

// IUnsetContext is an interface to support dynamic dispatch.
type IUnsetContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	UNSET() antlr.TerminalNode
	Resolver() IResolverContext
	Expr() IExprContext

	// IsUnsetContext differentiates from other interfaces.
	IsUnsetContext()
}

type UnsetContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnsetContext() *UnsetContext {
	var p = new(UnsetContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_unset
	return p
}

func (*UnsetContext) IsUnsetContext() {}

func NewUnsetContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnsetContext {
	var p = new(UnsetContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_unset

	return p
}

func (s *UnsetContext) GetParser() antlr.Parser { return s.parser }

func (s *UnsetContext) UNSET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserUNSET, 0)
}

func (s *UnsetContext) Resolver() IResolverContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IResolverContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IResolverContext)
}

func (s *UnsetContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *UnsetContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnsetContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnsetContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitUnset(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Unset() (localctx IUnsetContext) {
	this := p
	_ = this

	localctx = NewUnsetContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, LogCarverParserRULE_unset)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(109)
		p.Match(LogCarverParserUNSET)
	}
	{
		p.SetState(110)
		p.Resolver()
	}
	{
		p.SetState(111)
		p.Match(LogCarverParserT__0)
	}
	{
		p.SetState(112)
		p.Expr()
	}

	return localctx
}

// IResolverContext is an interface to support dynamic dispatch.
type IResolverContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	AllDOT() []antlr.TerminalNode
	DOT(i int) antlr.TerminalNode
	AllLBRACKET() []antlr.TerminalNode
	LBRACKET(i int) antlr.TerminalNode
	AllRPRACKET() []antlr.TerminalNode
	RPRACKET(i int) antlr.TerminalNode
	AllSTRING() []antlr.TerminalNode
	STRING(i int) antlr.TerminalNode

	// IsResolverContext differentiates from other interfaces.
	IsResolverContext()
}

type ResolverContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyResolverContext() *ResolverContext {
	var p = new(ResolverContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_resolver
	return p
}

func (*ResolverContext) IsResolverContext() {}

func NewResolverContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ResolverContext {
	var p = new(ResolverContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_resolver

	return p
}

func (s *ResolverContext) GetParser() antlr.Parser { return s.parser }

func (s *ResolverContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserIDENTIFIER)
}

func (s *ResolverContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, i)
}

func (s *ResolverContext) AllDOT() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserDOT)
}

func (s *ResolverContext) DOT(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserDOT, i)
}

func (s *ResolverContext) AllLBRACKET() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserLBRACKET)
}

func (s *ResolverContext) LBRACKET(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACKET, i)
}

func (s *ResolverContext) AllRPRACKET() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserRPRACKET)
}

func (s *ResolverContext) RPRACKET(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPRACKET, i)
}

func (s *ResolverContext) AllSTRING() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserSTRING)
}

func (s *ResolverContext) STRING(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserSTRING, i)
}

func (s *ResolverContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ResolverContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ResolverContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitResolver(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Resolver() (localctx IResolverContext) {
	this := p
	_ = this

	localctx = NewResolverContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, LogCarverParserRULE_resolver)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(114)
		p.Match(LogCarverParserIDENTIFIER)
	}
	p.SetState(122)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == LogCarverParserLBRACKET || _la == LogCarverParserDOT {
		p.SetState(120)
		p.GetErrorHandler().Sync(p)

		switch p.GetTokenStream().LA(1) {
		case LogCarverParserLBRACKET:
			{
				p.SetState(115)
				p.Match(LogCarverParserLBRACKET)
			}
			{
				p.SetState(116)
				_la = p.GetTokenStream().LA(1)

				if !(_la == LogCarverParserSTRING || _la == LogCarverParserIDENTIFIER) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(117)
				p.Match(LogCarverParserRPRACKET)
			}

		case LogCarverParserDOT:
			{
				p.SetState(118)
				p.Match(LogCarverParserDOT)
			}
			{
				p.SetState(119)
				p.Match(LogCarverParserIDENTIFIER)
			}

		default:
			panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		}

		p.SetState(124)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IActionContext is an interface to support dynamic dispatch.
type IActionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	AllArgument() []IArgumentContext
	Argument(i int) IArgumentContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsActionContext differentiates from other interfaces.
	IsActionContext()
}

type ActionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyActionContext() *ActionContext {
	var p = new(ActionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_action
	return p
}

func (*ActionContext) IsActionContext() {}

func NewActionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ActionContext {
	var p = new(ActionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_action

	return p
}

func (s *ActionContext) GetParser() antlr.Parser { return s.parser }

func (s *ActionContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *ActionContext) AllArgument() []IArgumentContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IArgumentContext); ok {
			len++
		}
	}

	tst := make([]IArgumentContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IArgumentContext); ok {
			tst[i] = t.(IArgumentContext)
			i++
		}
	}

	return tst
}

func (s *ActionContext) Argument(i int) IArgumentContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentContext)
}

func (s *ActionContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOMMA)
}

func (s *ActionContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, i)
}

func (s *ActionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ActionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ActionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitAction(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Action_() (localctx IActionContext) {
	this := p
	_ = this

	localctx = NewActionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, LogCarverParserRULE_action)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(125)
		p.Match(LogCarverParserIDENTIFIER)
	}
	p.SetState(134)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == LogCarverParserIDENTIFIER {
		{
			p.SetState(126)
			p.Argument()
		}
		p.SetState(131)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		for _la == LogCarverParserCOMMA {
			{
				p.SetState(127)
				p.Match(LogCarverParserCOMMA)
			}
			{
				p.SetState(128)
				p.Argument()
			}

			p.SetState(133)
			p.GetErrorHandler().Sync(p)
			_la = p.GetTokenStream().LA(1)
		}

	}

	return localctx
}

// IArgumentContext is an interface to support dynamic dispatch.
type IArgumentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Expr() IExprContext

	// IsArgumentContext differentiates from other interfaces.
	IsArgumentContext()
}

type ArgumentContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArgumentContext() *ArgumentContext {
	var p = new(ArgumentContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_argument
	return p
}

func (*ArgumentContext) IsArgumentContext() {}

func NewArgumentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentContext {
	var p = new(ArgumentContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_argument

	return p
}

func (s *ArgumentContext) GetParser() antlr.Parser { return s.parser }

func (s *ArgumentContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *ArgumentContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *ArgumentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArgumentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArgumentContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitArgument(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Argument() (localctx IArgumentContext) {
	this := p
	_ = this

	localctx = NewArgumentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, LogCarverParserRULE_argument)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(136)
		p.Match(LogCarverParserIDENTIFIER)
	}
	{
		p.SetState(137)
		p.Match(LogCarverParserT__0)
	}
	{
		p.SetState(138)
		p.Expr()
	}

	return localctx
}

// IExprContext is an interface to support dynamic dispatch.
type IExprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetOp returns the op token.
	GetOp() antlr.Token

	// SetOp sets the op token.
	SetOp(antlr.Token)

	// GetE returns the e rule contexts.
	GetE() IConditionalOrContext

	// GetE1 returns the e1 rule contexts.
	GetE1() IConditionalOrContext

	// GetE2 returns the e2 rule contexts.
	GetE2() IExprContext

	// SetE sets the e rule contexts.
	SetE(IConditionalOrContext)

	// SetE1 sets the e1 rule contexts.
	SetE1(IConditionalOrContext)

	// SetE2 sets the e2 rule contexts.
	SetE2(IExprContext)

	// Getter signatures
	AllConditionalOr() []IConditionalOrContext
	ConditionalOr(i int) IConditionalOrContext
	COLON() antlr.TerminalNode
	QUESTIONMARK() antlr.TerminalNode
	Expr() IExprContext

	// IsExprContext differentiates from other interfaces.
	IsExprContext()
}

type ExprContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	e      IConditionalOrContext
	op     antlr.Token
	e1     IConditionalOrContext
	e2     IExprContext
}

func NewEmptyExprContext() *ExprContext {
	var p = new(ExprContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_expr
	return p
}

func (*ExprContext) IsExprContext() {}

func NewExprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprContext {
	var p = new(ExprContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_expr

	return p
}

func (s *ExprContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprContext) GetOp() antlr.Token { return s.op }

func (s *ExprContext) SetOp(v antlr.Token) { s.op = v }

func (s *ExprContext) GetE() IConditionalOrContext { return s.e }

func (s *ExprContext) GetE1() IConditionalOrContext { return s.e1 }

func (s *ExprContext) GetE2() IExprContext { return s.e2 }

func (s *ExprContext) SetE(v IConditionalOrContext) { s.e = v }

func (s *ExprContext) SetE1(v IConditionalOrContext) { s.e1 = v }

func (s *ExprContext) SetE2(v IExprContext) { s.e2 = v }

func (s *ExprContext) AllConditionalOr() []IConditionalOrContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IConditionalOrContext); ok {
			len++
		}
	}

	tst := make([]IConditionalOrContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IConditionalOrContext); ok {
			tst[i] = t.(IConditionalOrContext)
			i++
		}
	}

	return tst
}

func (s *ExprContext) ConditionalOr(i int) IConditionalOrContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConditionalOrContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConditionalOrContext)
}

func (s *ExprContext) COLON() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOLON, 0)
}

func (s *ExprContext) QUESTIONMARK() antlr.TerminalNode {
	return s.GetToken(LogCarverParserQUESTIONMARK, 0)
}

func (s *ExprContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *ExprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitExpr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Expr() (localctx IExprContext) {
	this := p
	_ = this

	localctx = NewExprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, LogCarverParserRULE_expr)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(140)

		var _x = p.ConditionalOr()

		localctx.(*ExprContext).e = _x
	}
	p.SetState(146)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == LogCarverParserQUESTIONMARK {
		{
			p.SetState(141)

			var _m = p.Match(LogCarverParserQUESTIONMARK)

			localctx.(*ExprContext).op = _m
		}
		{
			p.SetState(142)

			var _x = p.ConditionalOr()

			localctx.(*ExprContext).e1 = _x
		}
		{
			p.SetState(143)
			p.Match(LogCarverParserCOLON)
		}
		{
			p.SetState(144)

			var _x = p.Expr()

			localctx.(*ExprContext).e2 = _x
		}

	}

	return localctx
}

// IConditionalOrContext is an interface to support dynamic dispatch.
type IConditionalOrContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetS10 returns the s10 token.
	GetS10() antlr.Token

	// SetS10 sets the s10 token.
	SetS10(antlr.Token)

	// GetOps returns the ops token list.
	GetOps() []antlr.Token

	// SetOps sets the ops token list.
	SetOps([]antlr.Token)

	// GetE returns the e rule contexts.
	GetE() IConditionalAndContext

	// Get_conditionalAnd returns the _conditionalAnd rule contexts.
	Get_conditionalAnd() IConditionalAndContext

	// SetE sets the e rule contexts.
	SetE(IConditionalAndContext)

	// Set_conditionalAnd sets the _conditionalAnd rule contexts.
	Set_conditionalAnd(IConditionalAndContext)

	// GetE1 returns the e1 rule context list.
	GetE1() []IConditionalAndContext

	// SetE1 sets the e1 rule context list.
	SetE1([]IConditionalAndContext)

	// Getter signatures
	AllConditionalAnd() []IConditionalAndContext
	ConditionalAnd(i int) IConditionalAndContext
	AllLOGICAL_OR() []antlr.TerminalNode
	LOGICAL_OR(i int) antlr.TerminalNode

	// IsConditionalOrContext differentiates from other interfaces.
	IsConditionalOrContext()
}

type ConditionalOrContext struct {
	*antlr.BaseParserRuleContext
	parser          antlr.Parser
	e               IConditionalAndContext
	s10             antlr.Token
	ops             []antlr.Token
	_conditionalAnd IConditionalAndContext
	e1              []IConditionalAndContext
}

func NewEmptyConditionalOrContext() *ConditionalOrContext {
	var p = new(ConditionalOrContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_conditionalOr
	return p
}

func (*ConditionalOrContext) IsConditionalOrContext() {}

func NewConditionalOrContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConditionalOrContext {
	var p = new(ConditionalOrContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_conditionalOr

	return p
}

func (s *ConditionalOrContext) GetParser() antlr.Parser { return s.parser }

func (s *ConditionalOrContext) GetS10() antlr.Token { return s.s10 }

func (s *ConditionalOrContext) SetS10(v antlr.Token) { s.s10 = v }

func (s *ConditionalOrContext) GetOps() []antlr.Token { return s.ops }

func (s *ConditionalOrContext) SetOps(v []antlr.Token) { s.ops = v }

func (s *ConditionalOrContext) GetE() IConditionalAndContext { return s.e }

func (s *ConditionalOrContext) Get_conditionalAnd() IConditionalAndContext { return s._conditionalAnd }

func (s *ConditionalOrContext) SetE(v IConditionalAndContext) { s.e = v }

func (s *ConditionalOrContext) Set_conditionalAnd(v IConditionalAndContext) { s._conditionalAnd = v }

func (s *ConditionalOrContext) GetE1() []IConditionalAndContext { return s.e1 }

func (s *ConditionalOrContext) SetE1(v []IConditionalAndContext) { s.e1 = v }

func (s *ConditionalOrContext) AllConditionalAnd() []IConditionalAndContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IConditionalAndContext); ok {
			len++
		}
	}

	tst := make([]IConditionalAndContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IConditionalAndContext); ok {
			tst[i] = t.(IConditionalAndContext)
			i++
		}
	}

	return tst
}

func (s *ConditionalOrContext) ConditionalAnd(i int) IConditionalAndContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConditionalAndContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConditionalAndContext)
}

func (s *ConditionalOrContext) AllLOGICAL_OR() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserLOGICAL_OR)
}

func (s *ConditionalOrContext) LOGICAL_OR(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserLOGICAL_OR, i)
}

func (s *ConditionalOrContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConditionalOrContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConditionalOrContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitConditionalOr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) ConditionalOr() (localctx IConditionalOrContext) {
	this := p
	_ = this

	localctx = NewConditionalOrContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, LogCarverParserRULE_conditionalOr)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(148)

		var _x = p.ConditionalAnd()

		localctx.(*ConditionalOrContext).e = _x
	}
	p.SetState(153)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == LogCarverParserLOGICAL_OR {
		{
			p.SetState(149)

			var _m = p.Match(LogCarverParserLOGICAL_OR)

			localctx.(*ConditionalOrContext).s10 = _m
		}
		localctx.(*ConditionalOrContext).ops = append(localctx.(*ConditionalOrContext).ops, localctx.(*ConditionalOrContext).s10)
		{
			p.SetState(150)

			var _x = p.ConditionalAnd()

			localctx.(*ConditionalOrContext)._conditionalAnd = _x
		}
		localctx.(*ConditionalOrContext).e1 = append(localctx.(*ConditionalOrContext).e1, localctx.(*ConditionalOrContext)._conditionalAnd)

		p.SetState(155)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IConditionalAndContext is an interface to support dynamic dispatch.
type IConditionalAndContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetS9 returns the s9 token.
	GetS9() antlr.Token

	// SetS9 sets the s9 token.
	SetS9(antlr.Token)

	// GetOps returns the ops token list.
	GetOps() []antlr.Token

	// SetOps sets the ops token list.
	SetOps([]antlr.Token)

	// GetE returns the e rule contexts.
	GetE() IRelationContext

	// Get_relation returns the _relation rule contexts.
	Get_relation() IRelationContext

	// SetE sets the e rule contexts.
	SetE(IRelationContext)

	// Set_relation sets the _relation rule contexts.
	Set_relation(IRelationContext)

	// GetE1 returns the e1 rule context list.
	GetE1() []IRelationContext

	// SetE1 sets the e1 rule context list.
	SetE1([]IRelationContext)

	// Getter signatures
	AllRelation() []IRelationContext
	Relation(i int) IRelationContext
	AllLOGICAL_AND() []antlr.TerminalNode
	LOGICAL_AND(i int) antlr.TerminalNode

	// IsConditionalAndContext differentiates from other interfaces.
	IsConditionalAndContext()
}

type ConditionalAndContext struct {
	*antlr.BaseParserRuleContext
	parser    antlr.Parser
	e         IRelationContext
	s9        antlr.Token
	ops       []antlr.Token
	_relation IRelationContext
	e1        []IRelationContext
}

func NewEmptyConditionalAndContext() *ConditionalAndContext {
	var p = new(ConditionalAndContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_conditionalAnd
	return p
}

func (*ConditionalAndContext) IsConditionalAndContext() {}

func NewConditionalAndContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConditionalAndContext {
	var p = new(ConditionalAndContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_conditionalAnd

	return p
}

func (s *ConditionalAndContext) GetParser() antlr.Parser { return s.parser }

func (s *ConditionalAndContext) GetS9() antlr.Token { return s.s9 }

func (s *ConditionalAndContext) SetS9(v antlr.Token) { s.s9 = v }

func (s *ConditionalAndContext) GetOps() []antlr.Token { return s.ops }

func (s *ConditionalAndContext) SetOps(v []antlr.Token) { s.ops = v }

func (s *ConditionalAndContext) GetE() IRelationContext { return s.e }

func (s *ConditionalAndContext) Get_relation() IRelationContext { return s._relation }

func (s *ConditionalAndContext) SetE(v IRelationContext) { s.e = v }

func (s *ConditionalAndContext) Set_relation(v IRelationContext) { s._relation = v }

func (s *ConditionalAndContext) GetE1() []IRelationContext { return s.e1 }

func (s *ConditionalAndContext) SetE1(v []IRelationContext) { s.e1 = v }

func (s *ConditionalAndContext) AllRelation() []IRelationContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRelationContext); ok {
			len++
		}
	}

	tst := make([]IRelationContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRelationContext); ok {
			tst[i] = t.(IRelationContext)
			i++
		}
	}

	return tst
}

func (s *ConditionalAndContext) Relation(i int) IRelationContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRelationContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRelationContext)
}

func (s *ConditionalAndContext) AllLOGICAL_AND() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserLOGICAL_AND)
}

func (s *ConditionalAndContext) LOGICAL_AND(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserLOGICAL_AND, i)
}

func (s *ConditionalAndContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConditionalAndContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConditionalAndContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitConditionalAnd(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) ConditionalAnd() (localctx IConditionalAndContext) {
	this := p
	_ = this

	localctx = NewConditionalAndContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, LogCarverParserRULE_conditionalAnd)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(156)

		var _x = p.relation(0)

		localctx.(*ConditionalAndContext).e = _x
	}
	p.SetState(161)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == LogCarverParserLOGICAL_AND {
		{
			p.SetState(157)

			var _m = p.Match(LogCarverParserLOGICAL_AND)

			localctx.(*ConditionalAndContext).s9 = _m
		}
		localctx.(*ConditionalAndContext).ops = append(localctx.(*ConditionalAndContext).ops, localctx.(*ConditionalAndContext).s9)
		{
			p.SetState(158)

			var _x = p.relation(0)

			localctx.(*ConditionalAndContext)._relation = _x
		}
		localctx.(*ConditionalAndContext).e1 = append(localctx.(*ConditionalAndContext).e1, localctx.(*ConditionalAndContext)._relation)

		p.SetState(163)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IRelationContext is an interface to support dynamic dispatch.
type IRelationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetOp returns the op token.
	GetOp() antlr.Token

	// SetOp sets the op token.
	SetOp(antlr.Token)

	// Getter signatures
	Calc() ICalcContext
	AllRelation() []IRelationContext
	Relation(i int) IRelationContext
	LESS() antlr.TerminalNode
	LESS_EQUALS() antlr.TerminalNode
	GREATER_EQUALS() antlr.TerminalNode
	GREATER() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	NOT_EQUALS() antlr.TerminalNode
	IN() antlr.TerminalNode

	// IsRelationContext differentiates from other interfaces.
	IsRelationContext()
}

type RelationContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	op     antlr.Token
}

func NewEmptyRelationContext() *RelationContext {
	var p = new(RelationContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_relation
	return p
}

func (*RelationContext) IsRelationContext() {}

func NewRelationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *RelationContext {
	var p = new(RelationContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_relation

	return p
}

func (s *RelationContext) GetParser() antlr.Parser { return s.parser }

func (s *RelationContext) GetOp() antlr.Token { return s.op }

func (s *RelationContext) SetOp(v antlr.Token) { s.op = v }

func (s *RelationContext) Calc() ICalcContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICalcContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICalcContext)
}

func (s *RelationContext) AllRelation() []IRelationContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRelationContext); ok {
			len++
		}
	}

	tst := make([]IRelationContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRelationContext); ok {
			tst[i] = t.(IRelationContext)
			i++
		}
	}

	return tst
}

func (s *RelationContext) Relation(i int) IRelationContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRelationContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRelationContext)
}

func (s *RelationContext) LESS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLESS, 0)
}

func (s *RelationContext) LESS_EQUALS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLESS_EQUALS, 0)
}

func (s *RelationContext) GREATER_EQUALS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserGREATER_EQUALS, 0)
}

func (s *RelationContext) GREATER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserGREATER, 0)
}

func (s *RelationContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserEQUALS, 0)
}

func (s *RelationContext) NOT_EQUALS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserNOT_EQUALS, 0)
}

func (s *RelationContext) IN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIN, 0)
}

func (s *RelationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *RelationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *RelationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitRelation(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Relation() (localctx IRelationContext) {
	return p.relation(0)
}

func (p *LogCarverParser) relation(_p int) (localctx IRelationContext) {
	this := p
	_ = this

	var _parentctx antlr.ParserRuleContext = p.GetParserRuleContext()
	_parentState := p.GetState()
	localctx = NewRelationContext(p, p.GetParserRuleContext(), _parentState)
	var _prevctx IRelationContext = localctx
	var _ antlr.ParserRuleContext = _prevctx // TODO: To prevent unused variable warning.
	_startState := 34
	p.EnterRecursionRule(localctx, 34, LogCarverParserRULE_relation, _p)
	var _la int

	defer func() {
		p.UnrollRecursionContexts(_parentctx)
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(165)
		p.calc(0)
	}

	p.GetParserRuleContext().SetStop(p.GetTokenStream().LT(-1))
	p.SetState(172)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 12, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			if p.GetParseListeners() != nil {
				p.TriggerExitRuleEvent()
			}
			_prevctx = localctx
			localctx = NewRelationContext(p, _parentctx, _parentState)
			p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_relation)
			p.SetState(167)

			if !(p.Precpred(p.GetParserRuleContext(), 1)) {
				panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 1)", ""))
			}
			{
				p.SetState(168)

				var _lt = p.GetTokenStream().LT(1)

				localctx.(*RelationContext).op = _lt

				_la = p.GetTokenStream().LA(1)

				if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&508) != 0) {
					var _ri = p.GetErrorHandler().RecoverInline(p)

					localctx.(*RelationContext).op = _ri
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(169)
				p.relation(2)
			}

		}
		p.SetState(174)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 12, p.GetParserRuleContext())
	}

	return localctx
}

// ICalcContext is an interface to support dynamic dispatch.
type ICalcContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetOp returns the op token.
	GetOp() antlr.Token

	// SetOp sets the op token.
	SetOp(antlr.Token)

	// Getter signatures
	Unary() IUnaryContext
	AllCalc() []ICalcContext
	Calc(i int) ICalcContext
	STAR() antlr.TerminalNode
	SLASH() antlr.TerminalNode
	PERCENT() antlr.TerminalNode
	PLUS() antlr.TerminalNode
	MINUS() antlr.TerminalNode

	// IsCalcContext differentiates from other interfaces.
	IsCalcContext()
}

type CalcContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	op     antlr.Token
}

func NewEmptyCalcContext() *CalcContext {
	var p = new(CalcContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_calc
	return p
}

func (*CalcContext) IsCalcContext() {}

func NewCalcContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CalcContext {
	var p = new(CalcContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_calc

	return p
}

func (s *CalcContext) GetParser() antlr.Parser { return s.parser }

func (s *CalcContext) GetOp() antlr.Token { return s.op }

func (s *CalcContext) SetOp(v antlr.Token) { s.op = v }

func (s *CalcContext) Unary() IUnaryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnaryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnaryContext)
}

func (s *CalcContext) AllCalc() []ICalcContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ICalcContext); ok {
			len++
		}
	}

	tst := make([]ICalcContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ICalcContext); ok {
			tst[i] = t.(ICalcContext)
			i++
		}
	}

	return tst
}

func (s *CalcContext) Calc(i int) ICalcContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICalcContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICalcContext)
}

func (s *CalcContext) STAR() antlr.TerminalNode {
	return s.GetToken(LogCarverParserSTAR, 0)
}

func (s *CalcContext) SLASH() antlr.TerminalNode {
	return s.GetToken(LogCarverParserSLASH, 0)
}

func (s *CalcContext) PERCENT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserPERCENT, 0)
}

func (s *CalcContext) PLUS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserPLUS, 0)
}

func (s *CalcContext) MINUS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserMINUS, 0)
}

func (s *CalcContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CalcContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CalcContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitCalc(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Calc() (localctx ICalcContext) {
	return p.calc(0)
}

func (p *LogCarverParser) calc(_p int) (localctx ICalcContext) {
	this := p
	_ = this

	var _parentctx antlr.ParserRuleContext = p.GetParserRuleContext()
	_parentState := p.GetState()
	localctx = NewCalcContext(p, p.GetParserRuleContext(), _parentState)
	var _prevctx ICalcContext = localctx
	var _ antlr.ParserRuleContext = _prevctx // TODO: To prevent unused variable warning.
	_startState := 36
	p.EnterRecursionRule(localctx, 36, LogCarverParserRULE_calc, _p)
	var _la int

	defer func() {
		p.UnrollRecursionContexts(_parentctx)
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(176)
		p.Unary()
	}

	p.GetParserRuleContext().SetStop(p.GetTokenStream().LT(-1))
	p.SetState(186)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 14, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			if p.GetParseListeners() != nil {
				p.TriggerExitRuleEvent()
			}
			_prevctx = localctx
			p.SetState(184)
			p.GetErrorHandler().Sync(p)
			switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 13, p.GetParserRuleContext()) {
			case 1:
				localctx = NewCalcContext(p, _parentctx, _parentState)
				p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_calc)
				p.SetState(178)

				if !(p.Precpred(p.GetParserRuleContext(), 2)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 2)", ""))
				}
				{
					p.SetState(179)

					var _lt = p.GetTokenStream().LT(1)

					localctx.(*CalcContext).op = _lt

					_la = p.GetTokenStream().LA(1)

					if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&117440512) != 0) {
						var _ri = p.GetErrorHandler().RecoverInline(p)

						localctx.(*CalcContext).op = _ri
					} else {
						p.GetErrorHandler().ReportMatch(p)
						p.Consume()
					}
				}
				{
					p.SetState(180)
					p.calc(3)
				}

			case 2:
				localctx = NewCalcContext(p, _parentctx, _parentState)
				p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_calc)
				p.SetState(181)

				if !(p.Precpred(p.GetParserRuleContext(), 1)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 1)", ""))
				}
				{
					p.SetState(182)

					var _lt = p.GetTokenStream().LT(1)

					localctx.(*CalcContext).op = _lt

					_la = p.GetTokenStream().LA(1)

					if !(_la == LogCarverParserMINUS || _la == LogCarverParserPLUS) {
						var _ri = p.GetErrorHandler().RecoverInline(p)

						localctx.(*CalcContext).op = _ri
					} else {
						p.GetErrorHandler().ReportMatch(p)
						p.Consume()
					}
				}
				{
					p.SetState(183)
					p.calc(2)
				}

			}

		}
		p.SetState(188)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 14, p.GetParserRuleContext())
	}

	return localctx
}

// IUnaryContext is an interface to support dynamic dispatch.
type IUnaryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsUnaryContext differentiates from other interfaces.
	IsUnaryContext()
}

type UnaryContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnaryContext() *UnaryContext {
	var p = new(UnaryContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_unary
	return p
}

func (*UnaryContext) IsUnaryContext() {}

func NewUnaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnaryContext {
	var p = new(UnaryContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_unary

	return p
}

func (s *UnaryContext) GetParser() antlr.Parser { return s.parser }

func (s *UnaryContext) CopyFrom(ctx *UnaryContext) {
	s.BaseParserRuleContext.CopyFrom(ctx.BaseParserRuleContext)
}

func (s *UnaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type LogicalNotContext struct {
	*UnaryContext
	s20 antlr.Token
	ops []antlr.Token
}

func NewLogicalNotContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *LogicalNotContext {
	var p = new(LogicalNotContext)

	p.UnaryContext = NewEmptyUnaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*UnaryContext))

	return p
}

func (s *LogicalNotContext) GetS20() antlr.Token { return s.s20 }

func (s *LogicalNotContext) SetS20(v antlr.Token) { s.s20 = v }

func (s *LogicalNotContext) GetOps() []antlr.Token { return s.ops }

func (s *LogicalNotContext) SetOps(v []antlr.Token) { s.ops = v }

func (s *LogicalNotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LogicalNotContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *LogicalNotContext) AllEXCLAM() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserEXCLAM)
}

func (s *LogicalNotContext) EXCLAM(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserEXCLAM, i)
}

func (s *LogicalNotContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitLogicalNot(s)

	default:
		return t.VisitChildren(s)
	}
}

type MemberExprContext struct {
	*UnaryContext
}

func NewMemberExprContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *MemberExprContext {
	var p = new(MemberExprContext)

	p.UnaryContext = NewEmptyUnaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*UnaryContext))

	return p
}

func (s *MemberExprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MemberExprContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *MemberExprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitMemberExpr(s)

	default:
		return t.VisitChildren(s)
	}
}

type NegateContext struct {
	*UnaryContext
	s19 antlr.Token
	ops []antlr.Token
}

func NewNegateContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *NegateContext {
	var p = new(NegateContext)

	p.UnaryContext = NewEmptyUnaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*UnaryContext))

	return p
}

func (s *NegateContext) GetS19() antlr.Token { return s.s19 }

func (s *NegateContext) SetS19(v antlr.Token) { s.s19 = v }

func (s *NegateContext) GetOps() []antlr.Token { return s.ops }

func (s *NegateContext) SetOps(v []antlr.Token) { s.ops = v }

func (s *NegateContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NegateContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *NegateContext) AllMINUS() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserMINUS)
}

func (s *NegateContext) MINUS(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserMINUS, i)
}

func (s *NegateContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitNegate(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Unary() (localctx IUnaryContext) {
	this := p
	_ = this

	localctx = NewUnaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, LogCarverParserRULE_unary)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.SetState(202)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 17, p.GetParserRuleContext()) {
	case 1:
		localctx = NewMemberExprContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(189)
			p.member(0)
		}

	case 2:
		localctx = NewLogicalNotContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		p.SetState(191)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		for ok := true; ok; ok = _la == LogCarverParserEXCLAM {
			{
				p.SetState(190)

				var _m = p.Match(LogCarverParserEXCLAM)

				localctx.(*LogicalNotContext).s20 = _m
			}
			localctx.(*LogicalNotContext).ops = append(localctx.(*LogicalNotContext).ops, localctx.(*LogicalNotContext).s20)

			p.SetState(193)
			p.GetErrorHandler().Sync(p)
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(195)
			p.member(0)
		}

	case 3:
		localctx = NewNegateContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		p.SetState(197)
		p.GetErrorHandler().Sync(p)
		_alt = 1
		for ok := true; ok; ok = _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			switch _alt {
			case 1:
				{
					p.SetState(196)

					var _m = p.Match(LogCarverParserMINUS)

					localctx.(*NegateContext).s19 = _m
				}
				localctx.(*NegateContext).ops = append(localctx.(*NegateContext).ops, localctx.(*NegateContext).s19)

			default:
				panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			}

			p.SetState(199)
			p.GetErrorHandler().Sync(p)
			_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 16, p.GetParserRuleContext())
		}
		{
			p.SetState(201)
			p.member(0)
		}

	}

	return localctx
}

// IMemberContext is an interface to support dynamic dispatch.
type IMemberContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsMemberContext differentiates from other interfaces.
	IsMemberContext()
}

type MemberContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMemberContext() *MemberContext {
	var p = new(MemberContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_member
	return p
}

func (*MemberContext) IsMemberContext() {}

func NewMemberContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MemberContext {
	var p = new(MemberContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_member

	return p
}

func (s *MemberContext) GetParser() antlr.Parser { return s.parser }

func (s *MemberContext) CopyFrom(ctx *MemberContext) {
	s.BaseParserRuleContext.CopyFrom(ctx.BaseParserRuleContext)
}

func (s *MemberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MemberContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type MemberCallContext struct {
	*MemberContext
	op   antlr.Token
	id   antlr.Token
	open antlr.Token
	args IExprListContext
}

func NewMemberCallContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *MemberCallContext {
	var p = new(MemberCallContext)

	p.MemberContext = NewEmptyMemberContext()
	p.parser = parser
	p.CopyFrom(ctx.(*MemberContext))

	return p
}

func (s *MemberCallContext) GetOp() antlr.Token { return s.op }

func (s *MemberCallContext) GetId() antlr.Token { return s.id }

func (s *MemberCallContext) GetOpen() antlr.Token { return s.open }

func (s *MemberCallContext) SetOp(v antlr.Token) { s.op = v }

func (s *MemberCallContext) SetId(v antlr.Token) { s.id = v }

func (s *MemberCallContext) SetOpen(v antlr.Token) { s.open = v }

func (s *MemberCallContext) GetArgs() IExprListContext { return s.args }

func (s *MemberCallContext) SetArgs(v IExprListContext) { s.args = v }

func (s *MemberCallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MemberCallContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *MemberCallContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPAREN, 0)
}

func (s *MemberCallContext) DOT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserDOT, 0)
}

func (s *MemberCallContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *MemberCallContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLPAREN, 0)
}

func (s *MemberCallContext) ExprList() IExprListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprListContext)
}

func (s *MemberCallContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitMemberCall(s)

	default:
		return t.VisitChildren(s)
	}
}

type SelectContext struct {
	*MemberContext
	op  antlr.Token
	opt antlr.Token
	id  antlr.Token
}

func NewSelectContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *SelectContext {
	var p = new(SelectContext)

	p.MemberContext = NewEmptyMemberContext()
	p.parser = parser
	p.CopyFrom(ctx.(*MemberContext))

	return p
}

func (s *SelectContext) GetOp() antlr.Token { return s.op }

func (s *SelectContext) GetOpt() antlr.Token { return s.opt }

func (s *SelectContext) GetId() antlr.Token { return s.id }

func (s *SelectContext) SetOp(v antlr.Token) { s.op = v }

func (s *SelectContext) SetOpt(v antlr.Token) { s.opt = v }

func (s *SelectContext) SetId(v antlr.Token) { s.id = v }

func (s *SelectContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SelectContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *SelectContext) DOT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserDOT, 0)
}

func (s *SelectContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *SelectContext) QUESTIONMARK() antlr.TerminalNode {
	return s.GetToken(LogCarverParserQUESTIONMARK, 0)
}

func (s *SelectContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitSelect(s)

	default:
		return t.VisitChildren(s)
	}
}

type PrimaryExprContext struct {
	*MemberContext
}

func NewPrimaryExprContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *PrimaryExprContext {
	var p = new(PrimaryExprContext)

	p.MemberContext = NewEmptyMemberContext()
	p.parser = parser
	p.CopyFrom(ctx.(*MemberContext))

	return p
}

func (s *PrimaryExprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrimaryExprContext) Primary() IPrimaryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimaryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimaryContext)
}

func (s *PrimaryExprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitPrimaryExpr(s)

	default:
		return t.VisitChildren(s)
	}
}

type IndexContext struct {
	*MemberContext
	op    antlr.Token
	opt   antlr.Token
	index IExprContext
}

func NewIndexContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *IndexContext {
	var p = new(IndexContext)

	p.MemberContext = NewEmptyMemberContext()
	p.parser = parser
	p.CopyFrom(ctx.(*MemberContext))

	return p
}

func (s *IndexContext) GetOp() antlr.Token { return s.op }

func (s *IndexContext) GetOpt() antlr.Token { return s.opt }

func (s *IndexContext) SetOp(v antlr.Token) { s.op = v }

func (s *IndexContext) SetOpt(v antlr.Token) { s.opt = v }

func (s *IndexContext) GetIndex() IExprContext { return s.index }

func (s *IndexContext) SetIndex(v IExprContext) { s.index = v }

func (s *IndexContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IndexContext) Member() IMemberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMemberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMemberContext)
}

func (s *IndexContext) RPRACKET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPRACKET, 0)
}

func (s *IndexContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACKET, 0)
}

func (s *IndexContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *IndexContext) QUESTIONMARK() antlr.TerminalNode {
	return s.GetToken(LogCarverParserQUESTIONMARK, 0)
}

func (s *IndexContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitIndex(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Member() (localctx IMemberContext) {
	return p.member(0)
}

func (p *LogCarverParser) member(_p int) (localctx IMemberContext) {
	this := p
	_ = this

	var _parentctx antlr.ParserRuleContext = p.GetParserRuleContext()
	_parentState := p.GetState()
	localctx = NewMemberContext(p, p.GetParserRuleContext(), _parentState)
	var _prevctx IMemberContext = localctx
	var _ antlr.ParserRuleContext = _prevctx // TODO: To prevent unused variable warning.
	_startState := 40
	p.EnterRecursionRule(localctx, 40, LogCarverParserRULE_member, _p)
	var _la int

	defer func() {
		p.UnrollRecursionContexts(_parentctx)
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	localctx = NewPrimaryExprContext(p, localctx)
	p.SetParserRuleContext(localctx)
	_prevctx = localctx

	{
		p.SetState(205)
		p.Primary()
	}

	p.GetParserRuleContext().SetStop(p.GetTokenStream().LT(-1))
	p.SetState(231)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 22, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			if p.GetParseListeners() != nil {
				p.TriggerExitRuleEvent()
			}
			_prevctx = localctx
			p.SetState(229)
			p.GetErrorHandler().Sync(p)
			switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 21, p.GetParserRuleContext()) {
			case 1:
				localctx = NewSelectContext(p, NewMemberContext(p, _parentctx, _parentState))
				p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_member)
				p.SetState(207)

				if !(p.Precpred(p.GetParserRuleContext(), 3)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 3)", ""))
				}
				{
					p.SetState(208)

					var _m = p.Match(LogCarverParserDOT)

					localctx.(*SelectContext).op = _m
				}
				p.SetState(210)
				p.GetErrorHandler().Sync(p)
				_la = p.GetTokenStream().LA(1)

				if _la == LogCarverParserQUESTIONMARK {
					{
						p.SetState(209)

						var _m = p.Match(LogCarverParserQUESTIONMARK)

						localctx.(*SelectContext).opt = _m
					}

				}
				{
					p.SetState(212)

					var _m = p.Match(LogCarverParserIDENTIFIER)

					localctx.(*SelectContext).id = _m
				}

			case 2:
				localctx = NewMemberCallContext(p, NewMemberContext(p, _parentctx, _parentState))
				p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_member)
				p.SetState(213)

				if !(p.Precpred(p.GetParserRuleContext(), 2)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 2)", ""))
				}
				{
					p.SetState(214)

					var _m = p.Match(LogCarverParserDOT)

					localctx.(*MemberCallContext).op = _m
				}
				{
					p.SetState(215)

					var _m = p.Match(LogCarverParserIDENTIFIER)

					localctx.(*MemberCallContext).id = _m
				}
				{
					p.SetState(216)

					var _m = p.Match(LogCarverParserLPAREN)

					localctx.(*MemberCallContext).open = _m
				}
				p.SetState(218)
				p.GetErrorHandler().Sync(p)
				_la = p.GetTokenStream().LA(1)

				if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8930178279424) != 0 {
					{
						p.SetState(217)

						var _x = p.ExprList()

						localctx.(*MemberCallContext).args = _x
					}

				}
				{
					p.SetState(220)
					p.Match(LogCarverParserRPAREN)
				}

			case 3:
				localctx = NewIndexContext(p, NewMemberContext(p, _parentctx, _parentState))
				p.PushNewRecursionContext(localctx, _startState, LogCarverParserRULE_member)
				p.SetState(221)

				if !(p.Precpred(p.GetParserRuleContext(), 1)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 1)", ""))
				}
				{
					p.SetState(222)

					var _m = p.Match(LogCarverParserLBRACKET)

					localctx.(*IndexContext).op = _m
				}
				p.SetState(224)
				p.GetErrorHandler().Sync(p)
				_la = p.GetTokenStream().LA(1)

				if _la == LogCarverParserQUESTIONMARK {
					{
						p.SetState(223)

						var _m = p.Match(LogCarverParserQUESTIONMARK)

						localctx.(*IndexContext).opt = _m
					}

				}
				{
					p.SetState(226)

					var _x = p.Expr()

					localctx.(*IndexContext).index = _x
				}
				{
					p.SetState(227)
					p.Match(LogCarverParserRPRACKET)
				}

			}

		}
		p.SetState(233)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 22, p.GetParserRuleContext())
	}

	return localctx
}

// IPrimaryContext is an interface to support dynamic dispatch.
type IPrimaryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsPrimaryContext differentiates from other interfaces.
	IsPrimaryContext()
}

type PrimaryContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrimaryContext() *PrimaryContext {
	var p = new(PrimaryContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_primary
	return p
}

func (*PrimaryContext) IsPrimaryContext() {}

func NewPrimaryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrimaryContext {
	var p = new(PrimaryContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_primary

	return p
}

func (s *PrimaryContext) GetParser() antlr.Parser { return s.parser }

func (s *PrimaryContext) CopyFrom(ctx *PrimaryContext) {
	s.BaseParserRuleContext.CopyFrom(ctx.BaseParserRuleContext)
}

func (s *PrimaryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrimaryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type CreateListContext struct {
	*PrimaryContext
	op    antlr.Token
	elems IListInitContext
}

func NewCreateListContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *CreateListContext {
	var p = new(CreateListContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *CreateListContext) GetOp() antlr.Token { return s.op }

func (s *CreateListContext) SetOp(v antlr.Token) { s.op = v }

func (s *CreateListContext) GetElems() IListInitContext { return s.elems }

func (s *CreateListContext) SetElems(v IListInitContext) { s.elems = v }

func (s *CreateListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CreateListContext) RPRACKET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPRACKET, 0)
}

func (s *CreateListContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACKET, 0)
}

func (s *CreateListContext) COMMA() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, 0)
}

func (s *CreateListContext) ListInit() IListInitContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IListInitContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IListInitContext)
}

func (s *CreateListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitCreateList(s)

	default:
		return t.VisitChildren(s)
	}
}

type CreateStructContext struct {
	*PrimaryContext
	op      antlr.Token
	entries IMapInitializerListContext
}

func NewCreateStructContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *CreateStructContext {
	var p = new(CreateStructContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *CreateStructContext) GetOp() antlr.Token { return s.op }

func (s *CreateStructContext) SetOp(v antlr.Token) { s.op = v }

func (s *CreateStructContext) GetEntries() IMapInitializerListContext { return s.entries }

func (s *CreateStructContext) SetEntries(v IMapInitializerListContext) { s.entries = v }

func (s *CreateStructContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CreateStructContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRBRACE, 0)
}

func (s *CreateStructContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACE, 0)
}

func (s *CreateStructContext) COMMA() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, 0)
}

func (s *CreateStructContext) MapInitializerList() IMapInitializerListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMapInitializerListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMapInitializerListContext)
}

func (s *CreateStructContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitCreateStruct(s)

	default:
		return t.VisitChildren(s)
	}
}

type ConstantLiteralContext struct {
	*PrimaryContext
}

func NewConstantLiteralContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ConstantLiteralContext {
	var p = new(ConstantLiteralContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *ConstantLiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConstantLiteralContext) Literal() ILiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILiteralContext)
}

func (s *ConstantLiteralContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitConstantLiteral(s)

	default:
		return t.VisitChildren(s)
	}
}

type NestedContext struct {
	*PrimaryContext
	e IExprContext
}

func NewNestedContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *NestedContext {
	var p = new(NestedContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *NestedContext) GetE() IExprContext { return s.e }

func (s *NestedContext) SetE(v IExprContext) { s.e = v }

func (s *NestedContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NestedContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLPAREN, 0)
}

func (s *NestedContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPAREN, 0)
}

func (s *NestedContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *NestedContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitNested(s)

	default:
		return t.VisitChildren(s)
	}
}

type CreateMessageContext struct {
	*PrimaryContext
	leadingDot  antlr.Token
	_IDENTIFIER antlr.Token
	ids         []antlr.Token
	s17         antlr.Token
	ops         []antlr.Token
	op          antlr.Token
	entries     IFieldInitializerListContext
}

func NewCreateMessageContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *CreateMessageContext {
	var p = new(CreateMessageContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *CreateMessageContext) GetLeadingDot() antlr.Token { return s.leadingDot }

func (s *CreateMessageContext) Get_IDENTIFIER() antlr.Token { return s._IDENTIFIER }

func (s *CreateMessageContext) GetS17() antlr.Token { return s.s17 }

func (s *CreateMessageContext) GetOp() antlr.Token { return s.op }

func (s *CreateMessageContext) SetLeadingDot(v antlr.Token) { s.leadingDot = v }

func (s *CreateMessageContext) Set_IDENTIFIER(v antlr.Token) { s._IDENTIFIER = v }

func (s *CreateMessageContext) SetS17(v antlr.Token) { s.s17 = v }

func (s *CreateMessageContext) SetOp(v antlr.Token) { s.op = v }

func (s *CreateMessageContext) GetIds() []antlr.Token { return s.ids }

func (s *CreateMessageContext) GetOps() []antlr.Token { return s.ops }

func (s *CreateMessageContext) SetIds(v []antlr.Token) { s.ids = v }

func (s *CreateMessageContext) SetOps(v []antlr.Token) { s.ops = v }

func (s *CreateMessageContext) GetEntries() IFieldInitializerListContext { return s.entries }

func (s *CreateMessageContext) SetEntries(v IFieldInitializerListContext) { s.entries = v }

func (s *CreateMessageContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CreateMessageContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRBRACE, 0)
}

func (s *CreateMessageContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserIDENTIFIER)
}

func (s *CreateMessageContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, i)
}

func (s *CreateMessageContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLBRACE, 0)
}

func (s *CreateMessageContext) COMMA() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, 0)
}

func (s *CreateMessageContext) AllDOT() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserDOT)
}

func (s *CreateMessageContext) DOT(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserDOT, i)
}

func (s *CreateMessageContext) FieldInitializerList() IFieldInitializerListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldInitializerListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFieldInitializerListContext)
}

func (s *CreateMessageContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitCreateMessage(s)

	default:
		return t.VisitChildren(s)
	}
}

type IdentOrGlobalCallContext struct {
	*PrimaryContext
	leadingDot antlr.Token
	id         antlr.Token
	op         antlr.Token
	args       IExprListContext
}

func NewIdentOrGlobalCallContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *IdentOrGlobalCallContext {
	var p = new(IdentOrGlobalCallContext)

	p.PrimaryContext = NewEmptyPrimaryContext()
	p.parser = parser
	p.CopyFrom(ctx.(*PrimaryContext))

	return p
}

func (s *IdentOrGlobalCallContext) GetLeadingDot() antlr.Token { return s.leadingDot }

func (s *IdentOrGlobalCallContext) GetId() antlr.Token { return s.id }

func (s *IdentOrGlobalCallContext) GetOp() antlr.Token { return s.op }

func (s *IdentOrGlobalCallContext) SetLeadingDot(v antlr.Token) { s.leadingDot = v }

func (s *IdentOrGlobalCallContext) SetId(v antlr.Token) { s.id = v }

func (s *IdentOrGlobalCallContext) SetOp(v antlr.Token) { s.op = v }

func (s *IdentOrGlobalCallContext) GetArgs() IExprListContext { return s.args }

func (s *IdentOrGlobalCallContext) SetArgs(v IExprListContext) { s.args = v }

func (s *IdentOrGlobalCallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IdentOrGlobalCallContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *IdentOrGlobalCallContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserRPAREN, 0)
}

func (s *IdentOrGlobalCallContext) DOT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserDOT, 0)
}

func (s *IdentOrGlobalCallContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(LogCarverParserLPAREN, 0)
}

func (s *IdentOrGlobalCallContext) ExprList() IExprListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprListContext)
}

func (s *IdentOrGlobalCallContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitIdentOrGlobalCall(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Primary() (localctx IPrimaryContext) {
	this := p
	_ = this

	localctx = NewPrimaryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, LogCarverParserRULE_primary)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(285)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 34, p.GetParserRuleContext()) {
	case 1:
		localctx = NewIdentOrGlobalCallContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		p.SetState(235)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserDOT {
			{
				p.SetState(234)

				var _m = p.Match(LogCarverParserDOT)

				localctx.(*IdentOrGlobalCallContext).leadingDot = _m
			}

		}
		{
			p.SetState(237)

			var _m = p.Match(LogCarverParserIDENTIFIER)

			localctx.(*IdentOrGlobalCallContext).id = _m
		}
		p.SetState(243)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 25, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(238)

				var _m = p.Match(LogCarverParserLPAREN)

				localctx.(*IdentOrGlobalCallContext).op = _m
			}
			p.SetState(240)
			p.GetErrorHandler().Sync(p)
			_la = p.GetTokenStream().LA(1)

			if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8930178279424) != 0 {
				{
					p.SetState(239)

					var _x = p.ExprList()

					localctx.(*IdentOrGlobalCallContext).args = _x
				}

			}
			{
				p.SetState(242)
				p.Match(LogCarverParserRPAREN)
			}

		}

	case 2:
		localctx = NewNestedContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(245)
			p.Match(LogCarverParserLPAREN)
		}
		{
			p.SetState(246)

			var _x = p.Expr()

			localctx.(*NestedContext).e = _x
		}
		{
			p.SetState(247)
			p.Match(LogCarverParserRPAREN)
		}

	case 3:
		localctx = NewCreateListContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(249)

			var _m = p.Match(LogCarverParserLBRACKET)

			localctx.(*CreateListContext).op = _m
		}
		p.SetState(251)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8930180376576) != 0 {
			{
				p.SetState(250)

				var _x = p.ListInit()

				localctx.(*CreateListContext).elems = _x
			}

		}
		p.SetState(254)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserCOMMA {
			{
				p.SetState(253)
				p.Match(LogCarverParserCOMMA)
			}

		}
		{
			p.SetState(256)
			p.Match(LogCarverParserRPRACKET)
		}

	case 4:
		localctx = NewCreateStructContext(p, localctx)
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(257)

			var _m = p.Match(LogCarverParserLBRACE)

			localctx.(*CreateStructContext).op = _m
		}
		p.SetState(259)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8930180376576) != 0 {
			{
				p.SetState(258)

				var _x = p.MapInitializerList()

				localctx.(*CreateStructContext).entries = _x
			}

		}
		p.SetState(262)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserCOMMA {
			{
				p.SetState(261)
				p.Match(LogCarverParserCOMMA)
			}

		}
		{
			p.SetState(264)
			p.Match(LogCarverParserRBRACE)
		}

	case 5:
		localctx = NewCreateMessageContext(p, localctx)
		p.EnterOuterAlt(localctx, 5)
		p.SetState(266)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserDOT {
			{
				p.SetState(265)

				var _m = p.Match(LogCarverParserDOT)

				localctx.(*CreateMessageContext).leadingDot = _m
			}

		}
		{
			p.SetState(268)

			var _m = p.Match(LogCarverParserIDENTIFIER)

			localctx.(*CreateMessageContext)._IDENTIFIER = _m
		}
		localctx.(*CreateMessageContext).ids = append(localctx.(*CreateMessageContext).ids, localctx.(*CreateMessageContext)._IDENTIFIER)
		p.SetState(273)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		for _la == LogCarverParserDOT {
			{
				p.SetState(269)

				var _m = p.Match(LogCarverParserDOT)

				localctx.(*CreateMessageContext).s17 = _m
			}
			localctx.(*CreateMessageContext).ops = append(localctx.(*CreateMessageContext).ops, localctx.(*CreateMessageContext).s17)
			{
				p.SetState(270)

				var _m = p.Match(LogCarverParserIDENTIFIER)

				localctx.(*CreateMessageContext)._IDENTIFIER = _m
			}
			localctx.(*CreateMessageContext).ids = append(localctx.(*CreateMessageContext).ids, localctx.(*CreateMessageContext)._IDENTIFIER)

			p.SetState(275)
			p.GetErrorHandler().Sync(p)
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(276)

			var _m = p.Match(LogCarverParserLBRACE)

			localctx.(*CreateMessageContext).op = _m
		}
		p.SetState(278)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserQUESTIONMARK || _la == LogCarverParserIDENTIFIER {
			{
				p.SetState(277)

				var _x = p.FieldInitializerList()

				localctx.(*CreateMessageContext).entries = _x
			}

		}
		p.SetState(281)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserCOMMA {
			{
				p.SetState(280)
				p.Match(LogCarverParserCOMMA)
			}

		}
		{
			p.SetState(283)
			p.Match(LogCarverParserRBRACE)
		}

	case 6:
		localctx = NewConstantLiteralContext(p, localctx)
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(284)
			p.Literal()
		}

	}

	return localctx
}

// IExprListContext is an interface to support dynamic dispatch.
type IExprListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Get_expr returns the _expr rule contexts.
	Get_expr() IExprContext

	// Set_expr sets the _expr rule contexts.
	Set_expr(IExprContext)

	// GetE returns the e rule context list.
	GetE() []IExprContext

	// SetE sets the e rule context list.
	SetE([]IExprContext)

	// Getter signatures
	AllExpr() []IExprContext
	Expr(i int) IExprContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsExprListContext differentiates from other interfaces.
	IsExprListContext()
}

type ExprListContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	_expr  IExprContext
	e      []IExprContext
}

func NewEmptyExprListContext() *ExprListContext {
	var p = new(ExprListContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_exprList
	return p
}

func (*ExprListContext) IsExprListContext() {}

func NewExprListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExprListContext {
	var p = new(ExprListContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_exprList

	return p
}

func (s *ExprListContext) GetParser() antlr.Parser { return s.parser }

func (s *ExprListContext) Get_expr() IExprContext { return s._expr }

func (s *ExprListContext) Set_expr(v IExprContext) { s._expr = v }

func (s *ExprListContext) GetE() []IExprContext { return s.e }

func (s *ExprListContext) SetE(v []IExprContext) { s.e = v }

func (s *ExprListContext) AllExpr() []IExprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprContext); ok {
			len++
		}
	}

	tst := make([]IExprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprContext); ok {
			tst[i] = t.(IExprContext)
			i++
		}
	}

	return tst
}

func (s *ExprListContext) Expr(i int) IExprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *ExprListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOMMA)
}

func (s *ExprListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, i)
}

func (s *ExprListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExprListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExprListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitExprList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) ExprList() (localctx IExprListContext) {
	this := p
	_ = this

	localctx = NewExprListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, LogCarverParserRULE_exprList)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(287)

		var _x = p.Expr()

		localctx.(*ExprListContext)._expr = _x
	}
	localctx.(*ExprListContext).e = append(localctx.(*ExprListContext).e, localctx.(*ExprListContext)._expr)
	p.SetState(292)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == LogCarverParserCOMMA {
		{
			p.SetState(288)
			p.Match(LogCarverParserCOMMA)
		}
		{
			p.SetState(289)

			var _x = p.Expr()

			localctx.(*ExprListContext)._expr = _x
		}
		localctx.(*ExprListContext).e = append(localctx.(*ExprListContext).e, localctx.(*ExprListContext)._expr)

		p.SetState(294)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IListInitContext is an interface to support dynamic dispatch.
type IListInitContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Get_optExpr returns the _optExpr rule contexts.
	Get_optExpr() IOptExprContext

	// Set_optExpr sets the _optExpr rule contexts.
	Set_optExpr(IOptExprContext)

	// GetElems returns the elems rule context list.
	GetElems() []IOptExprContext

	// SetElems sets the elems rule context list.
	SetElems([]IOptExprContext)

	// Getter signatures
	AllOptExpr() []IOptExprContext
	OptExpr(i int) IOptExprContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsListInitContext differentiates from other interfaces.
	IsListInitContext()
}

type ListInitContext struct {
	*antlr.BaseParserRuleContext
	parser   antlr.Parser
	_optExpr IOptExprContext
	elems    []IOptExprContext
}

func NewEmptyListInitContext() *ListInitContext {
	var p = new(ListInitContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_listInit
	return p
}

func (*ListInitContext) IsListInitContext() {}

func NewListInitContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ListInitContext {
	var p = new(ListInitContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_listInit

	return p
}

func (s *ListInitContext) GetParser() antlr.Parser { return s.parser }

func (s *ListInitContext) Get_optExpr() IOptExprContext { return s._optExpr }

func (s *ListInitContext) Set_optExpr(v IOptExprContext) { s._optExpr = v }

func (s *ListInitContext) GetElems() []IOptExprContext { return s.elems }

func (s *ListInitContext) SetElems(v []IOptExprContext) { s.elems = v }

func (s *ListInitContext) AllOptExpr() []IOptExprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOptExprContext); ok {
			len++
		}
	}

	tst := make([]IOptExprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOptExprContext); ok {
			tst[i] = t.(IOptExprContext)
			i++
		}
	}

	return tst
}

func (s *ListInitContext) OptExpr(i int) IOptExprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOptExprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOptExprContext)
}

func (s *ListInitContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOMMA)
}

func (s *ListInitContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, i)
}

func (s *ListInitContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ListInitContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ListInitContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitListInit(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) ListInit() (localctx IListInitContext) {
	this := p
	_ = this

	localctx = NewListInitContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, LogCarverParserRULE_listInit)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(295)

		var _x = p.OptExpr()

		localctx.(*ListInitContext)._optExpr = _x
	}
	localctx.(*ListInitContext).elems = append(localctx.(*ListInitContext).elems, localctx.(*ListInitContext)._optExpr)
	p.SetState(300)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 36, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(296)
				p.Match(LogCarverParserCOMMA)
			}
			{
				p.SetState(297)

				var _x = p.OptExpr()

				localctx.(*ListInitContext)._optExpr = _x
			}
			localctx.(*ListInitContext).elems = append(localctx.(*ListInitContext).elems, localctx.(*ListInitContext)._optExpr)

		}
		p.SetState(302)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 36, p.GetParserRuleContext())
	}

	return localctx
}

// IFieldInitializerListContext is an interface to support dynamic dispatch.
type IFieldInitializerListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetS22 returns the s22 token.
	GetS22() antlr.Token

	// SetS22 sets the s22 token.
	SetS22(antlr.Token)

	// GetCols returns the cols token list.
	GetCols() []antlr.Token

	// SetCols sets the cols token list.
	SetCols([]antlr.Token)

	// Get_optField returns the _optField rule contexts.
	Get_optField() IOptFieldContext

	// Get_expr returns the _expr rule contexts.
	Get_expr() IExprContext

	// Set_optField sets the _optField rule contexts.
	Set_optField(IOptFieldContext)

	// Set_expr sets the _expr rule contexts.
	Set_expr(IExprContext)

	// GetFields returns the fields rule context list.
	GetFields() []IOptFieldContext

	// GetValues returns the values rule context list.
	GetValues() []IExprContext

	// SetFields sets the fields rule context list.
	SetFields([]IOptFieldContext)

	// SetValues sets the values rule context list.
	SetValues([]IExprContext)

	// Getter signatures
	AllOptField() []IOptFieldContext
	OptField(i int) IOptFieldContext
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	AllExpr() []IExprContext
	Expr(i int) IExprContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsFieldInitializerListContext differentiates from other interfaces.
	IsFieldInitializerListContext()
}

type FieldInitializerListContext struct {
	*antlr.BaseParserRuleContext
	parser    antlr.Parser
	_optField IOptFieldContext
	fields    []IOptFieldContext
	s22       antlr.Token
	cols      []antlr.Token
	_expr     IExprContext
	values    []IExprContext
}

func NewEmptyFieldInitializerListContext() *FieldInitializerListContext {
	var p = new(FieldInitializerListContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_fieldInitializerList
	return p
}

func (*FieldInitializerListContext) IsFieldInitializerListContext() {}

func NewFieldInitializerListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FieldInitializerListContext {
	var p = new(FieldInitializerListContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_fieldInitializerList

	return p
}

func (s *FieldInitializerListContext) GetParser() antlr.Parser { return s.parser }

func (s *FieldInitializerListContext) GetS22() antlr.Token { return s.s22 }

func (s *FieldInitializerListContext) SetS22(v antlr.Token) { s.s22 = v }

func (s *FieldInitializerListContext) GetCols() []antlr.Token { return s.cols }

func (s *FieldInitializerListContext) SetCols(v []antlr.Token) { s.cols = v }

func (s *FieldInitializerListContext) Get_optField() IOptFieldContext { return s._optField }

func (s *FieldInitializerListContext) Get_expr() IExprContext { return s._expr }

func (s *FieldInitializerListContext) Set_optField(v IOptFieldContext) { s._optField = v }

func (s *FieldInitializerListContext) Set_expr(v IExprContext) { s._expr = v }

func (s *FieldInitializerListContext) GetFields() []IOptFieldContext { return s.fields }

func (s *FieldInitializerListContext) GetValues() []IExprContext { return s.values }

func (s *FieldInitializerListContext) SetFields(v []IOptFieldContext) { s.fields = v }

func (s *FieldInitializerListContext) SetValues(v []IExprContext) { s.values = v }

func (s *FieldInitializerListContext) AllOptField() []IOptFieldContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOptFieldContext); ok {
			len++
		}
	}

	tst := make([]IOptFieldContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOptFieldContext); ok {
			tst[i] = t.(IOptFieldContext)
			i++
		}
	}

	return tst
}

func (s *FieldInitializerListContext) OptField(i int) IOptFieldContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOptFieldContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOptFieldContext)
}

func (s *FieldInitializerListContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOLON)
}

func (s *FieldInitializerListContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOLON, i)
}

func (s *FieldInitializerListContext) AllExpr() []IExprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprContext); ok {
			len++
		}
	}

	tst := make([]IExprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprContext); ok {
			tst[i] = t.(IExprContext)
			i++
		}
	}

	return tst
}

func (s *FieldInitializerListContext) Expr(i int) IExprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *FieldInitializerListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOMMA)
}

func (s *FieldInitializerListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, i)
}

func (s *FieldInitializerListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FieldInitializerListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FieldInitializerListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitFieldInitializerList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) FieldInitializerList() (localctx IFieldInitializerListContext) {
	this := p
	_ = this

	localctx = NewFieldInitializerListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, LogCarverParserRULE_fieldInitializerList)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(303)

		var _x = p.OptField()

		localctx.(*FieldInitializerListContext)._optField = _x
	}
	localctx.(*FieldInitializerListContext).fields = append(localctx.(*FieldInitializerListContext).fields, localctx.(*FieldInitializerListContext)._optField)
	{
		p.SetState(304)

		var _m = p.Match(LogCarverParserCOLON)

		localctx.(*FieldInitializerListContext).s22 = _m
	}
	localctx.(*FieldInitializerListContext).cols = append(localctx.(*FieldInitializerListContext).cols, localctx.(*FieldInitializerListContext).s22)
	{
		p.SetState(305)

		var _x = p.Expr()

		localctx.(*FieldInitializerListContext)._expr = _x
	}
	localctx.(*FieldInitializerListContext).values = append(localctx.(*FieldInitializerListContext).values, localctx.(*FieldInitializerListContext)._expr)
	p.SetState(313)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 37, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(306)
				p.Match(LogCarverParserCOMMA)
			}
			{
				p.SetState(307)

				var _x = p.OptField()

				localctx.(*FieldInitializerListContext)._optField = _x
			}
			localctx.(*FieldInitializerListContext).fields = append(localctx.(*FieldInitializerListContext).fields, localctx.(*FieldInitializerListContext)._optField)
			{
				p.SetState(308)

				var _m = p.Match(LogCarverParserCOLON)

				localctx.(*FieldInitializerListContext).s22 = _m
			}
			localctx.(*FieldInitializerListContext).cols = append(localctx.(*FieldInitializerListContext).cols, localctx.(*FieldInitializerListContext).s22)
			{
				p.SetState(309)

				var _x = p.Expr()

				localctx.(*FieldInitializerListContext)._expr = _x
			}
			localctx.(*FieldInitializerListContext).values = append(localctx.(*FieldInitializerListContext).values, localctx.(*FieldInitializerListContext)._expr)

		}
		p.SetState(315)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 37, p.GetParserRuleContext())
	}

	return localctx
}

// IOptFieldContext is an interface to support dynamic dispatch.
type IOptFieldContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetOpt returns the opt token.
	GetOpt() antlr.Token

	// SetOpt sets the opt token.
	SetOpt(antlr.Token)

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	QUESTIONMARK() antlr.TerminalNode

	// IsOptFieldContext differentiates from other interfaces.
	IsOptFieldContext()
}

type OptFieldContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	opt    antlr.Token
}

func NewEmptyOptFieldContext() *OptFieldContext {
	var p = new(OptFieldContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_optField
	return p
}

func (*OptFieldContext) IsOptFieldContext() {}

func NewOptFieldContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OptFieldContext {
	var p = new(OptFieldContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_optField

	return p
}

func (s *OptFieldContext) GetParser() antlr.Parser { return s.parser }

func (s *OptFieldContext) GetOpt() antlr.Token { return s.opt }

func (s *OptFieldContext) SetOpt(v antlr.Token) { s.opt = v }

func (s *OptFieldContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(LogCarverParserIDENTIFIER, 0)
}

func (s *OptFieldContext) QUESTIONMARK() antlr.TerminalNode {
	return s.GetToken(LogCarverParserQUESTIONMARK, 0)
}

func (s *OptFieldContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OptFieldContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OptFieldContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitOptField(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) OptField() (localctx IOptFieldContext) {
	this := p
	_ = this

	localctx = NewOptFieldContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, LogCarverParserRULE_optField)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	p.SetState(317)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == LogCarverParserQUESTIONMARK {
		{
			p.SetState(316)

			var _m = p.Match(LogCarverParserQUESTIONMARK)

			localctx.(*OptFieldContext).opt = _m
		}

	}
	{
		p.SetState(319)
		p.Match(LogCarverParserIDENTIFIER)
	}

	return localctx
}

// IMapInitializerListContext is an interface to support dynamic dispatch.
type IMapInitializerListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetS22 returns the s22 token.
	GetS22() antlr.Token

	// SetS22 sets the s22 token.
	SetS22(antlr.Token)

	// GetCols returns the cols token list.
	GetCols() []antlr.Token

	// SetCols sets the cols token list.
	SetCols([]antlr.Token)

	// Get_optExpr returns the _optExpr rule contexts.
	Get_optExpr() IOptExprContext

	// Get_expr returns the _expr rule contexts.
	Get_expr() IExprContext

	// Set_optExpr sets the _optExpr rule contexts.
	Set_optExpr(IOptExprContext)

	// Set_expr sets the _expr rule contexts.
	Set_expr(IExprContext)

	// GetKeys returns the keys rule context list.
	GetKeys() []IOptExprContext

	// GetValues returns the values rule context list.
	GetValues() []IExprContext

	// SetKeys sets the keys rule context list.
	SetKeys([]IOptExprContext)

	// SetValues sets the values rule context list.
	SetValues([]IExprContext)

	// Getter signatures
	AllOptExpr() []IOptExprContext
	OptExpr(i int) IOptExprContext
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	AllExpr() []IExprContext
	Expr(i int) IExprContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsMapInitializerListContext differentiates from other interfaces.
	IsMapInitializerListContext()
}

type MapInitializerListContext struct {
	*antlr.BaseParserRuleContext
	parser   antlr.Parser
	_optExpr IOptExprContext
	keys     []IOptExprContext
	s22      antlr.Token
	cols     []antlr.Token
	_expr    IExprContext
	values   []IExprContext
}

func NewEmptyMapInitializerListContext() *MapInitializerListContext {
	var p = new(MapInitializerListContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_mapInitializerList
	return p
}

func (*MapInitializerListContext) IsMapInitializerListContext() {}

func NewMapInitializerListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MapInitializerListContext {
	var p = new(MapInitializerListContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_mapInitializerList

	return p
}

func (s *MapInitializerListContext) GetParser() antlr.Parser { return s.parser }

func (s *MapInitializerListContext) GetS22() antlr.Token { return s.s22 }

func (s *MapInitializerListContext) SetS22(v antlr.Token) { s.s22 = v }

func (s *MapInitializerListContext) GetCols() []antlr.Token { return s.cols }

func (s *MapInitializerListContext) SetCols(v []antlr.Token) { s.cols = v }

func (s *MapInitializerListContext) Get_optExpr() IOptExprContext { return s._optExpr }

func (s *MapInitializerListContext) Get_expr() IExprContext { return s._expr }

func (s *MapInitializerListContext) Set_optExpr(v IOptExprContext) { s._optExpr = v }

func (s *MapInitializerListContext) Set_expr(v IExprContext) { s._expr = v }

func (s *MapInitializerListContext) GetKeys() []IOptExprContext { return s.keys }

func (s *MapInitializerListContext) GetValues() []IExprContext { return s.values }

func (s *MapInitializerListContext) SetKeys(v []IOptExprContext) { s.keys = v }

func (s *MapInitializerListContext) SetValues(v []IExprContext) { s.values = v }

func (s *MapInitializerListContext) AllOptExpr() []IOptExprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOptExprContext); ok {
			len++
		}
	}

	tst := make([]IOptExprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOptExprContext); ok {
			tst[i] = t.(IOptExprContext)
			i++
		}
	}

	return tst
}

func (s *MapInitializerListContext) OptExpr(i int) IOptExprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOptExprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOptExprContext)
}

func (s *MapInitializerListContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOLON)
}

func (s *MapInitializerListContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOLON, i)
}

func (s *MapInitializerListContext) AllExpr() []IExprContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExprContext); ok {
			len++
		}
	}

	tst := make([]IExprContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExprContext); ok {
			tst[i] = t.(IExprContext)
			i++
		}
	}

	return tst
}

func (s *MapInitializerListContext) Expr(i int) IExprContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *MapInitializerListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(LogCarverParserCOMMA)
}

func (s *MapInitializerListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(LogCarverParserCOMMA, i)
}

func (s *MapInitializerListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MapInitializerListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MapInitializerListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitMapInitializerList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) MapInitializerList() (localctx IMapInitializerListContext) {
	this := p
	_ = this

	localctx = NewMapInitializerListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 52, LogCarverParserRULE_mapInitializerList)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(321)

		var _x = p.OptExpr()

		localctx.(*MapInitializerListContext)._optExpr = _x
	}
	localctx.(*MapInitializerListContext).keys = append(localctx.(*MapInitializerListContext).keys, localctx.(*MapInitializerListContext)._optExpr)
	{
		p.SetState(322)

		var _m = p.Match(LogCarverParserCOLON)

		localctx.(*MapInitializerListContext).s22 = _m
	}
	localctx.(*MapInitializerListContext).cols = append(localctx.(*MapInitializerListContext).cols, localctx.(*MapInitializerListContext).s22)
	{
		p.SetState(323)

		var _x = p.Expr()

		localctx.(*MapInitializerListContext)._expr = _x
	}
	localctx.(*MapInitializerListContext).values = append(localctx.(*MapInitializerListContext).values, localctx.(*MapInitializerListContext)._expr)
	p.SetState(331)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 39, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(324)
				p.Match(LogCarverParserCOMMA)
			}
			{
				p.SetState(325)

				var _x = p.OptExpr()

				localctx.(*MapInitializerListContext)._optExpr = _x
			}
			localctx.(*MapInitializerListContext).keys = append(localctx.(*MapInitializerListContext).keys, localctx.(*MapInitializerListContext)._optExpr)
			{
				p.SetState(326)

				var _m = p.Match(LogCarverParserCOLON)

				localctx.(*MapInitializerListContext).s22 = _m
			}
			localctx.(*MapInitializerListContext).cols = append(localctx.(*MapInitializerListContext).cols, localctx.(*MapInitializerListContext).s22)
			{
				p.SetState(327)

				var _x = p.Expr()

				localctx.(*MapInitializerListContext)._expr = _x
			}
			localctx.(*MapInitializerListContext).values = append(localctx.(*MapInitializerListContext).values, localctx.(*MapInitializerListContext)._expr)

		}
		p.SetState(333)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 39, p.GetParserRuleContext())
	}

	return localctx
}

// IOptExprContext is an interface to support dynamic dispatch.
type IOptExprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetOpt returns the opt token.
	GetOpt() antlr.Token

	// SetOpt sets the opt token.
	SetOpt(antlr.Token)

	// GetE returns the e rule contexts.
	GetE() IExprContext

	// SetE sets the e rule contexts.
	SetE(IExprContext)

	// Getter signatures
	Expr() IExprContext
	QUESTIONMARK() antlr.TerminalNode

	// IsOptExprContext differentiates from other interfaces.
	IsOptExprContext()
}

type OptExprContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
	opt    antlr.Token
	e      IExprContext
}

func NewEmptyOptExprContext() *OptExprContext {
	var p = new(OptExprContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_optExpr
	return p
}

func (*OptExprContext) IsOptExprContext() {}

func NewOptExprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OptExprContext {
	var p = new(OptExprContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_optExpr

	return p
}

func (s *OptExprContext) GetParser() antlr.Parser { return s.parser }

func (s *OptExprContext) GetOpt() antlr.Token { return s.opt }

func (s *OptExprContext) SetOpt(v antlr.Token) { s.opt = v }

func (s *OptExprContext) GetE() IExprContext { return s.e }

func (s *OptExprContext) SetE(v IExprContext) { s.e = v }

func (s *OptExprContext) Expr() IExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExprContext)
}

func (s *OptExprContext) QUESTIONMARK() antlr.TerminalNode {
	return s.GetToken(LogCarverParserQUESTIONMARK, 0)
}

func (s *OptExprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OptExprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OptExprContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitOptExpr(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) OptExpr() (localctx IOptExprContext) {
	this := p
	_ = this

	localctx = NewOptExprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, LogCarverParserRULE_optExpr)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	p.SetState(335)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == LogCarverParserQUESTIONMARK {
		{
			p.SetState(334)

			var _m = p.Match(LogCarverParserQUESTIONMARK)

			localctx.(*OptExprContext).opt = _m
		}

	}
	{
		p.SetState(337)

		var _x = p.Expr()

		localctx.(*OptExprContext).e = _x
	}

	return localctx
}

// ILiteralContext is an interface to support dynamic dispatch.
type ILiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsLiteralContext differentiates from other interfaces.
	IsLiteralContext()
}

type LiteralContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLiteralContext() *LiteralContext {
	var p = new(LiteralContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = LogCarverParserRULE_literal
	return p
}

func (*LiteralContext) IsLiteralContext() {}

func NewLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LiteralContext {
	var p = new(LiteralContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = LogCarverParserRULE_literal

	return p
}

func (s *LiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *LiteralContext) CopyFrom(ctx *LiteralContext) {
	s.BaseParserRuleContext.CopyFrom(ctx.BaseParserRuleContext)
}

func (s *LiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type BytesContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewBytesContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *BytesContext {
	var p = new(BytesContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *BytesContext) GetTok() antlr.Token { return s.tok }

func (s *BytesContext) SetTok(v antlr.Token) { s.tok = v }

func (s *BytesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BytesContext) BYTES() antlr.TerminalNode {
	return s.GetToken(LogCarverParserBYTES, 0)
}

func (s *BytesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitBytes(s)

	default:
		return t.VisitChildren(s)
	}
}

type UintContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewUintContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *UintContext {
	var p = new(UintContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *UintContext) GetTok() antlr.Token { return s.tok }

func (s *UintContext) SetTok(v antlr.Token) { s.tok = v }

func (s *UintContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UintContext) NUM_UINT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserNUM_UINT, 0)
}

func (s *UintContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitUint(s)

	default:
		return t.VisitChildren(s)
	}
}

type NullContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewNullContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *NullContext {
	var p = new(NullContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *NullContext) GetTok() antlr.Token { return s.tok }

func (s *NullContext) SetTok(v antlr.Token) { s.tok = v }

func (s *NullContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NullContext) NUL() antlr.TerminalNode {
	return s.GetToken(LogCarverParserNUL, 0)
}

func (s *NullContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitNull(s)

	default:
		return t.VisitChildren(s)
	}
}

type BoolFalseContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewBoolFalseContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *BoolFalseContext {
	var p = new(BoolFalseContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *BoolFalseContext) GetTok() antlr.Token { return s.tok }

func (s *BoolFalseContext) SetTok(v antlr.Token) { s.tok = v }

func (s *BoolFalseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BoolFalseContext) CEL_FALSE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCEL_FALSE, 0)
}

func (s *BoolFalseContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitBoolFalse(s)

	default:
		return t.VisitChildren(s)
	}
}

type StringContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewStringContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *StringContext {
	var p = new(StringContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *StringContext) GetTok() antlr.Token { return s.tok }

func (s *StringContext) SetTok(v antlr.Token) { s.tok = v }

func (s *StringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StringContext) STRING() antlr.TerminalNode {
	return s.GetToken(LogCarverParserSTRING, 0)
}

func (s *StringContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitString(s)

	default:
		return t.VisitChildren(s)
	}
}

type DoubleContext struct {
	*LiteralContext
	sign antlr.Token
	tok  antlr.Token
}

func NewDoubleContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *DoubleContext {
	var p = new(DoubleContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *DoubleContext) GetSign() antlr.Token { return s.sign }

func (s *DoubleContext) GetTok() antlr.Token { return s.tok }

func (s *DoubleContext) SetSign(v antlr.Token) { s.sign = v }

func (s *DoubleContext) SetTok(v antlr.Token) { s.tok = v }

func (s *DoubleContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DoubleContext) NUM_FLOAT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserNUM_FLOAT, 0)
}

func (s *DoubleContext) MINUS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserMINUS, 0)
}

func (s *DoubleContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitDouble(s)

	default:
		return t.VisitChildren(s)
	}
}

type BoolTrueContext struct {
	*LiteralContext
	tok antlr.Token
}

func NewBoolTrueContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *BoolTrueContext {
	var p = new(BoolTrueContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *BoolTrueContext) GetTok() antlr.Token { return s.tok }

func (s *BoolTrueContext) SetTok(v antlr.Token) { s.tok = v }

func (s *BoolTrueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BoolTrueContext) CEL_TRUE() antlr.TerminalNode {
	return s.GetToken(LogCarverParserCEL_TRUE, 0)
}

func (s *BoolTrueContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitBoolTrue(s)

	default:
		return t.VisitChildren(s)
	}
}

type IntContext struct {
	*LiteralContext
	sign antlr.Token
	tok  antlr.Token
}

func NewIntContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *IntContext {
	var p = new(IntContext)

	p.LiteralContext = NewEmptyLiteralContext()
	p.parser = parser
	p.CopyFrom(ctx.(*LiteralContext))

	return p
}

func (s *IntContext) GetSign() antlr.Token { return s.sign }

func (s *IntContext) GetTok() antlr.Token { return s.tok }

func (s *IntContext) SetSign(v antlr.Token) { s.sign = v }

func (s *IntContext) SetTok(v antlr.Token) { s.tok = v }

func (s *IntContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IntContext) NUM_INT() antlr.TerminalNode {
	return s.GetToken(LogCarverParserNUM_INT, 0)
}

func (s *IntContext) MINUS() antlr.TerminalNode {
	return s.GetToken(LogCarverParserMINUS, 0)
}

func (s *IntContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case LogCarverVisitor:
		return t.VisitInt(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *LogCarverParser) Literal() (localctx ILiteralContext) {
	this := p
	_ = this

	localctx = NewLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, LogCarverParserRULE_literal)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(353)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 43, p.GetParserRuleContext()) {
	case 1:
		localctx = NewIntContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		p.SetState(340)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserMINUS {
			{
				p.SetState(339)

				var _m = p.Match(LogCarverParserMINUS)

				localctx.(*IntContext).sign = _m
			}

		}
		{
			p.SetState(342)

			var _m = p.Match(LogCarverParserNUM_INT)

			localctx.(*IntContext).tok = _m
		}

	case 2:
		localctx = NewUintContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(343)

			var _m = p.Match(LogCarverParserNUM_UINT)

			localctx.(*UintContext).tok = _m
		}

	case 3:
		localctx = NewDoubleContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		p.SetState(345)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if _la == LogCarverParserMINUS {
			{
				p.SetState(344)

				var _m = p.Match(LogCarverParserMINUS)

				localctx.(*DoubleContext).sign = _m
			}

		}
		{
			p.SetState(347)

			var _m = p.Match(LogCarverParserNUM_FLOAT)

			localctx.(*DoubleContext).tok = _m
		}

	case 4:
		localctx = NewStringContext(p, localctx)
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(348)

			var _m = p.Match(LogCarverParserSTRING)

			localctx.(*StringContext).tok = _m
		}

	case 5:
		localctx = NewBytesContext(p, localctx)
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(349)

			var _m = p.Match(LogCarverParserBYTES)

			localctx.(*BytesContext).tok = _m
		}

	case 6:
		localctx = NewBoolTrueContext(p, localctx)
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(350)

			var _m = p.Match(LogCarverParserCEL_TRUE)

			localctx.(*BoolTrueContext).tok = _m
		}

	case 7:
		localctx = NewBoolFalseContext(p, localctx)
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(351)

			var _m = p.Match(LogCarverParserCEL_FALSE)

			localctx.(*BoolFalseContext).tok = _m
		}

	case 8:
		localctx = NewNullContext(p, localctx)
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(352)

			var _m = p.Match(LogCarverParserNUL)

			localctx.(*NullContext).tok = _m
		}

	}

	return localctx
}

func (p *LogCarverParser) Sempred(localctx antlr.RuleContext, ruleIndex, predIndex int) bool {
	switch ruleIndex {
	case 17:
		var t *RelationContext = nil
		if localctx != nil {
			t = localctx.(*RelationContext)
		}
		return p.Relation_Sempred(t, predIndex)

	case 18:
		var t *CalcContext = nil
		if localctx != nil {
			t = localctx.(*CalcContext)
		}
		return p.Calc_Sempred(t, predIndex)

	case 20:
		var t *MemberContext = nil
		if localctx != nil {
			t = localctx.(*MemberContext)
		}
		return p.Member_Sempred(t, predIndex)

	default:
		panic("No predicate with index: " + fmt.Sprint(ruleIndex))
	}
}

func (p *LogCarverParser) Relation_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	this := p
	_ = this

	switch predIndex {
	case 0:
		return p.Precpred(p.GetParserRuleContext(), 1)

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}

func (p *LogCarverParser) Calc_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	this := p
	_ = this

	switch predIndex {
	case 1:
		return p.Precpred(p.GetParserRuleContext(), 2)

	case 2:
		return p.Precpred(p.GetParserRuleContext(), 1)

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}

func (p *LogCarverParser) Member_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	this := p
	_ = this

	switch predIndex {
	case 3:
		return p.Precpred(p.GetParserRuleContext(), 3)

	case 4:
		return p.Precpred(p.GetParserRuleContext(), 2)

	case 5:
		return p.Precpred(p.GetParserRuleContext(), 1)

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}
