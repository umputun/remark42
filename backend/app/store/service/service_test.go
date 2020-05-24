package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
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
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Engine.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id))
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "text", res.Text)
	assert.True(t, time.Since(res.Timestamp).Seconds() < 1)
	assert.Equal(t, "user", res.User.ID)
	assert.Equal(t, "name", res.User.Name)
	assert.Equal(t, "23f97cf4d5c29ef788ca2bdd1c9e75656c0e4149", res.User.IP)
	assert.Equal(t, map[string]bool(nil), res.Votes)
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
		User:      store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
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

func TestService_CreateFromPartialWithTitle(t *testing.T) {
	ks := admin.NewStaticKeyStore("secret 123")
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second})}
	defer b.Close()

	postPath := "/post/42"
	postTitle := "Post Title 42"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == postPath {
			_, err := w.Write([]byte(fmt.Sprintf("<html><title>%s</title><body>...</body></html>", postTitle)))
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

func TestService_SetTitle(t *testing.T) {

	var titleEnable int32
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&titleEnable) == 0 {
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
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second})}
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

	atomic.StoreInt32(&titleEnable, 1)
	c, err := b.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.NoError(t, err)
	assert.Equal(t, "post1 blah 123", c.PostTitle)

	bErr := DataStore{Engine: eng, AdminStore: ks}
	defer bErr.Close()
	_, err = bErr.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.EqualError(t, err, "no title extractor")
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
	// check result as user2
	c, err = b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, store.User{ID: "user2"})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, 0, c.Vote, "can't see other user vote result")
	assert.Nil(t, c.Votes)

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
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: "user1", Val: true})
		}()
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{ID: "user1"})
	require.NoError(t, err)

	t.Logf("%+v", res[0])
	require.Equal(t, 3, len(res))
	assert.Equal(t, 2, res[0].Score, "add single +1")
	assert.Equal(t, 1, res[0].Vote, "user1 voted +1")
	assert.Equal(t, 0, len(res[0].Votes), "votes hidden")

	// random +1/-1 result should be [0..2]
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val := rand.Intn(2) > 0
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: "user1", Val: val})
		}()
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
	for i := 0; i < 100; i++ {
		wg.Add(1)
		ii := i
		go func() {
			defer wg.Done()
			_, _ = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: res[0].ID,
				UserID: fmt.Sprintf("user1-%d", ii), Val: true})
		}()
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Equal(t, 100, res[0].Score, "should have 100 score")
	assert.Equal(t, 0, len(res[0].Votes), "should hide votes")
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

	b.PositiveScore = false // allow negative voting
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

func TestService_VoteSameIP(t *testing.T) {

	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, AdminStore: admin.NewStaticKeyStore("secret 123"),
		MaxVotes: -1}
	b.RestrictSameIPVotes.Enabled = true

	c, err := b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user2", UserIP: "123", Val: true})
	assert.NoError(t, err)
	assert.Equal(t, 1, c.Score, "should have 1 score")

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user3", UserIP: "123", Val: true})
	assert.EqualError(t, err, "the same ip cce61be6e0a692420ae0de31dceca179123c3b8a already voted for id-2")
	assert.Equal(t, 1, c.Score, "still have 1 score")

	c, err = b.Vote(VoteReq{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, CommentID: "id-2",
		UserID: "user3", UserIP: "123", Val: false})
	assert.NoError(t, err)
	assert.Equal(t, 0, c.Score, "reset to 0 score, opposite vote allowed")
}

func TestService_VoteSameIPWithDuration(t *testing.T) {

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
	assert.EqualError(t, err, "the same ip cce61be6e0a692420ae0de31dceca179123c3b8a already voted for id-2")
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
		tt := tt
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

	time.Sleep(time.Second)

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
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}
	_, err = b.Create(reply)
	assert.NoError(t, err)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.EqualError(t, err, "parent comment with reply can't be edited, id-1")
}

