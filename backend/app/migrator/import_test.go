package migrator

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestImporters_UnreadableInput(t *testing.T) {
	disqusXML, err := os.ReadFile("testdata/disqus.xml")
	require.NoError(t, err)
	wpXML := xmlTestWP
	commentoJSON, err := os.ReadFile("testdata/commento.json")
	require.NoError(t, err)

	tbl := []struct {
		name     string
		importer func(ds Store) Importer
		input    string
		keep     bool // nothing could be read, so the site keeps its comments
	}{
		{"disqus malformed", func(ds Store) Importer { return &Disqus{DataStore: ds} }, "<disqus><thread", true},
		{"disqus truncated", func(ds Store) Importer { return &Disqus{DataStore: ds} }, string(disqusXML[:len(disqusXML)*2/3]), false},
		{"wordpress malformed", func(ds Store) Importer { return &WordPress{DataStore: ds} }, "<rss><channel><item", true},
		{"wordpress truncated", func(ds Store) Importer { return &WordPress{DataStore: ds} }, wpXML[:len(wpXML)*2/3], false},
		{"disqus empty", func(ds Store) Importer { return &Disqus{DataStore: ds} }, "", true},
		{"disqus whitespace", func(ds Store) Importer { return &Disqus{DataStore: ds} }, " \n\t\n", true},
		{"disqus given commento json", func(ds Store) Importer { return &Disqus{DataStore: ds} }, string(commentoJSON), true},
		{"disqus given wordpress xml", func(ds Store) Importer { return &Disqus{DataStore: ds} }, wpXML, true},
		{"wordpress empty", func(ds Store) Importer { return &WordPress{DataStore: ds} }, "", true},
		{"wordpress whitespace", func(ds Store) Importer { return &WordPress{DataStore: ds} }, " \n\t\n", true},
		{"wordpress given commento json", func(ds Store) Importer { return &WordPress{DataStore: ds} }, string(commentoJSON), true},
		{"wordpress given disqus xml", func(ds Store) Importer { return &WordPress{DataStore: ds} }, string(disqusXML), true},
		{"commento given empty object", func(ds Store) Importer { return &Commento{DataStore: ds} }, "{}", true},
		{"commento given native export", func(ds Store) Importer { return &Commento{DataStore: ds} },
			`{"version":1,"users":[{"id":"u1"}],"posts":[{"url":"https://example.com/p1"}]}` + "\n" + `{"id":"c1","text":"t"}` + "\n", true},
		{"commento given other json", func(ds Store) Importer { return &Commento{DataStore: ds} }, `{"name":"x","items":[1,2]}`, true},
		{"commento export followed by garbage", func(ds Store) Importer { return &Commento{DataStore: ds} },
			`{"comments":[],"commenters":[]}garbage`, true},
		{"commento export followed by a truncated object", func(ds Store) Importer { return &Commento{DataStore: ds} },
			`{"comments":[],"commenters":[]} {"comments":[`, true},
		{"disqus every record failed", func(ds Store) Importer { return &Disqus{DataStore: ds} },
			`<disqus><thread dsq:id="t1"><link>https://example.com/p2</link><createdAt>bad</createdAt></thread>` +
				`<post dsq:id="p1"><message>m</message><createdAt>bad</createdAt><thread dsq:id="t1"/></post></disqus>`, true},
		{"wordpress every item failed", func(ds Store) Importer { return &WordPress{DataStore: ds} },
			`<rss><channel><wp:wxr_version>1.2</wp:wxr_version><item><link>https://example.com/p2</link><wp:comment><wp:comment_id>1</wp:comment_id>` +
				`<wp:comment_date_gmt>bad</wp:comment_date_gmt><wp:comment_approved>1</wp:comment_approved></wp:comment></item></channel></rss>`, true},
		{"wordpress given rss feed", func(ds Store) Importer { return &WordPress{DataStore: ds} },
			`<?xml version="1.0"?><rss version="2.0"><channel><title>Blog</title><link>https://example.com</link>` +
				`<item><title>Post</title><link>https://example.com/p2</link><description>text</description></item>` +
				`</channel></rss>`, true},
		{"wordpress given empty rss feed", func(ds Store) Importer { return &WordPress{DataStore: ds} },
			`<rss version="2.0"><channel><title>Blog</title></channel></rss>`, true},
		{"commento empty", func(ds Store) Importer { return &Commento{DataStore: ds} }, "", true},
		{"commento malformed", func(ds Store) Importer { return &Commento{DataStore: ds} }, "{\"comments\": [", true},
		{"commento truncated", func(ds Store) Importer { return &Commento{DataStore: ds} }, string(commentoJSON[:len(commentoJSON)/2]), false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "test.db"), SiteID: "test"})
			require.NoError(t, err)
			ds := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
			defer ds.Close()

			_, err = ds.Create(store.Comment{ID: "existing", Text: "kept", Locator: store.Locator{SiteID: "test", URL: "https://example.com/p1"},
				User: store.User{ID: "u1", Name: "user"}})
			require.NoError(t, err)

			_, err = tt.importer(ds).Import(strings.NewReader(tt.input), "test")
			require.Error(t, err, "import of unreadable input fails")
			if tt.keep {
				c, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "existing", store.User{})
				require.NoError(t, err, "site keeps its comments when nothing could be read")
				assert.Equal(t, "kept", c.Text)
			}
		})
	}
}

