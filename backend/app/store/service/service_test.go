package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
)

func TestService_CreateFromEmpty(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks}
	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	beforeCreate := time.Now()
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "text", res.Text)
	assert.WithinRange(t, res.Timestamp, beforeCreate, time.Now(), "timestamp set during create")
	assert.Equal(t, "user", res.User.ID)
	assert.Equal(t, "name", res.User.Name)
	assert.Equal(t, "23f97cf4d5c29ef788ca2bdd1c9e75656c0e4149", res.User.IP)
	assert.Equal(t, map[string]bool(nil), res.Votes)
	assert.Equal(t, map[string]store.VotedIPInfo(nil), res.VotedIPs)
}

func TestService_CreateSiteDisabled(t *testing.T) {
	ks := admin.NewStaticStore("secret 123", []string{"xxx"}, nil, "email")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks}
	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.EqualError(t, err, "failed to prepare comment: can't get secret for site radio-t: site radio-t disabled")
}

func TestService_CreateFromPartial(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks}
	comment := store.Comment{
		Text:      "text",
		Timestamp: time.Date(2018, 3, 25, 16, 34, 33, 0, time.UTC),
		Votes:     map[string]bool{"u1": true, "u2": false},
		VotedIPs: map[string]store.VotedIPInfo{"xxx": {Value: true, Timestamp: time.Now()},
			"yyy": {Value: false, Timestamp: time.Now()}},
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "text", res.Text)
	assert.Equal(t, comment.Timestamp, res.Timestamp)
	assert.Equal(t, "user", res.User.ID)
	assert.Equal(t, "name", res.User.Name)
	assert.Equal(t, "23f97cf4d5c29ef788ca2bdd1c9e75656c0e4149", res.User.IP)
	assert.Equal(t, "", res.PostTitle)
	assert.Equal(t, comment.Votes, res.Votes)
}

func TestService_CreateWithQuotesInTitle(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks}
	comment := store.Comment{
		Text:      "text",
		PostTitle: "some example \"in quotes\"",
		Timestamp: time.Date(2018, 3, 25, 16, 34, 33, 0, time.UTC),
		Votes:     map[string]bool{"u1": true, "u2": false},
		VotedIPs: map[string]store.VotedIPInfo{"xxx": {Value: true, Timestamp: time.Now()},
			"yyy": {Value: false, Timestamp: time.Now()}},
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "text", res.Text)
	assert.Equal(t, comment.Timestamp, res.Timestamp)
	assert.Equal(t, "user", res.User.ID)
	assert.Equal(t, "name", res.User.Name)
	assert.Equal(t, "23f97cf4d5c29ef788ca2bdd1c9e75656c0e4149", res.User.IP)
	assert.Equal(t, "some example \"in quotes\"", res.PostTitle)
}

func TestService_CreateFromPartialWithTitle(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})}
	defer b.Close()

	postPath := "/post/42"
	postTitle := "Post Title 42"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == postPath {
			_, err := fmt.Fprintf(w, "<html><title>%s</title><body>...</body></html>", postTitle)
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	comment := store.Comment{
		Text:      "text",
		Timestamp: time.Date(2018, 3, 25, 16, 34, 33, 0, time.UTC),
		Votes:     map[string]bool{"u1": true, "u2": false},
		User:      store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator:   store.Locator{URL: ts.URL + postPath, SiteID: "radio-t"},
	}
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: ts.URL + postPath, SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, postTitle, res.PostTitle)

	comment.PostTitle = "post blah"
	id, err = b.Create(comment)
	assert.NoError(t, err)
	res, err = b.Engine.Get(getReq(store.Locator{URL: ts.URL + postPath, SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "post blah", res.PostTitle, "keep comment title")
}

func TestService_Put(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks}

	comment := store.Comment{
		ID:        "c-1",
		ParentID:  "id-1",
		Text:      "test text",
		User:      store.User{ID: "user2", Name: "user name 2"},
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
	}
	_, err := b.Create(comment)
	require.NoError(t, err)

	// create new comment with everything different to replace the first one with fields below
	updatedComment := store.Comment{
		ID:        "c-1",
		ParentID:  "id-new",
		Text:      "new text",
		User:      store.User{ID: "user3", Name: "user name 3"},
		Locator:   store.Locator{URL: "https://example.com", SiteID: "example"},
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
	}

	err = b.Put(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, updatedComment)
	require.NoError(t, err)

	// request with wrong user, should not affect the comment user
	got, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "c-1", store.User{ID: "user1", Name: "user name 1"})
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.ID)
	assert.Equal(t, "new text", got.Text)
	assert.Equal(t, "id-1", got.ParentID, "should be unaltered")
	assert.Equal(t, "user2", got.User.ID, "should be unaltered")
	assert.Equal(t, "user name 2", got.User.Name, "should be unaltered")
	assert.Equal(t, "https://radio-t.com", got.Locator.URL, "should be unaltered")
	assert.Equal(t, "radio-t", got.Locator.SiteID, "should be unaltered")
	assert.Equal(t, time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC), got.Timestamp, "should be unaltered")

}

func TestService_SetTitle(t *testing.T) {
	var titleEnable atomic.Int32
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if titleEnable.Load() == 0 {
			w.WriteHeader(404)
		}
		if r.URL.String() == "/post1" {
			_, err := w.Write([]byte("<html><title>post1 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		if r.URL.String() == "/post2" {
			_, err := w.Write([]byte("<html><title>post2 blah 123</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer tss.Close()

	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})}
	defer b.Close()
	comment := store.Comment{
		Text:      "text",
		Timestamp: time.Date(2018, 3, 25, 16, 34, 33, 0, time.UTC),
		Votes:     map[string]bool{"u1": true, "u2": false},
		User:      store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator:   store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"},
	}

	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "", res.PostTitle)

	b.TitleExtractor.cache.Purge()

	titleEnable.Store(1)
	c, err := b.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.NoError(t, err)
	assert.Equal(t, "post1 blah 123", c.PostTitle)

	bErr := DataStore{Engine: eng, AdminStore: ks}
	defer bErr.Close()
	_, err = bErr.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.EqualError(t, err, "no title extractor")
}

func TestService_CreateOnUnreadablePage(t *testing.T) {
	var hits atomic.Int32
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tss.Close()

	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"),
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})}
	defer b.Close()

	locator := store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}
	for i := range 3 {
		id, err := b.Create(store.Comment{Text: fmt.Sprintf("text %d", i), Locator: locator,
			User: store.User{IP: "192.168.1.1", ID: "user", Name: "name"}})
		require.NoError(t, err, "comment %d stored although the page can't be read", i)
		res, err := b.Engine.Get(getReq(locator, id))
		require.NoError(t, err)
		assert.Empty(t, res.PostTitle)
	}
	assert.Equal(t, int32(1), hits.Load(), "page requested once, the failure is cached")
}

func TestService_Vote(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, 0, res[0].Vote)
	assert.Equal(t, map[string]bool(nil), res[0].Votes, "no votes initially")

	// vote +1 as user1
	req := VoteReq{
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID,
		UserID:    "user1",
		UserIP:    "123",
		Val:       true,
	}
	c, err := b.Vote(req)
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, 1, c.Vote)
	assert.Equal(t, map[string]bool{"user1": true}, c.Votes, "user voted +")
	// check result as user1
	c, err = b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, store.User{ID: "user1"})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, 1, c.Vote, "can see own vote result")
	assert.Nil(t, c.Votes)
	assert.Nil(t, c.VotedIPs)
	// check result as user2
	c, err = b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, store.User{ID: "user2"})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, 0, c.Vote, "can't see other user vote result")
	assert.Nil(t, c.Votes)
	assert.Nil(t, c.VotedIPs)

	req = VoteReq{
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID,
		UserID:    "user",
		UserIP:    "123",
		Val:       true,
	}
	c, err = b.Vote(req)
	assert.Error(t, err, "self-voting not allowed")

	req = VoteReq{
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID,
		UserID:    "user1",
		UserIP:    "123",
		Val:       true,
	}
	_, err = b.Vote(req)
	assert.Error(t, err, "double-voting rejected")
	assert.True(t, strings.HasPrefix(err.Error(), "user user1 already voted"))

	// check in last as user1
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{ID: "user1"})
	assert.NoError(t, err)
	t.Logf("%+v", res[0])
	require.Equal(t, 3, len(res))
	assert.Equal(t, 1, res[0].Score)
	assert.Equal(t, 1, res[0].Vote)
	assert.Equal(t, 0.0, res[0].Controversy)

	// check in last as user2
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{ID: "user2"})
	assert.NoError(t, err)
	t.Logf("%+v", res[0])
	require.Equal(t, 3, len(res))
	assert.Equal(t, 1, res[0].Score)
	assert.Equal(t, 0, res[0].Vote)
	assert.Equal(t, 0.0, res[0].Controversy)

	req = VoteReq{
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID,
		UserID:    "user1",
		UserIP:    "123",
		Val:       false,
	}
	_, err = b.Vote(req)
	assert.NoError(t, err, "vote reset")
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	assert.NoError(t, err)
	require.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, 0, res[0].Vote)
	assert.Equal(t, map[string]bool(nil), res[0].Votes, "vote reset ok")
	assert.Equal(t, map[string]store.VotedIPInfo(nil), res[0].VotedIPs, "vote reset ok")
}

func TestService_VoteLimit(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: 2}

	_, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user2", Val: true})
	assert.NoError(t, err)

	_, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user3", Val: true})
	assert.NoError(t, err)

	_, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user4", Val: true})
	assert.Error(t, err, "vote limit reached")
	assert.True(t, strings.HasPrefix(err.Error(), "maximum number of votes exceeded for comment id-1"))

	_, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user4", Val: true})
	assert.NoError(t, err)
}

func TestService_VotesDisabled(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: 0}

	_, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user2", Val: true})
	assert.EqualError(t, err, "maximum number of votes exceeded for comment id-1")
}

func TestService_VoteAggressive(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	t.Logf("%+v", res[0])
	require.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool(nil), res[0].Votes, "no votes initially")

	// add a vote as user2
	_, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
		UserID: "user2", Val: true})
	require.NoError(t, err)

	// crazy vote +1 as user1
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: "user1", Val: true})
		})
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{ID: "user1"})
	require.NoError(t, err)

	t.Logf("%+v", res[0])
	require.Equal(t, 3, len(res))
	assert.Equal(t, 2, res[0].Score, "add single +1")
	assert.Equal(t, 1, res[0].Vote, "user1 voted +1")
	assert.Equal(t, 0, len(res[0].Votes), "votes hidden")
	assert.Equal(t, 0, len(res[0].VotedIPs), "vote ips hidden")

	// random +1/-1 result should be [0..2]
	for range 100 {
		wg.Go(func() {
			val := rand.Intn(2) > 0
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: "user1", Val: val})
		})
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	require.Equal(t, 3, len(res))
	t.Logf("%+v %d", res[0], res[0].Score)
	assert.True(t, res[0].Score >= 0 && res[0].Score <= 2, "unexpected score %d", res[0].Score)
}

