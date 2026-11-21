package migrator

import (
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

func TestWordPress_Import(t *testing.T) {
	siteID := "testWP"
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "remark-test.db"), SiteID: siteID})
	assert.NoError(t, err, "create store")

	dataStore := service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	wp := WordPress{DataStore: &dataStore, DisableFancyTextFormatting: false}
	size, err := wp.Import(strings.NewReader(xmlTestWP), siteID)
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	last, err := dataStore.Last(siteID, 10, time.Time{}, adminUser)
	assert.NoError(t, err)
	require.Equal(t, 2, len(last), "2 comments imported, the pingback skipped")

	c := last[0]
	assert.Equal(t, "14", c.ID)
	assert.Empty(t, c.ParentID, "the reply to the skipped pingback is a top-level comment")
	assert.Equal(t, "Post with comments. One is not approved", c.PostTitle)
	assert.Equal(t, store.Locator{URL: "https://realmenweardress.es/2010/07/do-you-rp/", SiteID: siteID}, c.Locator)
	assert.Equal(t, "wordpress_75b2b81081f82495d7af26759e67af6554ffda4a", c.User.ID)
	assert.Equal(t, "SuperUser3", c.User.Name)
	assert.Equal(t, "e8b1e92bbcf5b9bb88472f9bdb82d1b8c7ed39d6", c.User.IP)
	ts, _ := time.Parse(wpTimeLayout, "2010-08-18 15:19:14")
	assert.Equal(t, ts, c.Timestamp)
	assert.Equal(t, "<p>“Mekkatorque” was over in that tent up to the right</p>\n", c.Text)
	assert.True(t, c.Imported)

	posts, err := dataStore.List(siteID, 0, 0)
	assert.NoError(t, err)
	require.Equal(t, 1, len(posts))

	p := posts[0]
	assert.Equal(t, "https://realmenweardress.es/2010/07/do-you-rp/", p.URL)

	count, err := dataStore.Count(store.Locator{URL: "https://realmenweardress.es/2010/07/do-you-rp/", SiteID: siteID})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	// test with DisableFancyTextFormatting
	wp = WordPress{DataStore: &dataStore, DisableFancyTextFormatting: true}
	size, err = wp.Import(strings.NewReader(xmlTestWP), siteID)
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	last, err = dataStore.Last(siteID, 10, time.Time{}, adminUser)
	assert.NoError(t, err)
	require.Equal(t, 2, len(last), "2 comments imported")

	assert.Equal(t, "<p>&#34;Mekkatorque&#34; was over in that tent up to the right</p>\n", last[0].Text)
}

