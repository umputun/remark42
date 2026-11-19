package migrator

import (
	"bytes"
	"regexp"
	"slices"
	"strconv"
	"strings"

	bf "github.com/russross/blackfriday/v2"
)

// wpCommentHTML renders the text of a WordPress comment the way WordPress shows it to readers. WordPress keeps
// comments as HTML and does not interpret Markdown; its comment_text filters link bare URLs and emails, curl
// quotes and dashes, turn text smilies into emoji and add paragraphs and line breaks. fancy enables the quotes
// and dashes, which WordPress does with wptexturize and this function with SmartyPants.
func wpCommentHTML(text string, fancy bool) string {
	text = wpMakeClickable(text)
	if fancy {
		text = wpTexturize(text)
	}
	text = wpConvertChars(text)
	text = wpConvertSmilies(text)
	text = wpAutop(text)
	return wpCapitalPDangit(text)
}

var (
	wpTagSplit     = regexp.MustCompile(`<[^<>]+>`)
	wpURLClickable = regexp.MustCompile(`([\s(<.,;:!?])` + // leading whitespace or punctuation
		`(\w{1,20}://[\w\x{80}-\x{10FFFF}#%~/@\[\]*(+=&$-]*(?:['.,;:!?)][\w\x{80}-\x{10FFFF}#%~/@\[\]*(+=&$-]+)*)` + // url
		`(\)?)` + // trailing closing parenthesis, kept only when paired
		`(\.\w{2,6})?`) // file extension
	wpWebFtpClickable = regexp.MustCompile(`(?is)([\s>])((www|ftp)\.[\w\x{80}-\x{10FFFF}#$%&~/.\-;:=,?@\[\]+]+)`)
	wpEmailClickable  = regexp.MustCompile(`(?i)([\s>])([.0-9a-z_+-]+)@(([0-9a-z-]+\.)+[0-9a-z]{2,})`)
	wpNestedLinks     = regexp.MustCompile(`(?i)(<a([ \r\n\t]+[^>]+?>|>))<a [^>]+?>([^>]+?)</a></a>`)
	wpPseudoTagURL    = regexp.MustCompile(`^<\s*\w{1,20}://`)
	wpURLBadChars     = regexp.MustCompile(`(?i)[^a-z0-9\-~+_.?#=!&;,/:%@$|*'()\[\]\x{80}-\x{10FFFF}]`)
)

// wpMaxURLLen is the length of the longest url make_clickable links
const wpMaxURLLen = 2000

// wpProtocols are the URL schemes WordPress links, from wp_allowed_protocols
var wpProtocols = map[string]bool{"http": true, "https": true, "ftp": true, "ftps": true, "mailto": true, "news": true,
	"irc": true, "irc6": true, "ircs": true, "gopher": true, "nntp": true, "feed": true, "telnet": true, "mms": true,
	"rtsp": true, "sms": true, "svn": true, "tel": true, "fax": true, "xmpp": true, "webcal": true, "urn": true}

// wpMakeClickable links URLs, www and ftp hosts and email addresses outside tags, links, pre and code, as make_clickable does
func wpMakeClickable(text string) string {
	var r strings.Builder
	nested := 0 // how deep the text is inside pre, code, script or style
	for _, piece := range wpSplitTags(text, wpTagSplit) {
		lower := strings.ToLower(piece)
		switch {
		case wpOpensTag(lower, "code", "pre", "script", "style"):
			nested++
		case nested > 0 && (lower == "</code>" || lower == "</pre>" || lower == "</script>" || lower == "</style>"):
			nested--
		}
		if nested > 0 || piece == "" || (piece[0] == '<' && !wpPseudoTagURL.MatchString(piece)) {
			r.WriteString(piece)
			continue
		}
		ret := " " + piece + " " // padded with whitespace to simplify the regexes
		ret = wpReplaceSubmatch(wpURLClickable, ret, wpURLLink)
		ret = wpReplaceSubmatch(wpWebFtpClickable, ret, wpWebFtpLink)
		ret = wpReplaceSubmatch(wpEmailClickable, ret, func(m []string) string {
			email := m[2] + "@" + m[3]
			return m[1] + `<a href="mailto:` + email + `">` + email + `</a>`
		})
		r.WriteString(ret[1 : len(ret)-1])
	}
	return wpNestedLinks.ReplaceAllString(r.String(), "$1$3</a>")
}

