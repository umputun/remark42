package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi"
	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

type cleanedComments struct {
	ids  []string
	lock sync.Mutex
}

func TestCleanup_IsSpam(t *testing.T) {
	cc := CleanupCommand{
		BadWords: []string{"bad1", "bad2", "very bad", "xyz"},
		BadUsers: []string{"bu_"},
	}

	tbl := []struct {
		text      string
		user      string
		score     int
		isSpam    bool
		spamScore float64
		name      string
	}{
		{"", "", 1, false, 0, "empty passes"},
		{"one very bad two blah bad1 bad2 http://xyz.com", "bu_user", 0, true, 90, "3badwords link 0score baduser"},
		{"one very bad two blah bad1 bad2", "bu_user", 0, true, 67.5, "3 bad words 1score baduser"},
		{"bad1 bad2 xyz very bad", "bu_user", 0, true, 80, "4badwords 0score baduser"},
		{"bad1 bad2 xyz very bad", "user", 0, true, 70, "4badwords 0score"},
		{"bad1 bad2 xyz very bad", "user", 1, false, 50, "4badwords 1score"},
		{"bad1 test 12345", "user", 0, false, 32.5, "1badwords 0score"},
	}

	for n, tt := range tbl {
		checkName := fmt.Sprintf("check-%d-%s", n, tt.name)
		t.Run(checkName, func(t *testing.T) {
			c := store.Comment{ID: checkName, Text: tt.text, Score: tt.score}
			c.User.ID = tt.user
			r, score := cc.isSpam(c)
			assert.Equal(t, tt.isSpam, r)
			assert.InDelta(t, tt.spamScore, score, 0.01)
		})
	}
}

func TestCleanup_postsInRange(t *testing.T) {

	r := chi.NewRouter()
	cleanupRoutes(t, r, nil)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_", "--admin-passwd=secret"})
	require.Nil(t, err)
	posts, err := cmd.postsInRange("20181218", "20181219")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts))

	posts, err = cmd.postsInRange("", "")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	_, err = cmd.postsInRange("xxx", "yyy")
	assert.NotNil(t, err)
}

func TestCleanup_listComments(t *testing.T) {
	r := chi.NewRouter()
	cleanupRoutes(t, r, nil)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_", "--admin-passwd=secret"})
	require.Nil(t, err)

	comments, err := cmd.listComments("http://test.com/post1")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments))

	comments, err = cmd.listComments("http://test.com/post2")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments))

	comments, err = cmd.listComments("http://test.com/post-bad")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments))
}

func TestCleanup_ExecuteSpam(t *testing.T) {
	cleaned := cleanedComments{}
	r := chi.NewRouter()
	cleanupRoutes(t, r, &cleaned)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_",
		"--from=20181217", "--to=20181218", "--admin-passwd=secret"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
	t.Logf("deleted %+v", cleaned.ids)
	assert.Equal(t, []string{"/api/v1/admin/comment/1", "/api/v1/admin/comment/3", "/api/v1/admin/comment/11"}, cleaned.ids)
}

func TestCleanup_ExecuteTitle(t *testing.T) {
	titledComments := cleanedComments{}
	r := chi.NewRouter()
	cleanupRoutes(t, r, &titledComments)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--title", "--from=20181217", "--to=20181218", "--admin-passwd=secret"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
	t.Logf("set titles for %+v", titledComments.ids)
	assert.Equal(t, []string{"/api/v1/admin/title/1", "/api/v1/admin/title/2", "/api/v1/admin/title/3", "/api/v1/admin/title/11"}, titledComments.ids)
}

func cleanupRoutes(t *testing.T, r *chi.Mux, c *cleanedComments) {
	r.HandleFunc("/api/v1/list", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "site=remark&limit=10000", r.URL.RawQuery)
		list := []store.PostInfo{
			{
				URL:     "http://test.com/post1",
				FirstTS: time.Date(2018, 12, 17, 10, 0, 0, 0, time.Local),
				LastTS:  time.Date(2018, 12, 17, 10, 30, 0, 0, time.Local),
			},
			{
				URL:     "http://test.com/post2",
				FirstTS: time.Date(2018, 12, 18, 10, 0, 0, 0, time.Local),
				LastTS:  time.Date(2018, 12, 18, 10, 30, 0, 0, time.Local),
			},
			{
				URL:     "http://test.com/post3",
				FirstTS: time.Date(2018, 12, 19, 10, 0, 0, 0, time.Local),
				LastTS:  time.Date(2018, 12, 19, 10, 30, 0, 0, time.Local),
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(list))
	}))

	r.HandleFunc("/api/v1/find", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "remark", r.URL.Query().Get("site"))
		require.Equal(t, "plain", r.URL.Query().Get("format"))

		commentsWithInfo := struct {
			Comments []store.Comment `json:"comments"`
			Info     store.PostInfo  `json:"info,omitempty"`
		}{}

		switch r.URL.Query().Get("url") {
		case "http://test.com/post1":
			commentsWithInfo.Comments = []store.Comment{
				{ID: "1", Text: "one very bad two blah bad1 bad2 http://xyz.com", Score: 0, User: store.User{ID: "bu_user"}},
				{ID: "2", Text: "good one http://xyz.com", Score: 1, User: store.User{ID: "bu_user"}},
				{ID: "3", Text: "http://xyz.com bad1 bad2", Score: 0, User: store.User{ID: "user"}},
			}
		case "http://test.com/post2":
			commentsWithInfo.Comments = []store.Comment{
				{ID: "11", Text: "one very bad two blah bad1 bad2 http://xyz.com", Score: 0, User: store.User{ID: "bu_user"}},
			}
		case "http://test.com/post3":
			commentsWithInfo.Comments = []store.Comment{}
		}

		require.NoError(t, json.NewEncoder(w).Encode(commentsWithInfo))
	}))

	r.HandleFunc("/api/v1/admin/comment/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "DELETE", r.Method)
		t.Log("delete ", r.URL.Path)
		c.lock.Lock()
		c.ids = append(c.ids, r.URL.Path)
		c.lock.Unlock()
	}))

	r.HandleFunc("/api/v1/admin/title/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "PUT", r.Method)
		t.Log("title for ", r.URL.Path)
		c.lock.Lock()
		c.ids = append(c.ids, r.URL.Path)
		c.lock.Unlock()
	}))

}
