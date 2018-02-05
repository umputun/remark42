package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComment_GenID(t *testing.T) {
	c1 := Comment{}
	assert.Nil(t, c1.GenID())
	assert.True(t, len(c1.ID) > 8, "cid1 is long enough")

	c2 := Comment{}
	assert.Nil(t, c2.GenID())
	assert.True(t, len(c2.ID) > 8, "cid2 is long enough")

	assert.NotEqual(t, c1.ID, c2.ID, "cids different")
}

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
