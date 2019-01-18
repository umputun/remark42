package service

import (
	"strings"
	"unicode"
)

type Tokenizer interface {
	Tokenize(text string) []string
}

type WhitespaceTokenizer struct{}

func (_ WhitespaceTokenizer) Tokenize(text string) []string {
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
	if word && start < len(text)-1 {
		tokens = append(tokens, strings.ToLower(text[start:]))
	}

	return tokens
}

type Matcher interface {
	Match(words []string) bool
}

type SimpleMatcher struct {
	restricted map[string]bool
}

func NewSimpleMatcher(words []string) *SimpleMatcher {
	restricted := make(map[string]bool)
	for _, w := range words {
		restricted[strings.ToLower(w)] = true
	}

	return &SimpleMatcher{restricted}
}

func (l *SimpleMatcher) Match(words []string) bool {
	for _, w := range words {
		_, present := l.restricted[w]
		if present {
			return true
		}
	}
	return false
}

type Lister interface {
	List(siteID string) (restricted []string, err error)
}

type Checker interface {
	Check(siteID string, text string) bool
}

type RestrictedWordsChecker struct {
	tokenizer      Tokenizer
	lister         Lister
	matcherFactory func([]string) Matcher
	matchers       map[string]Matcher
}

func NewRestrictedWordsChecker(tokenizer Tokenizer, lister Lister, matcherFactory func([]string) Matcher) *RestrictedWordsChecker {
	return &RestrictedWordsChecker{tokenizer, lister, matcherFactory, make(map[string]Matcher)}
}

func (c *RestrictedWordsChecker) Check(siteID string, text string) bool {
	tokens := c.tokenizer.Tokenize(text)

	matcher, _ := c.matchers[siteID]
	if matcher == nil {
		words, err := c.lister.List(siteID)
		if err != nil {
			// todo log error
			return false
		}
		matcher = c.matcherFactory(words)
		c.matchers[siteID] = matcher
	}

	return matcher.Match(tokens)
}
