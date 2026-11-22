package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatcher_Tokenize(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})

	tbl := []struct {
		input  string
		output []string
	}{
		{
			"   word0 word1 word2, word3,,, !word4 !word5? word6-word7 word8",
			[]string{"word0", "word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8"},
		},
		{"русский 中文 française ไทย", []string{"русский", "中文", "française", "ไทย"}},
		{"word", []string{"word"}},
		{"", []string{}},
		{"\t\t\n\t \n\t \r\n    \t ,,, !#$%^&*()", []string{}},
		{"👍", []string{}},
	}

	for _, td := range tbl {
		tokens := matcher.tokenize(td.input)
		assert.Equal(t, td.output, tokens, "unexpected result for input '%v'", td.input)
	}
}

func TestWildcardTrie_Check(t *testing.T) {
	tbl := []struct {
		input   []string
		match   []string
		nomatch []string
	}{
		{[]string{"abc", "abb", "aab"}, []string{"abc", "abb", "aab"}, []string{"aaa", "aaaa", "a", "ab"}},
		{[]string{"abc", "*ck", "*z"}, []string{"abc", "duck", "quack", "ck", "xyz"}, []string{"quacker", "buzzer"}},
		{[]string{"abc", "du*", "c*"}, []string{"abc", "duck", "dungeon", "du", "cup"}, []string{"bbc", "ddu", "scuba"}},
		{[]string{"abc", "*uc*", "*x*"}, []string{"abc", "duck", "stuck", "uc", "wwxww", "xww", "wwx"}, []string{"bbc", "duke"}},
		{[]string{"abc", "d*k", "st*ck"}, []string{"abc", "duck", "dk", "stck", "stuck", "stiiick"}, []string{"bbc", "adka", "st", "ck"}},
		{[]string{"abc", "*a*a*"}, []string{"abc", "safari", "banana", "aa"}, []string{"bbc", "car", "a"}},
		{
			[]string{"ложить", "при*", "*ий", "*бег*", "про*жа", "*ไ*ย*", "*請*请*"},
			[]string{"ложить", "приклад", "ихний", "прибегать", "пропажа", "ไทย", "ทไย", "ไยท", "請問请问"},
			[]string{"положить", "гранпри", "бийск", "請", "ยไท"},
		},
	}

	for _, td := range tbl {
		n := newWildcardTrie(td.input...)

		for _, token := range td.match {
			assert.True(t, n.check(token), "should match token '%s' for restricted words '%v'", token, td.input)
		}

		for _, token := range td.nomatch {
			assert.False(t, n.check(token), "should not match token '%s' for restricted words '%v'", token, td.input)
		}
	}
}

func TestMatcher_MatchIfContainsRestrictedWords(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"duck"}})
	text := "What the duck it that?"
	assert.True(t, matcher.Match("fakeID", text))
}

func TestMatcher_DoNotMatchIfNoRestrictedWords(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"quack"}})
	text := "What the duck it that?"
	assert.False(t, matcher.Match("fakeID", text))
}

func TestFileRestrictedWordsLister(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.txt")
	require.NoError(t, os.WriteFile(path, []byte("\ufeff# spam words, saved with a BOM\n  viagra  \n\n*casino*\n"), 0o600))
	lister := &FileRestrictedWordsLister{Path: path, Words: []string{"duck"}}
	matcher := NewRestrictedWordsMatcher(lister)

	words, err := lister.List("site")
	require.NoError(t, err)
	assert.Equal(t, []string{"duck", "viagra", "*casino*"}, words, "static words first, comments and empty lines skipped")
	assert.True(t, matcher.Match("site", "best onlinecasino here"))
	assert.True(t, matcher.Match("site", "What the duck"))
	assert.False(t, matcher.Match("site", "a poker night"))

	// a changed file is read again without a restart; pin a later mtime instead of waiting for the clock
	require.NoError(t, os.WriteFile(path, []byte("poker\n"), 0o600))
	later := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(path, later, later))
	words, err = lister.List("site")
	require.NoError(t, err)
	assert.Equal(t, []string{"duck", "poker"}, words)
	assert.True(t, matcher.Match("site", "a poker night"))
	assert.False(t, matcher.Match("site", "best onlinecasino here"))

	// a file that disappears keeps the last patterns that were read
	require.NoError(t, os.Remove(path))
	words, err = lister.List("site")
	require.NoError(t, err)
	assert.Equal(t, []string{"duck", "poker"}, words)
	require.Error(t, lister.Load())
}

