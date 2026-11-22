package service

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
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
func (l StaticRestrictedWordsLister) List(_ string) (restricted []string, err error) {
	return l.Words, nil
}

// FileRestrictedWordsLister provides restricted words for every site from a file, one pattern per line,
// added to Words. Empty lines and lines starting with # are skipped. The file is read again when its
// modification time or size changes; if it can't be read, the last successfully read patterns stay in use.
type FileRestrictedWordsLister struct {
	Path  string
	Words []string

	mu    sync.Mutex
	mtime time.Time
	size  int64
	file  []string
}

// List provides restricted words from the file and Words (ignores siteID)
func (l *FileRestrictedWordsLister) List(_ string) (restricted []string, err error) {
	if err := l.Load(); err != nil {
		log.Printf("[WARN] restricted words file %s: %v", l.Path, err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Concat(l.Words, l.file), nil
}

// Load reads the file if its modification time or size changed since the last read
func (l *FileRestrictedWordsLister) Load() error {
	fi, err := os.Stat(l.Path)
	if err != nil {
		return fmt.Errorf("failed to stat: %w", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if fi.ModTime().Equal(l.mtime) && fi.Size() == l.size {
		return nil
	}
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}
	var words []string
	// editors that save "UTF-8 with BOM" put U+FEFF before the first line, which TrimSpace keeps
	for line := range strings.Lines(strings.TrimPrefix(string(data), "\ufeff")) {
		if w := strings.TrimSpace(line); w != "" && !strings.HasPrefix(w, "#") {
			words = append(words, w)
		}
	}
	l.mtime, l.size, l.file = fi.ModTime(), fi.Size(), words
	return nil
}

// RestrictedWordsMatcher matches comment text against restricted words
type RestrictedWordsMatcher struct {
	lister RestrictedWordsLister
}

// NewRestrictedWordsMatcher creates new RestrictedWordsMatcher using provided RestrictedWordsLister
func NewRestrictedWordsMatcher(lister RestrictedWordsLister) *RestrictedWordsMatcher {
	return &RestrictedWordsMatcher{lister: lister}
}

// Match matches rendered comment text against restricted words for specified site.
// The html is checked as it is and with percent-escapes decoded (for words in link and image addresses, which
// the sanitizer stores percent-encoded when they are not ASCII), and so is the text with the tags removed.
func (m *RestrictedWordsMatcher) Match(siteID, text string) bool {
	restrictedWords, err := m.lister.List(siteID)
	if err != nil {
		log.Printf("[WARN] failed to get restricted patterns for site %s: %v", siteID, err)
		return false
	}
	if len(restrictedWords) == 0 {
		return false
	}

	tokens := m.tokenize(text)
	tokens = append(tokens, m.tokenize(visibleText(text))...)
	tokens = append(tokens, m.tokenize(percentDecoded(text))...)
	trie := newWildcardTrie(restrictedWords...)

	return slices.ContainsFunc(tokens, trie.check)
}

// rePercentEscapes matches a run of percent-escaped bytes
var rePercentEscapes = regexp.MustCompile(`(?:%[0-9A-Fa-f]{2})+`)

// percentDecoded returns text with every run of percent-escaped bytes decoded. A "%" that starts no escape stays
// as it is, and an escape cut in the middle of a character decodes to bytes that are no letter, so they end a word.
func percentDecoded(text string) string {
	return rePercentEscapes.ReplaceAllStringFunc(text, func(run string) string {
		decoded, err := url.PathUnescape(run)
		if err != nil { // can't happen, a run of well-formed escapes always decodes
			return run
		}
		return decoded
	})
}

// tagStripper removes every tag; built once, a built policy is safe for concurrent use
var tagStripper = bluemonday.StrictPolicy()

// visibleText returns what a reader sees of rendered comment html: tags removed, entities decoded and
// characters which are not displayed (soft hyphen, zero-width space and joiners, variation selectors) dropped.
// Without it any markup or invisible character inside a word splits it in two tokens.
func visibleText(commentHTML string) string {
	text := html.UnescapeString(tagStripper.Sanitize(commentHTML))
	return strings.Map(func(r rune) rune {
		// a format character is not displayed, except the few that are signs put before a number
		if unicode.In(r, unicode.Cf, unicode.Variation_Selector, unicode.Other_Default_Ignorable_Code_Point) &&
			!unicode.Is(unicode.Prepended_Concatenation_Mark, r) {
			return -1
		}
		return r
	}, text)
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
		log.Printf("[WARN] invalid pattern length '%s': actual - %d, min allowed - 1, max allowed - 64", pattern, utf8.RuneCountInString(pattern))
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
	if token == "" {
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
		if suffix == "" {
			return false
		}

		if trie.check(suffix) {
			return true
		}

		_, width := utf8.DecodeRuneInString(suffix)
		suffix = suffix[width:]
	}
}
