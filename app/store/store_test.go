package store

import "testing"
import "github.com/stretchr/testify/assert"

func TestStore_MakeCommentID(t *testing.T) {
	cid1 := makeCommentID()
	assert.True(t, len(cid1) > 8, "cid1 is long enough")

	cid2 := makeCommentID()
	assert.True(t, len(cid2) > 8, "cid2 is long enough")

	assert.NotEqual(t, cid1, cid2, "cids different")
}

func TestStore_SanitizeComment(t *testing.T) {

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
		out := sanitizeComment(tt.inp)
		assert.Equal(t, tt.out, out, "check #%d", n)
	}
}
