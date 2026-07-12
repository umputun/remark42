package service

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestMakeTree(t *testing.T) {
	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(minute, second int) time.Time { return time.Date(2017, 12, 25, 19, minute, second, 0, time.UTC) }

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

	res := MakeTree(comments, "time", 0, "")
	resJSON, err := json.Marshal(&res)
	require.NoError(t, err)

	expJSON := mustLoadJSONFile(t, "testdata/tree.json")
	assert.Equal(t, expJSON, resJSON)

	res = MakeTree([]store.Comment{}, "time", 0, "")
	assert.Equal(t, &Tree{}, res)
}

func TestMakeEmptySubtree(t *testing.T) {
	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(minute, second int) time.Time { return time.Date(2017, 12, 25, 19, minute, second, 0, time.UTC) }

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

	res := MakeTree(comments, "time", 0, "")
	resJSON, err := json.Marshal(&res)
	require.NoError(t, err)
	t.Log(string(resJSON))

	expJSON := mustLoadJSONFile(t, "testdata/tree_del.json")
	assert.Equal(t, string(expJSON), string(resJSON))
}

func TestTreeSortNodes(t *testing.T) {
	// unsorted by purpose
	comments := []store.Comment{
		{ID: "14", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "132", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 46, 32, 0, time.UTC)},
		{ID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 1, 0, time.UTC), Score: 2, Controversy: 10},
		{ID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 2, 0, time.UTC), Score: 3, Controversy: 5},
		{ID: "11", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 11, 0, time.UTC)},
		{ID: "13", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 13, 0, time.UTC)},
		{ID: "12", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "131", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 50, 31, 0, time.UTC)},
		{ID: "21", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 21, 0, time.UTC)},
		{ID: "22", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "4", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC), Score: -2, Controversy: 7},
		{ID: "19", ParentID: "4", Timestamp: time.Date(2019, 12, 25, 19, 46, 14, 0, time.UTC), Deleted: true},
		{ID: "3", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 100, time.UTC)},
		{ID: "6", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 200, time.UTC)},
		{ID: "5", Deleted: true, Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 150, time.UTC)},
	}

	res := MakeTree(comments, "+active", 0, "")
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)

	res = MakeTree(comments, "-active", 0, "")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "+time", 0, "")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-time", 0, "")
	assert.Equal(t, "6", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "score", 0, "")
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)
	assert.Equal(t, "3", res.Nodes[1].Comment.ID)
	assert.Equal(t, "6", res.Nodes[2].Comment.ID)
	assert.Equal(t, "1", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "+score", 0, "")
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-score", 0, "")
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	assert.Equal(t, "1", res.Nodes[1].Comment.ID)
	assert.Equal(t, "3", res.Nodes[2].Comment.ID)
	assert.Equal(t, "6", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "+controversy", 0, "")
	assert.Equal(t, "3", res.Nodes[0].Comment.ID)
	assert.Equal(t, "6", res.Nodes[1].Comment.ID)
	assert.Equal(t, "2", res.Nodes[2].Comment.ID)
	assert.Equal(t, "4", res.Nodes[3].Comment.ID)
	assert.Equal(t, "1", res.Nodes[4].Comment.ID)

	res = MakeTree(comments, "-controversy", 0, "")
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)
	assert.Equal(t, "4", res.Nodes[1].Comment.ID)
	assert.Equal(t, "2", res.Nodes[2].Comment.ID)
	assert.Equal(t, "3", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "undefined", 0, "")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)
}

