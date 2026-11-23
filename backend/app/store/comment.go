package store

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Comment represents a single comment with optional reference to its parent
type Comment struct {
	ID          string                 `json:"id" bson:"_id"`
	ParentID    string                 `json:"pid"`
	Text        string                 `json:"text"`
	Orig        string                 `json:"orig,omitempty"` // important: never render this as HTML! It's not sanitized.
	User        User                   `json:"user"`
	Locator     Locator                `json:"locator"`
	Score       int                    `json:"score"`
	Votes       map[string]bool        `json:"votes,omitempty"`
	VotedIPs    map[string]VotedIPInfo `json:"voted_ips,omitempty"` // voted ips (hashes) with TS
	Vote        int                    `json:"vote"`                // vote for the current user, -1/1/0.
	Controversy float64                `json:"controversy,omitempty"`
	Timestamp   time.Time              `json:"time" bson:"time"`
	Edit        *Edit                  `json:"edit,omitempty" bson:"edit,omitempty"` // pointer to have empty default in json response
	Pin         bool                   `json:"pin,omitempty" bson:"pin,omitempty"`
	Deleted     bool                   `json:"delete,omitempty" bson:"delete"`
	Imported    bool                   `json:"imported,omitempty" bson:"imported"`
	PostTitle   string                 `json:"title,omitempty" bson:"title"`
}

// Locator keeps site and url of the post
type Locator struct {
	SiteID string `json:"site,omitempty" bson:"site"`
	URL    string `json:"url"`
}

// Edit indication
type Edit struct {
	Timestamp time.Time `json:"time" bson:"time"`
	Summary   string    `json:"summary"`
}

// PostInfo holds summary for given post url
type PostInfo struct {
	URL         string    `json:"url,omitempty"` // can be attached to site-wide comments but won't be set then
	Count       int       `json:"count"`
	CountLeft   int       `json:"count_left"`                                     // used only with returning search results limited by number, otherwise zero
	LastComment string    `json:"last_comment,omitempty"`                         // used only with returning search results limited by number
	ReadOnly    bool      `json:"read_only,omitempty" bson:"read_only,omitempty"` // can be attached to site-wide comments but won't be set then
	FirstTS     time.Time `json:"first_time" bson:"first_time,omitempty"`
	LastTS      time.Time `json:"last_time" bson:"last_time,omitempty"`
}

// BlockedUser holds id and ts for blocked user
type BlockedUser struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Until time.Time `json:"time"`
}

// VotedIPInfo keeps timestamp and voting value (direction). Used as VotedIPs value
type VotedIPInfo struct {
	Timestamp time.Time
	Value     bool
	UserID    string `json:",omitempty"` // whose vote it is; empty in records stored before it was kept
}

// DeleteMode defines how much comment info will be erased
type DeleteMode int

// DeleteMode enum
const (
	SoftDelete DeleteMode = 0
	HardDelete DeleteMode = 1
)

// Maximum length for URL text shortening.
const shortURLLen = 48
const snippetLen = 200

// PrepareUntrusted pre-processes a comment received from untrusted source by clearing all
// autogen fields and reset everything users not supposed to provide
func (c *Comment) PrepareUntrusted() {
	c.ID = ""                 // don't allow user to define ID, force auto-gen
	c.Timestamp = time.Time{} // reset time, force auto-gen
	c.Votes = make(map[string]bool)
	c.VotedIPs = make(map[string]VotedIPInfo)
	c.Score = 0
	c.Controversy = 0
	c.Edit = nil
	c.Pin = false
	c.Deleted = false
	c.Imported = false
}

// SetDeleted clears comment info, reset to deleted state. hard flag will clear all user info as well
func (c *Comment) SetDeleted(mode DeleteMode) {
	c.Text = ""
	c.Orig = ""
	c.PostTitle = "" // the commenter can set it, so it goes with the text
	c.Score = 0
	c.Controversy = 0
	c.Votes = map[string]bool{}
	c.VotedIPs = make(map[string]VotedIPInfo)
	c.Edit = nil
	c.Deleted = true
	c.Pin = false

	if mode == HardDelete {
		c.User.Name = "deleted"
		c.User.ID = "deleted"
		c.User.Picture = ""
		c.User.IP = ""
	}
}

