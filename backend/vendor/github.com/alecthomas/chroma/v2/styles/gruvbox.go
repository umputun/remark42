package styles

import (
	"github.com/alecthomas/chroma/v2"
)

// Gruvbox style.
var Gruvbox = Register(chroma.MustNewStyle("gruvbox", chroma.StyleEntries{
	chroma.CommentPreproc:      "noinherit #8ec07c",
	chroma.Comment:             "#928374 italic",
	chroma.GenericDeleted:      "noinherit #282828 bg:#fb4934",
	chroma.GenericEmph:         "#83a598 underline",
	chroma.GenericError:        "bg:#fb4934 bold",
	chroma.GenericHeading:      "#b8bb26 bold",
	chroma.GenericInserted:     "noinherit #282828 bg:#b8bb26",
	chroma.GenericOutput:       "noinherit #504945",
	chroma.GenericPrompt:       "#ebdbb2",
	chroma.GenericStrong:       "#ebdbb2",
	chroma.GenericSubheading:   "#b8bb26 bold",
	chroma.GenericTraceback:    "bg:#fb4934 bold",
	chroma.Generic:             "#ebdbb2",
	chroma.KeywordType:         "noinherit #fabd2f",
	chroma.Keyword:             "noinherit #fe8019",
	chroma.NameAttribute:       "#b8bb26 bold",
	chroma.NameBuiltin:         "#fabd2f",
	chroma.NameConstant:        "noinherit #d3869b",
	chroma.NameEntity:          "noinherit #fabd2f",
	chroma.NameException:       "noinherit #fb4934",
	chroma.NameFunction:        "#fabd2f",
	chroma.NameLabel:           "noinherit #fb4934",
	chroma.NameTag:             "noinherit #fb4934",
	chroma.NameVariable:        "noinherit #ebdbb2",
	chroma.Name:                "#ebdbb2",
	chroma.LiteralNumberFloat:  "noinherit #d3869b",
	chroma.LiteralNumber:       "noinherit #d3869b",
	chroma.Operator:            "#fe8019",
	chroma.LiteralStringSymbol: "#83a598",
	chroma.LiteralString:       "noinherit #b8bb26",
	chroma.Background:          "noinherit #ebdbb2 bg:#282828 bg:#282828",
}))