func TestMatcher_MatchRenderedText(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"duck", "quack*"}})

	tbl := []struct {
		text  string
		match bool
	}{
		{"<p>What the duck is that?</p>\n", true},
		{"<p>What the du<b></b>ck is that?</p>\n", true},
		{"<p>What the <strong>du</strong>ck is that?</p>\n", true},
		{"<p>What the du<span class=\"w\"></span>ck is that?</p>\n", true},
		{"<p>What the du<!-- -->ck is that?</p>\n", true},
		{"<p>What the du<wbr/>ck is that?</p>\n", true},
		{"<p>What the du<br/>ck is that?</p>\n", true}, // hand-written, shown on two lines but joined: tags are dropped without a space
		{"<p>What the qu<i></i>acker is that?</p>\n", true},
		{"<p>What the du\u00adck is that?</p>\n", true},  // soft hyphen
		{"<p>What the du&shy;ck is that?</p>\n", true},   // soft hyphen as entity
		{"<p>What the du\u200bck is that?</p>\n", true},  // zero-width space
		{"<p>What the du&#8203;ck is that?</p>\n", true}, // zero-width space as entity
		{"<p>What the du\u200dck is that?</p>\n", true},  // zero-width joiner
		{"<p>What the du\u034fck is that?</p>\n", true},  // combining grapheme joiner
		{"<p>What the du\ufe0fck is that?</p>\n", true},  // variation selector
		{"<p>What the d&#117;ck is that?</p>\n", true},   // letter as entity
		{"<p><a href=\"https://duck.example.com/\" rel=\"nofollow\">link</a></p>\n", true},
		{"<p><img src=\"https://example.com/duck.png\" alt=\"pic\"/></p>\n", true},
		{"<pre><code><span class=\"nx\">du</span><span class=\"nx\">ck</span></code></pre>", true},
		{"<p>What the du ck is that?</p>\n", false},
		{"<p>What the du-ck is that?</p>\n", false},
		{"<p>What the du</p>\n\n<p>ck is that?</p>\n", false},
		{"<p>What the du<br/>\nck is that?</p>\n", false},
		{"<ul>\n<li>du</li>\n<li>ck</li>\n</ul>\n", false},
		{"<p>What the du&lt;b&gt;&lt;/b&gt;ck is that?</p>\n", false}, // escaped tag is shown as is
		{"<p>What the du&amp;shy;ck is that?</p>\n", false},           // escaped entity is shown as is
		{"<p>What the <b>dock</b> is that?</p>\n", false},
		{"<p>What the du\u0600ck is that?</p>\n", false}, // the arabic number sign is a format character that is displayed
	}

	for _, td := range tbl {
		assert.Equal(t, td.match, matcher.Match("fakeID", td.text), "unexpected result for text %q", td.text)
	}
}

func TestMatcher_MatchPercentEncodedAddress(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"duck", "\u0443\u0442\u043a\u0430"}})

	tbl := []struct {
		text  string
		match bool
	}{
		// the sanitizer stores a link or image address with its non-ASCII letters percent-encoded
		{`<p><a href="https://example.com/%D1%83%D1%82%D0%BA%D0%B0" rel="nofollow">link</a></p>`, true},
		{`<p><a href="https://%D1%83%D1%82%D0%BA%D0%B0.example.com/" rel="nofollow">link</a></p>`, true},
		{`<p><img src="https://example.com/%D1%83%D1%82%D0%BA%D0%B0.png" alt="pic"/></p>`, true},
		{`<p><a href="https://example.com/d%75ck" rel="nofollow">link</a></p>`, true}, // an escaped ASCII letter
		{`<p><a href="https://example.com/%D1%83%D1%82%D0%BA%D0%BE" rel="nofollow">link</a></p>`, false},
		// a "%" that starts no escape is kept and does not switch the decoding off for the rest of the text
		{`<p>100% sure, see <a href="https://example.com/d%75ck" rel="nofollow">link</a></p>`, true},
		// an escape cut in the middle of a character decodes to a byte that ends the word before it
		{`<p><a href="https://example.com/%D1%83%D1%82%D0%BA%D0%B0%D1" rel="nofollow">link</a></p>`, true},
		// incomplete escapes with no banned word around them: no match
		{`<p>100% of 50%zz and %D1 alone, then %D1%83%D1</p>`, false},
	}
	for _, tt := range tbl {
		assert.Equal(t, tt.match, matcher.Match("radio-t", tt.text), "unexpected result for text %q", tt.text)
	}

	assert.Equal(t, "50%zz and \xd1 alone, d\u0443ck", percentDecoded("50%zz and %D1 alone, d%D1%83ck"))
}
