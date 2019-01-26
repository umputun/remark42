package service

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark/backend/app/store/admin"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

var testDb = "/tmp/test-remark.db"

func TestService_CreateFromEmpty(t *testing.T) {
	defer os.Remove(testDb)
	ks := admin.NewStaticKeyStore("secret 123")
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: ks}
	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id)
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "text", res.Text)
	assert.True(t, time.Since(res.Timestamp).Seconds() < 1)
	assert.Equal(t, "user", res.User.ID)
	assert.Equal(t, "name", res.User.Name)
	assert.Equal(t, "23f97cf4d5c29ef788ca2bdd1c9e75656c0e4149", res.User.IP)
	assert.Equal(t, map[string]bool{}, res.Votes)
}

func TestService_CreateFromPartial(t *testing.T) {
	defer os.Remove(testDb)
	ks := admin.NewStaticKeyStore("secret 123")
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: ks}
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

	res, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, id)
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
	defer os.Remove(testDb)
	ks := admin.NewStaticKeyStore("secret 123")
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second})}
	comment := store.Comment{
		Text:      "text",
		Timestamp: time.Date(2018, 3, 25, 16, 34, 33, 0, time.UTC),
		Votes:     map[string]bool{"u1": true, "u2": false},
		User:      store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator:   store.Locator{URL: "https://radio-t.com/p/2018/12/29/podcast-630/", SiteID: "radio-t"},
	}
	id, err := b.Create(comment)
	assert.NoError(t, err)
	assert.True(t, id != "", id)

	res, err := b.Get(store.Locator{URL: "https://radio-t.com/p/2018/12/29/podcast-630/", SiteID: "radio-t"}, id)
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "Радио-Т 630", res.PostTitle)

	comment.PostTitle = "post blah"
	id, err = b.Create(comment)
	assert.NoError(t, err)
	res, err = b.Get(store.Locator{URL: "https://radio-t.com/p/2018/12/29/podcast-630/", SiteID: "radio-t"}, id)
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "post blah", res.PostTitle, "keep comment title")
}

func TestService_SetTitle(t *testing.T) {
	defer os.Remove(testDb)

	var titleEnable int32
	tss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&titleEnable) == 0 {
			w.WriteHeader(404)
		}
		if r.URL.String() == "/post1" {
			w.Write([]byte("<html><title>post1 blah 123</title><body> 2222</body></html>"))
			return
		}
		if r.URL.String() == "/post2" {
			w.Write([]byte("<html><title>post2 blah 123</title><body> 2222</body></html>"))
			return
		}
		w.WriteHeader(404)
	}))
	defer tss.Close()

	ks := admin.NewStaticKeyStore("secret 123")
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: ks,
		TitleExtractor: NewTitleExtractor(http.Client{Timeout: 5 * time.Second})}
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

	res, err := b.Get(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	assert.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, "", res.PostTitle)

	atomic.StoreInt32(&titleEnable, 1)
	c, err := b.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.NoError(t, err)
	assert.Equal(t, "post1 blah 123", c.PostTitle)

	b = DataStore{Interface: prepStoreEngine(t), AdminStore: ks}
	_, err = b.SetTitle(store.Locator{URL: tss.URL + "/post1", SiteID: "radio-t"}, id)
	require.EqualError(t, err, "no title extractor")
}

func TestService_Vote(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes, "no votes initially")

	c, err := b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, map[string]bool{"user1": true}, c.Votes, "user voted +")

	c, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user", true)
	assert.NotNil(t, err, "self-voting not allowed")

	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
	assert.NotNil(t, err, "double-voting rejected")
	assert.True(t, strings.HasPrefix(err.Error(), "user user1 already voted"))

	res, err = b.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, 1, res[0].Score)

	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", false)
	assert.Nil(t, err, "vote reset")
	res, err = b.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes, "vote reset ok")
}