func TestService_ValidateComment(t *testing.T) {

	b := DataStore{MaxCommentSize: 2000, AdminStore: admin.NewStaticKeyStore("secret 123")}
	longText := fmt.Sprintf("%4000s", "X")

	tbl := []struct {
		inp store.Comment
		err error
	}{
		{inp: store.Comment{}, err: errors.New("empty comment text")},
		{inp: store.Comment{Orig: "something blah", User: store.User{ID: "myid", Name: "name"}}, err: nil},
		{inp: store.Comment{Orig: "something blah", User: store.User{ID: "myid"}}, err: errors.New("empty user info")},
		{inp: store.Comment{Orig: longText, User: store.User{ID: "myid", Name: "name"}}, err: errors.New("comment text exceeded max allowed size 2000 (4000)")},
	}

	for n, tt := range tbl {
		err := b.ValidateComment(&tt.inp)
		if tt.err == nil {
			assert.NoError(t, err, "check #%d", n)
			continue
		}
		require.Error(t, err)
		assert.EqualError(t, tt.err, err.Error(), "check #%d", n)
	}
}

func TestService_Counts(t *testing.T) {

	b, teardown := prepStoreEngine(t) // two comments for https://radio-t.com
	defer teardown()

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "123456",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
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

func TestService_UserDetailsOperations(t *testing.T) {

	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}

	// add single valid entry
	result, err := b.SetUserEmail("radio-t", "u1", "test@example.com")
	assert.NoError(t, err, "No error inserting entry expected")
	assert.Equal(t, "test@example.com", result)

	// read valid entry back
	result, err = b.GetUserEmail("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Equal(t, "test@example.com", result)

	// delete existing entry
	err = b.DeleteUserDetail("radio-t", "u1", engine.UserEmail)
	assert.NoError(t, err, "No error deleting entry expected")

	// read deleted entry
	result, err = b.GetUserEmail("radio-t", "u1")
	assert.NoError(t, err, "No error reading entry expected")
	assert.Empty(t, result)

	// insert entry with invalid site_id
	result, err = b.SetUserEmail("bad-site", "u3", "does_not_matter@example.com")
	assert.Error(t, err, "Site not found")
	assert.Empty(t, result)

	// read entry with invalid site_id
	result, err = b.GetUserEmail("bad-site", "u3")
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
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", []string{"radio-t"}, []string{"user2"}, "user@email.com")}
	defer b.Close()

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}

	assert.False(t, b.HasReplies(comment))

	reply := store.Comment{
		ID:        "123456",
		ParentID:  "id-1",
		Text:      "some text",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user2", Name: "user name 2"},
	}
	_, err := b.Create(reply)
	assert.NoError(t, err)
	assert.True(t, b.HasReplies(comment))
}

func TestService_UserReplies(t *testing.T) {

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

	_, err := b.Create(c1)
	require.NoError(t, err)
	_, err = b.Create(c2)
	require.NoError(t, err)
	_, err = b.Create(c3)
	require.NoError(t, err)
	_, err = b.Create(c4)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	_, err = b.Create(c5)
	require.NoError(t, err)

	cc, u, err := b.UserReplies("radio-t", "u1", 10, time.Hour)
	assert.NoError(t, err)
	require.Equal(t, 3, len(cc), "3 replies to u1")
	assert.Equal(t, "developer one u1", u)

	cc, u, err = b.UserReplies("radio-t", "u1", 10, time.Millisecond*199)
	assert.NoError(t, err)
	require.Equal(t, 1, len(cc), "1 reply to u1 in last 200ms")
	assert.Equal(t, "developer one u1", u)

	cc, u, err = b.UserReplies("radio-t", "u2", 10, time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(cc), "0 replies to u2")
	assert.Equal(t, "developer one u2", u)

	cc, u, err = b.UserReplies("radio-t", "uxxx", 10, time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(cc), "0 replies to uxxx")
	assert.Equal(t, "", u)

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
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
		Score:     1,
		Votes:     map[string]bool{"id-1": true, "id-2": true, "123456": false},
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
		time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local))
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

	info, err := b.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com", info.URL)
	assert.Equal(t, 2, info.Count)
	assert.False(t, info.ReadOnly)
	assert.True(t, info.LastTS.After(info.FirstTS))

	time.Sleep(1 * time.Second) // make post RO in 1sec
	info, err = b.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 1)
	require.NoError(t, err)
	assert.Equal(t, "https://radio-t.com", info.URL)
	assert.True(t, info.ReadOnly)
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
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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