func TestWordPress_Convert(t *testing.T) {
	wp := WordPress{}
	comments := make([]store.Comment, 0, 3)
	err := wp.convert(strings.NewReader(xmlTestWP), "testWP", func(c store.Comment) error {
		comments = append(comments, c)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(comments), "2 comments exported, 1 not approved and 1 pingback excluded")

	exp1 := store.Comment{
		ID: "14",
		Locator: store.Locator{
			SiteID: "testWP",
			URL:    "https://realmenweardress.es/2010/07/do-you-rp/",
		},
		Text: "<p>“Mekkatorque” was over in that tent up to the right</p>\n",
		User: store.User{
			Name: "SuperUser3",
			ID:   "wordpress_" + store.EncodeID("SuperUser3"),
			IP:   "128.243.253.117",
		},
		ParentID:  "13",
		PostTitle: "Post with comments. One is not approved",
		Imported:  true,
	}
	exp1.Timestamp, _ = time.Parse(wpTimeLayout, "2010-08-18 15:19:14")
	assert.Equal(t, exp1, comments[1])
}

func TestWordPress_ReaderCommentTypes(t *testing.T) {
	tbl := []struct {
		commentType string
		imported    bool
	}{
		{"", true}, {"comment", true}, {"review", true},
		{"pingback", false}, {"trackback", false}, {"order_note", false}, {"webhook_delivery", false},
	}
	for _, tt := range tbl {
		t.Run(tt.commentType, func(t *testing.T) {
			xml := `<rss><channel><wp:wxr_version>1.2</wp:wxr_version><item><title>Post</title><link>https://example.com/p1</link>` +
				`<wp:comment><wp:comment_id>1</wp:comment_id><wp:comment_author>A</wp:comment_author>` +
				`<wp:comment_date_gmt>2020-01-01 10:00:00</wp:comment_date_gmt><wp:comment_content>text</wp:comment_content>` +
				`<wp:comment_approved>1</wp:comment_approved><wp:comment_type>` + tt.commentType + `</wp:comment_type>` +
				`<wp:comment_parent>0</wp:comment_parent></wp:comment></item></channel></rss>`
			var res []store.Comment
			err := (&WordPress{}).convert(strings.NewReader(xml), "site", func(c store.Comment) error {
				res = append(res, c)
				return nil
			})
			require.NoError(t, err)
			if !tt.imported {
				assert.Empty(t, res)
				return
			}
			require.Len(t, res, 1)
			assert.Equal(t, "Post", res[0].PostTitle)
		})
	}
}

func TestWP_Convert_MD(t *testing.T) {
	wp := WordPress{}
	comments := make([]store.Comment, 0, 3)
	err := wp.convert(strings.NewReader(xmlTestWPmd), "siteID", func(c store.Comment) error {
		comments = append(comments, c)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(comments), "3 comments exported")

	// wordpress shows comments as html with paragraphs and links added, markdown stays text
	assert.Equal(t, "<p>Row1<br/>\nRow2</p>\n<p>Row4</p>\n", comments[0].Text)
	assert.Equal(t, "<p>markdown `text`</p>\n", comments[1].Text)
	expText := `<p>Row1 Link <a href="http://releases.rancher.com/os/latest" rel="nofollow ugc">http://releases.rancher.com/os/latest</a>`
	expText += " markdown `text` blah</p>\n<p>Row3 markdown“`md block“`</p>\n"
	assert.Equal(t, expText, comments[2].Text)
}

func TestWordPress_CommentTextIndependent(t *testing.T) {
	// quotes are curled for each comment on its own, as wptexturize does
	first, second := wpCommentHTML(`5 " x`, true), wpCommentHTML(`5 " x`, true)
	assert.Equal(t, first, second)
}

func TestWordPress_CommentTextLongURL(t *testing.T) {
	// wordpress does not link a url longer than about 2000 characters, so a long one never reaches the
	// parenthesis balancing, which is quadratic in the url length
	url := "http://example.com/" + strings.Repeat(")a", 200000)
	assert.NotContains(t, wpCommentHTML("see "+url, false), "<a ")
	short := "http://example.com/" + strings.Repeat("a", 1900)
	assert.Contains(t, wpCommentHTML("see "+short, false), `<a href="`+short+`"`)
}

// TestWordPress_CommentText checks the text of imported comments against what WordPress shows for them;
// each expected value is the output of WordPress' own comment_text filters for the same input.
func TestWordPress_CommentText(t *testing.T) {
	tbl := []struct {
		name, in, fancy, plain string
	}{
		{name: "backslashes stay", in: `shrug ¯\_(ツ)_/¯ and C:\temp\new`,
			fancy: "<p>shrug ¯\\_(ツ)_/¯ and C:\\temp\\new</p>\n"},
		{name: "markdown stays text", in: "*not emphasis* _nor this_ `nor code` # not heading\n1. not list",
			fancy: "<p>*not emphasis* _nor this_ `nor code` # not heading<br/>\n1. not list</p>\n"},
		{name: "quotes, apostrophes and primes", in: `He said "hello" and 'bye'. It's 5'10" tall, 9" nails, the '80s`,
			fancy: "<p>He said “hello” and ‘bye’. It’s 5’10” tall, 9″ nails, the ’80s</p>\n",
			plain: "<p>He said &#34;hello&#34; and &#39;bye&#39;. It&#39;s 5&#39;10&#34; tall, 9&#34; nails, the &#39;80s</p>\n"},
		{name: "primes", in: `9" nails and 6' board`, fancy: "<p>9″ nails and 6′ board</p>\n",
			plain: "<p>9&#34; nails and 6&#39; board</p>\n"},
		{name: "dashes", in: "a -- b, a---b, 10-20, - alone, xn--test, :-)",
			fancy: "<p>a — b, a—b, 10-20, – alone, xn--test, 🙂</p>\n",
			plain: "<p>a -- b, a---b, 10-20, - alone, xn--test, 🙂</p>\n"},
		{name: "symbols", in: "ellipsis... and (tm) and (c) (r) 1/2 3/4 2x3 0x1f",
			fancy: "<p>ellipsis… and ™ and (c) (r) 1/2 3/4 2×3 0x1f</p>\n",
			plain: "<p>ellipsis... and (tm) and (c) (r) 1/2 3/4 2x3 0x1f</p>\n"},
		{name: "smilies", in: "after :) :-) ;) :D :mrgreen: 8) :x smilies:) (:)",
			fancy: "<p>after 🙂 🙂 😉 😀 :mrgreen: 8) 😡 smilies:) (:)</p>\n"},
		{name: "links", in: "see http://example.com/a?b=1&c=2. and www.example.org, mail me@example.com", // rel is the sanitizer's
			fancy: `<p>see <a href="http://example.com/a?b=1&amp;c=2" rel="nofollow ugc">http://example.com/a?b=1&amp;c=2</a>.` +
				` and <a href="http://www.example.org" rel="nofollow ugc">http://www.example.org</a>,` +
				` mail <a href="mailto:me@example.com" rel="nofollow ugc">me@example.com</a></p>` + "\n"},
		{name: "paired parentheses", in: "(see https://en.wikipedia.org/wiki/Foo_(bar)) end",
			fancy: `<p>(see <a href="https://en.wikipedia.org/wiki/Foo_(bar)" rel="nofollow ugc">` +
				`https://en.wikipedia.org/wiki/Foo_(bar)</a>) end</p>` + "\n"},
		{name: "no links in links and code", in: `<a href="http://x.com">http://y.com</a> and <code>http://z.com 'q'</code>`,
			fancy: `<p><a href="http://x.com" rel="nofollow ugc">http://y.com</a> and <code>http://z.com &#39;q&#39;</code></p>` + "\n"},
		{name: "blockquote", in: "<blockquote>quoted ¯\\_(ツ)_/¯</blockquote>\n\nafter it",
			fancy: "<blockquote><p>quoted ¯\\_(ツ)_/¯</p></blockquote>\n<p>after it</p>\n"},
		{name: "several pre blocks", in: "<pre>one :)</pre>\n\nbetween :)\n\n<pre>two -- 2</pre>",
			fancy: "<pre>one :)</pre>\n<p>between 🙂</p>\n<pre>two -- 2</pre>\n"},
		{name: "pre keeps its text", in: "<pre>\ncode \"line\"\n\nmore -- here :)\n</pre>\nafter",
			fancy: "<pre>code &#34;line&#34;\n\nmore -- here :)\n</pre>\n<p>after</p>\n"}, // html drops the newline after <pre>
		{name: "windows line ends", in: "line one\r\nline two\r\n\r\npara two",
			fancy: "<p>line one<br/>\nline two</p>\n<p>para two</p>\n"},
		{name: "lone angle brackets", in: "x < 10 and y < 1000 and a << b",
			fancy: "<p>x &lt; 10 and y &lt; 1000 and a &lt;&lt; b</p>\n"},
		{name: "capital P", in: "Wordpress rules (Wordpress) aWordpress", fancy: "<p>WordPress rules (WordPress) aWordpress</p>\n"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			xml := `<rss version="2.0" xmlns:wp="http://wordpress.org/export/1.2/"><channel><wp:wxr_version>1.2</wp:wxr_version>` +
				`<item><link>https://example.com/p1</link><wp:comment><wp:comment_id>1</wp:comment_id>` +
				`<wp:comment_author>A</wp:comment_author><wp:comment_date>2020-01-01 10:00:00</wp:comment_date>` +
				`<wp:comment_content><![CDATA[` + tt.in + `]]></wp:comment_content>` +
				`<wp:comment_approved>1</wp:comment_approved></wp:comment></item></channel></rss>`
			for _, fancy := range []bool{true, false} {
				want := tt.fancy
				if !fancy && tt.plain != "" {
					want = tt.plain
				}
				ds := newImportStore(t)
				_, err := (&WordPress{DataStore: ds, DisableFancyTextFormatting: !fancy}).Import(strings.NewReader(xml), "test")
				require.NoError(t, err)
				c, err := ds.Get(store.Locator{SiteID: "test", URL: "https://example.com/p1"}, "1", adminUser)
				require.NoError(t, err)
				assert.Equal(t, want, c.Text, "fancy %v", fancy)
			}
		})
	}
}

var xmlTestWP = `
<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0"
	xmlns:excerpt="http://wordpress.org/export/1.2/excerpt/"
	xmlns:content="http://purl.org/rss/1.0/modules/content/"
	xmlns:wfw="http://wellformedweb.org/CommentAPI/"
	xmlns:dc="http://purl.org/dc/elements/1.1/"
	xmlns:wp="http://wordpress.org/export/1.2/"
>

<channel>
	<title>Real Men Wear Dress.es</title>
	<link>https://realmenweardress.es</link>
	<description>SuperAdmin&#039;s gaming and technological musings</description>
	<pubDate>Mon, 23 Jul 2018 10:21:47 +0000</pubDate>
	<language>en-US</language>
	<wp:wxr_version>1.2</wp:wxr_version>
	<wp:base_site_url>https://realmenweardress.es</wp:base_site_url>
	<wp:base_blog_url>https://realmenweardress.es</wp:base_blog_url>

	<wp:author><wp:author_id>2</wp:author_id><wp:author_login><![CDATA[SuperAdmin]]></wp:author_login><wp:author_email><![CDATA[superadmin@super.eu]]></wp:author_email><wp:author_display_name><![CDATA[SuperAdmin]]></wp:author_display_name><wp:author_first_name><![CDATA[SuperAdmin]]></wp:author_first_name><wp:author_last_name><![CDATA[superadmin]]></wp:author_last_name></wp:author>
	<wp:author><wp:author_id>1</wp:author_id><wp:author_login><![CDATA[admin]]></wp:author_login><wp:author_email><![CDATA[superadmin@superadmin.co.uk]]></wp:author_email><wp:author_display_name><![CDATA[admin]]></wp:author_display_name><wp:author_first_name><![CDATA[]]></wp:author_first_name><wp:author_last_name><![CDATA[]]></wp:author_last_name></wp:author>

	<wp:category>
		<wp:term_id>25</wp:term_id>
		<wp:category_nicename><![CDATA[cataclysm]]></wp:category_nicename>
		<wp:category_parent><![CDATA[]]></wp:category_parent>
		<wp:cat_name><![CDATA[Cataclysm]]></wp:cat_name>
	</wp:category>

	<wp:tag>
		<wp:term_id>39</wp:term_id>
		<wp:tag_slug><![CDATA[addons]]></wp:tag_slug>
		<wp:tag_name><![CDATA[addons]]></wp:tag_name>
	</wp:tag>

	<generator>https://wordpress.org/?v=4.8.1</generator>

	<item>
		<title>Post without comments</title>
		<link>https://realmenweardress.es/2010/06/hello-world/screenshot_013110_200413/</link>
		<pubDate>Sat, 19 Jun 2010 08:34:13 +0000</pubDate>
		<dc:creator><![CDATA[admin]]></dc:creator>
		<guid isPermaLink="false">http://realmenweardress.es/wp-content/uploads/2010/06/ScreenShot_013110_200413.jpeg</guid>
		<description></description>
		<content:encoded><![CDATA[So you can actually fly into the well it appears and if your lucky you stay mounted. I imagine it terrifies the poor rats.]]></content:encoded>
		<excerpt:encoded><![CDATA[]]></excerpt:encoded>
		<wp:post_id>6</wp:post_id>
		<wp:post_date><![CDATA[2010-06-19 08:34:13]]></wp:post_date>
		<wp:post_date_gmt><![CDATA[2010-06-19 08:34:13]]></wp:post_date_gmt>
		<wp:comment_status><![CDATA[open]]></wp:comment_status>
		<wp:ping_status><![CDATA[open]]></wp:ping_status>
		<wp:post_name><![CDATA[screenshot_013110_200413]]></wp:post_name>
		<wp:status><![CDATA[inherit]]></wp:status>
		<wp:post_parent>1</wp:post_parent>
		<wp:menu_order>0</wp:menu_order>
		<wp:post_type><![CDATA[attachment]]></wp:post_type>
		<wp:post_password><![CDATA[]]></wp:post_password>
		<wp:is_sticky>0</wp:is_sticky>
		<wp:attachment_url><![CDATA[https://realmenweardress.es/wp-content/uploads/2010/06/ScreenShot_013110_200413-e1277214413194.jpeg]]></wp:attachment_url>
		<wp:postmeta>
			<wp:meta_key><![CDATA[_wp_attached_file]]></wp:meta_key>
			<wp:meta_value><![CDATA[2010/06/ScreenShot_013110_200413-e1277214413194.jpeg]]></wp:meta_value>
		</wp:postmeta>
	</item>
	<item>
		<title>Post with comments. One is not approved</title>
		<link>https://realmenweardress.es/2010/07/do-you-rp/</link>
		<pubDate>Mon, 19 Jul 2010 14:24:22 +0000</pubDate>
		<dc:creator><![CDATA[SuperAdmin]]></dc:creator>
		<guid isPermaLink="false">http://realmenweardress.es/?p=100</guid>
		<description></description>
		<content:encoded><![CDATA[<a href="http://realmenweardress.es/wp-content/uploads/2010/07/ScreenShot_071410_230307-e1279546180886.jpeg"><img class="size-thumbnail wp-image-102 alignleft" title="I need to stand on things else I can't reach" src="http://realmenweardress.es/wp-content/uploads/2010/07/ScreenShot_071410_230307-e1279546270587-120x120.jpg" alt="I need to stand on things else I can't reach" width="120" height="120" /></a>Meet Grokknomel?]]></content:encoded>
		<excerpt:encoded><![CDATA[]]></excerpt:encoded>
		<wp:post_id>100</wp:post_id>
		<wp:post_date><![CDATA[2010-07-19 14:24:22]]></wp:post_date>
		<wp:post_date_gmt><![CDATA[2010-07-19 14:24:22]]></wp:post_date_gmt>
		<wp:comment_status><![CDATA[open]]></wp:comment_status>
		<wp:ping_status><![CDATA[open]]></wp:ping_status>
		<wp:post_name><![CDATA[do-you-rp]]></wp:post_name>
		<wp:status><![CDATA[publish]]></wp:status>
		<wp:post_parent>0</wp:post_parent>
		<wp:menu_order>0</wp:menu_order>
		<wp:post_type><![CDATA[post]]></wp:post_type>
		<wp:post_password><![CDATA[]]></wp:post_password>
		<wp:is_sticky>0</wp:is_sticky>
		<category domain="post_tag" nicename="alts"><![CDATA[alts]]></category>
		<category domain="post_tag" nicename="role-playing"><![CDATA[role playing]]></category>
		<category domain="category" nicename="stuff"><![CDATA[Stuff]]></category>
		<category domain="post_tag" nicename="weird-in-a-cant-quite-help-myself-way"><![CDATA[weird in a can't quite help myself way]]></category>
		<wp:postmeta>
			<wp:meta_key><![CDATA[_edit_last]]></wp:meta_key>
			<wp:meta_value><![CDATA[2]]></wp:meta_value>
		</wp:postmeta>
		<wp:comment>
			<wp:comment_id>8</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser1]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[superuser1@aol.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://superuser1.blogspot.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[79.141.141.73]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-20 12:08:08]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-20 12:08:08]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[I do catch myself]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>9</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser2]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[superuser2@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://thewowstorm.wordpress.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[97.36.113.1]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-20 13:09:25]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-20 13:09:25]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[I think it us inherent in the game to start seeing your character as a personality]]></wp:comment_content>
			<wp:comment_approved><![CDATA[0]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>13</wp:comment_id>
			<wp:comment_author><![CDATA[Wednesday Reading &laquo; Cynwise&#039;s Battlefield Manual]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[]]></wp:comment_author_email>
			<wp:comment_author_url>http://cynwise.wordpress.com/2010/07/21/wednesday-reading-8/</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[74.200.244.101]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-21 14:02:08]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-21 14:02:08]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[[...] I know I&#8217;m a bit loony with my attachment to my bankers.  I&#8217;m glad I&#8217;m not the only one. [...]]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[pingback]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>14</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser3]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[blablah@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://realmenweardress.es</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[128.243.253.117]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-08-18 15:19:14]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-08-18 15:19:14]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA["Mekkatorque" was over in that tent up to the right]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>13</wp:comment_parent>
			<wp:comment_user_id>2</wp:comment_user_id>
		</wp:comment>
	</item>
	</channel>
</rss>
`

// parts of unused xml tags are omitted
var xmlTestWPmd = `
<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
	<wp:wxr_version>1.2</wp:wxr_version>
	<item>
		<title>Deploying RancherOS on Vultr instances</title>
		<link>https://realmenweardress.es/2016/07/deploying-rancheros-on-vultr-instances/</link>

		<wp:comment>
			<wp:comment_id>1</wp:comment_id>
			<wp:comment_author><![CDATA[user1]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[eric@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>https://eric.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[96.54.240.57]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2017-12-11 00:08:56]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2017-12-11 00:08:56]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[Row1
Row2

Row4]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>

		<wp:comment>
			<wp:comment_id>2</wp:comment_id>
			<wp:comment_author><![CDATA[user1]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[eric@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>https://eric.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[96.54.240.57]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2017-12-11 00:08:56]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2017-12-11 00:08:56]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[markdown ` + "`" + "text" + "`" + `]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>

		<wp:comment>
			<wp:comment_id>2</wp:comment_id>
			<wp:comment_author><![CDATA[user1]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[eric@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>https://eric.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[96.54.240.57]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2017-12-11 00:08:56]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2017-12-11 00:08:56]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[Row1 Link http://releases.rancher.com/os/latest markdown ` + "`" + "text" + "`" + ` blah

Row3 markdown` +
	"```" +
	"md block" +
	"```" +
	`]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>

	</item>
</channel>
</rss>
`
