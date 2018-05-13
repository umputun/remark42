package g

import (
	"strings"

	. "github.com/alecthomas/chroma" // nolint
	"github.com/alecthomas/chroma/lexers/internal"
)

// Go lexer.
var Go = internal.Register(MustNewLexer(
	&Config{
		Name:      "Go",
		Aliases:   []string{"go", "golang"},
		Filenames: []string{"*.go"},
		MimeTypes: []string{"text/x-gosrc"},
	},
	Rules{
		"root": {
			{`\n`, Text, nil},
			{`\s+`, Text, nil},
			{`\\\n`, Text, nil},
			{`//(.*?)\n`, CommentSingle, nil},
			{`/(\\\n)?[*](.|\n)*?[*](\\\n)?/`, CommentMultiline, nil},
			{`(import|package)\b`, KeywordNamespace, nil},
			{`(var|func|struct|map|chan|type|interface|const)\b`, KeywordDeclaration, nil},
			{Words(``, `\b`, `break`, `default`, `select`, `case`, `defer`, `go`, `else`, `goto`, `switch`, `fallthrough`, `if`, `range`, `continue`, `for`, `return`), Keyword, nil},
			{`(true|false|iota|nil)\b`, KeywordConstant, nil},
			{Words(``, `\b(\()`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`, `print`, `println`, `panic`, `recover`, `close`, `complex`, `real`, `imag`, `len`, `cap`, `append`, `copy`, `delete`, `new`, `make`), ByGroups(NameBuiltin, Punctuation), nil},
			{Words(``, `\b`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`), KeywordType, nil},
			{`\d+i`, LiteralNumber, nil},
			{`\d+\.\d*([Ee][-+]\d+)?i`, LiteralNumber, nil},
			{`\.\d+([Ee][-+]\d+)?i`, LiteralNumber, nil},
			{`\d+[Ee][-+]\d+i`, LiteralNumber, nil},
			{`\d+(\.\d+[eE][+\-]?\d+|\.\d*|[eE][+\-]?\d+)`, LiteralNumberFloat, nil},
			{`\.\d+([eE][+\-]?\d+)?`, LiteralNumberFloat, nil},
			{`0[0-7]+`, LiteralNumberOct, nil},
			{`0[xX][0-9a-fA-F]+`, LiteralNumberHex, nil},
			{`(0|[1-9][0-9]*)`, LiteralNumberInteger, nil},
			{`'(\\['"\\abfnrtv]|\\x[0-9a-fA-F]{2}|\\[0-7]{1,3}|\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}|[^\\])'`, LiteralStringChar, nil},
			{"`[^`]*`", LiteralString, nil},
			{`"(\\\\|\\"|[^"])*"`, LiteralString, nil},
			{`(<<=|>>=|<<|>>|<=|>=|&\^=|&\^|\+=|-=|\*=|/=|%=|&=|\|=|&&|\|\||<-|\+\+|--|==|!=|:=|\.\.\.|[+\-*/%&])`, Operator, nil},
			{`[|^<>=!()\[\]{}.,;:]`, Punctuation, nil},
			{`[^\W\d]\w*`, NameOther, nil},
		},
	},
).SetAnalyser(func(text string) float32 {
	if strings.Contains(text, "fmt.") && strings.Contains(text, "package ") {
		return 0.5
	}
	if strings.Contains(text, "package ") {
		return 0.1
	}
	return 0.0
}))