func TestService_List(t *testing.T) {

	// two comments for https://radio-t.com, no reply
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", nil, []string{"user2"}, "user@email.com")}

	// add one more for user2
	comment := store.Comment{
		ID:        "id-3",
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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
	assert.Equal(t, time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local), res[0].FirstTS)
	assert.Equal(t, time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local), res[0].LastTS)

	assert.Equal(t, "https://radio-t.com", res[1].URL)
	assert.Equal(t, 2, res[1].Count)
	assert.Equal(t, time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local), res[1].FirstTS)
	assert.Equal(t, time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local), res[1].LastTS)
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
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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
		Timestamp: time.Date(2018, 12, 20, 15, 18, 22, 0, time.Local),
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

	mockStore := image.MockStore{}
	mockStore.On("Commit", "dev/pic1.png").Once().Return(nil)
	mockStore.On("Commit", "dev/pic2.png").Once().Return(nil)
	imgSvc := image.NewService(&mockStore,
		image.ServiceParams{
			EditDuration: 50 * time.Millisecond,
			ImageAPI:     "/",
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
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Engine.Create(c) // create directly with engine, doesn't call submitImages
	assert.NoError(t, err)

	b.submitImages(c.Locator, c.ID)
	time.Sleep(250 * time.Millisecond)
	mockStore.AssertNumberOfCalls(t, "Commit", 2)
}

func TestService_ResubmitStagingImages(t *testing.T) {
	mockStore := image.MockStore{}
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
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Engine.Create(comment)
	require.NoError(t, err)

	// resubmit single comment with three images, of which two are in staging storage
	mockStore.On("Info").Once().Return(image.StoreInfo{FirstStagingImageTS: time.Time{}.Add(time.Second)}, nil)
	err = b.ResubmitStagingImages([]string{"radio-t"})
	assert.NoError(t, err)

	// wait for Submit goroutine to commit image
	mockStore.On("Commit", "dev_user/bqf122eq9r8ad657n3ng").Once().Return(nil)
	mockStore.On("Commit", "dev_user/bqf321eq9r8ad657n3ng").Once().Return(nil)
	mockStore.On("Commit", "cached_images/12318fbd4c55e9d177b8b5ae197bc89c5afd8e07-a41fcb00643f28d700504256ec81cbf2e1aac53e").Once().Return(nil)
	time.Sleep(time.Millisecond * 100)

	mockStore.AssertNumberOfCalls(t, "Info", 1)
	mockStore.AssertNumberOfCalls(t, "Commit", 3)

	// empty answer
	mockStoreEmpty := image.MockStore{}
	imgSvcEmpty := image.NewService(&mockStoreEmpty,
		image.ServiceParams{
			EditDuration: 10 * time.Millisecond,
			ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
		})
	defer imgSvcEmpty.Close(context.TODO())
	bEmpty := DataStore{Engine: eng, EditDuration: 10 * time.Millisecond, ImageService: imgSvcEmpty}

	// resubmit receive empty timestamp and should do nothing
	mockStoreEmpty.On("Info").Once().Return(image.StoreInfo{FirstStagingImageTS: time.Time{}}, nil)
	err = bEmpty.ResubmitStagingImages([]string{"radio-t", "non_existent"})
	assert.NoError(t, err)

	mockStoreEmpty.AssertNumberOfCalls(t, "Info", 1)

	// 	error from image storage
	mockStoreError := image.MockStore{}
	imgSvcError := image.NewService(&mockStoreError,
		image.ServiceParams{
			EditDuration: 10 * time.Millisecond,
			ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
		})
	defer imgSvcError.Close(context.TODO())
	bError := DataStore{Engine: eng, EditDuration: 10 * time.Millisecond, ImageService: imgSvcError}

	// resubmit will receive error from image storage and should return it
	mockStoreError.On("Info").Once().Return(image.StoreInfo{}, errors.New("mock_err"))
	err = bError.ResubmitStagingImages([]string{"radio-t"})
	assert.EqualError(t, err, "mock_err")

	mockStoreError.AssertNumberOfCalls(t, "Info", 1)
}

func TestService_ResubmitStagingImages_EngineError(t *testing.T) {
	mockStore := image.MockStore{}
	imgSvc := image.NewService(&mockStore,
		image.ServiceParams{
			EditDuration: 10 * time.Millisecond,
			ImageAPI:     "http://127.0.0.1:8080/api/v1/picture/",
		})
	defer imgSvc.Close(context.TODO())

	engineMock := engine.MockInterface{}
	site1Req := engine.FindRequest{Locator: store.Locator{SiteID: "site1", URL: ""}, Sort: "time", Since: time.Time{}.Add(time.Second)}
	site2Req := engine.FindRequest{Locator: store.Locator{SiteID: "site2", URL: ""}, Sort: "time", Since: time.Time{}.Add(time.Second)}
	engineMock.On("Find", site1Req).Return(nil, nil)
	engineMock.On("Find", site2Req).Return(nil, errors.New("mockError"))
	b := DataStore{Engine: &engineMock, EditDuration: 10 * time.Millisecond, ImageService: imgSvc}

	// One call without error and one with error
	mockStore.On("Info").Once().Return(image.StoreInfo{FirstStagingImageTS: time.Time{}.Add(time.Second)}, nil)
	err := b.ResubmitStagingImages([]string{"site1", "site2"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "problem finding comments for site site2: mockError")

	mockStore.AssertNumberOfCalls(t, "Info", 1)
}

func TestService_alterComment(t *testing.T) {

	engineMock := engine.MockInterface{}
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}).Return(false, nil)
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}).Return(false, nil)
	svc := DataStore{Engine: &engineMock}

	r := svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", ID: "devid"}}, r, "ip cleaned")
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"}},
		store.User{Name: "dev", ID: "devid", Admin: true})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid"}}, r, "ip not cleaned")

	engineMock = engine.MockInterface{}
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}).Return(false, nil)
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}).Return(true, nil)
	svc = DataStore{Engine: &engineMock}
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid", Verified: true}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", ID: "devid", Verified: true}}, r, "verified set")

	engineMock = engine.MockInterface{}
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Blocked, UserID: "devid"}).Return(true, nil)
	engineMock.On("Flag", engine.FlagRequest{Flag: engine.Verified, UserID: "devid"}).Return(false, nil)
	svc = DataStore{Engine: &engineMock}
	r = svc.alterComment(store.Comment{ID: "123", User: store.User{IP: "127.0.0.1", ID: "devid", Verified: true}},
		store.User{Name: "dev", ID: "devid", Admin: false})
	assert.Equal(t, store.Comment{ID: "123", User: store.User{IP: "", Verified: true, Blocked: true, ID: "devid"},
		Deleted: false}, r, "blocked")
}