func TestService_VoteLimit(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: 2}

	_, err := b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user2", true)
	assert.Nil(t, err)

	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user3", true)
	assert.Nil(t, err)

	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user4", true)
	assert.NotNil(t, err, "vote limit reached")
	assert.True(t, strings.HasPrefix(err.Error(), "maximum number of votes exceeded for comment id-1"))

	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-2", "user4", true)
	assert.Nil(t, err)
}

func TestService_VotesDisabled(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: 0}

	_, err := b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user2", true)
	assert.EqualError(t, err, "maximum number of votes exceeded for comment id-1")
}

func TestService_VoteAggressive(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	res, err := b.Last("radio-t", 0)
	require.Nil(t, err)
	t.Logf("%+v", res[0])
	assert.Equal(t, 3, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes, "no votes initially")

	// add a vote as user2
	_, err = b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user2", true)
	require.Nil(t, err)

	// crazy vote +1 as user1
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", true)
		}()
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0)
	require.NoError(t, err)

	t.Logf("%+v", res[0])
	assert.Equal(t, 3, len(res))
	assert.Equal(t, 2, res[0].Score, "add single +1")
	assert.Equal(t, 2, len(res[0].Votes), "made a single vote")

	// random +1/-1 result should be [0..2]
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val := rand.Intn(2) > 0
			b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, "user1", val)
		}()
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0)
	require.NoError(t, err)
	assert.Equal(t, 3, len(res))
	t.Logf("%+v %d", res[0], res[0].Score)
	assert.True(t, res[0].Score >= 0 && res[0].Score <= 2, "unexpected score %d", res[0].Score)
}

func TestService_VoteConcurrent(t *testing.T) {

	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"), MaxVotes: -1}

	comment := store.Comment{
		Text:    "text",
		User:    store.User{IP: "192.168.1.1", ID: "user", Name: "name"},
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)
	res, err := b.Last("radio-t", 0)
	require.Nil(t, err)

	// concurrent vote +1 as multiple users for the same comment
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, fmt.Sprintf("user1-%d", i), true)
		}()
	}
	wg.Wait()
	res, err = b.Last("radio-t", 0)
	require.NoError(t, err)
	assert.Equal(t, 100, res[0].Score, "should have 1000 score")
	assert.Equal(t, 100, len(res[0].Votes), "should have 1000 votes")
}

func TestService_VotePositive(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"),
		MaxVotes: -1, PositiveScore: true}

	_, err := b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user2", false)
	assert.EqualError(t, err, "minimal score reached for comment id-1")

	b = DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123"),
		MaxVotes: -1, PositiveScore: false}
	c, err := b.Vote(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "id-1", "user2", false)
	assert.Nil(t, err, "minimal score ignored")
	assert.Equal(t, -1, c.Score)
}

func TestService_Pin(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, res[0].Pin)

	err = b.SetPin(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, true)
	assert.Nil(t, err)

	c, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, true, c.Pin)

	err = b.SetPin(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, false)
	assert.Nil(t, err)
	c, err = b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, false, c.Pin)
}

func TestService_EditComment(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	comment, err := b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.Nil(t, err)
	assert.Equal(t, "my edit", comment.Edit.Summary)
	assert.Equal(t, "xxx", comment.Text)
	assert.Equal(t, "yyy", comment.Orig)

	c, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, "my edit", c.Edit.Summary)
	assert.Equal(t, "xxx", c.Text)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.Nil(t, err, "allow second edit")
}

func TestService_DeleteComment(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID, EditRequest{Delete: true})
	assert.Nil(t, err)

	c, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID)
	assert.Nil(t, err)
	assert.True(t, c.Deleted)
	t.Logf("%+v", c)
}

func TestService_EditCommentDurationFailed(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), EditDuration: 100 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Nil(t, res[0].Edit)

	time.Sleep(time.Second)

	_, err = b.EditComment(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID,
		EditRequest{Orig: "yyy", Text: "xxx", Summary: "my edit"})
	assert.NotNil(t, err)
}