func wpURLLink(m []string) string {
	url, suffix := m[2], m[3]
	if m[3] == ")" && strings.Index(url, "(") > 0 {
		url, suffix = url+m[3], ""
	}
	url += m[4]
	if len(url) > wpMaxURLLen {
		return m[0]
	}
	for strings.Count(url, "(") < strings.Count(url, ")") { // parentheses belong to the url only when paired
		i := strings.LastIndex(url, ")")
		suffix = url[i:] + suffix
		url = url[:i]
	}
	if url = wpEscURL(url); url == "" {
		return m[0]
	}
	return m[1] + `<a href="` + url + `"` + wpRel(url) + `>` + url + `</a>` + suffix
}

func wpWebFtpLink(m []string) string {
	dest, ret := "http://"+m[2], ""
	if last := dest[len(dest)-1:]; strings.ContainsAny(last, ".,;:)") {
		dest, ret = dest[:len(dest)-1], last
	}
	if dest = wpEscURL(dest); dest == "" {
		return m[0]
	}
	return m[1] + `<a href="` + dest + `"` + wpRel(dest) + `>` + dest + `</a>` + ret
}

// wpRel is the rel attribute WordPress gives a link in a comment
func wpRel(url string) string {
	scheme, _, _ := strings.Cut(url, ":")
	if scheme = strings.ToLower(scheme); scheme == "http" || scheme == "https" {
		return ` rel="nofollow ugc"`
	}
	return ` rel="ugc"`
}

// wpEscURL cleans a url for display as esc_url does, and returns an empty string for a scheme WordPress does not link
func wpEscURL(url string) string {
	url = strings.ReplaceAll(strings.TrimSpace(url), " ", "%20")
	url = wpURLBadChars.ReplaceAllString(url, "")
	if url == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(url), "mailto:") {
		for _, s := range []string{"%0d", "%0a", "%0D", "%0A"} {
			url = strings.ReplaceAll(url, s, "")
		}
	}
	url = strings.ReplaceAll(url, ";//", "://")
	if scheme, _, ok := strings.Cut(url, ":"); ok && !wpProtocols[strings.ToLower(scheme)] {
		return ""
	}
	url = wpConvertChars(url)
	return strings.ReplaceAll(strings.ReplaceAll(url, "&amp;", "&#038;"), "'", "&#039;")
}

// wpTexturize curls quotes and turns dashes and ellipses into typographic ones outside tags and outside
// pre, code, kbd, style, script and tt, where wptexturize leaves the text alone
func wpTexturize(text string) string {
	var r strings.Builder
	var skip []string
	for _, piece := range wpSplitTags(text, wpTagSplit) {
		lower := strings.ToLower(piece)
		switch {
		case strings.HasPrefix(piece, "<!--"):
		case strings.HasPrefix(piece, "</"):
			if name := strings.TrimSuffix(lower[2:], ">"); len(skip) > 0 && skip[len(skip)-1] == name {
				skip = skip[:len(skip)-1]
			}
		case strings.HasPrefix(piece, "<"):
			for _, name := range []string{"pre", "code", "kbd", "style", "script", "tt"} {
				if wpOpensTag(lower, name) && !strings.HasSuffix(piece, "/>") {
					skip = append(skip, name)
				}
			}
		case len(skip) == 0 && strings.TrimSpace(piece) != "":
			piece = wpTexturizeText(piece)
		}
		r.WriteString(piece)
	}
	return r.String()
}

