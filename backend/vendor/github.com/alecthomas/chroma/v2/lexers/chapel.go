package lexers

import (
	. "github.com/alecthomas/chroma/v2" // nolint
)

// Chapel lexer.
var Chapel = Register(MustNewLexer(
	&Config{
		Name:      "Chapel",
		Aliases:   []string{"chapel", "chpl"},
		Filenames: []string{"*.chpl"},
		MimeTypes: []string{},
	},
	func() Rules {
		return Rules{
			"root": {
				{`\n`, TextWhitespace, nil},
				{`\s+`, TextWhitespace, nil},
				{`\\\n`, Text, nil},
				{`//(.*?)\n`, CommentSingle, nil},
				{`/(\\\n)?[*](.|\n)*?[*](\\\n)?/`, CommentMultiline, nil},
				{Words(``, `\b`, `config`, `const`, `in`, `inout`, `out`, `param`, `ref`, `type`, `var`), KeywordDeclaration, nil},
				{Words(``, `\b`, `false`, `nil`, `none`, `true`), KeywordConstant, nil},
				{Words(``, `\b`, `bool`, `bytes`, `complex`, `imag`, `int`, `locale`, `nothing`, `opaque`, `range`, `real`, `string`, `uint`, `void`), KeywordType, nil},
				{Words(``, `\b`, `atomic`, `single`, `sync`, `borrowed`, `owned`, `shared`, `unmanaged`, `align`, `as`, `begin`, `break`, `by`, `catch`, `cobegin`, `coforall`, `continue`, `defer`, `delete`, `dmapped`, `do`, `domain`, `else`, `enum`, `except`, `export`, `extern`, `for`, `forall`, `foreach`, `forwarding`, `if`, `implements`, `import`, `index`, `init`, `inline`, `label`, `lambda`, `let`, `lifetime`, `local`, `new`, `noinit`, `on`, `only`, `otherwise`, `override`, `pragma`, `primitive`, `private`, `prototype`, `public`, `reduce`, `require`, `return`, `scan`, `select`, `serial`, `sparse`, `subdomain`, `then`, `this`, `throw`, `throws`, `try`, `use`, `when`, `where`, `while`, `with`, `yield`, `zip`), Keyword, nil},
				{`(iter)(\s+)`, ByGroups(Keyword, TextWhitespace), Push("procname")},
				{`(proc)(\s+)`, ByGroups(Keyword, TextWhitespace), Push("procname")},
				{`(operator)(\s+)`, ByGroups(Keyword, TextWhitespace), Push("procname")},
				{`(class|interface|module|record|union)(\s+)`, ByGroups(Keyword, TextWhitespace), Push("classname")},
				{`\d+i`, LiteralNumber, nil},
				{`\d+\.\d*([Ee][-+]\d+)?i`, LiteralNumber, nil},
				{`\.\d+([Ee][-+]\d+)?i`, LiteralNumber, nil},
				{`\d+[Ee][-+]\d+i`, LiteralNumber, nil},
				{`(\d*\.\d+)([eE][+-]?[0-9]+)?i?`, LiteralNumberFloat, nil},
				{`\d+[eE][+-]?[0-9]+i?`, LiteralNumberFloat, nil},
				{`0[bB][01]+`, LiteralNumberBin, nil},
				{`0[xX][0-9a-fA-F]+`, LiteralNumberHex, nil},
				{`0[oO][0-7]+`, LiteralNumberOct, nil},
				{`[0-9]+`, LiteralNumberInteger, nil},
				{`"(\\\\|\\"|[^"])*"`, LiteralString, nil},
				{`'(\\\\|\\'|[^'])*'`, LiteralString, nil},
				{`(=|\+=|-=|\*=|/=|\*\*=|%=|&=|\|=|\^=|&&=|\|\|=|<<=|>>=|<=>|<~>|\.\.|by|#|\.\.\.|&&|\|\||!|&|\||\^|~|<<|>>|==|!=|<=|>=|<|>|[+\-*/%]|\*\*)`, Operator, nil},
				{`[:;,.?()\[\]{}]`, Punctuation, nil},
				{`[a-zA-Z_][\w$]*`, NameOther, nil},
			},
			"classname": {
				{`[a-zA-Z_][\w$]*`, NameClass, Pop(1)},
			},
			"procname": {
				{`([a-zA-Z_][.\w$]*|\~[a-zA-Z_][.\w$]*|[+*/!~%<>=&^|\-:]{1,2})`, NameFunction, Pop(1)},
				{`\(`, Punctuation, Push("receivertype")},
				{`\)+\.`, Punctuation, nil},
			},
			"receivertype": {
				{Words(``, `\b`, `atomic`, `single`, `sync`, `borrowed`, `owned`, `shared`, `unmanaged`), Keyword, nil},
				{Words(``, `\b`, `bool`, `bytes`, `complex`, `imag`, `int`, `locale`, `nothing`, `opaque`, `range`, `real`, `string`, `uint`, `void`), KeywordType, nil},
				{`[^()]*`, NameOther, Pop(1)},
			},
		}
	},
))
