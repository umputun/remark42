package lexers

import (
	. "github.com/alecthomas/chroma/v2" // nolint
)

// Docker lexer.
var Docker = Register(MustNewLexer(
	&Config{
		Name:            "Docker",
		Aliases:         []string{"docker", "dockerfile"},
		Filenames:       []string{"Dockerfile", "*.docker"},
		MimeTypes:       []string{"text/x-dockerfile-config"},
		CaseInsensitive: true,
	},
	dockerRules,
))

func dockerRules() Rules {
	return Rules{
		"root": {
			{`#.*`, Comment, nil},
			{`(ONBUILD)((?:\s*\\?\s*))`, ByGroups(Keyword, Using("Bash")), nil},
			{`(HEALTHCHECK)(((?:\s*\\?\s*)--\w+=\w+(?:\s*\\?\s*))*)`, ByGroups(Keyword, Using("Bash")), nil},
			{`(VOLUME|ENTRYPOINT|CMD|SHELL)((?:\s*\\?\s*))(\[.*?\])`, ByGroups(Keyword, Using("Bash"), Using("JSON")), nil},
			{`(LABEL|ENV|ARG)((?:(?:\s*\\?\s*)\w+=\w+(?:\s*\\?\s*))*)`, ByGroups(Keyword, Using("Bash")), nil},
			{`((?:FROM|MAINTAINER|EXPOSE|WORKDIR|USER|STOPSIGNAL)|VOLUME)\b(.*)`, ByGroups(Keyword, LiteralString), nil},
			{`((?:RUN|CMD|ENTRYPOINT|ENV|ARG|LABEL|ADD|COPY))`, Keyword, nil},
			{`(.*\\\n)*.+`, Using("Bash"), nil},
		},
	}
}