func TestMakeTreeLimit(t *testing.T) {
	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(sec int) time.Time { return time.Date(2017, 12, 25, 19, 0, sec, 0, time.UTC) }

	// tree with four top-level comments and subtree sizes 3, 2, 1, 3 (total 9):
	//   c1 -> c1a, c1b
	//   c2 -> c2a
	//   c3
	//   c4 -> c4a -> c4a1
	comments := []store.Comment{
		{Locator: loc, ID: "c1", Timestamp: ts(1)},
		{Locator: loc, ID: "c1a", ParentID: "c1", Timestamp: ts(11)},
		{Locator: loc, ID: "c1b", ParentID: "c1", Timestamp: ts(12)},
		{Locator: loc, ID: "c2", Timestamp: ts(2)},
		{Locator: loc, ID: "c2a", ParentID: "c2", Timestamp: ts(21)},
		{Locator: loc, ID: "c3", Timestamp: ts(3)},
		{Locator: loc, ID: "c4", Timestamp: ts(4)},
		{Locator: loc, ID: "c4a", ParentID: "c4", Timestamp: ts(41)},
		{Locator: loc, ID: "c4a1", ParentID: "c4a", Timestamp: ts(42)},
	}

	nodeIDs := func(nodes []*Node) []string {
		ids := make([]string, 0, len(nodes))
		for _, n := range nodes {
			ids = append(ids, n.Comment.ID)
		}
		return ids
	}

	tests := []struct {
		name      string
		limit     int
		offsetID  string
		wantNodes []string
		wantLeft  int
		wantLast  string
	}{
		{"no limit, no offset returns all", 0, "", []string{"c1", "c2", "c3", "c4"}, 0, ""},
		{"limit equals first subtree size", 3, "", []string{"c1"}, 6, "c1"},
		{"limit smaller than first subtree returns it whole", 2, "", []string{"c1"}, 6, "c1"},
		{"limit between first and second boundary stops after first", 4, "", []string{"c1"}, 6, "c1"},
		{"limit at exact two-subtree boundary includes both", 5, "", []string{"c1", "c2"}, 4, "c2"},
		{"limit reaches third subtree exactly", 6, "", []string{"c1", "c2", "c3"}, 3, "c3"},
		{"limit equal to total returns all", 9, "", []string{"c1", "c2", "c3", "c4"}, 0, "c4"},
		{"limit larger than total returns all", 100, "", []string{"c1", "c2", "c3", "c4"}, 0, "c4"},
		{"offset only, no limit slices remainder", 0, "c1", []string{"c2", "c3", "c4"}, 0, ""},
		{"offset at last node clears result", 0, "c4", []string{}, 0, ""},
		{"offset at last node with limit clears result", 5, "c4", []string{}, 0, ""},
		{"offset not found starts from beginning", 0, "missing", []string{"c1", "c2", "c3", "c4"}, 0, ""},
		{"offset plus limit returns single subtree", 2, "c1", []string{"c2"}, 4, "c2"},
		{"offset plus limit stops before last subtree", 3, "c2", []string{"c3"}, 3, "c3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := MakeTree(comments, "+time", tc.limit, tc.offsetID)
			assert.Equal(t, tc.wantNodes, nodeIDs(res.Nodes), "top-level nodes")
			assert.Equal(t, tc.wantLeft, res.CountLeft(), "count left")
			assert.Equal(t, tc.wantLast, res.LastComment(), "last comment")
		})
	}
}

func TestCountReplies(t *testing.T) {
	loc := store.Locator{URL: "url", SiteID: "site"}
	ts := func(sec int) time.Time { return time.Date(2017, 12, 25, 19, 0, sec, 0, time.UTC) }
	comments := []store.Comment{
		{Locator: loc, ID: "c1", Timestamp: ts(1)},
		{Locator: loc, ID: "c1a", ParentID: "c1", Timestamp: ts(11)},
		{Locator: loc, ID: "c1b", ParentID: "c1", Timestamp: ts(12)},
		{Locator: loc, ID: "c4", Timestamp: ts(4)},
		{Locator: loc, ID: "c4a", ParentID: "c4", Timestamp: ts(41)},
		{Locator: loc, ID: "c4a1", ParentID: "c4a", Timestamp: ts(42)},
	}
	res := MakeTree(comments, "+time", 0, "")

	byID := map[string]*Node{}
	for _, n := range res.Nodes {
		byID[n.Comment.ID] = n
	}

	// guard presence and shape first so a regression in MakeTree fails with a clear
	// assertion instead of a nil-pointer panic on the map lookups below
	require.Contains(t, byID, "c1")
	require.Contains(t, byID, "c4")
	require.Len(t, byID["c1"].Replies, 2)
	require.Len(t, byID["c4"].Replies, 1)

	assert.Equal(t, 2, countReplies(byID["c1"]), "c1 has two direct replies, no nesting")
	assert.Equal(t, 2, countReplies(byID["c4"]), "c4 counts nested reply recursively")
	assert.Equal(t, 1, countReplies(byID["c4"].Replies[0]), "c4a has one nested reply")
	assert.Equal(t, 0, countReplies(byID["c1"].Replies[0]), "leaf reply has no replies")
}

func BenchmarkTree(b *testing.B) {
	comments := []store.Comment{}
	data, err := os.ReadFile("testdata/tree_bench.json")
	assert.NoError(b, err)
	err = json.Unmarshal(data, &comments)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		res := MakeTree(comments, "time", 0, "")
		assert.NotNil(b, res)
	}
}

// loadJsonFile read fixtrue file and clear any custom json formatting
func mustLoadJSONFile(t *testing.T, file string) []byte {
	expJSON, err := os.ReadFile(file)
	require.NoError(t, err)
	expTree := Tree{}
	err = json.Unmarshal(expJSON, &expTree)
	require.NoError(t, err)
	expJSON, err = json.Marshal(expTree)
	require.NoError(t, err)
	return expJSON
}
