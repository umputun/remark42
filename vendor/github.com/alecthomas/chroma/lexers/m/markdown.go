package m

import (
	. "github.com/alecthomas/chroma" // nolint
	"github.com/alecthomas/chroma/lexers/internal"
)

// Markdown lexer.
var Markdown = internal.Register(MustNewLexer(
	&Config{
		Name:      "markdown",
		Aliases:   []string{"md", "mkd"},
		Filenames: []string{"*.md", "*.mkd", "*.markdown"},
		MimeTypes: []string{"text/x-markdown"},
	},
	Rules{
		"root": {
			{`^(#)([^#].+\n)`, ByGroups(GenericHeading, Text), nil},
			{`^(#{2,6})(.+\n)`, ByGroups(GenericSubheading, Text), nil},
			{`^(\s*)([*-] )(\[[ xX]\])( .+\n)`, ByGroups(Text, Keyword, Keyword, UsingSelf("inline")), nil},
			{`^(\s*)([*-])(\s)(.+\n)`, ByGroups(Text, Keyword, Text, UsingSelf("inline")), nil},
			{`^(\s*)([0-9]+\.)( .+\n)`, ByGroups(Text, Keyword, UsingSelf("inline")), nil},
			{`^(\s*>\s)(.+\n)`, ByGroups(Keyword, GenericEmph), nil},
			{"^(```\\n)([\\w\\W]*?)(^```$)", ByGroups(LiteralString, Text, LiteralString), nil},
			{"^(```)(\\w+)(\\n)([\\w\\W]*?)(^```$)", EmitterFunc(markdownCodeBlock), nil},
			Include("inline"),
		},
		"inline": {
			{`\\.`, Text, nil},
			{`(\s)([*_][^*_]+[*_])(\W|\n)`, ByGroups(Text, GenericEmph, Text), nil},
			{`(\s)((\*\*|__).*\3)((?=\W|\n))`, ByGroups(Text, GenericStrong, None, Text), nil},
			{`(\s)(~~[^~]+~~)((?=\W|\n))`, ByGroups(Text, GenericDeleted, Text), nil},
			{"`[^`]+`", LiteralStringBacktick, nil},
			{`[@#][\w/:]+`, NameEntity, nil},
			{`(!?\[)([^]]+)(\])(\()([^)]+)(\))`, ByGroups(Text, NameTag, Text, Text, NameAttribute, Text), nil},
			{`[^\\\s]+`, Text, nil},
			{`.`, Text, nil},
		},
	},
))

func markdownCodeBlock(groups []string, lexer Lexer) Iterator {
	iterators := []Iterator{}
	tokens := []*Token{
		{String, groups[1]},
		{String, groups[2]},
		{Text, groups[3]},
	}
	code := groups[4]
	lexer = internal.Get(groups[2])
	if lexer == nil {
		tokens = append(tokens, &Token{String, code})
		iterators = append(iterators, Literator(tokens...))
	} else {
		sub, err := lexer.Tokenise(nil, code)
		if err != nil {
			panic(err)
		}
		iterators = append(iterators, Literator(tokens...), sub)
	}
	iterators = append(iterators, Literator(&Token{String, groups[5]}))
	return Concaterator(iterators...)
}