func TestService_VoteConcurrent(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)
	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)

	// concurrent vote +1 as multiple users for the same comment
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: fmt.Sprintf("user1-%d", i), Val: true})
		})
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Equal(t, 100, res[0].Score, "should have 100 score")
	assert.Equal(t, 0, len(res[0].Votes), "should hide votes")
	assert.Equal(t, 0, len(res[0].VotedIPs), "should hide vote ips")
	assert.Equal(t, 0.0, res[0].Controversy, "should have 0 controversy")
}

func TestService_VotePositive(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"),
		MaxVotes: -1, PositiveScore: true} // allow positive voting only

	_, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user2", Val: false})
	assert.EqualError(t, err, "minimal score reached for comment id-1")

	c, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user3", Val: true})
	assert.NoError(t, err, "minimal score doesn't affect positive vote")
	assert.Equal(t, 1, c.Score)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user2", Val: true})
	assert.NoError(t, err, "vote set to +2")
	assert.Equal(t, 2, c.Score)

	// check +, -, -
	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user5", Val: true})
	assert.NoError(t, err, "user5 +1, score 3")
	assert.Equal(t, 3, c.Score)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user5", Val: false})
	assert.NoError(t, err, "user5 -1, score reset to 2")
	assert.Equal(t, 2, c.Score)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user5", Val: false})
	assert.NoError(t, err, "user5 -1, score 1")
	assert.Equal(t, 1, c.Score)

	// allow negative voting
	b.PositiveScore = false
	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user2", Val: false})
	assert.NoError(t, err, "minimal score ignored")
	assert.Equal(t, 0, c.Score)
	assert.Equal(t, 2.0, c.Controversy)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-1",
		UserID: "user4", Val: false})
	assert.NoError(t, err, "minimal score ignored")
	assert.Equal(t, -1, c.Score)
}

func TestService_VoteControversy(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	c, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user2", Val: false})
	assert.NoError(t, err)
	assert.Equal(t, -1, c.Score, "should have -1 score")
	assert.InDelta(t, 0.00, c.Controversy, 0.01)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user3", Val: true})
	assert.NoError(t, err)
	assert.Equal(t, 0, c.Score, "should have 0 score")
	assert.InDelta(t, 2.00, c.Controversy, 0.01)

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user4", Val: true})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score, "should have 1 score")
	assert.InDelta(t, 1.73, c.Controversy, 0.01)

	// check if stored
	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Equal(t, 1, res[0].Score, "should have 1 score")
	assert.InDelta(t, 1.73, res[0].Controversy, 0.01)
}

func TestService_RestrictedWords(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	restictedWordLister := StaticRestrictedWordsLister{Words: []string{"restricted"}}
	b := DataStore{Engine: eng, AdminStore: ks, RestrictedWordsMatcher: NewRestrictedWordsMatcher(restictedWordLister)}

	// test creating a comment with restricted words which should fail with appropriate error
	reply := store.Comment{
		ID:        "c-1",
		ParentID:  "id-1",
		Text:      "restricted word",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}
	id, err := b.Create(reply)
	assert.EqualError(t, err, ErrRestrictedWordsFound.Error(), "should fail with RestrictedWordError")
	assert.Empty(t, id)

	// an import brings back comments the site already had, whatever words are restricted now
	reply.Imported = true
	id, err = b.Create(reply)
	require.NoError(t, err)
	assert.Equal(t, "c-1", id)
}

func TestDataStore_AdminStoreErrors(t *testing.T) {
	badKey := true
	badEnabled := true
	as := admin.StoreMock{
		OnEventFunc: func(string, admin.EventType) error { return errors.New("err") },
		KeyFunc: func(string) (string, error) {
			if badKey {
				return "", errors.New("mock key err")
			}
			return "secret", nil
		},
		EnabledFunc: func(string) (bool, error) {
			if badEnabled {
				return false, errors.New("mock enabled err")
			}
			return true, nil
		},
		AdminsFunc: func(string) ([]string, error) { return nil, errors.New("mock admins err") },
	}
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: &as, MaxVotes: -1}
	comment := store.Comment{
		ID:        "c-1",
		ParentID:  "id-1",
		Text:      "restricted word",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}

	// key call error
	id, err := b.Create(comment)
	assert.ErrorContainsf(t, err, "mock key err", "should fail with mock error")
	assert.Empty(t, id)
	assert.Equal(t, len(as.KeyCalls()), 1)
	assert.Equal(t, len(as.EnabledCalls()), 0)
	assert.Equal(t, len(as.OnEventCalls()), 0)

	// enabled call error
	badKey = false
	id, err = b.Create(comment)
	assert.ErrorContains(t, err, "mock enabled err", "should fail with mock error")
	assert.Empty(t, id)
	assert.Equal(t, len(as.KeyCalls()), 2)
	assert.Equal(t, len(as.EnabledCalls()), 1)
	assert.Equal(t, len(as.OnEventCalls()), 0)

	// only OnEvent error
	badEnabled = false
	id, err = b.Create(comment)
	assert.NoError(t, err, "OnEvent error should be just logged")
	assert.Equal(t, id, "c-1")
	assert.Equal(t, len(as.KeyCalls()), 3)
	assert.Equal(t, len(as.EnabledCalls()), 2)
	assert.Equal(t, len(as.OnEventCalls()), 1)

	// OnEvent error on Vote call
	_, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "c-1",
		UserID: "user4", Val: true})
	assert.NoError(t, err, "OnEvent error should be just logged")
	assert.Equal(t, len(as.KeyCalls()), 4)
	assert.Equal(t, len(as.EnabledCalls()), 3)
	assert.Equal(t, len(as.OnEventCalls()), 2)

	// admins error
	isAdmin := b.IsAdmin("radio-t", "user2")
	assert.False(t, isAdmin)
	assert.Equal(t, len(as.AdminsCalls()), 1)
	assert.Equal(t, len(as.OnEventCalls()), 2)

	// OnEvent error on EditComment call
	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "c-1", EditRequest{Text: "new text"})
	assert.NoError(t, err, "OnEvent error should be just logged")

	// OnEvent error on Delete call
	err = b.Delete(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "c-1", store.SoftDelete)
	assert.NoError(t, err, "OnEvent error should be just logged")

	// OnEvent error on EditComment Delete call
	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "c-1", EditRequest{Delete: true})
	assert.NoError(t, err, "OnEvent error should be just logged")
}

func TestService_VoteSameIP(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}
	b.RestrictSameIPVotes.Enabled = true

	c, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user2", UserIP: "123", Val: true})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score, "should have 1 score")

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user3", UserIP: "123", Val: true})
	assert.EqualError(t, err, "the same ip already voted for id-2")
	assert.Equal(t, 1, c.Score, "still have 1 score, rejected")

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user2", UserIP: "123", Val: false})
	assert.NoError(t, err)
	assert.Equal(t, 0, c.Score, "reset to 0 score, opposite vote allowed")

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user2", UserIP: "123", Val: false})
	assert.NoError(t, err)
	assert.Equal(t, -1, c.Score, "set to -1 score, correction vote allowed")
}

func TestService_VoteSameIPWithDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		eng, teardown := prepStoreEngine(t)
		defer teardown()
		b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"),
			MaxVotes: -1}
		b.RestrictSameIPVotes.Enabled = true
		b.RestrictSameIPVotes.Duration = 500 * time.Millisecond

		c, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
			UserID: "user2", UserIP: "123", Val: true})
		assert.NoError(t, err)
		assert.Equal(t, 1, c.Score, "should have 1 score")

		c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
			UserID: "user3", UserIP: "123", Val: true})
		assert.EqualError(t, err, "the same ip already voted for id-2")
		assert.Equal(t, 1, c.Score, "still have 1 score")

		c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
			UserID: "user4", UserIP: "12345", Val: true})
		assert.NoError(t, err)
		assert.Equal(t, 2, c.Score, "have 2 score")

		time.Sleep(501 * time.Millisecond)

		c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
			UserID: "user3", UserIP: "123", Val: true})
		assert.NoError(t, err)
		assert.Equal(t, 3, c.Score, "have 3 score")
	})
}

func TestService_VotePrunesExpiredIPs(t *testing.T) {
	const secret = "secret 123"
	locator := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	vote := func(b *DataStore, userID, ip string) error {
		_, err := b.Vote(VoteReq{Locator: locator, CommentID: "id-2", UserID: userID, UserIP: ip, Val: true})
		return err
	}
	// the service hides VotedIPs from its callers, so the stored comment is read from the engine
	storedIPs := func(t *testing.T, eng engine.Interface) map[string]store.VotedIPInfo {
		c, err := eng.Get(getReq(locator, "id-2"))
		require.NoError(t, err)
		return c.VotedIPs
	}
	oldIP, freshIP, lastIP := store.HashValue("10.0.0.1", secret), store.HashValue("10.0.0.2", secret), store.HashValue("10.0.0.3", secret)

	tbl := []struct {
		name     string
		enabled  bool
		duration time.Duration
		pruned   bool
	}{
		{name: "restriction off", enabled: false, duration: 5 * time.Minute, pruned: true},
		{name: "restriction on", enabled: true, duration: 5 * time.Minute, pruned: true},
		{name: "restriction off, unlimited window", enabled: false, duration: 0, pruned: false},
		{name: "restriction on, unlimited window", enabled: true, duration: 0, pruned: false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				eng, teardown := prepStoreEngine(t)
				defer teardown()
				b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore(secret), MaxVotes: -1}
				b.RestrictSameIPVotes.Enabled = tt.enabled
				b.RestrictSameIPVotes.Duration = tt.duration

				require.NoError(t, vote(&b, "user2", "10.0.0.1"))
				time.Sleep(3 * time.Minute)
				require.NoError(t, vote(&b, "user3", "10.0.0.2"))
				assert.Len(t, storedIPs(t, eng), 2, "both votes are inside the window, nothing to drop")

				time.Sleep(3 * time.Minute) // the first vote is 6 minutes old now, the second one 3 minutes
				require.NoError(t, vote(&b, "user4", "10.0.0.3"))

				ips := storedIPs(t, eng)
				assert.Contains(t, ips, freshIP, "ip of a vote inside the window stays")
				assert.Contains(t, ips, lastIP, "ip of the vote just made is recorded")
				if tt.pruned {
					assert.NotContains(t, ips, oldIP, "ip of a vote older than the window is dropped by the next vote")
				} else {
					assert.Contains(t, ips, oldIP, "zero duration never expires, so nothing is dropped")
				}

				// the same-ip limit keeps working on what is left
				err := vote(&b, "user5", "10.0.0.2")
				if tt.enabled {
					assert.EqualError(t, err, "the same ip already voted for id-2", "ip inside the window is still limited")
				} else {
					assert.NoError(t, err, "no limit with the restriction off")
				}
				err = vote(&b, "user6", "10.0.0.1")
				if tt.enabled && !tt.pruned {
					assert.EqualError(t, err, "the same ip already voted for id-2", "unlimited window still limits the old ip")
				} else {
					assert.NoError(t, err, "ip outside the window can vote again")
				}
			})
		})
	}
}