// the sanitizing policies are built once and shared: building one compiles a set of regular expressions,
// too expensive to repeat for every comment read from the store, and a built policy is safe for concurrent use
var (
	commentPolicy = newCommentPolicy()        // used by Sanitize for the comment text
	urlPolicy     = bluemonday.UGCPolicy()    // used by SanitizeAsURL
	textPolicy    = bluemonday.StrictPolicy() // used by SanitizeText
)

// newCommentPolicy makes the UGC policy extended with the classes and attributes the comment renderer produces
func newCommentPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(regexp.MustCompile("^chroma$")).OnElements("pre")
	// special case for embedding the quotes from Twitter
	p.AllowAttrs("class").Matching(regexp.MustCompile("^twitter-tweet$")).OnElements("blockquote")
	// this is list of <span> tag classes which could be produced by chroma code renderer
	// source: https://github.com/alecthomas/chroma/blob/c263f6f/types.go#L209-L306
	const codeSpanClassRegex = "^(bg|chroma|line|ln|lnt|hl|lntable|lntd|lnlinks|cl|w|err|x|k|kc" +
		"|kd|kn|kp|kr|kt|n|na|nb|bp|nc|no|nd|ni|ne|nf|fm|py|nl|nn|nx|nt|nv|vc|vg" +
		"|vi|vm|l|ld|s|sa|sb|sc|dl|sd|s2|se|sh|si|sx|sr|s1|ss|m|mb|mf|mh|mi|il" +
		"|mo|o|ow|p|c|ch|cm|cp|cpf|c1|cs|g|gd|ge|gr|gh|gi|go|gp|gs|gu|gt|gl)$"
	p.AllowAttrs("class").Matching(regexp.MustCompile(codeSpanClassRegex)).OnElements("span")
	p.AllowAttrs("loading").Matching(regexp.MustCompile("^(lazy|eager)$")).OnElements("img")
	// image.SetSizes marks an image it gave width and height to, so the widget can size it by them
	p.AllowAttrs("class").Matching(regexp.MustCompile("^sized$")).OnElements("img")
	return p
}

// Sanitize clean dangerous html/js from the comment.
// Comment.Orig which is used to store the original comment text is not sanitized
// as we expect to never render it as HTML and render Comment.Text instead
func (c *Comment) Sanitize() {
	c.Text = sanitizeBalanced(c.Text)
	c.MarkLinksUGC()
	c.User.ID = template.HTMLEscapeString(c.User.ID)
	c.User.Name = c.SanitizeText(c.User.Name)
	c.User.Picture = c.SanitizeAsURL(c.User.Picture)
	c.Locator.URL = c.SanitizeAsURL(c.Locator.URL)
	c.PostTitle = c.SanitizeText(c.PostTitle)
	c.SanitizeEditSummary()
}

// sanitizeBalanced sanitizes comment HTML and makes its tags pair up. The sanitizer filters tags one at a
// time and does not match them, so its result can hold an end tag without its start tag, which closes an
// element of whatever the text is placed into. Text whose tags already pair up is kept byte for byte.
// Text that had to be balanced goes through the sanitizer once more, so the sanitizer has the last word
// on what is stored. Text that does not settle, or is nested deeper than the parser accepts, keeps only
// its words.
func sanitizeBalanced(text string) string {
	clean := commentPolicy.Sanitize(text)
	balanced, err := BalanceHTML(clean)
	if err == nil && sameTags(clean, balanced) {
		return clean
	}
	if err == nil {
		clean = commentPolicy.Sanitize(balanced)
		if balanced, err = BalanceHTML(clean); err == nil && sameTags(clean, balanced) {
			return clean
		}
	}
	return bluemonday.StrictPolicy().Sanitize(clean)
}

// sameTags reports whether two HTML texts hold the same start and end tags in the same order. Rendering
// parsed HTML can reorder attributes and rewrite <br> as <br/> with nothing added, dropped or moved, and
// then the text it was parsed from was balanced already.
func sameTags(a, b string) bool {
	tags := func(text string) (res []string, ok bool) {
		z := nethtml.NewTokenizer(strings.NewReader(text))
		for {
			switch z.Next() {
			case nethtml.ErrorToken:
				return res, errors.Is(z.Err(), io.EOF)
			case nethtml.StartTagToken, nethtml.SelfClosingTagToken:
				name, _ := z.TagName()
				res = append(res, string(name))
			case nethtml.EndTagToken:
				name, _ := z.TagName()
				res = append(res, "/"+string(name))
			}
		}
	}
	tagsA, okA := tags(a)
	tagsB, okB := tags(b)
	return okA && okB && slices.Equal(tagsA, tagsB)
}

