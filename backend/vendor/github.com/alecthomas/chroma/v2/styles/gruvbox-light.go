package styles

import (
	"github.com/alecthomas/chroma/v2"
)

// Gruvbox light style.
var GruvboxLight = Register(chroma.MustNewStyle("gruvbox-light", chroma.StyleEntries{
	chroma.CommentPreproc:      "noinherit #427B58",
	chroma.Comment:             "#928374 italic",
	chroma.GenericDeleted:      "noinherit #282828 bg:#9D0006",
	chroma.GenericEmph:         "#076678 underline",
	chroma.GenericError:        "bg:#9D0006 bold",
	chroma.GenericHeading:      "#79740E bold",
	chroma.GenericInserted:     "noinherit #282828 bg:#79740E",
	chroma.GenericOutput:       "noinherit #504945",
	chroma.GenericPrompt:       "#3C3836",
	chroma.GenericStrong:       "#3C3836",
	chroma.GenericSubheading:   "#79740E bold",
	chroma.GenericTraceback:    "bg:#3C3836 bold",
	chroma.Generic:             "#3C3836",
	chroma.KeywordType:         "noinherit #B57614",
	chroma.Keyword:             "noinherit #AF3A03",
	chroma.NameAttribute:       "#79740E bold",
	chroma.NameBuiltin:         "#B57614",
	chroma.NameConstant:        "noinherit #d3869b",
	chroma.NameEntity:          "noinherit #B57614",
	chroma.NameException:       "noinherit #fb4934",
	chroma.NameFunction:        "#B57614",
	chroma.NameLabel:           "noinherit #9D0006",
	chroma.NameTag:             "noinherit #9D0006",
	chroma.NameVariable:        "noinherit #3C3836",
	chroma.Name:                "#3C3836",
	chroma.LiteralNumberFloat:  "noinherit #8F3F71",
	chroma.LiteralNumber:       "noinherit #8F3F71",
	chroma.Operator:            "#AF3A03",
	chroma.LiteralStringSymbol: "#076678",
	chroma.LiteralString:       "noinherit #79740E",
	chroma.Background:          "noinherit #3C3836 bg:#FBF1C7 bg:#FBF1C7",
}))