func Benchmark_ServiceCreate(b *testing.B) {
	dbFile := fmt.Sprintf("%s/test-remark42-%d.db", os.TempDir(), rand.Intn(9999999999))
	defer func() { _ = os.Remove(dbFile) }()

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: dbFile, SiteID: "radio-t"})
	svc := DataStore{Engine: boltStore, EditDuration: 50 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}
	require.NoError(b, err)
	defer func() { assert.NoError(b, svc.Close()) }()

	for i := 0; i < b.N; i++ {
		comment := store.Comment{
			ID:        "id-" + strconv.Itoa(i),
			Text:      `some text, <a href="http://radio-t.com">link</a>`,
			Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
			Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
			User:      store.User{ID: "user1", Name: "user name"},
		}
		_, err = svc.Create(comment)
		require.NoError(b, err)
	}
}

// makes new boltdb, put two records
func prepStoreEngine(t *testing.T) (e engine.Interface, teardown func()) {
	testDBLoc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err)
	testDB := path.Join(testDBLoc, "test.db")
	_ = os.Remove(testDB)

	st := time.Now()
	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: testDB, SiteID: "radio-t"})
	assert.NoError(t, err)

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = boltStore.Create(comment)
	assert.NoError(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = boltStore.Create(comment)
	assert.NoError(t, err)
	t.Logf("prepared store engine in %v", time.Since(st))
	return boltStore, func() {
		assert.NoError(t, boltStore.Close())
		_ = os.Remove(testDB)
	}
}

func getReq(locator store.Locator, commentID string) engine.GetRequest {
	return engine.GetRequest{
		Locator:   locator,
		CommentID: commentID,
	}
}