var (
	wpCockney = strings.NewReplacer("``", "&#8220;", "''", "&#8221;", " (tm)", " &#8482;", "'tain't", "&#8217;tain&#8217;t", "'twere", "&#8217;twere",
		"'twas", "&#8217;twas", "'tis", "&#8217;tis", "'twill", "&#8217;twill", "'til", "&#8217;til", "'bout", "&#8217;bout",
		"'nuff", "&#8217;nuff", "'round", "&#8217;round", "'cause", "&#8217;cause", "'em", "&#8217;em")
	wpMasked = strings.NewReplacer("(c)", "\x01c)", "(C)", "\x01C)", "(r)", "\x01r)", "(R)", "\x01R)", "(tm)", "\x01tm)",
		"(TM)", "\x01TM)", "/", "\x02", "<", "\x03")
	wpUnmasked = strings.NewReplacer("\x01", "(", "\x02", "/", "\x03", "<")
	wpTimes    = regexp.MustCompile(`\b(\d[\d.,]*)x(\d[\d.,]*)\b`)
)

// wpTexturizeText texturizes a piece of text between tags. SmartyPants also turns (c), (r) and 1/2 into symbols,
// which wptexturize does not, so those parentheses and slashes are hidden from it, and so is <, which SmartyPants
// expects to start a tag. Each call gets its own renderer, as SmartyPants keeps the open quotes it has seen.
func wpTexturizeText(text string) string {
	text = wpCockney.Replace(text)
	if strings.ContainsAny(text, "\x01\x02\x03") {
		return text
	}
	text = wpDecadeApostrophes(wpPrimes(wpPrimes(text, '"', "&#8243;"), '\'', "&#8242;"))
	var buf bytes.Buffer
	bf.NewSmartypantsRenderer(bf.Smartypants).Process(&buf, []byte(wpMasked.Replace(text)))
	text = wpDashes(wpUnmasked.Replace(buf.String()))
	return wpReplaceSubmatch(wpTimes, text, func(m []string) string {
		if m[1] == "0" { // 0x1f is a hex number
			return m[0]
		}
		return m[1] + "&#215;" + m[2]
	})
}

// wpPrimes replaces the quotes that wptexturize_primes turns into primes, as in 9" or 6'. Text is split into
// sentences at opening quotes; in a sentence that has to close its quote, a quote after a digit stays a quote
// when it is the only one that can close it.
func wpPrimes(text string, quote byte, prime string) string {
	if !strings.Contains(text, string(quote)) {
		return text
	}
	sentences, closedEarly := wpSentences(text, quote)
	primes := map[int]bool{}
	for key, sn := range sentences {
		for _, i := range wpSentencePrimes(text, sn, key == 0, closedEarly, quote) {
			primes[i] = true
		}
	}
	if len(primes) == 0 {
		return text
	}
	var r strings.Builder
	for i := 0; i < len(text); i++ {
		if primes[i] {
			r.WriteString(prime)
			continue
		}
		r.WriteByte(text[i])
	}
	return r.String()
}

// wpSpan is a part of text from at to end
type wpSpan struct{ at, end int }

// wpSentences splits text into the sentences wptexturize_primes works on, which start after an opening quote,
// and finds the quotes wptexturize closes before it looks for primes, as in '99"
func wpSentences(text string, quote byte) (sentences []wpSpan, closedEarly map[int]bool) {
	closedEarly = map[int]bool{}
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] != quote {
			continue
		}
		if i >= 3 && text[i-3] == '\'' && wpDigit(text[i-2]) && wpDigit(text[i-1]) && wpQuoteEnds(text[i+1:]) {
			closedEarly[i] = true
			continue
		}
		if wpOpensQuote(text, i, quote) {
			sentences = append(sentences, wpSpan{start, i})
			start = i + 1
		}
	}
	return append(sentences, wpSpan{start, len(text)}), closedEarly
}

// wpSentencePrimes returns the positions of the quotes in a sentence that become primes
func wpSentencePrimes(text string, sn wpSpan, first bool, closedEarly map[int]bool, quote byte) []int {
	var afterDigit, ending []int
	closed := false
	for i := sn.at; i < sn.end; i++ {
		switch {
		case text[i] != quote:
			continue
		case closedEarly[i]:
			closed = true
			continue
		case quote == '\'' && i > 0 && !wpSpaceBefore(text[:i]) && !wpApostropheEnds(text[i+1:]):
			continue // an apostrophe, as in 90's or 5'10
		}
		if i > 0 && wpDigit(text[i-1]) {
			afterDigit = append(afterDigit, i)
		}
		if wpQuoteEnds(text[i+1:]) {
			ending = append(ending, i)
		}
	}
	if first || closed || len(ending) == 0 {
		return afterDigit
	}
	closer := wpCloser(text, ending)
	return slices.DeleteFunc(afterDigit, func(i int) bool { return i == closer })
}

