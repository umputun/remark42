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
				Text: `blah <a href="javascript:alert('XSS1')" onmouseover="alert('XSS2')">XSS<a>` + "\n\t",
				User: User{ID: `<a href="http://blah.com">username</a>`},
			},
			out: Comment{
				Text: `blah XSS`,
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

func TestComment_HashUserFields(t *testing.T) {
	tbl := []struct {
		inp Comment
		out Comment
	}{
		{inp: Comment{}, out: Comment{}},
		{
			inp: Comment{
				Text: "blah",
				User: User{ID: "my id", IP: "127.0.0.1"},
			},
			out: Comment{
				Text: "blah",
				User: User{ID: "de58071dda71e1783b6deb931ddb48bb66966f79", IP: "4b84b15bff6ee5796152495a230e45e3d7e947d9"},
			},
		},
	}

	for n, tt := range tbl {
		tt.inp.HashUserFields()
		assert.Equal(t, tt.out, tt.inp, "check #%d", n)
	}
}