func TestService_Controversy(t *testing.T) {
	tbl := []struct {
		ups, downs int
		res        float64
	}{
		{0, 0, 0},
		{10, 5, 3.87},
		{20, 5, 2.24},
		{20, 50, 5.47},
		{20, 0, 0},
		{1100, 500, 28.60},
		{1100, 12100, 2.37},
		{100, 100, 200},
		{101, 101, 202},
	}

	b := DataStore{}
	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d-%d:%d", i, tt.ups, tt.downs), func(t *testing.T) {
			assert.InDelta(t, tt.res, b.controversy(tt.ups, tt.downs), 0.01)
		})
	}
}

func TestService_Pin(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, false, res[0].Pin)

	err = b.SetPin(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, true)
	assert.NoError(t, err)

	c, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID))
	assert.NoError(t, err)
	assert.Equal(t, true, c.Pin)

	err = b.SetPin(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, false)
	assert.NoError(t, err)
	c, err = b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID))
	assert.NoError(t, err)
	assert.Equal(t, false, c.Pin)
}

func TestService_EditComment(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}
	defer b.Close()

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	comment, err := b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.NoError(t, err)
	assert.Equal(t, "my edit", comment.Edit.Summary)
	assert.Equal(t, "xxx", comment.Text)
	assert.Equal(t, "yyy", comment.Orig)

	c, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID))
	assert.NoError(t, err)
	assert.Equal(t, "my edit", c.Edit.Summary)
	assert.Equal(t, "xxx", c.Text)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.NoError(t, err, "allow second edit")
}

func TestService_EditCommentSanitizesSummary(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}
	defer b.Close()

	locator := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	comment, err := b.EditComment(locator, "id-1",
		EditRequest{Orig: "yyy", Text: "xxx", Summary: `<script>alert(1)</script><b>why</b> & "how"`})
	require.NoError(t, err)
	require.NotNil(t, comment.Edit)
	assert.Equal(t, `why &amp; "how"`, comment.Edit.Summary, "returned by the edit")

	c, err := b.Engine.Get(getReq(locator, "id-1"))
	require.NoError(t, err)
	require.NotNil(t, c.Edit)
	assert.Equal(t, `why &amp; "how"`, c.Edit.Summary, "stored by the edit")

	// a delete sent through an edit keeps no summary
	_, err = b.EditComment(locator, "id-2", EditRequest{Delete: true, Summary: "<b>gone</b>"})
	require.NoError(t, err)
	c, err = b.Engine.Get(getReq(locator, "id-2"))
	require.NoError(t, err)
	assert.True(t, c.Deleted)
	assert.Nil(t, c.Edit, "no edit record stored on delete")
}

func TestService_ReadSanitizesStoredSummary(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}
	defer b.Close()

	// a summary stored before edits cleaned it
	locator := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	const raw = `<script>alert(1)</script><b>why</b> & "how"`
	const clean = `why &amp; "how"`
	c, err := b.Engine.Get(getReq(locator, "id-1"))
	require.NoError(t, err)
	c.Edit = &store.Edit{Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Summary: raw}
	require.NoError(t, b.Engine.Update(c))

	summaryOf := func(t *testing.T, cc []store.Comment) string {
		t.Helper()
		for _, c := range cc {
			if c.ID == "id-1" {
				require.NotNil(t, c.Edit)
				return c.Edit.Summary
			}
		}
		require.FailNow(t, "id-1 not in the result")
		return ""
	}

	got, err := b.Get(locator, "id-1", store.User{})
	require.NoError(t, err)
	require.NotNil(t, got.Edit)
	assert.Equal(t, clean, got.Edit.Summary, "Get")

	cc, err := b.Find(locator, "time", store.User{})
	require.NoError(t, err)
	assert.Equal(t, clean, summaryOf(t, cc), "Find")

	cc, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Equal(t, clean, summaryOf(t, cc), "Last")

	cc, err = b.User("radio-t", "user1", 0, 0, store.User{})
	require.NoError(t, err)
	assert.Equal(t, clean, summaryOf(t, cc), "User")

	c, err = b.Engine.Get(getReq(locator, "id-1"))
	require.NoError(t, err)
	assert.Equal(t, raw, c.Edit.Summary, "reading does not rewrite the stored record")
}

func TestService_DeleteComment(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}
	defer b.Close()

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, EditRequest{Delete: true})
	assert.NoError(t, err)

	c, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID))
	assert.NoError(t, err)
	assert.True(t, c.Deleted)
	t.Logf("%+v", c)
}

func TestService_EditCommentDurationFailed(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.Error(t, err)
}

func TestService_EditCommentReplyFailed(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}
	defer b.Close()

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[1])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Nil(t, res[1].Edit)

	reply := store.Comment{
		ID:        "123456",
		ParentID:  "id-1",
		Text:      "some text",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}
	_, err = b.Create(reply)
	assert.NoError(t, err)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.EqualError(t, err, "parent comment with reply can't be edited, id-1")
}

func TestService_EditCommentAdmin(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123"), AdminEdits: true}

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	t.Logf("%+v", res[0])
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit", Admin: true})
	assert.NoError(t, err)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit", Admin: false})
	assert.Error(t, err)
}

func TestService_EditCommentDeleted(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), AdminEdits: true}
	defer b.Close()

	locator := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	require.Len(t, res, 2)
	id := res[0].ID
	require.NoError(t, b.Delete(locator, id, store.SoftDelete))

	// admin edits skip the edit window and the replies check, but not this one
	for _, isAdmin := range []bool{false, true} {
		_, err = b.EditComment(locator, id, EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit", Admin: isAdmin})
		assert.EqualError(t, err, "deleted comment can't be edited, "+id, "admin %v", isAdmin)
	}

	c, err := b.Engine.Get(getReq(locator, id))
	require.NoError(t, err)
	assert.True(t, c.Deleted)
	assert.Empty(t, c.Text, "stored text stays empty")
	assert.Empty(t, c.Orig, "stored original text stays empty")
	assert.Nil(t, c.Edit, "no edit record stored")

	c, err = b.EditComment(locator, id, EditRequest{Delete: true})
	require.NoError(t, err, "a repeated delete is accepted")
	assert.True(t, c.Deleted)
	assert.Empty(t, c.Text)
}

func TestService_ValidateComment(t *testing.T) {
	b := DataStore{MinCommentSize: 6, MaxCommentSize: 2000, AdminStore: admin.NewStaticKeyStore("secret 123")}
	longText := fmt.Sprintf("%4000s", "X")

	tbl := []struct {
		inp store.Comment
		err string
	}{
		{inp: store.Comment{}, err: "empty comment text"},
		{inp: store.Comment{Orig: "something blah", User: store.User{ID: "myid", Name: "name"}}, err: ""},
		{inp: store.Comment{Orig: "something blah", User: store.User{ID: "myid"}}, err: "empty user info"},
		{inp: store.Comment{Orig: "short", User: store.User{ID: "myid", Name: "name"}}, err: "comment text is smaller than min allowed size 6 (5)"},
		{inp: store.Comment{Orig: longText, User: store.User{ID: "myid", Name: "name"}}, err: "comment text exceeded max allowed size 2000 (4000)"},
		{inp: store.Comment{Orig: "here is a link with relative URL: [google.com](url)", User: store.User{ID: "myid", Name: "name"}}, err: "links should start with mailto:, http:// or https://"},
		{inp: store.Comment{Orig: "here is a link with relative URL: [google.com](url)", User: store.User{ID: "myid", Name: "name"}}, err: "links should start with mailto:, http:// or https://"},
		{inp: store.Comment{Orig: "multiple links, one is bad: [test](http://test) [test2](bad_url) [test3](https://test3)", User: store.User{ID: "myid", Name: "name"}}, err: "links should start with mailto:, http:// or https://"},
		{inp: store.Comment{Orig: "some text\n[0]:<", User: store.User{ID: "myid", Name: "name"}}, err: "can't parse comment text"}, // the parser panics on it
	}

	for n, tt := range tbl {
		var err error
		require.NotPanics(t, func() { err = b.ValidateComment(&tt.inp) }, "check #%d", n)
		if tt.err == "" {
			assert.NoError(t, err, "check #%d", n)
			continue
		}
		require.Error(t, err)
		assert.EqualError(t, err, tt.err, "check #%d", n)
	}
}

func TestService_Counts(t *testing.T) {
	b, teardown := prepStoreEngine(t) // two comments for https://radio-t.com
	defer teardown()

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "123456",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	svc := DataStore{Engine: b}
	res, err := svc.Counts("radio-t", []string{"https://radio-t.com/2"})
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1}}, res)

	res, err = svc.Counts("radio-t", []string{"https://radio-t.com", "https://radio-t.com/2", "blah"})
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{
		{URL: "https://radio-t.com", Count: 2},
		{URL: "https://radio-t.com/2", Count: 1},
		{URL: "blah", Count: 0},
	}, res)
}

func TestService_GetMetas(t *testing.T) {
	// two comments for https://radio-t.com
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}

	um, pm, err := b.Metas("radio-t")
	require.NoError(t, err)
	assert.Equal(t, 0, len(um))
	assert.Equal(t, 0, len(pm))

	assert.NoError(t, b.SetVerified("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user1", true, time.Hour))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true, time.Hour))
	assert.NoError(t, b.SetReadOnly(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, true))

	// set email for one existing and one non-existing user
	req := engine.UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user2", Detail: engine.UserEmail, Update: "test@example.org"}
	value, err := b.Engine.UserDetail(req)
	assert.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "user2", Email: "test@example.org"}}, value)
	req.UserID = "user3"
	value, err = b.Engine.UserDetail(req)
	assert.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "user3", Email: "test@example.org"}}, value)

	um, pm, err = b.Metas("radio-t")
	require.NoError(t, err)

	require.Equal(t, 3, len(um))
	assert.Equal(t, "user1", um[0].ID)
	assert.Equal(t, true, um[0].Verified)
	assert.Equal(t, engine.UserDetailEntry{Email: ""}, um[0].Details)
	assert.Equal(t, true, um[0].Blocked.Status)
	assert.Equal(t, false, um[1].Verified)
	assert.Equal(t, true, um[1].Blocked.Status)
	assert.Equal(t, "test@example.org", um[1].Details.Email)
	assert.Equal(t, "user3", um[2].ID)
	assert.Equal(t, "test@example.org", um[2].Details.Email)

	require.Equal(t, 1, len(pm))
	assert.Equal(t, "https://radio-t.com", pm[0].URL)
	assert.Equal(t, true, pm[0].ReadOnly)
}

