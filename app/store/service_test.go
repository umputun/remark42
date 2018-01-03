package store

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Vote(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes)

	c, err := b.Vote(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, map[string]bool{"user1": true}, c.Votes)

	_, err = b.Vote(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.NotNil(t, err, "double-voting rejected")

	res, err = b.Last(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 1, res[0].Score)
}

func TestBoltDB_Pin(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, res[0].Pin)

	err = b.SetPin(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, true)
	assert.Nil(t, err)

	c, err := b.GetByID(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, true, c.Pin)

	err = b.SetPin(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, false)
	assert.Nil(t, err)
	c, err = b.GetByID(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, false, c.Pin)
}

func TestBoltDB_EditComment(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, Edit{}, res[0].Edit)

	comment, err := b.EditComment(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "xxx", Edit{Summary: "my edit"})
	assert.Nil(t, err)
	assert.Equal(t, "my edit", comment.Edit.Summary)
	assert.Equal(t, "xxx", comment.Text)

	c, err := b.GetByID(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, "my edit", c.Edit.Summary)
	assert.Equal(t, "xxx", c.Text)
}