// wpCloser returns the quote among the ending ones that closes the sentence's quotation when it follows a digit,
// or -1 when a quote that does not follow a digit closes it
func wpCloser(text string, ending []int) int {
	if len(ending) == 1 {
		return ending[0]
	}
	for _, i := range ending {
		if !wpDigit(text[i-1]) {
			return -1
		}
	}
	closer := ending[len(ending)-1]
	for _, i := range ending {
		if strings.HasPrefix(text[i+1:], ".") {
			closer = i
		}
	}
	return closer
}

// wpDecadeApostrophes turns a single quote before two digits into an apostrophe, as in '99 or the '80s, unless
// the digits go on as a number
func wpDecadeApostrophes(text string) string {
	var r strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] == '\'' && i+2 < len(text) && wpDigit(text[i+1]) && wpDigit(text[i+2]) {
			rest := text[i+3:]
			number := rest != "" && (rest[0] == '%' || wpDigit(rest[0]) || (len(rest) > 1 && (rest[0] == '.' || rest[0] == ',') && wpDigit(rest[1])))
			if !number {
				r.WriteString("&#8217;")
				continue
			}
		}
		r.WriteByte(text[i])
	}
	return r.String()
}

// wpOpensQuote reports whether the quote at i opens a quotation by the rules of wptexturize: it follows the
// start, a space, an opening bracket, a dash or &lt; (and a double quote for a single one), and a double
// quote is not followed by a space
func wpOpensQuote(text string, i int, quote byte) bool {
	before := text[:i]
	opens := wpSpaceBefore(before) || strings.ContainsAny(before[len(before)-1:], "([{-") ||
		strings.HasSuffix(before, "&lt;") || (quote == '\'' && strings.HasSuffix(before, `"`))
	if quote == '"' {
		return opens && (i+1 >= len(text) || !wpSpaceAfter(text[i+1:]))
	}
	return opens && (i+2 >= len(text) || !wpDigit(text[i+1]) || !wpDigit(text[i+2])) // '99 is an apostrophe
}

// wpQuoteEnds reports whether the text after a quote lets it close a quotation: the end, a space, &gt; or
// closing punctuation
func wpQuoteEnds(rest string) bool {
	return wpSpaceAfter(rest) || strings.ContainsAny(rest[:1], ".,:;!?)}-]") || strings.HasPrefix(rest, "&gt;")
}

// wpApostropheEnds reports whether the text after a single quote keeps wptexturize from reading it as an apostrophe
func wpApostropheEnds(rest string) bool {
	return wpSpaceAfter(rest) || strings.ContainsAny(rest[:1], `.,:;!?"'(){}[]-`) || strings.HasPrefix(rest, "&lt;") ||
		strings.HasPrefix(rest, "&gt;")
}

func wpDigit(c byte) bool { return c >= '0' && c <= '9' }

// wpDashes applies the dash rules of wptexturize: --- and a spaced -- are em dashes, other -- (except in
// punycode xn--) and a spaced - are en dashes
func wpDashes(text string) string {
	if !strings.Contains(text, "-") {
		return text
	}
	text = strings.ReplaceAll(text, "---", "&#8212;")
	var r strings.Builder
	for i := 0; i < len(text); {
		if text[i] != '-' {
			r.WriteByte(text[i])
			i++
			continue
		}
		double := i+1 < len(text) && text[i+1] == '-'
		end := i + 1
		if double {
			end = i + 2
		}
		spaced := wpSpaceBefore(text[:i]) && wpSpaceAfter(text[end:])
		switch {
		case double && spaced:
			r.WriteString("&#8212;")
		case double && !strings.HasSuffix(text[:i], "xn"):
			r.WriteString("&#8211;")
		case double:
			r.WriteString("--")
		case spaced:
			r.WriteString("&#8211;")
		default:
			r.WriteByte('-')
		}
		i = end
	}
	return r.String()
}