func TestService_EditCommentReplyFailed(t *testing.T) {
	defer os.Remove(testDb)
	b := DataStore{Interface: prepStoreEngine(t), AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.Last("radio-t", 0)
	t.Logf("%+v", res[1])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
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
		e := b.ValidateComment(&tt.inp)
		if tt.err == nil {
			assert.Nil(t, e, "check #%d", n)
			continue
		}
		assert.EqualError(t, tt.err, e.Error(), "check #%d", n)
	}
}

func TestService_Counts(t *testing.T) {
	defer os.Remove(testDb)
	b := prepStoreEngine(t) // two comments for https://radio-t.com

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "123456",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.Nil(t, err)

	svc := DataStore{Interface: b}
	res, err := svc.Counts("radio-t", []string{"https://radio-t.com/2"})
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1}}, res)

	res, err = svc.Counts("radio-t", []string{"https://radio-t.com", "https://radio-t.com/2", "blah"})
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{
		{URL: "https://radio-t.com", Count: 2},
		{URL: "https://radio-t.com/2", Count: 1},
		{URL: "blah", Count: 0},
	}, res)
}

func TestService_GetMetas(t *testing.T) {
	defer os.Remove(testDb)
	// two comments for https://radio-t.com
	b := DataStore{Interface: prepStoreEngine(t), EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}

	um, pm, err := b.Metas("radio-t")
	require.NoError(t, err)
	assert.Equal(t, 0, len(um))
	assert.Equal(t, 0, len(pm))

	assert.NoError(t, b.SetVerified("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user1", true, time.Hour))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true, time.Hour))
	assert.NoError(t, b.SetReadOnly(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, true))

	um, pm, err = b.Metas("radio-t")
	require.NoError(t, err)

	assert.Equal(t, 2, len(um))
	assert.Equal(t, "user1", um[0].ID)
	assert.Equal(t, true, um[0].Verified)
	assert.Equal(t, true, um[0].Blocked.Status)
	assert.Equal(t, false, um[1].Verified)
	assert.Equal(t, true, um[1].Blocked.Status)

	assert.Equal(t, 1, len(pm))
	assert.Equal(t, "https://radio-t.com", pm[0].URL)
	assert.Equal(t, true, pm[0].ReadOnly)
}

func TestService_SetMetas(t *testing.T) {
	defer os.Remove(testDb)
	// two comments for https://radio-t.com
	b := DataStore{Interface: prepStoreEngine(t), EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticKeyStore("secret 123")}
	umetas := []UserMetaData{}
	pmetas := []PostMetaData{}
	err := b.SetMetas("radio-t", umetas, pmetas)
	assert.NoError(t, err, "empty metas")

	um1 := UserMetaData{ID: "user1", Verified: true}
	um2 := UserMetaData{ID: "user2"}
	um2.Blocked.Status = true
	um2.Blocked.Until = time.Now().AddDate(0, 1, 1)

	pmetas = []PostMetaData{{URL: "https://radio-t.com", ReadOnly: true}}
	err = b.SetMetas("radio-t", []UserMetaData{um1, um2}, pmetas)
	assert.NoError(t, err)

	assert.True(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}))
	assert.True(t, b.IsVerified("radio-t", "user1"))
	assert.True(t, b.IsBlocked("radio-t", "user2"))
}

func TestService_IsAdmin(t *testing.T) {
	defer os.Remove(testDb)
	// two comments for https://radio-t.com
	b := DataStore{Interface: prepStoreEngine(t), EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", []string{"user2"}, "user@email.com")}

	assert.False(t, b.IsAdmin("radio-t", "user1"))
	assert.True(t, b.IsAdmin("radio-t", "user2"))
}

func TestService_HasReplies(t *testing.T) {
	defer os.Remove(testDb)

	// two comments for https://radio-t.com, no reply
	b := DataStore{Interface: prepStoreEngine(t), EditDuration: 100 * time.Millisecond,
		AdminStore: admin.NewStaticStore("secret 123", []string{"user2"}, "user@email.com")}

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

// makes new boltdb, put two records
func prepStoreEngine(t *testing.T) engine.Interface {
	os.Remove(testDb)

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/test-remark.db", SiteID: "radio-t"})
	assert.Nil(t, err)
	b := boltStore

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