func TestService_SetMetas(t *testing.T) {
	// two comments for https://radio-t.com
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}
	umetas := []UserMetaData{}
	pmetas := []PostMetaData{}
	err := b.SetMetas("radio-t", umetas, pmetas)
	assert.NoError(t, err, "empty metas")

	um1 := UserMetaData{ID: "user1", Verified: true, Details: engine.UserDetailEntry{Email: "test@example.org"}}
	um2 := UserMetaData{ID: "user2"}
	um2.Blocked.Status = true
	um2.Blocked.Until = time.Now().AddDate(0, 1, 1)

	pmetas = []PostMetaData{{URL: "https://radio-t.com", ReadOnly: true}}
	err = b.SetMetas("radio-t", []UserMetaData{um1, um2}, pmetas)
	assert.NoError(t, err)

	assert.True(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}))
	assert.True(t, b.IsVerified("radio-t", "user1"))
	assert.True(t, b.IsBlocked("radio-t", "user2"))
	val, err := b.Engine.UserDetail(engine.UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Detail: engine.UserEmail})
	assert.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "user1", Email: "test@example.org"}}, val)
}

func TestService_SetMetasExpiredBlock(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123")}

	expired := UserMetaData{ID: "user1"}
	expired.Blocked.Status = true
	expired.Blocked.Until = time.Now().Add(-time.Hour)
	permanent := UserMetaData{ID: "user2"} // no end time, as an export without one carries it
	permanent.Blocked.Status = true

	require.NoError(t, b.SetMetas("radio-t", []UserMetaData{expired, permanent}, nil))
	assert.False(t, b.IsBlocked("radio-t", "user1"), "a block that ran out before the restore stays out")
	assert.True(t, b.IsBlocked("radio-t", "user2"), "a block without an end time stays permanent")
	blocked, err := b.BlockedUsers("radio-t")
	require.NoError(t, err)
	require.Len(t, blocked, 1)
	assert.Equal(t, "user2", blocked[0].ID)
}

func TestService_UserDetailsOperations(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}

	// add single valid entry
	result, err := b.SetUserEmail("radio-t", "u1", "test@example.com")
	assert.NoError(t, err, "No error inserting entry expected")
	assert.Equal(t, "test@example.com", result)
	result, err = b.SetUserTelegram("radio-t", "u1", "test@example.com")
	assert.NoError(t, err, "No error inserting entry expected")
	assert.Equal(t, "test@example.com", result)

	// read valid entry back
	result, err = b.GetUserEmail("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Equal(t, "test@example.com", result)
	result, err = b.GetUserTelegram("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Equal(t, "test@example.com", result)

	// delete existing entry
	err = b.DeleteUserDetail("radio-t", "u1", engine.UserEmail)
	assert.NoError(t, err, "No error deleting entry expected")
	err = b.DeleteUserDetail("radio-t", "u1", engine.UserTelegram)
	assert.NoError(t, err, "No error deleting entry expected")

	// read deleted entry
	result, err = b.GetUserEmail("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Empty(t, result)
	result, err = b.GetUserTelegram("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Empty(t, result)

	// insert entry with invalid site_id
	result, err = b.SetUserEmail("bad-site", "u3", "does_not_matter@example.com")
	assert.Error(t, err, "Site not found")
	assert.Empty(t, result)
	result, err = b.SetUserTelegram("bad-site", "u3", "does_not_matter@example.com")
	assert.Error(t, err, "Site not found")
	assert.Empty(t, result)

	// read entry with invalid site_id
	result, err = b.GetUserEmail("bad-site", "u3")
	assert.Error(t, err, "Site not found")
	assert.Empty(t, result)
	result, err = b.GetUserTelegram("bad-site", "u3")
	assert.Error(t, err, "Site not found")
	assert.Empty(t, result)
}

func TestService_IsAdmin(t *testing.T) {
	// two comments for https://radio-t.com
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", []string{"radio-t"}, []string{"user2"}, "user@email.com")}

	assert.False(t, b.IsAdmin("radio-t", "user1"))
	assert.True(t, b.IsAdmin("radio-t", "user2"))
	assert.False(t, b.IsAdmin("radio-t-bad", "user1"))
}

func TestService_HasReplies(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng,
		AdminStore: admin.NewStaticStore("secret 123", []string{"radio-t"}, []string{"user2"}, "user@email.com")}
	defer b.Close()

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	assert.False(t, b.HasReplies(comment))

	reply := store.Comment{
		ID:        "c-1",
		ParentID:  "id-1",
		Text:      "some text",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}
	_, err := b.Create(reply)
	assert.NoError(t, err)
	assert.True(t, b.HasReplies(comment))

	// deletion of the parent comment shouldn't work as the comment has replies
	_, err = b.EditComment(reply.Locator, comment.ID, EditRequest{Orig: comment.ID, Delete: true, Summary: "user deletes the comment"})
	assert.EqualError(t, err, "parent comment with reply can't be edited, "+comment.ID)

	// should not report replies after deletion of the child
	err = b.Delete(reply.Locator, reply.ID, store.HardDelete)
	assert.NoError(t, err)
	assert.False(t, b.HasReplies(comment))

	// recreate reply with the new ID
	reply.ID = "c-2"
	_, err = b.Create(reply)
	assert.NoError(t, err)
	assert.True(t, b.HasReplies(comment))

	// should not report replies after deletion of the child using Edit mechanism
	_, err = b.EditComment(reply.Locator, reply.ID, EditRequest{Orig: reply.ID, Delete: true, Summary: "user deletes the comment"})
	assert.NoError(t, err)
	assert.False(t, b.HasReplies(comment))
}

func TestService_HasRepliesAfterManyLaterComments(t *testing.T) {
	// the reply check must see a reply however many comments the site got after it on other posts
	eng, err := engine.NewBoltDB(bolt.Options{NoSync: true}, engine.BoltSite{FileName: path.Join(t.TempDir(), "test.db"), SiteID: "radio-t"})
	require.NoError(t, err)
	b := DataStore{Engine: eng,
		AdminStore: admin.NewStaticStore("secret 123", []string{"radio-t"}, []string{"user2"}, "user@email.com")}
	defer func() { assert.NoError(t, b.Close()) }()

	ts := time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC)
	post := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	_, err = eng.Create(store.Comment{ID: "id-1", Text: `some text, <a href="http://radio-t.com">link</a>`, Timestamp: ts,
		Locator: post, User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)
	_, err = eng.Create(store.Comment{ID: "reply-1", ParentID: "id-1", Text: "reply", Timestamp: ts.Add(time.Minute),
		Locator: post, User: store.User{ID: "user2", Name: "user name 2"}})
	require.NoError(t, err)

	// more later comments on another post than the engine returns as the site's last comments
	other := store.Locator{URL: "https://radio-t.com/other", SiteID: "radio-t"}
	for i := range 1001 {
		c := store.Comment{ID: fmt.Sprintf("other-%d", i), Text: "text", Timestamp: ts.Add(time.Hour + time.Duration(i)*time.Second),
			Locator: other, User: store.User{ID: "user3", Name: "user name 3"}}
		_, err = eng.Create(c)
		require.NoError(t, err)
	}

	parent, err := eng.Get(engine.GetRequest{Locator: post, CommentID: "id-1"})
	require.NoError(t, err)
	assert.True(t, b.HasReplies(parent), "the reply to id-1 is on its post")

	// the author of id-1 can neither delete nor edit it while it has a reply
	_, err = b.EditComment(post, "id-1", EditRequest{Delete: true})
	require.EqualError(t, err, "parent comment with reply can't be edited, id-1")
	_, err = b.EditComment(post, "id-1", EditRequest{Text: "new text", Orig: "new text"})
	require.EqualError(t, err, "parent comment with reply can't be edited, id-1")
	parent, err = eng.Get(engine.GetRequest{Locator: post, CommentID: "id-1"})
	require.NoError(t, err)
	assert.False(t, parent.Deleted)
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, parent.Text)
}

func TestService_HasRepliesAfterUserDelete(t *testing.T) {
	// deleting the author of the only reply leaves the parent without replies straight away
	for name, mode := range map[string]store.DeleteMode{"soft": store.SoftDelete, "hard": store.HardDelete} {
		t.Run(name, func(t *testing.T) {
			eng, teardown := prepStoreEngine(t)
			defer teardown()
			b := DataStore{Engine: eng,
				AdminStore: admin.NewStaticStore("secret 123", []string{"radio-t"}, []string{"user2"}, "user@email.com")}
			defer b.Close()

			post := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
			_, err := b.Create(store.Comment{ID: "reply-1", ParentID: "id-1", Text: "reply", Locator: post,
				User: store.User{ID: "user2", Name: "user name 2"}})
			require.NoError(t, err)
			parent, err := eng.Get(engine.GetRequest{Locator: post, CommentID: "id-1"})
			require.NoError(t, err)
			require.True(t, b.HasReplies(parent))

			require.NoError(t, b.DeleteUser("radio-t", "user2", mode))
			assert.False(t, b.HasReplies(parent), "the only reply is deleted with its author")
			_, err = b.EditComment(post, "id-1", EditRequest{Delete: true})
			require.NoError(t, err, "the author can delete a comment that has no replies left")
		})
	}
}

func TestService_UserReplies(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// two comments for https://radio-t.com, no reply
		eng, teardown := prepStoreEngine(t)
		defer teardown()
		b := DataStore{Engine: eng,
			AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

		c1 := store.Comment{
			ID:      "comment-id-1",
			Text:    "test 123",
			Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
			User:    store.User{ID: "u1", Name: "developer one u1"},
		}
		c2 := store.Comment{
			ID:       "comment-id-2",
			ParentID: "comment-id-1",
			Text:     "xyz test",
			Locator:  store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
			User:     store.User{ID: "u2", Name: "developer one u2"},
		}
		c3 := store.Comment{
			ID:       "comment-id-3",
			ParentID: "comment-id-1",
			Text:     "xyz test",
			Locator:  store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
			User:     store.User{ID: "u2", Name: "developer one u3"},
		}
		c4 := store.Comment{
			ID:       "comment-id-4",
			ParentID: "",
			Text:     "xyz test",
			Locator:  store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
			User:     store.User{ID: "u4", Name: "developer one u4"},
		}
		c5 := store.Comment{
			ID:       "comment-id-5",
			ParentID: "comment-id-1",
			Text:     "xyz test",
			Locator:  store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
			User:     store.User{ID: "u2", Name: "developer one u2"},
		}

		// small sleeps give each Create a unique nanosecond timestamp under synctest's fake clock,
		// since Bolt keys the "last" bucket by comment timestamp
		_, err := b.Create(c1)
		require.NoError(t, err)
		time.Sleep(time.Nanosecond)
		_, err = b.Create(c2)
		require.NoError(t, err)
		time.Sleep(time.Nanosecond)
		_, err = b.Create(c3)
		require.NoError(t, err)
		time.Sleep(time.Nanosecond)
		_, err = b.Create(c4)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		_, err = b.Create(c5)
		require.NoError(t, err)

		cc, u, err := b.UserReplies("radio-t", "u1", 10, time.Hour)
		assert.NoError(t, err)
		require.Equal(t, 3, len(cc), "3 replies to u1")
		assert.Equal(t, "developer one u1", u)

		// advance fake clock so c2 and c3 (created 100ms before c5) age past the 299ms window,
		// leaving only c5 as a recent reply
		time.Sleep(200 * time.Millisecond)
		cc, u, err = b.UserReplies("radio-t", "u1", 10, time.Millisecond*299)
		require.NoError(t, err)
		require.Equal(t, "developer one u1", u)
		require.Equal(t, 1, len(cc), "1 reply to u1 in the last 299ms")

		cc, u, err = b.UserReplies("radio-t", "u2", 10, time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cc), "0 replies to u2")
		assert.Equal(t, "developer one u2", u)

		cc, u, err = b.UserReplies("radio-t", "uxxx", 10, time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cc), "0 replies to uxxx")
		assert.Equal(t, "", u)

		// with u1's only comment deleted, the name comes from the replied-to comment, which keeps it
		require.NoError(t, b.Delete(c1.Locator, c1.ID, store.SoftDelete))
		cc, u, err = b.UserReplies("radio-t", "u1", 10, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 3, len(cc), "replies to a deleted comment are still listed")
		assert.Equal(t, "developer one u1", u)
	})
}

