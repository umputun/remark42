package store

import (
	"strconv"
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
				User: User{ID: `<a href="http://blah.com">username</a>`, Name: "name <b/>"},
			},
			out: Comment{
				Text: "blah XSS\n\t",
				User: User{ID: `&lt;a href=&#34;http://blah.com&#34;&gt;username&lt;/a&gt;`, Name: "name &lt;b/&gt;"},
			},
		},
		{
			inp: Comment{
				Text: "blah 123" + "\n\t",
				User: User{ID: "id", Name: "xyz-123"},
			},
			out: Comment{
				Text: `blah 123` + "\n\t",
				User: User{ID: "id", Name: "xyz-123"},
			},
		},
		{
			inp: Comment{Text: "blah & & 123 &mdash; &mdash;"},
			out: Comment{Text: `blah &amp; &amp; 123 — —`},
		},
		{
			inp: Comment{Text: "blah & & 123 — —"},
			out: Comment{Text: `blah &amp; &amp; 123 — —`},
		},
		{
			inp: Comment{Text: "blah & & 123", User: User{Name: "name <> & ' ` \""}},
			out: Comment{Text: `blah &amp; &amp; 123`, User: User{Name: "name &lt;&gt; & ' ` \""}},
		},

		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "javascript:alert('XSS1')"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: ""}},
		},
		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "javascript:alert(document.domain)//"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: ""}},
		},
		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "<script>alert()</script>"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: "%3Cscript%3Ealert%28%29%3C/script%3E"}},
		},
		{
			inp: Comment{Text: "blah blah",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
			out: Comment{Text: "blah blah",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
		},
		{
			inp: Comment{Text: "<scrİpt>&lt;img src=x onerror=alert(1)&gt;",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
			out: Comment{Text: "&lt;img src=x onerror=alert(1)&gt;",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
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
	assert.Equal(t, make(map[string]VotedIPInfo), comment.VotedIPs)
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
	assert.Equal(t, map[string]VotedIPInfo{}, comment.VotedIPs)
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
	assert.Equal(t, map[string]VotedIPInfo{}, comment.VotedIPs)
	assert.Equal(t, 0, comment.Score)
	assert.True(t, comment.Deleted)
	assert.Nil(t, comment.Edit)
	assert.False(t, comment.Pin)
	assert.Equal(t, User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comment.User)
}

func TestComment_Snippet(t *testing.T) {
	tbl := []struct {
		limit int
		inp   string
		out   string
	}{
		{0, "", ""},
		{-1, "test\nblah", "test blah"},
		{5, "test\nblah", "test ..."},
		{5, "xyz12345 xxx", "xyz12345 ..."},
		{10, "xyz12345 xxx\ntest 123456", "xyz12345 xxx test ..."},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c := Comment{Text: tt.inp}
			out := c.Snippet(tt.limit)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestComment_sanitizeAsURL(t *testing.T) {

	tbl := []struct {
		inp, out string
	}{
		{
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"javascript:alert(document.domain)//",
			"",
		},
		{
			"<script>alert()</script>",
			"%3Cscript%3Ealert%28%29%3C/script%3E",
		},
		{
			"<a href=javascript:alert(document.domain)//>xxx</a>",
			"",
		},
	}

	for i, tt := range tbl {
		tt := tt
		c := Comment{}
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, c.SanitizeAsURL(tt.inp))
		})
	}

}