func TestImporters_ValidInputWithoutComments(t *testing.T) {
	tbl := []struct {
		name     string
		importer func(ds Store) Importer
		input    string
	}{
		{"disqus", func(ds Store) Importer { return &Disqus{DataStore: ds} }, "<disqus></disqus>"},
		{"wordpress", func(ds Store) Importer { return &WordPress{DataStore: ds} }, "<rss><channel><wp:wxr_version>1.2</wp:wxr_version></channel></rss>"},
		{"commento", func(ds Store) Importer { return &Commento{DataStore: ds} }, `{"comments": [], "commenters": []}`},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "test.db"), SiteID: "test"})
			require.NoError(t, err)
			ds := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
			defer ds.Close()

			_, err = ds.Create(store.Comment{ID: "existing", Text: "removed", Locator: store.Locator{SiteID: "test", URL: "https://example.com/p1"},
				User: store.User{ID: "u1", Name: "user"}})
			require.NoError(t, err)

			size, err := tt.importer(ds).Import(strings.NewReader(tt.input), "test")
			require.NoError(t, err)
			assert.Equal(t, 0, size)
			_, err = ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "existing", store.User{})
			assert.Error(t, err, "an export without comments still replaces the site's comments")
		})
	}
}

// newImportStore makes a data store on a fresh bolt db with the site "test"
func newImportStore(t *testing.T) *service.DataStore {
	t.Helper()
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "test.db"), SiteID: "test"})
	require.NoError(t, err)
	ds := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	t.Cleanup(func() { _ = ds.Close() })
	return ds
}

// shownTree returns the comments of https://example.com/p1 the way the widget's tree shows them, as "id"
// for a top-level comment and "parent>id" for a reply, with "deleted:" before a deleted placeholder
func shownTree(t *testing.T, ds *service.DataStore) []string {
	t.Helper()
	comments, err := ds.Find(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "time", adminUser)
	require.NoError(t, err)
	var res []string
	var walk func(nodes []*service.Node, parent string)
	walk = func(nodes []*service.Node, parent string) {
		for _, n := range nodes {
			id := n.Comment.ID
			if parent != "" {
				id = parent + ">" + id
			}
			if n.Comment.Deleted {
				id = "deleted:" + id
			}
			res = append(res, id)
			walk(n.Replies, n.Comment.ID)
		}
	}
	walk(service.MakeTree(comments, "time", 0, "").Nodes, "")
	return res
}

