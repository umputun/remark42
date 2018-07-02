package rest

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

func TestMakeTree(t *testing.T) {

	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(min int, sec int) time.Time { return time.Date(2017, 12, 25, 19, min, sec, 0, time.UTC) }

	// unsorted by purpose
	comments := []store.Comment{
		{Locator: loc, ID: "14", ParentID: "1", Timestamp: ts(46, 14)},
		{Locator: loc, ID: "1", Timestamp: ts(46, 1)},
		{Locator: loc, ID: "2", Timestamp: ts(47, 2)},
		{Locator: loc, ID: "11", ParentID: "1", Timestamp: ts(46, 11)},
		{Locator: loc, ID: "13", ParentID: "1", Timestamp: ts(46, 13)},
		{Locator: loc, ID: "12", ParentID: "1", Timestamp: ts(46, 12)},
		{Locator: loc, ID: "131", ParentID: "13", Timestamp: ts(46, 31)},
		{Locator: loc, ID: "132", ParentID: "13", Timestamp: ts(46, 32)},
		{Locator: loc, ID: "21", ParentID: "2", Timestamp: ts(47, 21)},
		{Locator: loc, ID: "22", ParentID: "2", Timestamp: ts(47, 22)},
		{Locator: loc, ID: "4", Timestamp: ts(47, 22)},
		{Locator: loc, ID: "3", Timestamp: ts(47, 22)},
		{Locator: loc, ID: "5", Deleted: true},
		{Locator: loc, ID: "6", Deleted: true},
		{Locator: loc, ID: "61", ParentID: "6", Deleted: true},
		{Locator: loc, ID: "62", ParentID: "6", Deleted: true},
		{Locator: loc, ID: "611", ParentID: "61", Deleted: true},
	}

	res := MakeTree(comments, "time", 0)
	resJSON, err := json.Marshal(&res)
	require.Nil(t, err)

	expJSON := mustLoadJSONFile(t, "testdata/tree.json")
	assert.Equal(t, expJSON, resJSON)
	assert.Equal(t, store.PostInfo{URL: "url", Count: 12, FirstTS: ts(46, 1), LastTS: ts(47, 22)}, res.Info)

	res = MakeTree([]store.Comment{}, "time", 0)
	assert.Equal(t, &Tree{}, res)

	res = MakeTree(comments, "time", 10)
	assert.Equal(t, store.PostInfo{URL: "url", Count: 12, FirstTS: ts(46, 1), LastTS: ts(47, 22), ReadOnly: true}, res.Info)
}

func TestMakeEmptySubtree(t *testing.T) {
	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(min int, sec int) time.Time { return time.Date(2017, 12, 25, 19, min, sec, 0, time.UTC) }

	// unsorted by purpose
	comments := []store.Comment{
		{Locator: loc, ID: "1", Timestamp: ts(46, 1)},
		{Locator: loc, ID: "11", ParentID: "1", Timestamp: ts(46, 11)},
		{Locator: loc, ID: "111", ParentID: "11", Timestamp: ts(46, 12)},
		{Locator: loc, ID: "112", ParentID: "11", Deleted: true}, // subtree deleted
		{Locator: loc, ID: "1121", ParentID: "112", Deleted: true},
		{Locator: loc, ID: "1122", ParentID: "112", Deleted: true},
		{Locator: loc, ID: "12", ParentID: "12", Deleted: true}, // subcomment deleted

		{Locator: loc, ID: "2", Timestamp: ts(47, 1)},
		{Locator: loc, ID: "21", ParentID: "2", Deleted: true}, // subtree deleted
		{Locator: loc, ID: "211", ParentID: "21", Deleted: true},
		{Locator: loc, ID: "212", ParentID: "21", Deleted: true},
		{Locator: loc, ID: "22", ParentID: "2", Timestamp: ts(47, 2)},
		{Locator: loc, ID: "221", ParentID: "22", Timestamp: ts(47, 3)},
		{Locator: loc, ID: "222", ParentID: "22", Timestamp: ts(47, 4)},
		{Locator: loc, ID: "223", ParentID: "22", Deleted: true},
		{Locator: loc, ID: "224", ParentID: "22", Deleted: true},
		{Locator: loc, ID: "2241", ParentID: "223", Timestamp: ts(47, 5)},
		{Locator: loc, ID: "3", Timestamp: ts(48, 1), Deleted: true}, // deleted top level
	}

	res := MakeTree(comments, "time", 0)
	resJSON, err := json.Marshal(&res)
	require.Nil(t, err)
	log.Print(string(resJSON))

	expJSON := mustLoadJSONFile(t, "testdata/tree_del.json")
	assert.Equal(t, string(expJSON), string(resJSON))

}

func TestTreeSortNodes(t *testing.T) {
	// unsorted by purpose
	comments := []store.Comment{
		{ID: "14", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "132", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 46, 32, 0, time.UTC)},
		{ID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 1, 0, time.UTC), Score: 2},
		{ID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 2, 0, time.UTC), Score: 3},
		{ID: "11", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 11, 0, time.UTC)},
		{ID: "13", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 13, 0, time.UTC)},
		{ID: "12", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "131", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 50, 31, 0, time.UTC)},
		{ID: "21", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 21, 0, time.UTC)},
		{ID: "22", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "4", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC), Score: -2},
		{ID: "3", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 100, time.UTC)},
		{ID: "6", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 200, time.UTC)},
		{ID: "5", Deleted: true, Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 150, time.UTC)},
	}

	res := MakeTree(comments, "+active", 0)
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)

	res = MakeTree(comments, "-active", 0)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "+time", 0)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-time", 0)
	assert.Equal(t, "6", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "score", 0)
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)
	assert.Equal(t, "3", res.Nodes[1].Comment.ID)
	assert.Equal(t, "6", res.Nodes[2].Comment.ID)
	assert.Equal(t, "1", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "+score", 0)
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-score", 0)
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	assert.Equal(t, "1", res.Nodes[1].Comment.ID)
	assert.Equal(t, "3", res.Nodes[2].Comment.ID)
	assert.Equal(t, "6", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "undefined", 0)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)
}

func BenchmarkTree(b *testing.B) {
	comments := []store.Comment{}
	data, err := ioutil.ReadFile("testdata/tree_bench.json")
	assert.Nil(b, err)
	err = json.Unmarshal(data, &comments)
	assert.Nil(b, err)

	for i := 0; i < b.N; i++ {
		res := MakeTree(comments, "time", 0)
		assert.NotNil(b, res)
	}
}

// loadJsonFile read fixtrue file and clear any custom json formatting
func mustLoadJSONFile(t *testing.T, file string) []byte {
	expJSON, err := ioutil.ReadFile(file)
	require.Nil(t, err)
	expTree := Tree{}
	err = json.Unmarshal(expJSON, &expTree)
	require.Nil(t, err)
	expJSON, err = json.Marshal(expTree)
	require.Nil(t, err)
	return expJSON
}