// wpSpaceBefore reports whether s is empty or ends with a space as wp_spaces_regexp defines it
func wpSpaceBefore(s string) bool {
	return s == "" || strings.ContainsAny(s[len(s)-1:], "\r\n\t ") || strings.HasSuffix(s, "\u00a0") ||
		strings.HasSuffix(s, "&nbsp;")
}

// wpSpaceAfter reports whether s is empty or starts with a space as wp_spaces_regexp defines it
func wpSpaceAfter(s string) bool {
	return s == "" || strings.ContainsAny(s[:1], "\r\n\t ") || strings.HasPrefix(s, "\u00a0") ||
		strings.HasPrefix(s, "&nbsp;")
}

// wpConvertChars escapes an ampersand that does not start an entity, as convert_chars does
func wpConvertChars(text string) string {
	if !strings.Contains(text, "&") {
		return text
	}
	var r strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] != '&' || i+1 >= len(text) || text[i+1] == '#' || wpEntityName(text[i+1:]) {
			r.WriteByte(text[i])
			continue
		}
		r.WriteString("&#038;")
	}
	return r.String()
}

// wpEntityName reports whether s starts with 1 to 8 of [a-z1-4], case-insensitive, and a semicolon
func wpEntityName(s string) bool {
	for i := 0; i < len(s) && i <= 8; i++ {
		c := s[i] | 0x20
		switch {
		case c >= 'a' && c <= 'z', s[i] >= '1' && s[i] <= '4':
			continue
		case s[i] == ';':
			return i > 0
		}
		return false
	}
	return false
}

// wpSmilies maps text smilies to the emoji WordPress shows, from smilies_init. WordPress shows :mrgreen: as an
// image on the blog's own server, so it is left as text.
var wpSmilies = map[string]string{
	":neutral:": "😐", ":twisted:": "😈", ":arrow:": "➡", ":shock:": "😯", ":smile:": "🙂", ":???:": "😕",
	":cool:": "😎", ":evil:": "👿", ":grin:": "😀", ":idea:": "💡", ":oops:": "😳", ":razz:": "😛",
	":roll:": "🙄", ":wink:": "😉", ":cry:": "😥", ":eek:": "😮", ":lol:": "😆", ":mad:": "😡",
	":sad:": "🙁", "8-)": "😎", "8-O": "😯", ":-(": "🙁", ":-)": "🙂", ":-?": "😕", ":-D": "😀",
	":-P": "😛", ":-o": "😮", ":-x": "😡", ":-|": "😐", ";-)": "😉", "8O": "😯", ":(": "🙁",
	":)": "🙂", ":?": "😕", ":D": "😀", ":P": "😛", ":o": "😮", ":x": "😡", ":|": "😐", ";)": "😉",
	":!:": "❗", ":?:": "❓",
}

var (
	wpAnyTag   = regexp.MustCompile(`<[^>]*>`)
	wpSpaces   = regexp.MustCompile(`[\r\n\t ]|\x{a0}|&nbsp;`)
	wpIgnoreRe = regexp.MustCompile(`^<(code|pre|style|script|textarea)[^>]*>`)
)

// wpConvertSmilies replaces smilies that stand between spaces, outside code, pre, style, script and textarea,
// as convert_smilies does
func wpConvertSmilies(text string) string {
	var r strings.Builder
	ignore := ""
	for _, piece := range wpSplitTags(text, wpAnyTag) {
		if m := wpIgnoreRe.FindStringSubmatch(piece); ignore == "" && m != nil {
			ignore = m[1]
		}
		if ignore == "" && piece != "" && piece[0] != '<' {
			piece = wpReplaceWords(piece)
		}
		if ignore != "" && piece == "</"+ignore+">" {
			ignore = ""
		}
		r.WriteString(piece)
	}
	return r.String()
}

// wpReplaceWords replaces each smiley that is a whole space-separated word of text
func wpReplaceWords(text string) string {
	var r strings.Builder
	prev := 0
	for _, loc := range append(wpSpaces.FindAllStringIndex(text, -1), []int{len(text), len(text)}) {
		word := text[prev:loc[0]]
		if emoji, ok := wpSmilies[word]; ok {
			word = emoji
		}
		r.WriteString(word)
		r.WriteString(text[loc[0]:loc[1]])
		prev = loc[1]
	}
	return r.String()
}