func TestImportComments_RepliesStayVisible(t *testing.T) {
	const postURL = "https://example.com/p1"
	ts := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)
	mk := func(id, parent string, minute int, deleted bool) store.Comment {
		c := store.Comment{ID: id, ParentID: parent, Text: "text " + id, Locator: store.Locator{SiteID: "test", URL: postURL},
			User: store.User{ID: "u1", Name: "user"}, Timestamp: ts.Add(time.Duration(minute) * time.Minute), Imported: true}
		if deleted {
			c.SetDeleted(store.HardDelete)
		}
		return c
	}
	tbl := []struct {
		name     string
		comments []store.Comment
		shown    []string
		count    int
	}{
		{"reply before its parent", []store.Comment{mk("r1", "p1", 2, false), mk("p1", "", 1, false)},
			[]string{"p1", "p1>r1"}, 2},
		{"reply to a parent not in the export", []store.Comment{mk("p1", "", 1, false), mk("r1", "gone", 2, false)},
			[]string{"p1", "r1"}, 2},
		{"chain under a parent not in the export", []store.Comment{mk("r2", "r1", 3, false), mk("r1", "gone", 2, false)},
			[]string{"r1", "r1>r2"}, 2},
		{"reply to a deleted comment", []store.Comment{mk("p1", "", 1, true), mk("r1", "p1", 2, false)},
			[]string{"deleted:p1", "p1>r1"}, 1},
		{"reply before the deleted comment it replies to", []store.Comment{mk("r1", "p1", 2, false), mk("p1", "", 1, true)},
			[]string{"deleted:p1", "p1>r1"}, 1},
		{"reply under two deleted comments", []store.Comment{mk("p1", "", 1, true), mk("p2", "p1", 2, true), mk("p3", "p2", 3, true),
			mk("r1", "p2", 4, false)}, []string{"deleted:p1", "deleted:p1>p2", "p2>r1"}, 1},
		{"deleted comments replying to each other", []store.Comment{mk("p1", "p2", 1, true), mk("p2", "p1", 2, true),
			mk("r1", "p1", 3, false)}, []string{"deleted:p1", "p1>r1"}, 1},
		{"deleted comment without replies", []store.Comment{mk("p1", "", 1, false), mk("p2", "", 2, true)},
			[]string{"p1"}, 1},
		{"chain under a missing parent that sorts after it", []store.Comment{mk("a", "z", 1, false), mk("b", "a", 2, false),
			mk("c", "b", 3, false)}, []string{"a", "a>b", "b>c"}, 3},
		{"chain under a deleted reply to a missing parent", []store.Comment{mk("a", "z", 1, true), mk("b", "a", 2, false)},
			[]string{"deleted:a", "a>b"}, 1},
		{"reply to a comment in a cycle", []store.Comment{mk("a", "b", 1, false), mk("b", "a", 2, false), mk("c", "a", 3, false)},
			[]string{"b", "b>a", "a>c"}, 3},
		{"replies hanging off a cycle, sorting before it", []store.Comment{mk("x", "y", 1, false), mk("y", "x", 2, false),
			mk("a", "x", 3, false), mk("e", "a", 4, false)}, []string{"x", "x>y", "x>a", "a>e"}, 4},
		{"chain leading into a cycle", []store.Comment{mk("a", "y", 1, false), mk("b", "a", 2, false), mk("y", "z", 3, false),
			mk("z", "y", 4, false)}, []string{"y", "y>a", "a>b", "y>z"}, 4},
		{"reply with replies, emitted before the cycle closes", []store.Comment{mk("a", "b", 1, false), mk("c", "a", 2, false),
			mk("d", "c", 3, false), mk("b", "a", 4, false)}, []string{"a", "a>c", "c>d", "a>b"}, 4},
		{"replies to each other", []store.Comment{mk("a", "b", 1, false), mk("b", "a", 2, false)},
			[]string{"b", "b>a"}, 2},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ds := newImportStore(t)
			passed, err := importComments(ds, strings.NewReader(""), "test", func(_ io.Reader, _ string, emit func(store.Comment) error) error {
				for _, c := range tt.comments {
					require.NoError(t, emit(c))
				}
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, tt.count, passed, "deleted placeholders are not counted as imported")
			assert.Equal(t, tt.shown, shownTree(t, ds))
			count, err := ds.Count(store.Locator{SiteID: "test", URL: postURL})
			require.NoError(t, err)
			assert.Equal(t, tt.count, count, "the post counts its comments, not the placeholders")
			posts, err := ds.List("test", 0, 0)
			require.NoError(t, err)
			assert.Len(t, posts, 1)
		})
	}

	t.Run("input breaks off", func(t *testing.T) {
		ds := newImportStore(t)
		passed, err := importComments(ds, strings.NewReader(""), "test", func(_ io.Reader, _ string, emit func(store.Comment) error) error {
			require.NoError(t, emit(mk("p1", "", 1, false)))
			require.NoError(t, emit(mk("r1", "gone", 2, false)))
			return errors.New("unexpected EOF")
		})
		require.EqualError(t, err, "unexpected EOF")
		assert.Equal(t, 2, passed)
		assert.Equal(t, []string{"p1", "r1"}, shownTree(t, ds), "comments read before the input broke off are imported")
	})

	t.Run("post with deleted comments only", func(t *testing.T) {
		ds := newImportStore(t)
		_, err := importComments(ds, strings.NewReader(""), "test", func(_ io.Reader, _ string, emit func(store.Comment) error) error {
			require.NoError(t, emit(mk("p1", "", 1, false)))
			d1, d2 := mk("d1", "", 2, true), mk("d2", "d1", 3, true)
			d1.Locator.URL, d2.Locator.URL = "https://example.com/spam-only", "https://example.com/spam-only"
			require.NoError(t, emit(d1))
			require.NoError(t, emit(d2))
			return nil
		})
		require.NoError(t, err)
		posts, err := ds.List("test", 0, 0)
		require.NoError(t, err)
		require.Len(t, posts, 1, "a post whose comments were all removed is not listed")
		assert.Equal(t, postURL, posts[0].URL)
	})
}

