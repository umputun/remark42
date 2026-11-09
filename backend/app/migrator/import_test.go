package migrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