const wpAllBlocks = `(?:table|thead|tfoot|caption|col|colgroup|tbody|tr|td|th|div|dl|dd|dt|ul|ol|li|pre|form|map|area|` +
	`blockquote|address|style|p|h[1-6]|hr|fieldset|legend|section|article|aside|hgroup|header|footer|nav|figure|` +
	`figcaption|details|menu|summary)`

var (
	wpDoubleBr      = regexp.MustCompile(`<br\s*/?>\s*<br\s*/?>`)
	wpBlockOpen     = regexp.MustCompile(`(<` + wpAllBlocks + `[\s/>])`)
	wpBlockClose    = regexp.MustCompile(`(</` + wpAllBlocks + `>)`)
	wpHr            = regexp.MustCompile(`(<hr\s*?/?>)`)
	wpElements      = regexp.MustCompile(`(?s)<!--.*?-->|<[^>]*>`)
	wpOptionBefore  = regexp.MustCompile(`\s*<option`)
	wpOptionAfter   = regexp.MustCompile(`</option>\s*`)
	wpFigcapBefore  = regexp.MustCompile(`\s*(<figcaption[^>]*>)`)
	wpFigcapAfter   = regexp.MustCompile(`</figcaption>\s*`)
	wpManyBreaks    = regexp.MustCompile("\n\n+")
	wpParagraphs    = regexp.MustCompile(`\n\s*\n`)
	wpEmptyP        = regexp.MustCompile(`<p>\s*</p>`)
	wpUnclosedP     = regexp.MustCompile(`<p>([^<]+)</(div|address|form)>`)
	wpWrappedBlock  = regexp.MustCompile(`<p>\s*(</?` + wpAllBlocks + `[^>]*>)\s*</p>`)
	wpWrappedLi     = regexp.MustCompile(`<p>(<li.+?)</p>`)
	wpPBlockquote   = regexp.MustCompile(`(?i)<p><blockquote((?:[^>"']|"[^"]*"|'[^']*')*)>`)
	wpPBeforeBlock  = regexp.MustCompile(`<p>\s*(</?` + wpAllBlocks + `[^>]*>)`)
	wpPAfterBlock   = regexp.MustCompile(`(</?` + wpAllBlocks + `[^>]*>)\s*</p>`)
	wpNewline       = regexp.MustCompile(`(?:<br />)?\s*\n`)
	wpBrAfterBlock  = regexp.MustCompile(`(</?` + wpAllBlocks + `[^>]*>)\s*<br />`)
	wpBrBeforeBlock = regexp.MustCompile(`<br />(\s*</?(?:p|li|div|dl|dd|dt|th|pre|td|ul|ol)[^>]*>)`)
)

