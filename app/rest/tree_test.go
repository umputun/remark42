package rest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
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

	res := MakeTree(comments, "time")

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	assert.Nil(t, err)
	expected, actual := cleanFormatting(expJSON, buf.String())
	assert.Equal(t, expected, actual)
	assert.Equal(t, store.PostInfo{URL: "url", Count: 17, FirstTS: ts(46, 1), LastTS: ts(47, 22)}, res.Info)

	res = MakeTree([]store.Comment{}, "time")
	assert.Equal(t, &Tree{}, res)
}

func TestTreeSortNodes(t *testing.T) {
	// unsorted by purpose
	comments := []store.Comment{
		{ID: "14", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 1, 0, time.UTC), Score: 2},
		{ID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 2, 0, time.UTC), Score: 3},
		{ID: "11", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 11, 0, time.UTC)},
		{ID: "13", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 13, 0, time.UTC)},
		{ID: "12", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "131", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 50, 31, 0, time.UTC)},
		{ID: "132", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 46, 32, 0, time.UTC)},
		{ID: "21", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 21, 0, time.UTC)},
		{ID: "22", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "4", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC), Score: -2},
		{ID: "3", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 100, time.UTC)},
		{ID: "6", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 200, time.UTC)},
		{ID: "5", Deleted: true},
	}

	res := MakeTree(comments, "+active")
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)

	res = MakeTree(comments, "-active")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "+time")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-time")
	assert.Equal(t, "6", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "score")
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)
	assert.Equal(t, "3", res.Nodes[1].Comment.ID)
	assert.Equal(t, "6", res.Nodes[2].Comment.ID)
	assert.Equal(t, "1", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "+score")
	assert.Equal(t, "4", res.Nodes[0].Comment.ID)

	res = MakeTree(comments, "-score")
	assert.Equal(t, "2", res.Nodes[0].Comment.ID)
	assert.Equal(t, "1", res.Nodes[1].Comment.ID)
	assert.Equal(t, "3", res.Nodes[2].Comment.ID)
	assert.Equal(t, "6", res.Nodes[3].Comment.ID)

	res = MakeTree(comments, "undefined")
	t.Log(res.Nodes[0].Comment.ID, res.Nodes[0].tsModified)
	assert.Equal(t, "1", res.Nodes[0].Comment.ID)

}

func BenchmarkTree(b *testing.B) {
	comments := []store.Comment{}
	data, err := ioutil.ReadFile("testfile.json")
	assert.Nil(b, err)
	err = json.Unmarshal(data, &comments)
	assert.Nil(b, err)

	for i := 0; i < b.N; i++ {
		res := MakeTree(comments, "time")
		assert.NotNil(b, res)
	}
}

const expJSON = `{
  "comments": [
    {
      "comment": {
        "id": "1",
        "pid": "",
        "text": "",
        "user": {
          "name": "",
          "id": "",
          "picture": "",
          "admin": false
        },
        "locator": { 
          "site": "site",
          "url": "url"
        },
        "score": 0,
        "votes": null,
        "time": "2017-12-25T19:46:01Z"
      },
      "replies": [
        {
          "comment": {
            "id": "11",
            "pid": "1",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
          	  "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:46:11Z"
          }
        },
        {
          "comment": {
            "id": "12",
            "pid": "1",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
              "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:46:12Z"
          }
        },
        {
          "comment": {
            "id": "13",
            "pid": "1",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
              "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:46:13Z"
          },
          "replies": [
            {
              "comment": {
                "id": "131",
                "pid": "13",
                "text": "",
                "user": {
                  "name": "",
                  "id": "",
                  "picture": "",
                  "admin": false
                },
                "locator": {
                  "site": "site",
                  "url": "url"
                },
                "score": 0,
                "votes": null,
                "time": "2017-12-25T19:46:31Z"
              }
            },
            {
              "comment": {
                "id": "132",
                "pid": "13",
                "text": "",
                "user": {
                  "name": "",
                  "id": "",
                  "picture": "",
                  "admin": false
                },
                "locator": {
                  "site": "site",
                  "url": "url"
                },
                "score": 0,
                "votes": null,
                "time": "2017-12-25T19:46:32Z"
              }
            }
          ]
        },
        {
          "comment": {
            "id": "14",
            "pid": "1",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
              "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:46:14Z"
          }
        }
      ]
    },
    {
      "comment": {
        "id": "2",
        "pid": "",
        "text": "",
        "user": {
          "name": "",
          "id": "",
          "picture": "",
          "admin": false
        },
        "locator": {
          "site": "site",
          "url": "url"
        },
        "score": 0,
        "votes": null,
        "time": "2017-12-25T19:47:02Z"
      },
      "replies": [
        {
          "comment": {
            "id": "21",
            "pid": "2",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
              "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:47:21Z"
          }
        },
        {
          "comment": {
            "id": "22",
            "pid": "2",
            "text": "",
            "user": {
              "name": "",
              "id": "",
              "picture": "",
              "admin": false
            },
            "locator": {
              "site": "site",
              "url": "url"
            },
            "score": 0,
            "votes": null,
            "time": "2017-12-25T19:47:22Z"
          }
        }
      ]
    },
    {
      "comment": {
        "id": "4",
        "pid": "",
        "text": "",
        "user": {
          "name": "",
          "id": "",
          "picture": "",
          "admin": false
        },
        "locator": {
          "site": "site",
          "url": "url"
        },
        "score": 0,
        "votes": null,
        "time": "2017-12-25T19:47:22Z"
      }
    },
    {
      "comment": {
        "id": "3",
        "pid": "",
        "text": "",
        "user": {
          "name": "",
          "id": "",
          "picture": "",
          "admin": false
        },
        "locator": {
          "site": "site",
          "url": "url"
        },
        "score": 0,
        "votes": null,
        "time": "2017-12-25T19:47:22Z"
      }
    }
  ],
  "info": {
	 "url": "url",
	 "count": 17,
     "first_time": "2017-12-25T19:46:01Z",
     "last_time": "2017-12-25T19:47:22Z"
   }
}
`

func cleanFormatting(expected, actual string) (string, string) {
	reSpaces := regexp.MustCompile(`[\s\p{Zs}]{2,}`)

	expected = strings.Replace(expected, "\n", " ", -1)
	expected = strings.Replace(expected, "\t", " ", -1)
	expected = reSpaces.ReplaceAllString(expected, " ")

	actual = strings.Replace(actual, "\n", " ", -1)
	actual = reSpaces.ReplaceAllString(actual, " ")
	return expected, actual
}