func TestService_UserRepliesLimit(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticStore("secret 123", nil, []string{}, "user@email.com")}

	loc := store.Locator{URL: "https://radio-t.com/limit", SiteID: "radio-t"}
	base := time.Now().Add(-time.Hour)
	_, err := eng.Create(store.Comment{ID: "parent", Locator: loc, Timestamp: base, User: store.User{ID: "u1", Name: "user one"}})
	require.NoError(t, err)
	for i := 1; i <= 3; i++ {
		_, err = eng.Create(store.Comment{ID: fmt.Sprintf("reply-%d", i), ParentID: "parent", Locator: loc,
			Timestamp: base.Add(time.Duration(i) * time.Second), User: store.User{ID: "u2", Name: "user two"}})
		require.NoError(t, err)
	}

	cc, _, err := b.UserReplies("radio-t", "u1", 2, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, cc, 2, "no more replies than the limit")
	assert.Equal(t, "reply-3", cc[0].ID, "newest first")
	assert.Equal(t, "reply-2", cc[1].ID)
}

func TestService_UserRepliesSkipsOrphanReply(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticStore("secret 123", nil, []string{}, "user@email.com")}

	loc := store.Locator{URL: "https://radio-t.com/orphan", SiteID: "radio-t"}
	base := time.Now().Add(-time.Hour)
	_, err := eng.Create(store.Comment{ID: "parent", Locator: loc, Timestamp: base, User: store.User{ID: "u1", Name: "user one"}})
	require.NoError(t, err)
	_, err = eng.Create(store.Comment{ID: "reply", ParentID: "parent", Locator: loc, Timestamp: base.Add(time.Second),
		User: store.User{ID: "u2", Name: "user two"}})
	require.NoError(t, err)
	// a later reply by another user to a comment that is not in the store
	_, err = eng.Create(store.Comment{ID: "orphan", ParentID: "missing", Locator: loc, Timestamp: base.Add(2 * time.Second),
		User: store.User{ID: "u3", Name: "user three"}})
	require.NoError(t, err)

	cc, u, err := b.UserReplies("radio-t", "u1", 10, 24*time.Hour)
	require.NoError(t, err, "a reply to a missing comment does not fail the replies of another user")
	assert.Equal(t, "user one", u)
	require.Len(t, cc, 1)
	assert.Equal(t, "reply", cc[0].ID)

	// more replies to missing comments on the same post cost one read of the post, not one each
	for i := range 3 {
		_, err = eng.Create(store.Comment{ID: "orphan" + strconv.Itoa(i), ParentID: "missing" + strconv.Itoa(i), Locator: loc,
			Timestamp: base.Add(time.Duration(3+i) * time.Second), User: store.User{ID: "u3", Name: "user three"}})
		require.NoError(t, err)
	}
	counted := &failingReads{Interface: eng}
	b.Engine = counted
	cc, _, err = b.UserReplies("radio-t", "u1", 10, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, cc, 1)
	assert.Equal(t, 1, counted.postReads, "the post is read once for all its replies to missing comments")

	// a parent that can't be told missing because the post can't be read still fails the call
	b.Engine = &failingReads{Interface: eng, post: true}
	_, _, err = b.UserReplies("radio-t", "u1", 10, 24*time.Hour)
	require.Error(t, err, "a store error while checking the parent is returned")
	assert.Contains(t, err.Error(), "can't get parent comment")

	// a parent that is in the post but can't be read fails the call too
	b.Engine = &failingReads{Interface: eng, get: true}
	_, _, err = b.UserReplies("radio-t", "u1", 10, 24*time.Hour)
	require.Error(t, err, "a store error while reading an existing parent is returned")
	assert.Contains(t, err.Error(), "can't get parent comment")
}

// failingReads fails reads of single comments (get) or of a post's comments (post), and counts the latter
type failingReads struct {
	engine.Interface
	get, post bool
	postReads int // reads of a post's comments
}

func (f *failingReads) Get(req engine.GetRequest) (store.Comment, error) {
	if f.get {
		return store.Comment{}, errors.New("read failed")
	}
	return f.Interface.Get(req)
}

func (f *failingReads) Find(req engine.FindRequest) ([]store.Comment, error) {
	if req.Locator.URL != "" {
		f.postReads++
		if f.post {
			return nil, errors.New("read failed")
		}
	}
	return f.Interface.Find(req)
}

func TestService_Find(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	res, err := b.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time", store.User{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(res))

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "123456",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
		Score:     1,
		Votes:     map[string]bool{"id-1": true, "id-2": true, "123456": false},
		PostTitle: `some title, <a href="http://radio-t.com">link</a>`,
	}
	_, err = b.Engine.Create(comment) // create directly with engine, doesn't set Controversy
	assert.NoError(t, err)

	// make sure Controversy altered
	res, err = b.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "-controversy", store.User{})
	require.NoError(t, err)
	require.Equal(t, 3, len(res))
	assert.Equal(t, "123456", res[0].ID)
	assert.InDelta(t, 1.73, res[0].Controversy, 0.01)
	assert.Equal(t, "id-1", res[1].ID)
	assert.InDelta(t, 0, res[1].Controversy, 0.01)

	// make sure title sanitized
	assert.Equal(t, "some title, link", res[0].PostTitle)
}

