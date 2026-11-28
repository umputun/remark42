package stringprep

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Profile represents a stringprep profile.
type Profile struct {
	Mappings  []Mapping
	Normalize bool
	Prohibits []Set
	CheckBiDi bool
}

var errProhibited = "prohibited character"
var errTooManyCombining = "more than 30 consecutive combining characters (normalization inserted CGJ)"

// Prepare transforms an input string to an output string following
// the rules defined in the profile as defined by RFC-3454.
//
// If the profile normalizes, Prepare returns an Error when the decomposed
// input has more than 30 consecutive combining characters, the limit of
// the UAX #15 stream-safe text format.
func (p Profile) Prepare(s string) (string, error) {
	// Optimistically, assume output will be same length as input
	temp := make([]rune, 0, len(s))

	// Apply maps
	for _, r := range s {
		rs, ok := p.applyMaps(r)
		if ok {
			temp = append(temp, rs...)
		} else {
			temp = append(temp, r)
		}
	}

	// Normalize
	mapped := string(temp)
	out := mapped
	if p.Normalize {
		out = norm.NFKC.String(mapped)
		if insertedGraphemeJoiner(mapped, out) {
			return "", Error{Msg: errTooManyCombining, Rune: 0x034F}
		}
	}

	// Check prohibited
	for _, r := range out {
		if p.runeIsProhibited(r) {
			return "", Error{Msg: errProhibited, Rune: r}
		}
	}

	// Check BiDi allowed
	if p.CheckBiDi {
		if err := passesBiDiRules(out); err != nil {
			return "", err
		}
	}

	return out, nil
}

func (p Profile) applyMaps(r rune) ([]rune, bool) {
	for _, m := range p.Mappings {
		rs, ok := m.Map(r)
		if ok {
			return rs, true
		}
	}
	return nil, false
}

// insertedGraphemeJoiner reports whether NFKC added norm.GraphemeJoiner,
// which x/text inserts after 30 consecutive non-starters (UAX #15
// stream-safe format). Such output is not strict NFKC, and with table B.1
// a second Prepare deletes the joiner and reorders the marks differently.
// Stripping the joiner instead would leave non-NFKC output.
func insertedGraphemeJoiner(in, out string) bool {
	return strings.Count(out, norm.GraphemeJoiner) > strings.Count(in, norm.GraphemeJoiner)
}

func (p Profile) runeIsProhibited(r rune) bool {
	for _, s := range p.Prohibits {
		if s.Contains(r) {
			return true
		}
	}
	return false
}
