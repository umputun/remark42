package rest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

func TestStore_MakeTree(t *testing.T) {

	// unsorted by purpose
	comments := []store.Comment{
		{ID: "14", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 14, 0, time.UTC)},
		{ID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 1, 0, time.UTC)},
		{ID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 2, 0, time.UTC)},
		{ID: "11", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 11, 0, time.UTC)},
		{ID: "13", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 13, 0, time.UTC)},
		{ID: "12", ParentID: "1", Timestamp: time.Date(2017, 12, 25, 19, 46, 12, 0, time.UTC)},
		{ID: "131", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 46, 31, 0, time.UTC)},
		{ID: "132", ParentID: "13", Timestamp: time.Date(2017, 12, 25, 19, 46, 32, 0, time.UTC)},
		{ID: "21", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 21, 0, time.UTC)},
		{ID: "22", ParentID: "2", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "4", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "3", Timestamp: time.Date(2017, 12, 25, 19, 47, 22, 0, time.UTC)},
		{ID: "5", Deleted: true},
	}

	res := MakeTree(comments, "time")

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	assert.Nil(t, err)
	assert.Equal(t, expJSON, buf.String())
	// t.Log(string(buf.Bytes()))
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
          "url": ""
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
              "url": ""
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
              "url": ""
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
              "url": ""
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
                  "url": ""
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
                  "url": ""
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
              "url": ""
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
          "url": ""
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
              "url": ""
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
              "url": ""
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
          "url": ""
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
          "url": ""
        },
        "score": 0,
        "votes": null,
        "time": "2017-12-25T19:47:22Z"
      }
    }
  ]
}
`
