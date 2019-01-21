package service

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RestrictedWordsLister provides restricted words in comments per site
type RestrictedWordsLister interface {
	List(siteID string) (restricted []string, err error)
}

// StaticRestrictedWordsLister provides same restricted words in comments for every site
type StaticRestrictedWordsLister struct {
	Words []string
}

// List provides restricted words in comments (ignores siteID)
func (l StaticRestrictedWordsLister) List(siteID string) (restricted []string, err error) {
	return l.Words, nil
}

// RestrictedWordsMatcher matches comment text against restricted words
type RestrictedWordsMatcher struct {
	lister RestrictedWordsLister
}

// NewRestrictedWordsMatcher creates new RestrictedWordsMatcher using provided RestrictedWordsLister
func NewRestrictedWordsMatcher(lister RestrictedWordsLister) *RestrictedWordsMatcher {
	return &RestrictedWordsMatcher{lister: lister}
}

// Match matches comment text against restricted words for specified site
func (m *RestrictedWordsMatcher) Match(siteID string, text string) bool {
	tokens := m.tokenize(text)

	restrictedWords, err := m.lister.List(siteID)
	if err != nil {
		fmt.Printf("failed to get restricted patterns for site %s: %v", siteID, err)
		return false
	}

	trie := newWildcardTrie(restrictedWords...)

	for _, token := range tokens {
		if trie.check(token) {
			return true
		}
	}
	return false
}

func (m *RestrictedWordsMatcher) tokenize(text string) []string {
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
		}

		if word && start < pos {
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

type wildcardTrie struct {
	terminal bool
	children map[rune]*wildcardTrie
}

func newWildcardTrie(patterns ...string) *wildcardTrie {
	trie := &wildcardTrie{terminal: false, children: make(map[rune]*wildcardTrie)}
	for _, p := range patterns {
		trie.addPattern(p)
	}
	return trie
}

func (trie *wildcardTrie) addPattern(pattern string) {
	// since pattern matching algorithm is recursive we do not allow long patterns
	if utf8.RuneCountInString(pattern) < 1 || utf8.RuneCountInString(pattern) > 64 {
		fmt.Printf("[WARN] invalid pattern length '%s': actual - %d, min allowed - 1, max allowed - 64", pattern, utf8.RuneCountInString(pattern))
		return
	}

	node := trie

	for _, r := range strings.ToLower(strings.TrimSpace(pattern)) {
		if childNode, exists := node.children[r]; exists {
			node = childNode
			continue
		}

		childNode := newWildcardTrie()
		node.children[r] = childNode
		node = childNode
	}

	node.terminal = true
}

// check tests if any pattern stored in trie matches the token. Recursive. Max depth is longest pattern in trie.
func (trie *wildcardTrie) check(token string) bool {
	if len(token) == 0 {
		if trie.terminal {
			return true
		}

		if childNode, exists := trie.children['*']; exists && childNode.terminal {
			return true
		}

		return false
	}

	r, width := utf8.DecodeRuneInString(token)

	if childNode, exists := trie.children[r]; exists {
		if childNode.check(token[width:]) {
			return true
		}
	}

	if childNode, exists := trie.children['*']; exists {
		if childNode.terminal {
			return true
		}
		if childNode.checkAllSuffixes(token) {
			return true
		}
	}

	return false
}

func (trie *wildcardTrie) checkAllSuffixes(token string) bool {
	suffix := token
	for {
		if len(suffix) == 0 {
			return false
		}

		if trie.check(suffix) {
			return true
		}

		_, width := utf8.DecodeRuneInString(suffix)
		suffix = suffix[width:]
	}
}
