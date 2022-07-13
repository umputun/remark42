package lexers

import (
	"regexp"
)

// TODO(moorereason): can this be factored away?
var bashAnalyserRe = regexp.MustCompile(`(?m)^#!.*/bin/(?:env |)(?:bash|zsh|sh|ksh)`)

func init() { // nolint: gochecknoinits
	Get("bash").SetAnalyser(func(text string) float32 {
		if bashAnalyserRe.FindString(text) != "" {
			return 1.0
		}
		return 0.0
	})
}
