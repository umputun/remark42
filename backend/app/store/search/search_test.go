package search

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-pkgz/syncs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func createTestServiceInDir(t *testing.T, sites []string, idxPath string, keepIdx bool) (searcher *Service, teardown func()) {
	searcher, err := NewService(sites, ServiceParams{
		IndexPath: idxPath,
		Analyzer:  "standard",
	})

	require.NoError(t, err)

	teardown = func() {
		err := searcher.Close()
		require.NoError(t, err)
		if !keepIdx {
			_ = os.RemoveAll(idxPath)
		}
	}
	return searcher, teardown
}

func createTestService(t *testing.T, sites []string) (searcher *Service, teardown func()) {
	idxPath := os.TempDir() + "/search-remark42"
	_ = os.RemoveAll(idxPath)
	return createTestServiceInDir(t, sites, idxPath, false)
}

func TestSearch_Site(t *testing.T) {
	idxPath := os.TempDir() + "/search-remark42"
	searcher, teardown := createTestServiceInDir(t, []string{"test-site", "test-site2", "test-site3"}, idxPath, true)
	defer teardown()

	err := searcher.Index(store.Comment{
		ID:        "123456",
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text:      "text 123",
		User:      store.User{ID: "u1", Name: "user1"},
		Timestamp: time.Date(2017, 12, 20, 15, 18, 24, 0, time.Local),
	})
	assert.NoError(t, err)
	err = searcher.Index(store.Comment{
		ID:        "123456",
		Locator:   store.Locator{SiteID: "test-site2", URL: "http://example.com/post1"},
		Text:      "text 345",
		User:      store.User{ID: "u1", Name: "user1"},
		Timestamp: time.Date(2017, 12, 20, 15, 20, 24, 0, time.Local),
	})
	assert.NoError(t, err)
	err = searcher.Index(store.Comment{
		ID:        "123457",
		Locator:   store.Locator{SiteID: "test-site2", URL: "http://example.com/post1"},
		Text:      "foobar 345",
		User:      store.User{ID: "u1", Name: "user1"},
		Timestamp: time.Date(2017, 12, 20, 15, 20, 28, 0, time.Local),
	})
	assert.NoError(t, err)

	var res *Result

	res, err = searcher.Search(&Request{SiteID: "test-site", Query: "123", Limit: 3})
	require.NoError(t, err)
	require.Len(t, res.Keys, 1)
	assert.Equal(t, "123456", res.Keys[0].ID)

	res, err = searcher.Search(&Request{SiteID: "test-site", Query: "345", Limit: 3})
	require.NoError(t, err)
	require.Len(t, res.Keys, 0)

	res, err = searcher.Search(&Request{SiteID: "test-site2", Query: "345", SortBy: "-time", Limit: 3})
	require.NoError(t, err)
	require.Len(t, res.Keys, 2)
	assert.Equal(t, "123457", res.Keys[0].ID)
	assert.Equal(t, "123456", res.Keys[1].ID)

	res, err = searcher.Search(&Request{SiteID: "test-site2", Query: "345", SortBy: "-time", Limit: 1})
	require.NoError(t, err)
	require.Len(t, res.Keys, 1)
	assert.Equal(t, "123457", res.Keys[0].ID)

	res, err = searcher.Search(&Request{SiteID: "test-site2", Query: "345", SortBy: "-time", Limit: 1, Skip: 1})
	require.NoError(t, err)
	require.Len(t, res.Keys, 1)
	assert.Equal(t, "123456", res.Keys[0].ID)

	res, err = searcher.Search(&Request{SiteID: "test-site2", Query: "123", Limit: 3})
	require.NoError(t, err)
	assert.Len(t, res.Keys, 0)

	res, err = searcher.Search(&Request{SiteID: "test-site3", Query: "345", Limit: 3})
	require.NoError(t, err)
	assert.Len(t, res.Keys, 0)

	_, err = searcher.Search(&Request{SiteID: "test-site2", Query: "345", SortBy: "badfield"})
	require.Error(t, err)

	res, err = searcher.Search(&Request{SiteID: "test-site3", Query: "345", Limit: 3})
	require.NoError(t, err)
	assert.Len(t, res.Keys, 0)

	// restart service to check that saved index is loaded
	err = searcher.Close()
	require.NoError(t, err)

	searcher2, teardown2 := createTestServiceInDir(t, []string{"test-site", "test-site2", "test-site3"}, idxPath, false)
	defer teardown2()

	res, err = searcher2.Search(&Request{SiteID: "test-site2", Query: "345", SortBy: "-time", Limit: 3})
	require.NoError(t, err)
	require.Len(t, res.Keys, 2)
	assert.Equal(t, "123457", res.Keys[0].ID)
	assert.Equal(t, "123456", res.Keys[1].ID)
}

func TestSearch_Paginate(t *testing.T) {
	searcher, teardown := createTestService(t, []string{"test-site"})
	defer teardown()

	t0 := time.Date(2017, 12, 20, 15, 18, 24, 0, time.Local)
	for shift := 0; shift < 4; shift++ {
		cid := fmt.Sprintf("comment%d", shift)
		err := searcher.Index(store.Comment{
			ID:        cid,
			Locator:   store.Locator{SiteID: "test-site", URL: fmt.Sprintf("http://example.com/post%d", shift%2)},
			Text:      "text 123",
			User:      store.User{ID: "u1", Name: "user1"},
			Timestamp: t0.Add(time.Duration(shift) * time.Minute),
		})
		assert.NoError(t, err)
	}

	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "123", Limit: 1, SortBy: "time", Skip: 0})
		require.NoError(t, err)
		require.Len(t, res.Keys, 1)
		assert.Equal(t, "comment0", res.Keys[0].ID)
	}
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "123", Limit: 1, SortBy: "+time", Skip: 1})
		require.NoError(t, err)
		require.Len(t, res.Keys, 1)
		assert.Equal(t, "comment1", res.Keys[0].ID)
	}
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "123", Limit: 1, SortBy: "+time", Skip: 3})
		require.NoError(t, err)
		require.Len(t, res.Keys, 1)
		assert.Equal(t, "comment3", res.Keys[0].ID)
	}
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "123", Limit: 2, Skip: 1, SortBy: "-time"})
		require.NoError(t, err)
		require.Len(t, res.Keys, 2)
		assert.Equal(t, []string{"comment2", "comment1"}, []string{res.Keys[0].ID, res.Keys[1].ID})
	}
}