// BalanceHTML parses an HTML fragment the way it will be parsed inside a <div> and renders it back,
// so end tags without a matching start tag are dropped and unclosed elements are closed.
// A tree built from misnested tags can be rendered to HTML which is parsed into another tree,
// e.g. <h1><div><h2><b><h3></b> gives <h3> inside <h2> and the next parser closes <h1> and <div> early.
// Because of this the result is returned only when parsing it again doesn't change it.
func BalanceHTML(fragment string) (string, error) {
	const maxPasses = 4 // two passes are enough for any well-formed input
	for range maxPasses {
		nodes, err := nethtml.ParseFragment(strings.NewReader(fragment),
			&nethtml.Node{Type: nethtml.ElementNode, Data: "div", DataAtom: atom.Div})
		if err != nil {
			return "", fmt.Errorf("can't parse html fragment: %w", err)
		}
		res := strings.Builder{}
		for _, n := range nodes {
			if err = nethtml.Render(&res, n); err != nil {
				return "", fmt.Errorf("can't render html fragment: %w", err)
			}
		}
		if res.String() == fragment {
			return fragment, nil
		}
		fragment = res.String()
	}
	return "", errors.New("html fragment is not stable")
}

// MarkLinksUGC makes the rel attribute of every <a> and <area> in the text hold both nofollow and ugc,
// telling search engines the link was written by a commenter and passes no ranking. bluemonday writes
// rel on every such link with an address and keeps the rel a commenter wrote on an <area>, appending
// nofollow only when the value does not already contain those letters, so rel="nofollowed" would stay
// followed. The text is tokenized because an attribute value such as id="item rel=" can look like the
// start of a rel attribute. Only the tags whose rel changes are written again, the same way bluemonday
// writes them, and every other token is copied as it is. Text stored by earlier releases came out of the
// same sanitizer, so the read path applies this too.
func (c *Comment) MarkLinksUGC() {
	if !strings.Contains(c.Text, "rel=") {
		return
	}
	z := nethtml.NewTokenizer(strings.NewReader(c.Text))
	var b strings.Builder
	for {
		tt := z.Next()
		if tt == nethtml.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return // keep the text as it is rather than write out part of it
			}
			break
		}
		if tt != nethtml.StartTagToken && tt != nethtml.SelfClosingTagToken {
			b.Write(z.Raw())
			continue
		}
		raw := string(z.Raw())
		tok := z.Token()
		if !markRelUGC(&tok) {
			b.WriteString(raw)
			continue
		}
		b.WriteString(tok.String())
	}
	c.Text = b.String()
}

// markRelUGC adds nofollow and ugc to the rel attribute of an <a> or <area> token, reporting whether
// the token changed
func markRelUGC(tok *nethtml.Token) bool {
	if tok.DataAtom != atom.A && tok.DataAtom != atom.Area {
		return false
	}
	changed := false
	for i, attr := range tok.Attr {
		if attr.Key != "rel" {
			continue
		}
		tokens := strings.Fields(attr.Val)
		for _, want := range []string{"nofollow", "ugc"} {
			if !slices.Contains(tokens, want) {
				tokens = append(tokens, want)
				changed = true
			}
		}
		tok.Attr[i].Val = strings.Join(tokens, " ")
	}
	return changed
}

// SanitizeEditSummary cleans the edit summary as plain text. It replaces the Edit record instead of
// changing it in place, as a comment handed out by an engine can share that record with the stored one.
func (c *Comment) SanitizeEditSummary() {
	if c.Edit == nil || c.Edit.Summary == "" {
		return
	}
	edit := *c.Edit
	edit.Summary = c.SanitizeText(edit.Summary)
	c.Edit = &edit
}

