package lexers

import (
	. "github.com/alecthomas/chroma/v2" // nolint
)

// FortranFixed lexer.
var FortranFixed = Register(MustNewLexer(
	&Config{
		Name:            "FortranFixed",
		Aliases:         []string{"fortranfixed"},
		Filenames:       []string{"*.f", "*.F"},
		MimeTypes:       []string{"text/x-fortran"},
		NotMultiline:    true,
		CaseInsensitive: true,
	},
	func() Rules {
		return Rules{
			"root": {
				{`[C*].*\n`, Comment, nil},
				{`#.*\n`, CommentPreproc, nil},
				{` {0,4}!.*\n`, Comment, nil},
				{`(.{5})`, NameLabel, Push("cont-char")},
				{`.*\n`, Using("Fortran"), nil},
			},
			"cont-char": {
				{` `, TextWhitespace, Push("code")},
				{`.`, GenericStrong, Push("code")},
			},
			"code": {
				{`(.{66})(.*)(\n)`, ByGroups(Using("Fortran"), Comment, TextWhitespace), Push("root")},
				{`(.*)(!.*)(\n)`, ByGroups(Using("Fortran"), Comment, TextWhitespace), Push("root")},
				{`(.*)(\n)`, ByGroups(Using("Fortran"), TextWhitespace), Push("root")},
				Default(Push("root")),
			},
		}
	},
))
