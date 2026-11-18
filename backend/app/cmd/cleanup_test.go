package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
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
	r := http.NewServeMux()
	cleanupRoutes(t, r, nil)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_", "--admin-passwd=secret")
	posts, err := cmd.postsInRange(t.Context(), "20181218", "20181219")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts))

	posts, err = cmd.postsInRange(t.Context(), "", "")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(posts))

	_, err = cmd.postsInRange(t.Context(), "xxx", "yyy")
	assert.Error(t, err)
}

func TestCleanup_listComments(t *testing.T) {
	r := http.NewServeMux()
	cleanupRoutes(t, r, nil)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_", "--admin-passwd=secret")

	comments, err := cmd.listComments(t.Context(), "http://test.com/post1")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(comments))

	comments, err = cmd.listComments(t.Context(), "http://test.com/post2")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments))

	comments, err = cmd.listComments(t.Context(), "http://test.com/post-bad")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments))
}

func TestCleanup_ExecuteSpam(t *testing.T) {
	cleaned := cleanedComments{}
	r := http.NewServeMux()
	cleanupRoutes(t, r, &cleaned)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--bword=bad2", "--buser=bu_",
		"--from=20181217", "--to=20181218", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	assert.NoError(t, err)
	t.Logf("deleted %+v", cleaned.ids)
	assert.Equal(t, []string{"/api/v1/admin/comment/1", "/api/v1/admin/comment/3", "/api/v1/admin/comment/11"}, cleaned.ids)
}

func TestCleanup_ExecuteTitle(t *testing.T) {
	titledComments := cleanedComments{}
	r := http.NewServeMux()
	cleanupRoutes(t, r, &titledComments)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--title", "--from=20181217", "--to=20181218", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	assert.NoError(t, err)
	t.Logf("set titles for %+v", titledComments.ids)
	assert.Equal(t, []string{"/api/v1/admin/title/1", "/api/v1/admin/title/2", "/api/v1/admin/title/3", "/api/v1/admin/title/11"}, titledComments.ids)
}

func TestCleanup_ExecuteFailsWhenCommentsUnreadable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/list", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"url":"http://test.com/post1","first_time":"2018-12-17T10:00:00Z","last_time":"2018-12-17T10:30:00Z"}]`))
	})
	mux.HandleFunc("/api/v1/find", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "store failed", http.StatusInternalServerError)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	require.Error(t, err, "a post whose comments could not be read fails the run")
	assert.Contains(t, err.Error(), "1 of 1 posts")
}

func TestCleanup_ExecuteHonoursTimeout(t *testing.T) {
	r := http.NewServeMux()
	cleanupRoutes(t, r, &cleanedComments{})
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	// a timeout that has run out before the first request
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--admin-passwd=secret", "--timeout=1ns")
	err := cmd.Execute(nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// the later requests take the same context
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = cmd.listComments(ctx, "http://test.com/post1")
	require.ErrorIs(t, err, context.Canceled)
	c := store.Comment{ID: "1", Locator: store.Locator{URL: "http://test.com/post1"}}
	require.ErrorIs(t, cmd.deleteComment(ctx, c), context.Canceled)
	require.ErrorIs(t, cmd.setTitle(ctx, c), context.Canceled)
}

func TestCleanup_ExecuteFailsWhenTimeoutHitsLastPost(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/list", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"url":"http://test.com/post1","first_time":"2018-12-17T10:00:00Z","last_time":"2018-12-17T10:30:00Z"}]`))
	})
	mux.HandleFunc("/api/v1/find", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"comments":[{"id":"1","text":"bad1 bad2 http://xyz.com","user":{"id":"user"}}]}`))
	})
	// the delete of the spam comment found in the last post outlasts the timeout
	mux.HandleFunc("/api/v1/admin/comment/{id}", func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--bword=bad1", "--bword=bad2", "--admin-passwd=secret", "--timeout=500ms")
	err := cmd.Execute(nil)
	require.ErrorIs(t, err, context.DeadlineExceeded, "a run cut short by the timeout is not a success")
}

func TestCleanup_ExecuteClearsSecretsFromEnv(t *testing.T) {
	t.Setenv("SECRET", "123456")
	t.Setenv("ADMIN_PASSWD", "secret")
	r := http.NewServeMux()
	cleanupRoutes(t, r, &cleanedComments{})
	ts := httptest.NewServer(r)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--dry")
	require.NoError(t, cmd.Execute(nil))
	for _, name := range []string{"SECRET", "ADMIN_PASSWD"} {
		_, ok := os.LookupEnv(name)
		assert.False(t, ok, "%s is cleared from the environment", name)
	}
}

func cleanupRoutes(t *testing.T, r *http.ServeMux, c *cleanedComments) {
	r.HandleFunc("/api/v1/list", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "site=remark&limit=10000", r.URL.RawQuery)
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
		assert.NoError(t, json.NewEncoder(w).Encode(list))
	})

	r.HandleFunc("/api/v1/find", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "remark", r.URL.Query().Get("site"))
		assert.Equal(t, "plain", r.URL.Query().Get("format"))

		commentsWithInfo := struct {
			Comments []store.Comment `json:"comments"`
			Info     store.PostInfo  `json:"info"`
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

		assert.NoError(t, json.NewEncoder(w).Encode(commentsWithInfo))
	})

	r.HandleFunc("/api/v1/admin/comment/{id}", func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		t.Log("delete ", r.URL.Path)
		c.lock.Lock()
		c.ids = append(c.ids, r.URL.Path)
		c.lock.Unlock()
	})

	r.HandleFunc("/api/v1/admin/title/{id}", func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		t.Log("title for ", r.URL.Path)
		c.lock.Lock()
		c.ids = append(c.ids, r.URL.Path)
		c.lock.Unlock()
	})
}

func TestCleanup_RequestsKeepPostURLQuery(t *testing.T) {
	const postURL = "https://example.com/post?id=1&lang=en"
	const site = "site&x"
	type received struct{ method, path, site, url, format, user string }
	var lock sync.Mutex
	var got []received
	record := func(r *http.Request) {
		lock.Lock()
		defer lock.Unlock()
		user, _, _ := r.BasicAuth()
		got = append(got, received{method: r.Method, path: r.URL.Path, site: r.URL.Query().Get("site"),
			url: r.URL.Query().Get("url"), format: r.URL.Query().Get("format"), user: user})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/find", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		_, _ = w.Write([]byte(`{"comments":[]}`))
	})
	mux.HandleFunc("/api/v1/admin/comment/{id}", func(_ http.ResponseWriter, r *http.Request) { record(r) })
	mux.HandleFunc("/api/v1/admin/title/{id}", func(_ http.ResponseWriter, r *http.Request) { record(r) })
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cmd := CleanupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site="+site, "--admin-passwd=secret")

	_, err := cmd.listComments(t.Context(), postURL)
	require.NoError(t, err)
	c := store.Comment{ID: "1", Locator: store.Locator{URL: postURL}}
	require.NoError(t, cmd.deleteComment(t.Context(), c))
	require.NoError(t, cmd.setTitle(t.Context(), c))

	assert.Equal(t, []received{
		{method: "GET", path: "/api/v1/find", site: site, url: postURL, format: "plain"},
		{method: "DELETE", path: "/api/v1/admin/comment/1", site: site, url: postURL, format: "plain", user: "admin"},
		{method: "PUT", path: "/api/v1/admin/title/1", site: site, url: postURL, format: "plain", user: "admin"},
	}, got, "site and post URL arrive intact, including the post URL's own query")
}
