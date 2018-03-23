package store

import (
	"os"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"
)

func TestService_Vote(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes, "no votes initially")

	c, err := b.Vote(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, map[string]bool{"user1": true}, c.Votes, "user voted +")

	_, err = b.Vote(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.NotNil(t, err, "double-voting rejected")

	res, err = b.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 1, res[0].Score)

	_, err = b.Vote(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", false)
	assert.Nil(t, err, "vote reset")
	res, err = b.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes, "vote reset ok")
}

func TestService_Pin(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, res[0].Pin)

	err = b.SetPin(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, true)
	assert.Nil(t, err)

	c, err := b.Get(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, true, c.Pin)

	err = b.SetPin(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, false)
	assert.Nil(t, err)
	c, err = b.Get(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, false, c.Pin)
}

func TestService_EditComment(t *testing.T) {
	defer os.Remove(testDb)
	b := Service{Interface: prep(t)}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	comment, err := b.EditComment(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "xxx", Edit{Summary: "my edit"})
	assert.Nil(t, err)
	assert.Equal(t, "my edit", comment.Edit.Summary)
	assert.Equal(t, "xxx", comment.Text)

	c, err := b.Get(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, "my edit", c.Edit.Summary)
	assert.Equal(t, "xxx", c.Text)

	_, err = b.EditComment(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "xxx", Edit{Summary: "my edit"})
	assert.NotNil(t, err, "allow edit once")
}

func TestService_EditCommentDurationFailed(t *testing.T) {
	defer os.Remove(testDb)

	blt, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: "/tmp/test-remark.db", SiteID: "radio-t"})
	assert.Nil(t, err)

	comment := Comment{
		ID:      "id-1",
		Text:    `some text, <a href="http://radio-t.com">link</a>`,
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:    User{ID: "user1", Name: "user name"},
	}
	_, err = blt.Create(comment)
	assert.Nil(t, err)

	b := Service{Interface: blt, EditDuration: 100 * time.Millisecond}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Nil(t, res[0].Edit)

	time.Sleep(time.Second)

	comment, err = b.EditComment(Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "xxx",
		Edit{Summary: "my edit"})
	assert.NotNil(t, err)
}

func TestService_Counts(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t) // two comments for https://radio-t.com

	// add one more for https://radio-t.com/2
	comment := Comment{
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.Nil(t, err)

	svc := Service{Interface: b}
	res, err := svc.Counts("radio-t", []string{"https://radio-t.com/2"})
	assert.Nil(t, err)
	assert.Equal(t, []PostInfo{{URL: "https://radio-t.com/2", Count: 1}}, res)

	res, err = svc.Counts("radio-t", []string{"https://radio-t.com", "https://radio-t.com/2", "blah"})
	assert.Nil(t, err)
	assert.Equal(t, []PostInfo{
		{URL: "https://radio-t.com", Count: 2},
		{URL: "https://radio-t.com/2", Count: 1},
		{URL: "blah", Count: 0},
	}, res)
}