// comments stored by a release that marked links nofollow only are read back with ugc added
func TestService_ReadMarksStoredLinksUGC(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	locator := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	_, err := b.Engine.Create(store.Comment{ID: "stored", Text: `see <a href="http://radio-t.com" rel="nofollow">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 24, 0, time.UTC), Locator: locator,
		User: store.User{ID: "user1", Name: "user name"}}) // written directly, as an earlier release stored it
	require.NoError(t, err)
	want := `see <a href="http://radio-t.com" rel="nofollow ugc">link</a>`

	res, err := b.Find(locator, "time", store.User{})
	require.NoError(t, err)
	require.Len(t, res, 3)
	assert.Equal(t, want, res[2].Text, "find")

	c, err := b.Get(locator, "stored", store.User{})
	require.NoError(t, err)
	assert.Equal(t, want, c.Text, "get")
}

func TestService_FindSince(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	res, err := b.FindSince(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time", store.User{}, time.Time{})
	require.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "id-1", res[0].ID)

	res, err = b.FindSince(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time", store.User{},
		time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, 1, len(res))
	assert.Equal(t, "id-2", res[0].ID)
}

func TestService_Info(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more comment for another URL to test site-wide Info request
	comment := store.Comment{
		ID:        "123456xyz",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com/another", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	require.NoError(t, err)

	// get non-existing URL info
	info, err := b.Info(store.Locator{URL: "bad", SiteID: "radio-t"}, 0)
	assert.Error(t, err)
	assert.Empty(t, info)

	// get non-existing site info
	info, err = b.Info(store.Locator{SiteID: "bad"}, 0)
	assert.Error(t, err)
	assert.Empty(t, info)

	// test two initially created comments and store first comment FirstTS
	info, err = b.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com", info.URL)
	assert.Equal(t, 2, info.Count)
	assert.False(t, info.ReadOnly)
	assert.True(t, info.LastTS.After(info.FirstTS))
	firstTS := info.FirstTS

	info, err = b.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 1)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com", info.URL)
	assert.True(t, info.ReadOnly)

	// get last created comment LastTS
	info, err = b.Info(store.Locator{URL: "https://radio-t.com/another", SiteID: "radio-t"}, 0)
	require.NoError(t, err)
	lastTS := info.LastTS

	// site-level request
	info, err = b.Info(store.Locator{SiteID: "radio-t"}, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, info.Count)
	assert.Empty(t, info.URL, "site-level request should not set URL")
	assert.False(t, info.ReadOnly, "site-level request should not set ReadOnly")
	assert.Equal(t, firstTS, info.FirstTS, "site-level request should have FirstTS from the first post")
	assert.Equal(t, lastTS, info.LastTS, "site-level request should have LastTS from the last post")
}

func TestService_Delete(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	require.Equal(t, 2, len(res))
	assert.NoError(t, err)

	err = b.Delete(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, store.SoftDelete)
	assert.NoError(t, err)

	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	assert.Equal(t, 1, len(res), "one left")
	assert.NoError(t, err)
}

func TestService_deleteImagesOnCommentDelete(t *testing.T) {
	lgr.Setup(lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

	synctest.Test(t, func(t *testing.T) {
		mockStore := image.StoreMock{
			DeleteFunc:            func(string) error { return nil },
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		imgSvc := image.NewService(&mockStore,
			image.ServiceParams{
				EditDuration: 50 * time.Millisecond,
				ImageAPI:     "/images/dev/",
				ProxyAPI:     "/non_existent",
			})
		defer imgSvc.Close(context.TODO())

		// two comments for https://radio-t.com
		eng, teardown := prepStoreEngine(t)
		defer teardown()
		b := DataStore{Engine: eng, EditDuration: 50 * time.Millisecond,
			AdminStore: admin.NewStaticKeyStore("secret 123"), ImageService: imgSvc}

		c := store.Comment{
			ID:        "id-22",
			Text:      `some text <img src="/images/dev/pic1.png"/> xx <img src="/images/dev/pic2.png"/>`,
			Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
			Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:      store.User{ID: "dev", Name: "user name"}, // the author of the dev/ uploads
		}
		_, err := b.Engine.Create(c) // create directly with engine, doesn't call submitImages
		assert.NoError(t, err)
		b.submitImages(c)
		// reply to the first comment with one new image and one existing one
		c = store.Comment{
			ID:       "id-23",
			ParentID: "id-22",
			Text:     `some text <img src="/images/dev/pic2.png"/> xx <img src="/images/dev/pic3.png"/>`,
			Locator:  store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:     store.User{ID: "dev", Name: "user name"},
		}
		_, err = b.Engine.Create(c) // create directly with engine, doesn't call submitImages
		assert.NoError(t, err)
		b.submitImages(c)

		// verify that images are in staging store
		assert.Equal(t, 4, len(mockStore.ResetCleanupTimerCalls()))
		assert.Equal(t, "dev/pic1.png", mockStore.ResetCleanupTimerCalls()[0].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.ResetCleanupTimerCalls()[1].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.ResetCleanupTimerCalls()[2].ID)
		assert.Equal(t, "dev/pic3.png", mockStore.ResetCleanupTimerCalls()[3].ID)
		time.Sleep(b.EditDuration + 100*time.Millisecond)
		// verify that they got into the main store
		assert.Equal(t, 4, len(mockStore.CommitCalls()))
		assert.Equal(t, "dev/pic1.png", mockStore.CommitCalls()[0].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.CommitCalls()[1].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.CommitCalls()[2].ID)
		assert.Equal(t, "dev/pic3.png", mockStore.CommitCalls()[3].ID)

		// delete the first comment
		err = b.Delete(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-22", store.SoftDelete)
		assert.NoError(t, err)
		// verify that images are deleted from the main store
		assert.Equal(t, 1, len(mockStore.DeleteCalls()))
		assert.Equal(t, "dev/pic1.png", mockStore.DeleteCalls()[0].ID)

		// delete the second comment
		err = b.Delete(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-23", store.SoftDelete)
		assert.NoError(t, err)
		// verify that images are deleted from the main store
		assert.Equal(t, 3, len(mockStore.DeleteCalls()))
		assert.Equal(t, "dev/pic2.png", mockStore.DeleteCalls()[1].ID)
		assert.Equal(t, "dev/pic3.png", mockStore.DeleteCalls()[2].ID)
	})
}

// an admin deleting a comment removes only the pictures its author uploaded, and a user's own delete removes none
func TestService_DeleteCommentPictures(t *testing.T) {
	proxied := `<img src="/non_existent?src=` + base64.URLEncoding.EncodeToString([]byte("https://example.com/cat.png")) + `"/>`
	prep := func(t *testing.T) (*DataStore, *image.StoreMock) {
		mockStore := &image.StoreMock{DeleteFunc: func(string) error { return nil }}
		imgSvc := image.NewService(mockStore, image.ServiceParams{ImageAPI: "/images/", ProxyAPI: "/non_existent"})
		t.Cleanup(func() { imgSvc.Close(context.TODO()) })
		eng, teardown := prepStoreEngine(t)
		t.Cleanup(teardown)
		b := &DataStore{Engine: eng, EditDuration: time.Hour, AdminStore: admin.NewStaticKeyStore("secret 123"), ImageService: imgSvc}
		for _, c := range []store.Comment{
			// user1 shows their own upload, user2's upload and a cached external image
			{ID: "u1-a", Text: `<img src="/images/user1/own.png"/> <img src="/images/user2/up.png"/> ` + proxied,
				Locator: store.Locator{URL: "https://radio-t.com/a", SiteID: "radio-t"}, User: store.User{ID: "user1"},
				Timestamp: time.Now()},
			{ID: "u2-b", Text: `<img src="/images/user2/up.png"/>`,
				Locator: store.Locator{URL: "https://radio-t.com/b", SiteID: "radio-t"}, User: store.User{ID: "user2"},
				Timestamp: time.Now().Add(time.Second)},
		} {
			_, err := b.Engine.Create(c) // create directly with engine, doesn't call submitImages
			require.NoError(t, err)
		}
		return b, mockStore
	}

	for _, mode := range []store.DeleteMode{store.HardDelete, store.SoftDelete} {
		t.Run(fmt.Sprintf("admin delete, mode %d", mode), func(t *testing.T) {
			b, mockStore := prep(t)
			require.NoError(t, b.Delete(store.Locator{URL: "https://radio-t.com/a", SiteID: "radio-t"}, "u1-a", mode))
			// user2's upload and the cached external image stay although only the deleted comment shows them on page a
			assert.Equal(t, []string{"user1/own.png"}, deletedImages(mockStore))
			c, err := b.Engine.Get(engine.GetRequest{Locator: store.Locator{URL: "https://radio-t.com/b", SiteID: "radio-t"}, CommentID: "u2-b"})
			require.NoError(t, err)
			assert.Contains(t, c.Text, "/images/user2/up.png", "user2's comment untouched")
		})
	}

	t.Run("user's own delete", func(t *testing.T) {
		b, mockStore := prep(t)
		_, err := b.EditComment(store.Locator{URL: "https://radio-t.com/a", SiteID: "radio-t"}, "u1-a", EditRequest{Delete: true})
		require.NoError(t, err)
		assert.Empty(t, deletedImages(mockStore), "a user's own delete removes no pictures")
	})
}

// DeleteUser removes all comments from user
func TestService_DeleteUser(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "123456xyz",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	assert.Equal(t, 3, len(res), "3 comments initially, for 2 diff users and 2 posts")
	assert.NoError(t, err)

	err = b.DeleteUser("radio-t", "user1", store.HardDelete)
	assert.NoError(t, err)

	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	require.Equal(t, 1, len(res), "only one comment left for user2")
	assert.NoError(t, err)
	assert.Equal(t, "user2", res[0].User.ID)
}

func TestService_DeleteUserPictures(t *testing.T) {
	for _, mode := range []store.DeleteMode{store.HardDelete, store.SoftDelete} {
		t.Run(fmt.Sprintf("mode %d", mode), func(t *testing.T) {
			mockStore := image.StoreMock{DeleteFunc: func(string) error { return nil }}
			imgSvc := image.NewService(&mockStore, image.ServiceParams{ImageAPI: "/images/", ProxyAPI: "/non_existent"})
			defer imgSvc.Close(context.TODO())

			eng, teardown := prepStoreEngine(t)
			defer teardown()
			b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), ImageService: imgSvc}

			ts := time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC)
			for _, c := range []store.Comment{
				{ID: "u1-a", Text: `<img src="/images/user1/own1.png"/> <img src="/images/user1/shared.png"/>`,
					Locator: store.Locator{URL: "https://radio-t.com/a", SiteID: "radio-t"}, User: store.User{ID: "user1"}},
				{ID: "u1-b", Text: `<img src="/images/user1/own2.png"/> <img src="/images/user1/own1.png"/>`,
					Locator: store.Locator{URL: "https://radio-t.com/b", SiteID: "radio-t"}, User: store.User{ID: "user1"}},
				{ID: "u2-a", Text: `<img src="/images/user2/other.png"/> <img src="/images/user1/shared.png"/>`,
					Locator: store.Locator{URL: "https://radio-t.com/a", SiteID: "radio-t"}, User: store.User{ID: "user2"}},
				{ID: "u2-c", Text: `<img src="/images/user2/other2.png"/>`,
					Locator: store.Locator{URL: "https://radio-t.com/c", SiteID: "radio-t"}, User: store.User{ID: "user2"}},
				// user1 shows user2's upload and a cached external image on a page where nobody else uses them
				{ID: "u1-d", Text: `<img src="/images/user2/other2.png"/> <img src="/non_existent?src=` +
					base64.URLEncoding.EncodeToString([]byte("https://example.com/cat.png")) + `"/>`,
					Locator: store.Locator{URL: "https://radio-t.com/d", SiteID: "radio-t"}, User: store.User{ID: "user1"}},
			} {
				c.Timestamp = ts
				ts = ts.Add(time.Minute)
				_, err := b.Engine.Create(c) // create directly with engine, doesn't call submitImages
				require.NoError(t, err)
			}

			require.NoError(t, b.DeleteUser("radio-t", "user1", mode))

			deleted := deletedImages(&mockStore)
			// only user1's own uploads go: user1/shared.png stays as user2 reuses it on the same page,
			// user2's upload and the cached external image stay although only user1 shows them on page d
			assert.ElementsMatch(t, []string{"user1/own1.png", "user1/own2.png"}, deleted)

			u2, err := b.User("radio-t", "user2", 0, 0, store.User{})
			require.NoError(t, err)
			require.Len(t, u2, 2)
			for _, c := range u2 {
				assert.Contains(t, c.Text, "/images/user2/", "user2's comments untouched")
			}
		})
	}
}

func TestService_DeleteUserPicturesNoComments(t *testing.T) {
	mockStore := image.StoreMock{DeleteFunc: func(string) error { return nil }}
	imgSvc := image.NewService(&mockStore, image.ServiceParams{ImageAPI: "/images/", ProxyAPI: "/non_existent"})
	defer imgSvc.Close(context.TODO())

	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"), ImageService: imgSvc}

	require.NoError(t, b.DeleteUser("radio-t", "no-such-user", store.HardDelete))
	assert.Empty(t, mockStore.DeleteCalls())
}

func TestService_DeleteUserPicturesEngine(t *testing.T) {
	// the user's comments come in pages, each page asked for with a growing skip
	pages := [][]store.Comment{
		{{ID: "c1", Text: `<img src="/images/user1/p1.png"/>`, Locator: store.Locator{SiteID: "s", URL: "u1"}, User: store.User{ID: "user1"}}},
		{{ID: "c2", Text: `<img src="/images/user1/p2.png"/>`, Locator: store.Locator{SiteID: "s", URL: "u2"}, User: store.User{ID: "user1"}}},
	}
	findFn := func(req engine.FindRequest) []store.Comment {
		if req.UserID == "" { // page lookup
			for _, p := range pages {
				if p[0].Locator.URL == req.Locator.URL {
					return p
				}
			}
			return nil
		}
		if req.Skip < len(pages) {
			return pages[req.Skip]
		}
		return []store.Comment{}
	}

	tbl := []struct {
		name      string
		findErr   error
		deleteErr error
		wantErr   bool
		deleted   []string
	}{
		{name: "all pages", deleted: []string{"user1/p1.png", "user1/p2.png"}},
		{name: "engine delete fails", deleteErr: errors.New("delete failed"), wantErr: true, deleted: []string{}},
		{name: "find fails", findErr: errors.New("find failed"), deleted: []string{}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := image.StoreMock{DeleteFunc: func(string) error { return nil }}
			imgSvc := image.NewService(&mockStore, image.ServiceParams{ImageAPI: "/images/", ProxyAPI: "/non_existent"})
			defer imgSvc.Close(context.TODO())
			eng := engine.InterfaceMock{
				FindFunc: func(req engine.FindRequest) ([]store.Comment, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return findFn(req), nil
				},
				DeleteFunc: func(engine.DeleteRequest) error { return tt.deleteErr },
			}
			b := DataStore{Engine: &eng, ImageService: imgSvc}

			err := b.DeleteUser("s", "user1", store.HardDelete)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, eng.DeleteCalls(), 1, "comments deleted even when the picture lookup fails")
			deleted := deletedImages(&mockStore)
			assert.ElementsMatch(t, tt.deleted, deleted)
		})
	}

	t.Run("engine ignores skip", func(t *testing.T) {
		mockStore := image.StoreMock{DeleteFunc: func(string) error { return nil }}
		imgSvc := image.NewService(&mockStore, image.ServiceParams{ImageAPI: "/images/", ProxyAPI: "/non_existent"})
		defer imgSvc.Close(context.TODO())
		userFinds := 0
		eng := engine.InterfaceMock{
			FindFunc: func(req engine.FindRequest) ([]store.Comment, error) {
				if req.UserID == "" {
					return nil, nil
				}
				userFinds++
				if userFinds > 5 {
					return nil, errors.New("paging never stopped")
				}
				return []store.Comment{pages[0][0], pages[1][0]}, nil // every comment, whatever the skip
			},
			DeleteFunc: func(engine.DeleteRequest) error { return nil },
		}
		b := DataStore{Engine: &eng, ImageService: imgSvc}

		require.NoError(t, b.DeleteUser("s", "user1", store.HardDelete))
		assert.Equal(t, 2, userFinds, "second page brings nothing new and ends the paging")
		deleted := deletedImages(&mockStore)
		assert.ElementsMatch(t, []string{"user1/p1.png", "user1/p2.png"}, deleted)
	})
}