// Snippet returns the start of the comment's text as plain text, escaped to be put into HTML.
// Text is HTML and can't be shortened as it is: the cut may fall inside a tag or an attribute value,
// and the tags before the cut are not necessarily balanced. So the tags are dropped before the cut.
func (c *Comment) Snippet(limit int) string {
	if limit <= 0 {
		limit = snippetLen
	}
	cleanText := html.UnescapeString(bluemonday.StrictPolicy().Sanitize(spaceBlocks(c.Text)))
	cleanText = strings.Join(strings.Fields(cleanText), " ") // dropped tags leave runs of spaces and newlines
	size := len([]rune(cleanText))
	if size < limit {
		return html.EscapeString(cleanText)
	}
	snippet := []rune(cleanText)[:limit]
	// go back in snippet and found the first space
	for i, s := range slices.Backward(snippet) {
		if s == ' ' {
			snippet = snippet[:i]
			break
		}
	}
	// don't add a space if comment is just a one single word which has been truncated.
	if len(snippet) == limit {
		return html.EscapeString(string(snippet)) + "..."
	}
	return html.EscapeString(string(snippet)) + " ..."
}

// snippetBreaks are the elements whose start and end separate words: with their tags dropped,
// the last word of a paragraph would run into the first word of the next one
var snippetBreaks = map[atom.Atom]bool{
	atom.P: true, atom.Br: true, atom.Div: true, atom.Li: true, atom.Ul: true, atom.Ol: true, atom.Blockquote: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Pre: true, atom.Hr: true,
	atom.Table: true, atom.Tr: true, atom.Td: true, atom.Th: true, atom.Dl: true, atom.Dt: true, atom.Dd: true,
	atom.Section: true, atom.Article: true, atom.Aside: true, atom.Figure: true, atom.Figcaption: true,
	atom.Summary: true, atom.Details: true, atom.Hgroup: true, atom.Caption: true,
	atom.Header: true, atom.Footer: true, atom.Nav: true, atom.Main: true, atom.Address: true,
}

// spaceBlocks adds a space after each tag of the elements in snippetBreaks and copies everything else as it is
func spaceBlocks(text string) string {
	z := nethtml.NewTokenizer(strings.NewReader(text))
	var b strings.Builder
	for {
		tt := z.Next()
		if tt == nethtml.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return text // the tags are dropped anyway, only the spacing is lost
			}
			return b.String()
		}
		b.Write(z.Raw())
		if tt != nethtml.StartTagToken && tt != nethtml.EndTagToken && tt != nethtml.SelfClosingTagToken {
			continue
		}
		if name, _ := z.TagName(); snippetBreaks[atom.Lookup(name)] {
			b.WriteByte(' ')
		}
	}
}

var reHref = regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href="([^"]*)"`)

// SanitizeAsURL drops dangerous code from a url.
// It wraps input with href to trigger bluemonday sanitizer and cleans href after sanitizing done.
// White space around the url is dropped. Sanitizing the result again returns it unchanged, as a locator's url is
// sanitized on create and on every read: a url that a second pass would change is refused. So is a relative
// path that comes out starting with "//", which a browser reads as another host.
func (c *Comment) SanitizeAsURL(inp string) string {
	res := sanitizeURL(inp)
	if res == inp {
		return res
	}
	if sanitizeURL(res) != res || (strings.HasPrefix(res, "//") && !strings.HasPrefix(strings.TrimSpace(inp), "//")) {
		return ""
	}
	return res
}

// sanitizeURL makes one pass of the href filter over the url
func sanitizeURL(inp string) string {
	// quoted as html, not with %q: go escapes (\\, \u200b) don't exist in html and would end up in the url
	h := `<a href="` + html.EscapeString(inp) + `">`
	clean := urlPolicy.Sanitize(h)
	if match := reHref.FindStringSubmatch(clean); len(match) > 1 {
		// bluemonday trims spaces before parsing only, trim the ones left at the end after an empty fragment is dropped
		return strings.TrimSpace(html.UnescapeString(match[1]))
	}
	return "" // this shouldn't happen as we build the href
}

func (c *Comment) escapeHTMLWithSome(inp string) string {
	res := template.HTMLEscapeString(inp)
	res = strings.ReplaceAll(res, "&amp;", "&")
	res = strings.ReplaceAll(res, "&#34;", "\"")
	res = strings.ReplaceAll(res, "&#39;", "'")
	return res
}

// SanitizeText used to sanitize any input string, and removes any HTML tags
func (c *Comment) SanitizeText(inp string) string {
	clean := textPolicy.Sanitize(inp)
	return strings.TrimSpace(c.escapeHTMLWithSome(clean))
}
