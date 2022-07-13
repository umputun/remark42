package lexers

import (
	. "github.com/alecthomas/chroma/v2" // nolint
)

// BashSession lexer.
var BashSession = Register(MustNewLexer(
	&Config{
		Name:      "BashSession",
		Aliases:   []string{"bash-session", "console", "shell-session"},
		Filenames: []string{".sh-session"},
		MimeTypes: []string{"text/x-sh"},
		EnsureNL:  true,
	},
	bashsessionRules,
))

func bashsessionRules() Rules {
	return Rules{
		"root": {
			{`^((?:\[[^]]+@[^]]+\]\s?)?[#$%>])(\s*)(.*\n?)`, ByGroups(GenericPrompt, Text, Using("Bash")), nil},
			{`^.+\n?`, GenericOutput, nil},
		},
	}
}