// deletedImages lists the IDs of the pictures removed from the image store
func deletedImages(m *image.StoreMock) []string {
	res := make([]string, 0, len(m.DeleteCalls()))
	for _, call := range m.DeleteCalls() {
		res = append(res, call.ID)
	}
	return res
}

func TestService_List(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.List("radio-t", 0, 0)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res), "2 posts")
	assert.Equal(t, "https://radio-t.com/2", res[0].URL)
	assert.Equal(t, 1, res[0].Count)
	assert.Equal(t, time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC), res[0].FirstTS)
	assert.Equal(t, time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC), res[0].LastTS)

	assert.Equal(t, "https://radio-t.com", res[1].URL)
	assert.Equal(t, 2, res[1].Count)
	assert.Equal(t, time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC), res[1].FirstTS)
	assert.Equal(t, time.Date(2017, 12, 20, 15, 18, 23, 0, time.UTC), res[1].LastTS)
}

func TestService_Count(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	c, err := b.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.NoError(t, err)
	assert.Equal(t, 2, c)

	c, err = b.Count(store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"})
	assert.NoError(t, err)
	assert.Equal(t, 1, c)

	c, err = b.Count(store.Locator{URL: "https://radio-t.com/3", SiteID: "radio-t"})
	assert.NoError(t, err)
	assert.Equal(t, 0, c)
}

func TestService_UserComments(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	cc, err := b.User("radio-t", "user1", 0, 0, store.User{})
	assert.NoError(t, err)
	require.Equal(t, 2, len(cc), "two recs for user1")
	assert.Equal(t, "id-2", cc[0].ID, "reverse sort")
	assert.Equal(t, "id-1", cc[1].ID, "reverse sort")
}

func TestService_UserCount(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	c, err := b.UserCount("radio-t", "user1")
	assert.NoError(t, err)
	assert.Equal(t, 2, c)

	c, err = b.UserCount("radio-t", "user2")
	assert.NoError(t, err)
	assert.Equal(t, 1, c)

	_, err = b.UserCount("radio-t", "userBad")
	assert.EqualError(t, err, "no comments for user userBad in store for radio-t site")
}

func TestService_DeleteAll(t *testing.T) {
	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.UTC),
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	err = b.DeleteAll("radio-t")
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestService_submitImages(t *testing.T) {
	lgr.Setup(lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

	synctest.Test(t, func(t *testing.T) {
		mockStore := image.StoreMock{
			CommitFunc:            func(string) error { return nil },
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		imgSvc := image.NewService(&mockStore,
			image.ServiceParams{
				EditDuration: 50 * time.Millisecond,
				ImageAPI:     "/images/dev/",
				ProxyAPI:     "/non_existent",
			})
		defer imgSvc.Close(context.TODO())

		// two comments for https://radio-t.com
		eng, teardown := prepStoreEngine(t)
		defer teardown()
		b := DataStore{Engine: eng, EditDuration: 50 * time.Millisecond,
			AdminStore: admin.NewStaticKeyStore("secret 123"), ImageService: imgSvc}

		c := store.Comment{
			ID:        "id-22",
			Text:      `some text <img src="/images/dev/pic1.png"/> xx <img src="/images/dev/pic2.png"/>`,
			Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
			Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:      store.User{ID: "user1", Name: "user name"},
		}
		_, err := b.Engine.Create(c) // create directly with engine, doesn't call submitImages
		assert.NoError(t, err)

		b.submitImages(c)
		assert.Equal(t, 2, len(mockStore.ResetCleanupTimerCalls()))
		assert.Equal(t, "dev/pic1.png", mockStore.ResetCleanupTimerCalls()[0].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.ResetCleanupTimerCalls()[1].ID)
		time.Sleep(b.EditDuration + 100*time.Millisecond)
		assert.Equal(t, 2, len(mockStore.CommitCalls()))
		assert.Equal(t, "dev/pic1.png", mockStore.CommitCalls()[0].ID)
		assert.Equal(t, "dev/pic2.png", mockStore.CommitCalls()[1].ID)
	})
}

func TestService_ResubmitStagingImages(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mockStore := image.StoreMock{
			InfoFunc: func() (image.StoreInfo, error) {
				return image.StoreInfo{FirstStagingImageTS: time.Time{}.Add(time.Second)}, nil
			},
			CommitFunc: func(string) error {
				return nil
			},
			ResetCleanupTimerFunc: func(string) error { return nil },
		}
		imgSvc := image.NewService(&mockStore,
			image.ServiceParams{
				EditDuration: 10 * time.Millisecond,
				ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
				ProxyAPI:     "http://127.0.0.1:8080/api/v1/img",
			})
		defer imgSvc.Close(context.TODO())

		eng, teardown := prepStoreEngine(t)
		defer teardown()
		b := DataStore{Engine: eng, EditDuration: 10 * time.Millisecond, ImageService: imgSvc}

		// create comment with three images without preparing it properly
		comment := store.Comment{
			ID: "id-0",
			Text: `<img src="http://127.0.0.1:8080/api/v1/picture/dev_user/bqf122eq9r8ad657n3ng" alt="startrails_01.jpg"><br/>
               <img src="http://127.0.0.1:8080/api/v1/picture/dev_user/bqf321eq9r8ad657n3ng" alt="cat.png"><br/>
               <img src="http://127.0.0.1:8080/api/v1/img?src=aHR0cHM6Ly9ob21lcGFnZXMuY2FlLndpc2MuZWR1L35lY2U1MzMvaW1hZ2VzL2JvYXQucG5n" alt="cat.png"><br/>
               <img src="https://homepages.cae.wisc.edu/~ece533/images/boat.png" alt="boat.png">`,
			Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
			Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:      store.User{ID: "user1", Name: "user name"},
		}
		_, err := b.Engine.Create(comment)
		require.NoError(t, err)

		// resubmit single comment with three images, of which two are in staging storage
		err = b.ResubmitStagingImages([]string{"radio-t"})
		assert.NoError(t, err)

		// wait for Submit goroutine to commit image
		time.Sleep(b.EditDuration + time.Millisecond*100)

		assert.Equal(t, 1, len(mockStore.InfoCalls()))
		assert.Equal(t, 3, len(mockStore.CommitCalls()))

		// empty answer
		mockStoreEmpty := image.StoreMock{InfoFunc: func() (image.StoreInfo, error) {
			return image.StoreInfo{FirstStagingImageTS: time.Time{}}, nil
		}}
		imgSvcEmpty := image.NewService(&mockStoreEmpty,
			image.ServiceParams{
				EditDuration: 10 * time.Millisecond,
				ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
			})
		defer imgSvcEmpty.Close(context.TODO())
		bEmpty := DataStore{Engine: eng, EditDuration: 10 * time.Millisecond, ImageService: imgSvcEmpty}

		// resubmit receive empty timestamp and should do nothing )
		err = bEmpty.ResubmitStagingImages([]string{"radio-t", "non_existent"})
		assert.NoError(t, err)

		assert.Equal(t, 1, len(mockStore.InfoCalls()))

		// error from image storage
		mockStoreError := image.StoreMock{InfoFunc: func() (image.StoreInfo, error) {
			return image.StoreInfo{}, fmt.Errorf("mock_err")
		}}
		imgSvcError := image.NewService(&mockStoreError,
			image.ServiceParams{
				EditDuration: 10 * time.Millisecond,
				ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
			})
		defer imgSvcError.Close(context.TODO())
		bError := DataStore{Engine: eng, EditDuration: 10 * time.Millisecond, ImageService: imgSvcError}

		// resubmit will receive error from image storage and should return it
		err = bError.ResubmitStagingImages([]string{"radio-t"})
		assert.EqualError(t, err, "mock_err")

		assert.Equal(t, 1, len(mockStore.InfoCalls()))
		assert.Equal(t, 3, len(mockStore.ResetCleanupTimerCalls()))
		assert.Equal(t, "dev_user/bqf122eq9r8ad657n3ng", mockStore.ResetCleanupTimerCalls()[0].ID)
		assert.Equal(t, "dev_user/bqf321eq9r8ad657n3ng", mockStore.ResetCleanupTimerCalls()[1].ID)
		assert.Equal(t, "cached_images/12318fbd4c55e9d177b8b5ae197bc89c5afd8e07-a41fcb00643f28d700504256ec81cbf2e1aac53e", mockStore.ResetCleanupTimerCalls()[2].ID)
	})
}

func TestService_ResubmitStagingImages_EngineError(t *testing.T) {
	mockStore := image.StoreMock{InfoFunc: func() (image.StoreInfo, error) {
		return image.StoreInfo{FirstStagingImageTS: time.Time{}.Add(time.Second)}, nil
	}}
	imgSvc := image.NewService(&mockStore,
		image.ServiceParams{
			EditDuration: 10 * time.Millisecond,
			ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
		})
	defer imgSvc.Close(context.TODO())

	first := true
	engineMock := engine.InterfaceMock{
		FindFunc: func(engine.FindRequest) ([]store.Comment, error) {
			if first {
				first = false
				return nil, nil
			}
			return nil, fmt.Errorf("mockError")
		},
	}
	site1Req := engine.FindRequest{Locator: store.Locator{SiteID: "site1", URL: ""}, Sort: "time", Since: time.Time{}.Add(time.Second)}
	site2Req := engine.FindRequest{Locator: store.Locator{SiteID: "site2", URL: ""}, Sort: "time", Since: time.Time{}.Add(time.Second)}
	b := DataStore{Engine: &engineMock, EditDuration: 10 * time.Millisecond, ImageService: imgSvc}

	// one call without error and one with error
	err := b.ResubmitStagingImages([]string{"site1", "site2"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "problem finding comments for site site2: mockError")

	assert.Equal(t, 1, len(mockStore.InfoCalls()))
	assert.Equal(t, 2, len(engineMock.FindCalls()))
	assert.Equal(t, site1Req, engineMock.FindCalls()[0].Req)
	assert.Equal(t, site2Req, engineMock.FindCalls()[1].Req)
}

func TestService_alterComment(t *testing.T) {
	engineMock := engine.InterfaceMock{
		FlagFunc: func(engine.FlagRequest) (bool, error) {
			return false, nil
		},
	}
	svc := DataStore{Engine: &engineMock}

	r := svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"},
		Locator: store.Locator{URL: "http://example.com?foo=bar"}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", ID: "devid"},
		Locator: store.Locator{URL: "http://example.com?foo=bar"}}, r, "ip cleaned")
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"}},
		store.User{Name: "dev", ID: "devid", Admin: true})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"}}, r, "ip not cleaned")
	assert.Equal(t, 4, len(engineMock.FlagCalls()))
	assert.Equal(t, engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}, engineMock.FlagCalls()[0].Req)
	assert.Equal(t, engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}, engineMock.FlagCalls()[1].Req)
	assert.Equal(t, engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}, engineMock.FlagCalls()[2].Req)
	assert.Equal(t, engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}, engineMock.FlagCalls()[3].Req)

	first := true
	engineMock = engine.InterfaceMock{
		FlagFunc: func(engine.FlagRequest) (bool, error) {
			if first {
				first = false
				return false, nil
			}
			return true, nil
		},
	}
	svc = DataStore{Engine: &engineMock}
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid", Verified: true}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", ID: "devid", Verified: true}}, r, "verified set")
	assert.Equal(t, 2, len(engineMock.FlagCalls()))
	assert.Equal(t, engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}, engineMock.FlagCalls()[0].Req)
	assert.Equal(t, engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}, engineMock.FlagCalls()[1].Req)

	first = true
	engineMock = engine.InterfaceMock{
		FlagFunc: func(engine.FlagRequest) (bool, error) {
			if first {
				first = false
				return true, nil
			}
			return false, nil
		},
	}
	svc = DataStore{Engine: &engineMock}
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid", Verified: true},
		Locator: store.Locator{URL: "javascript:alert('XSS1')"}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", Verified: true, Blocked: true, ID: "devid"},
		Deleted: false}, r, "blocked")
	assert.Equal(t, 1, len(engineMock.FlagCalls()))
	assert.Equal(t, engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}, engineMock.FlagCalls()[0].Req)
}

