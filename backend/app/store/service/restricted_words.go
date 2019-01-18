package service

import (
	"fmt"
	"strings"
	"unicode"
)

type Lister interface {
	List(siteID string) (restricted []string, err error)
}

type StaticLister struct {
	Words map[string][]string
}

func (l *StaticLister) List(siteID string) (restricted []string, err error) {
	restricted, exists := l.Words[siteID]
	if !exists {
		return restricted, fmt.Errorf("no restricted words configured for site '%s'", siteID)
	}
	return
}

type Matcher struct {
	lister Lister
	data   map[string]restrictedWordsSet
}

type restrictedWordsSet struct {
	restricted map[string]bool
}

func NewMatcher(lister Lister) *Matcher {
	return &Matcher{lister, make(map[string]restrictedWordsSet)}
}

func (m *Matcher) Match(siteID string, text string) bool {
	tokens := m.tokenize(text)

	data, exists := m.data[siteID]
	if !exists {
		words, err := m.lister.List(siteID)
		if err != nil {
			fmt.Printf("failed to get restricted words for site %s: %v", siteID, err)
			return false
		}
		restricted := make(map[string]bool)
		for _, w := range words {
			restricted[strings.ToLower(w)] = true
		}
		data = restrictedWordsSet{restricted}
		m.data[siteID] = data
	}

	for _, token := range tokens {
		_, present := data.restricted[token]
		if present {
			return true
		}
	}
	return false
}

func (_ *Matcher) tokenize(text string) []string {
	tokens := make([]string, 0, 10) // accumulator for tokens
	word := false                   // flag shows if current range is word
	start := 0                      // beginning of the current range

	for pos, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !word {
				// everything from start to pos - 1 is not a word, so reset start and start word tracking
				start = pos
				word = true
			}
			continue
		} else if word && start < pos {
			// everything from start to pos - 1 is a word, so add it as a token and reset start
			tokens = append(tokens, strings.ToLower(text[start:pos]))
			start = pos
		}

		// exited the word
		word = false
	}

	// since we append tokens when we already left the word (on next iteration),
	// we need to do it manually for the last iteration
	if word {
		tokens = append(tokens, strings.ToLower(text[start:]))
	}

	return tokens
}
