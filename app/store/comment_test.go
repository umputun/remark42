package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComment_Sanitize(t *testing.T) {

	tbl := []struct {
		inp Comment
		out Comment
	}{
		{inp: Comment{}, out: Comment{}},
		{
			inp: Comment{
				Text: `blah <a href="javascript:alert('XSS1')" onmouseover="alert('XSS2')">XSS</a>` + "\n\t",
				User: User{ID: `<a href="http://blah.com">username</a>`},
			},
			out: Comment{
				Text: "blah XSS\n\t",
				User: User{ID: `&lt;a href=&#34;http://blah.com&#34;&gt;username&lt;/a&gt;`},
			},
		},
		{
			inp: Comment{
				Text: `blah <a href="https://www.reddit.com/r/golang/comments/8jdo2l/remark42_is_a_selfhosted_lightweight_and_simple/">https://www.reddit.com/r/golang/comments/8jdo2l/remark42_is_a_selfhosted_lightweight_and_simple/</a>` + "\n\t",
				User: User{ID: `<a href="http://blah.com">username</a>`},
			},
			out: Comment{
				Text: `blah <a href="https://www.reddit.com/r/golang/comments/8jdo2l/remark42_is_a_selfhosted_lightweight_and_simple/" rel="nofollow">https://www.reddit.com/r/golang/comments/8jdo...</a>` + "\n\t",
				User: User{ID: `&lt;a href=&#34;http://blah.com&#34;&gt;username&lt;/a&gt;`},
			},
		},
	}

	for n, tt := range tbl {
		tt.inp.Sanitize()
		assert.Equal(t, tt.out, tt.inp, "check #%d", n)
	}
}

func TestComment_PrepareUntrusted(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "username"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Pin:       true,
		Deleted:   true,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.Local),
		Votes:     map[string]bool{"uu": true},
	}

	comment.PrepareUntrusted()
	assert.Equal(t, "", comment.ID)
	assert.Equal(t, "p123", comment.ParentID)
	assert.Equal(t, "blah", comment.Text)
	assert.Equal(t, 0, comment.Score)
	assert.Equal(t, false, comment.Pin)
	assert.Equal(t, time.Time{}, comment.Timestamp)
	assert.Equal(t, false, comment.Deleted)
	assert.Equal(t, make(map[string]bool), comment.Votes)
	assert.Equal(t, User{ID: "username"}, comment.User)

}

func TestComment_SetDeleted(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "userid", Name: "username", IP: "123", Picture: "pic"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Deleted:   false,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.Local),
		Votes:     map[string]bool{"uu": true},
		Pin:       true,
	}

	comment.SetDeleted(SoftDelete)

	assert.Equal(t, "", comment.Text)
	assert.Equal(t, "", comment.Orig)
	assert.Equal(t, map[string]bool{}, comment.Votes)
	assert.Equal(t, 0, comment.Score)
	assert.True(t, comment.Deleted)
	assert.Nil(t, comment.Edit)
	assert.False(t, comment.Pin)
	assert.Equal(t, User{Name: "username", ID: "userid", Picture: "pic", Admin: false, Blocked: false, IP: "123"}, comment.User)
}

func TestComment_SetDeletedHard(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "userid", Name: "username", IP: "123", Picture: "pic"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Deleted:   false,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.Local),
		Votes:     map[string]bool{"uu": true},
		Pin:       true,
	}

	comment.SetDeleted(HardDelete)

	assert.Equal(t, "", comment.Text)
	assert.Equal(t, "", comment.Orig)
	assert.Equal(t, map[string]bool{}, comment.Votes)
	assert.Equal(t, 0, comment.Score)
	assert.True(t, comment.Deleted)
	assert.Nil(t, comment.Edit)
	assert.False(t, comment.Pin)
	assert.Equal(t, User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comment.User)
}

func TestComment_ShortenAutoLinks(t *testing.T) {
	tbl := []struct {
		max     int
		in, out string
	}{
		{32, "", ""},
		{32, "text", "text"},
		{32, "<p>asd</p>", "<p>asd</p>"},
		{5, `<a href="incorrect-url">incorrect-url</a>`, `<a href="incorrect-url">incorrect-url</a>`},
		{32, `<a href="https://blah.com">some text, not href</a>`, `<a href="https://blah.com">some text, not href</a>`},
		{
			32,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
		},
		{
			31,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=1...</a>`,
		},
		{
			15,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com...</a>`,
		},
		{
			3,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com...</a>`,
		},
		{
			-1,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
		},
	}

	for n, tt := range tbl {
		got := shortenAutoLinks(tt.in, tt.max)
		assert.Equalf(t, tt.out, got, "check #%d", n)
	}
}