func TestService_alterCommentsFlagCaching(t *testing.T) {
	t.Run("repeated user looked up once", func(t *testing.T) {
		engineMock := engine.InterfaceMock{
			FlagFunc: func(engine.FlagRequest) (bool, error) { return false, nil },
		}
		svc := DataStore{Engine: &engineMock}

		comments := make([]store.Comment, 0, 5)
		for i := range 5 {
			comments = append(comments, store.Comment{ID: fmt.Sprintf("c%d", i),
				User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}})
		}
		svc.alterComments(comments, store.User{ID: "u1"})

		// one Blocked + one Verified lookup for the single user, not two per comment
		assert.Equal(t, 2, len(engineMock.FlagCalls()), "5 comments by one user -> 2 flag lookups")
	})

	t.Run("distinct users looked up per user", func(t *testing.T) {
		engineMock := engine.InterfaceMock{
			FlagFunc: func(req engine.FlagRequest) (bool, error) {
				return req.Flag == engine.Blocked && req.UserID == "blocked", nil // "blocked" user is blocked
			},
		}
		svc := DataStore{Engine: &engineMock}

		comments := []store.Comment{
			{ID: "c1", User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}},
			{ID: "c2", User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}},
			{ID: "c3", User: store.User{ID: "blocked"}, Locator: store.Locator{SiteID: "site1"}},
			{ID: "c4", User: store.User{ID: "blocked"}, Locator: store.Locator{SiteID: "site1"}},
		}
		res := svc.alterComments(comments, store.User{ID: "admin", Admin: true})

		// u1: Blocked+Verified (2); blocked user: Blocked only, Verified skipped (1) = 3 total
		assert.Equal(t, 3, len(engineMock.FlagCalls()), "two distinct users -> 3 flag lookups")
		assert.True(t, res[2].User.Blocked && res[3].User.Blocked, "blocked user marked blocked")
		assert.False(t, res[0].User.Blocked, "u1 not blocked")
	})

	t.Run("flag read error is not cached", func(t *testing.T) {
		var blockedCalls int
		engineMock := engine.InterfaceMock{
			FlagFunc: func(req engine.FlagRequest) (bool, error) {
				if req.Flag == engine.Blocked {
					blockedCalls++
					if blockedCalls == 1 {
						return false, fmt.Errorf("transient flag read error")
					}
					return true, nil
				}
				return false, nil
			},
		}
		svc := DataStore{Engine: &engineMock}

		comments := []store.Comment{
			{ID: "c0", User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}},
			{ID: "c1", User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}},
			{ID: "c2", User: store.User{ID: "u1"}, Locator: store.Locator{SiteID: "site1"}},
		}
		res := svc.alterComments(comments, store.User{ID: "admin", Admin: true})

		// the errored first lookup must not be cached, so the next comment retries and
		// picks up the real blocked state; once it succeeds the result is cached
		assert.False(t, res[0].User.Blocked, "errored lookup treated as not blocked")
		assert.True(t, res[1].User.Blocked, "retry after error picks up blocked state")
		assert.True(t, res[2].User.Blocked, "successful read is cached")
		assert.Equal(t, 2, blockedCalls, "blocked retried once after the error, then cached")
	})
}

// a token's admin flag is set for the site it was issued for, and the open routes accept the token
// for every site of the instance, so on another site the viewer counts as admin only if the admin
// store says so for that site
func TestService_alterCommentAdminPerSite(t *testing.T) {
	engineMock := &engine.InterfaceMock{FlagFunc: func(engine.FlagRequest) (bool, error) { return false, nil }}
	comments := func() []store.Comment {
		return []store.Comment{
			{ID: "a1", User: store.User{ID: "u1", IP: "ip-a1"}, Locator: store.Locator{SiteID: "site-a"}},
			{ID: "b1", User: store.User{ID: "u1", IP: "ip-b1"}, Locator: store.Locator{SiteID: "site-b"}},
			{ID: "b2", User: store.User{ID: "u2", IP: "ip-b2"}, Locator: store.Locator{SiteID: "site-b"}},
		}
	}
	ips := func(cc []store.Comment) []string {
		res := make([]string, len(cc))
		for i, c := range cc {
			res[i] = c.User.IP
		}
		return res
	}
	adminOfA := store.User{ID: "adm", Name: "adm", Admin: true, SiteID: "site-a"}

	t.Run("admin of one site only", func(t *testing.T) {
		as := &admin.StoreMock{AdminsFunc: func(siteID string) ([]string, error) {
			if siteID == "site-a" {
				return []string{"adm"}, nil
			}
			return []string{"other"}, nil
		}}
		svc := DataStore{Engine: engineMock, AdminStore: as}

		res := svc.alterComments(comments(), adminOfA)
		assert.Equal(t, []string{"ip-a1", "", ""}, ips(res), "ip kept on the token's site only")
		require.Len(t, as.AdminsCalls(), 1, "other site's admins read once per listing")
		assert.Equal(t, "site-b", as.AdminsCalls()[0].SiteID)

		c := svc.alterComment(comments()[1], adminOfA)
		assert.Empty(t, c.User.IP, "single comment of the other site")
	})

	t.Run("shared admins see every site as today", func(t *testing.T) {
		as := &admin.StoreMock{AdminsFunc: func(string) ([]string, error) { return []string{"adm"}, nil }}
		svc := DataStore{Engine: engineMock, AdminStore: as}
		res := svc.alterComments(comments(), adminOfA)
		assert.Equal(t, []string{"ip-a1", "ip-b1", "ip-b2"}, ips(res))
	})

	t.Run("admin store error hides ip", func(t *testing.T) {
		as := &admin.StoreMock{AdminsFunc: func(string) ([]string, error) { return nil, errors.New("rpc failed") }}
		svc := DataStore{Engine: engineMock, AdminStore: as}
		res := svc.alterComments(comments(), adminOfA)
		assert.Equal(t, []string{"ip-a1", "", ""}, ips(res))
	})

	t.Run("basic-auth admin sees every site", func(t *testing.T) {
		as := &admin.StoreMock{AdminsFunc: func(string) ([]string, error) { return []string{"other"}, nil }}
		svc := DataStore{Engine: engineMock, AdminStore: as}
		res := svc.alterComments(comments(), store.User{ID: "admin", Name: "admin", Admin: true})
		assert.Equal(t, []string{"ip-a1", "ip-b1", "ip-b2"}, ips(res))
		assert.Empty(t, as.AdminsCalls(), "no lookup for a user without a site")
	})

	t.Run("non-admin sees no ip anywhere", func(t *testing.T) {
		as := &admin.StoreMock{AdminsFunc: func(string) ([]string, error) { return []string{"adm"}, nil }}
		svc := DataStore{Engine: engineMock, AdminStore: as}
		res := svc.alterComments(comments(), store.User{ID: "adm", SiteID: "site-a"})
		assert.Equal(t, []string{"", "", ""}, ips(res))
		assert.Empty(t, as.AdminsCalls(), "the token's flag is not widened by the store")
	})
}

func Benchmark_ServiceCreate(b *testing.B) {
	dbFile := path.Join(b.TempDir(), "test-remark42.db")

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: dbFile, SiteID: "radio-t"})
	svc := DataStore{Engine: boltStore, EditDuration: 50 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}
	require.NoError(b, err)
	defer func() { assert.NoError(b, svc.Close()) }()

	for i := 0; i < b.N; i++ {
		comment := store.Comment{
			ID:        "id-" + strconv.Itoa(i),
			Text:      `some text, <a href="http://radio-t.com">link</a>`,
			Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
			Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:      store.User{ID: "user1", Name: "user name"},
		}
		_, err = svc.Create(comment)
		require.NoError(b, err)
	}
}

func TestService_DoubleClose_Bolt(t *testing.T) {
	dbFile := path.Join(t.TempDir(), "test-remark42.db")

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: dbFile, SiteID: "radio-t"})
	svc := DataStore{Engine: boltStore, EditDuration: 50 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}
	require.NoError(t, err)
	assert.NoError(t, boltStore.Close())
	assert.NoError(t, boltStore.Close(), "second call should not result in panic or errors")
	assert.NoError(t, svc.Close())
	assert.NoError(t, svc.Close(), "second call should not result in panic or errors")
}

func TestService_DoubleClose_Static(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{})}
	assert.NoError(t, b.Close())
	// second call should not result in panic or errors
	assert.NoError(t, b.Close())
}

// makes new boltdb, put two records
func prepStoreEngine(t *testing.T) (e engine.Interface, teardown func()) {
	testDB := path.Join(t.TempDir(), "test.db")

	st := time.Now()
	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDB, SiteID: "radio-t"})
	assert.NoError(t, err)

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = boltStore.Create(comment)
	assert.NoError(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = boltStore.Create(comment)
	assert.NoError(t, err)
	t.Logf("prepared store engine in %v", time.Since(st))
	return boltStore, func() {
		assert.NoError(t, boltStore.Close())
	}
}

func getReq(locator store.Locator, commentID string) engine.GetRequest {
	return engine.GetRequest{
		Locator:   locator,
		CommentID: commentID,
	}
}
