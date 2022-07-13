package lexers

import (
	. "github.com/alecthomas/chroma/v2" // nolint
)

// Smarty lexer.
var Smarty = Register(MustNewLexer(
	&Config{
		Name:      "Smarty",
		Aliases:   []string{"smarty"},
		Filenames: []string{"*.tpl"},
		MimeTypes: []string{"application/x-smarty"},
		DotAll:    true,
	},
	smartyRules,
))

func smartyRules() Rules {
	return Rules{
		"root": {
			{`[^{]+`, Other, nil},
			{`(\{)(\*.*?\*)(\})`, ByGroups(CommentPreproc, Comment, CommentPreproc), nil},
			{`(\{php\})(.*?)(\{/php\})`, ByGroups(CommentPreproc, Using("PHP"), CommentPreproc), nil},
			{`(\{)(/?[a-zA-Z_]\w*)(\s*)`, ByGroups(CommentPreproc, NameFunction, Text), Push("smarty")},
			{`\{`, CommentPreproc, Push("smarty")},
		},
		"smarty": {
			{`\s+`, Text, nil},
			{`\{`, CommentPreproc, Push()},
			{`\}`, CommentPreproc, Pop(1)},
			{`#[a-zA-Z_]\w*#`, NameVariable, nil},
			{`\$[a-zA-Z_]\w*(\.\w+)*`, NameVariable, nil},
			{`[~!%^&*()+=|\[\]:;,.<>/?@-]`, Operator, nil},
			{`(true|false|null)\b`, KeywordConstant, nil},
			{`[0-9](\.[0-9]*)?(eE[+-][0-9])?[flFLdD]?|0[xX][0-9a-fA-F]+[Ll]?`, LiteralNumber, nil},
			{`"(\\\\|\\"|[^"])*"`, LiteralStringDouble, nil},
			{`'(\\\\|\\'|[^'])*'`, LiteralStringSingle, nil},
			{`[a-zA-Z_]\w*`, NameAttribute, nil},
		},
	}
}