// wpAutop wraps paragraphs separated by blank lines in <p> and turns other newlines into <br />, leaving
// block elements and the content of pre alone, as wpautop does
func wpAutop(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	text += "\n"

	pre := map[string]string{}
	if strings.Contains(text, "<pre") {
		parts := strings.Split(text, "</pre>")
		var b strings.Builder
		for i, part := range parts[:len(parts)-1] {
			start := strings.Index(part, "<pre")
			if start < 0 { // malformed html
				b.WriteString(part)
				continue
			}
			name := "<pre wp-pre-tag-" + strconv.Itoa(i) + "></pre>"
			pre[name] = part[start:] + "</pre>"
			b.WriteString(part[:start] + name)
		}
		text = b.String() + parts[len(parts)-1]
	}

	text = wpDoubleBr.ReplaceAllString(text, "\n\n")
	text = wpBlockOpen.ReplaceAllString(text, "\n\n$1")
	text = wpBlockClose.ReplaceAllString(text, "$1\n\n")
	text = wpHr.ReplaceAllString(text, "$1\n\n")
	text = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text)
	text = wpElements.ReplaceAllStringFunc(text, func(el string) string { return strings.ReplaceAll(el, "\n", " <!-- wpnl --> ") })
	if strings.Contains(text, "<option") {
		text = wpOptionAfter.ReplaceAllString(wpOptionBefore.ReplaceAllString(text, "<option"), "</option>")
	}
	if strings.Contains(text, "<figcaption") {
		text = wpFigcapAfter.ReplaceAllString(wpFigcapBefore.ReplaceAllString(text, "$1"), "</figcaption>")
	}
	text = wpManyBreaks.ReplaceAllString(text, "\n\n")

	var b strings.Builder
	for _, paragraph := range wpParagraphs.Split(text, -1) {
		if paragraph != "" {
			b.WriteString("<p>" + strings.Trim(paragraph, "\n") + "</p>\n")
		}
	}
	text = b.String()

	text = wpEmptyP.ReplaceAllString(text, "")
	text = wpUnclosedP.ReplaceAllString(text, "<p>$1</p></$2>")
	text = wpWrappedBlock.ReplaceAllString(text, "$1")
	text = wpWrappedLi.ReplaceAllString(text, "$1")
	text = wpPBlockquote.ReplaceAllString(text, "<blockquote$1><p>")
	text = strings.ReplaceAll(text, "</blockquote></p>", "</p></blockquote>")
	text = wpPBeforeBlock.ReplaceAllString(text, "$1")
	text = wpPAfterBlock.ReplaceAllString(text, "$1")
	text = strings.NewReplacer("<br>", "<br />", "<br/>", "<br />").Replace(text)
	text = wpNewline.ReplaceAllString(text, "<br />\n")
	text = wpBrAfterBlock.ReplaceAllString(text, "$1")
	text = wpBrBeforeBlock.ReplaceAllString(text, "$1")
	if strings.HasSuffix(text, "\n</p>") {
		text = strings.TrimSuffix(text, "\n</p>") + "</p>"
	}
	restore := []string{" <!-- wpnl --> ", "\n", "<!-- wpnl -->", "\n"}
	for name, content := range pre {
		restore = append(restore, name, content)
	}
	return strings.NewReplacer(restore...).Replace(text)
}

// wpCapitalPDangit fixes the capitalization of WordPress after a space, quote, tag or parenthesis
func wpCapitalPDangit(text string) string {
	return strings.NewReplacer(" Wordpress", " WordPress", "&#8216;Wordpress", "&#8216;WordPress",
		"&#8220;Wordpress", "&#8220;WordPress", "&lsquo;Wordpress", "&lsquo;WordPress",
		"&ldquo;Wordpress", "&ldquo;WordPress", ">Wordpress", ">WordPress", "(Wordpress", "(WordPress").Replace(text)
}

// wpSplitTags splits text into the pieces between matches of tag and the matches themselves, in order
func wpSplitTags(text string, tag *regexp.Regexp) []string {
	var res []string
	prev := 0
	for _, loc := range tag.FindAllStringIndex(text, -1) {
		res = append(res, text[prev:loc[0]], text[loc[0]:loc[1]])
		prev = loc[1]
	}
	return append(res, text[prev:])
}

// wpOpensTag reports whether the lowercased piece opens one of the named elements
func wpOpensTag(lower string, names ...string) bool {
	for _, name := range names {
		if rest, ok := strings.CutPrefix(lower, "<"+name); ok && rest != "" && (rest[0] == '>' || rest[0] == ' ' ||
			rest[0] == '\t' || rest[0] == '\n' || rest[0] == '/') {
			return true
		}
	}
	return false
}

// wpReplaceSubmatch replaces every match of re in s with repl applied to its submatches
func wpReplaceSubmatch(re *regexp.Regexp, s string, repl func(m []string) string) string {
	var r strings.Builder
	prev := 0
	for _, loc := range re.FindAllStringSubmatchIndex(s, -1) {
		m := make([]string, len(loc)/2)
		for i := range m {
			if loc[2*i] >= 0 {
				m[i] = s[loc[2*i]:loc[2*i+1]]
			}
		}
		r.WriteString(s[prev:loc[0]])
		r.WriteString(repl(m))
		prev = loc[1]
	}
	r.WriteString(s[prev:])
	return r.String()
}