func TestSearch_ExtraFields(t *testing.T) {
	searcher, teardown := createTestService(t, []string{"test-site", "test-site2", "test-site3"})
	defer teardown()

	err := searcher.Index(store.Comment{
		ID:        "123456",
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text:      "text 123",
		User:      store.User{ID: "u1", Name: "user foo"},
		Timestamp: time.Date(2017, 12, 18, 15, 18, 24, 0, time.Local),
	})
	assert.NoError(t, err)
	err = searcher.Index(store.Comment{
		ID:        "123457",
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text:      "text 345",
		User:      store.User{ID: "u2", Name: "User Bar"},
		Timestamp: time.Date(2017, 12, 21, 15, 20, 24, 0, time.Local),
	})
	assert.NoError(t, err)
	err = searcher.Index(store.Comment{
		ID:        "123458",
		Locator:   store.Locator{SiteID: "test-site", URL: "http://example.com/post1"},
		Text:      "foobar text",
		User:      store.User{ID: "u2", Name: "User Bar"},
		Timestamp: time.Date(2017, 12, 25, 16, 20, 28, 0, time.Local),
	})
	assert.NoError(t, err)

	// username
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "text +user:\"user bar\"", Limit: 20})
		require.NoError(t, err)
		assert.Len(t, res.Keys, 2)
	}
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "text +user:\"user foo\"", Limit: 20})
		require.NoError(t, err)
		assert.Len(t, res.Keys, 1)
	}
	{
		// order matters in username field, match only whole token
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "text +user:\"foo user\"", Limit: 20})
		require.NoError(t, err)
		assert.Len(t, res.Keys, 0)
	}

	// time range
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "text +time:>\"2017-12-20\"", Limit: 20})
		require.NoError(t, err)
		assert.Len(t, res.Keys, 2)
	}
	{
		res, err := searcher.Search(&Request{SiteID: "test-site", Query: "text +time:<\"2017-12-20\"", Limit: 20})
		require.NoError(t, err)
		assert.Len(t, res.Keys, 1)
	}
}

func TestSearch_IndexStartup(t *testing.T) {
	sites := []string{"test-site", "remark", "test-site42"}

	searcher, serviceTeardown := createTestService(t, sites)
	defer serviceTeardown()

	storeEngine, dbTeardown := createDB(t, 42, sites)
	defer dbTeardown()

	grp := syncs.NewErrSizedGroup(4)
	for _, siteID := range sites {
		err := IndexSite(context.Background(), siteID, searcher, storeEngine, grp)
		require.NoError(t, err)
	}
	err := grp.Wait()
	require.NoError(t, err)

	for _, siteID := range sites {
		serp, err := searcher.Search(&Request{
			SiteID: siteID,
			Query:  "text",
			Limit:  19,
		})
		assert.NoError(t, err)
		assert.Len(t, serp.Keys, 19)
	}
}

func TestSearch_BadAnalyzerName(t *testing.T) {
	sites := []string{"test-site"}
	idxPath := os.TempDir() + "/search-remark42"

	_, err := NewService(sites, ServiceParams{
		IndexPath: idxPath,
		Analyzer:  "badanalyzer",
	})

	require.Error(t, err)
}

func createDB(t *testing.T, commentsPerSite int, sites []string) (e engine.Interface, teardown func()) {
	testDB := os.TempDir() + "/remark-db"
	_ = os.RemoveAll(testDB)
	err := os.MkdirAll(testDB, 0o700)
	require.NoError(t, err)
	bsites := []engine.BoltSite{}
	for _, s := range sites {
		bsites = append(bsites, engine.BoltSite{FileName: testDB + "/" + s, SiteID: s})
	}
	b, err := engine.NewBoltDB(bbolt.Options{}, bsites...)
	require.NoError(t, err)
	teardown = func() {
		require.NoError(t, b.Close())
		_ = os.RemoveAll(testDB)
	}

	rng := rand.New(rand.NewSource(42))

	t0 := time.Date(2017, 12, 20, 15, 18, 24, 0, time.Local)
	for _, siteID := range sites {
		for shift := 0; shift < commentsPerSite; shift++ {
			cid := fmt.Sprintf("comment%d", shift)
			uid := rng.Intn(15)
			comment := store.Comment{
				ID:        cid,
				Locator:   store.Locator{SiteID: siteID, URL: fmt.Sprintf("http://example.com/post%d", rng.Intn(9))},
				Text:      fmt.Sprintf("%d text %d", rng.Int63(), rng.Int63()),
				User:      store.User{ID: fmt.Sprintf("u%d", uid), Name: fmt.Sprintf("user %d", uid)},
				Timestamp: t0.Add(time.Duration(shift) * time.Hour),
			}
			ccid, err := b.Create(comment)
			require.NoError(t, err)
			require.Equal(t, cid, ccid)
		}
	}

	return b, teardown
}