func TestDisqus_ImportRepliesToRemovedComments(t *testing.T) {
	post := func(id, parent, deleted, spam string) string {
		res := `<post dsq:id="` + id + `"><message><![CDATA[<p>text ` + id + `</p>]]></message><createdAt>2020-01-01T10:00:0` + id[1:] + `Z</createdAt>` +
			`<isDeleted>` + deleted + `</isDeleted><isSpam>` + spam + `</isSpam><author><name>user ` + id + `</name><username>u` + id + `</username></author>` +
			`<ipAddress>10.0.0.1</ipAddress><thread dsq:id="t1"/>`
		if parent != "" {
			res += `<parent dsq:id="` + parent + `"/>`
		}
		return res + `</post>`
	}
	xml := `<disqus xmlns="http://disqus.com" xmlns:dsq="http://disqus.com/disqus-internals">` +
		`<thread dsq:id="t1"><link>https://example.com/p1</link><title>Post one</title><createdAt>2020-01-01T09:00:00Z</createdAt></thread>` +
		post("p1", "", "true", "false") + post("p2", "p1", "false", "false") + // reply to a deleted comment
		post("p3", "", "false", "true") + post("p4", "p3", "false", "false") + // reply to spam
		post("p5", "gone", "false", "false") + // reply to a comment not in the export
		post("p6", "", "true", "false") + // deleted, no replies
		`</disqus>`

	ds := newImportStore(t)
	size, err := (&Disqus{DataStore: ds}).Import(strings.NewReader(xml), "test")
	require.NoError(t, err)
	assert.Equal(t, 3, size)
	assert.Equal(t, []string{"deleted:p1", "p1>p2", "deleted:p3", "p3>p4", "p5"}, shownTree(t, ds))

	count, err := ds.Count(store.Locator{SiteID: "test", URL: "https://example.com/p1"})
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	removed, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "p3", adminUser)
	require.NoError(t, err)
	assert.Empty(t, removed.Text, "the text of a removed comment is not stored")
	assert.Equal(t, "deleted", removed.User.ID, "the author of a removed comment is not stored")

	reply, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "p2", adminUser)
	require.NoError(t, err)
	assert.Equal(t, "Post one", reply.PostTitle, "the title comes from the export")
}

func TestCommento_ImportFidelity(t *testing.T) {
	export := `{"version":1,"comments":[
{"commentHex":"c1","domain":"example.com","url":"/p1","commenterHex":"u1","markdown":"**bold** and [a link](https://example.com)\n\nsecond paragraph","parentHex":"root","state":"approved","creationDate":"2021-03-17T12:00:01Z"},
{"commentHex":"c2","domain":"example.com","url":"/p1","commenterHex":"anonymous","markdown":"[deleted]","parentHex":"root","state":"approved","creationDate":"2021-03-17T12:00:02Z"},
{"commentHex":"c3","domain":"example.com","url":"/p1","commenterHex":"u2","markdown":"reply to deleted","parentHex":"c2","state":"approved","creationDate":"2021-03-17T12:00:03Z"},
{"commentHex":"c4","domain":"example.com","url":"/p1","commenterHex":"anonymous","markdown":"[deleted]","parentHex":"root","state":"approved","creationDate":"2021-03-17T12:00:04Z"},
{"commentHex":"c5","domain":"example.com","url":"/p1","commenterHex":"u2","markdown":"waits for moderation","parentHex":"root","state":"unapproved","creationDate":"2021-03-17T12:00:05Z"},
{"commentHex":"c6","domain":"example.com","url":"/p1","commenterHex":"u2","markdown":"spam","parentHex":"root","state":"flagged","creationDate":"2021-03-17T12:00:06Z"},
{"commentHex":"c7","domain":"example.com","url":"/p1","commenterHex":"u1","markdown":"reply to spam","parentHex":"c6","state":"approved","creationDate":"2021-03-17T12:00:07Z"}
],"commenters":[
{"commenterHex":"u1","name":"With photo","photo":"https://example.com/u1.png"},
{"commenterHex":"u2","name":"No photo","photo":"undefined"}
]}`
	ds := newImportStore(t)
	size, err := (&Commento{DataStore: ds}).Import(strings.NewReader(export), "test")
	require.NoError(t, err)
	assert.Equal(t, 3, size)
	assert.Equal(t, []string{"c1", "deleted:c2", "c2>c3", "c7"}, shownTree(t, ds))

	get := func(id string) store.Comment {
		c, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, id, adminUser)
		require.NoError(t, err)
		return c
	}
	assert.Equal(t, `<p><strong>bold</strong> and <a href="https://example.com" rel="nofollow ugc">a link</a></p>`+"\n\n<p>second paragraph</p>\n", get("c1").Text)
	assert.Equal(t, "https://example.com/u1.png", get("c1").User.Picture)
	assert.Empty(t, get("c3").User.Picture, `commento's "undefined" photo is not an avatar`)
	assert.Empty(t, get("c2").Text)
}
