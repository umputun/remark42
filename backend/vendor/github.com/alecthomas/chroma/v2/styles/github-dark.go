package styles

import (
	"fmt"

	"github.com/alecthomas/chroma/v2"
)

var (
	// colors used from https://github.com/primer/primitives
	ghRed2      = "#ffa198"
	ghRed3      = "#ff7b72"
	ghRed9      = "#490202"
	ghOrange2   = "#ffa657"
	ghOrange3   = "#f0883e"
	ghGreen1    = "#7ee787"
	ghGreen2    = "#56d364"
	ghGreen7    = "#0f5323"
	ghBlue1     = "#a5d6ff"
	ghBlue2     = "#79c0ff"
	ghPurple2   = "#d2a8ff"
	ghGray3     = "#8b949e"
	ghGray4     = "#6e7681"
	ghFgSubtle  = "#6e7681"
	ghFgDefault = "#c9d1d9"
	ghBgDefault = "#0d1117"
	ghDangerFg  = "#f85149"
)

// GitHub Dark style.
var GitHubDark = Register(chroma.MustNewStyle("github-dark", chroma.StyleEntries{
	// Default Token Style
	chroma.Background: fmt.Sprintf("bg:%s %s", ghBgDefault, ghFgDefault),

	chroma.LineNumbers: ghGray4,
	// has transparency in VS Code theme as `colors.codemirror.activelineBg`
	chroma.LineHighlight: ghGray4,

	chroma.Error: ghDangerFg,

	chroma.Keyword:         ghRed3,
	chroma.KeywordConstant: ghBlue2,
	chroma.KeywordPseudo:   ghBlue2,

	chroma.Name:          ghFgDefault,
	chroma.NameClass:     "bold " + ghOrange3,
	chroma.NameConstant:  "bold " + ghBlue2,
	chroma.NameDecorator: "bold " + ghPurple2,
	chroma.NameEntity:    ghOrange2,
	chroma.NameException: "bold " + ghOrange3,
	chroma.NameFunction:  "bold " + ghPurple2,
	chroma.NameLabel:     "bold " + ghBlue2,
	chroma.NameNamespace: ghRed3,
	chroma.NameProperty:  ghBlue2,
	chroma.NameTag:       ghGreen1,
	chroma.NameVariable:  ghBlue2,

	chroma.Literal:                ghBlue1,
	chroma.LiteralDate:            ghBlue2,
	chroma.LiteralStringAffix:     ghBlue2,
	chroma.LiteralStringDelimiter: ghBlue2,
	chroma.LiteralStringEscape:    ghBlue2,
	chroma.LiteralStringHeredoc:   ghBlue2,
	chroma.LiteralStringRegex:     ghBlue2,

	chroma.Operator: "bold " + ghRed3,

	chroma.Comment:        "italic " + ghGray3,
	chroma.CommentPreproc: "bold " + ghGray3,
	chroma.CommentSpecial: "bold italic " + ghGray3,

	chroma.Generic:           ghFgDefault,
	chroma.GenericDeleted:    fmt.Sprintf("bg:%s %s", ghRed9, ghRed2),
	chroma.GenericEmph:       "italic",
	chroma.GenericError:      ghRed2,
	chroma.GenericHeading:    "bold " + ghBlue2,
	chroma.GenericInserted:   fmt.Sprintf("bg:%s %s", ghGreen7, ghGreen2),
	chroma.GenericOutput:     ghGray3,
	chroma.GenericPrompt:     ghGray3,
	chroma.GenericStrong:     "bold",
	chroma.GenericSubheading: ghBlue2,
	chroma.GenericTraceback:  ghRed3,
	chroma.GenericUnderline:  "underline",

	chroma.TextWhitespace: ghFgSubtle,
}))
