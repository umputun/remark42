package auth

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

func TestPut(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pic.png" {
			w.Header().Set("Content-Type", "image/*")
			fmt.Fprint(w, "some picture bin data")
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	p := AvatarProxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar", RemarkURL: "http://localhost:8080"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name", Picture: ts.URL + "/pic.png"}
	res, err := p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/avatar/b3daa77b4c04a9551b8781d03191fe098f325e67.image", res)
	fi, err := os.Stat("/tmp/avatars.test/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(21), fi.Size())

	u.ID = "user2"
	res, err = p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/avatar/a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image", res)
	fi, err = os.Stat("/tmp/avatars.test/84/a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(21), fi.Size())
}

func TestPutNoAvatar(t *testing.T) {
	p := AvatarProxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar"}
	u := store.User{ID: "user1", Name: "user1 name"}
	_, err := p.Put(u)
	assert.Error(t, err)
}

func TestRoutes(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pic.png" {
			w.Header().Set("Content-Type", "image/*")
			fmt.Fprint(w, "some picture bin data")
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	p := AvatarProxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name", Picture: ts.URL + "/pic.png"}
	_, err := p.Put(u)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/b3daa77b4c04a9551b8781d03191fe098f325e67.image", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	_, routes := p.Routes()
	handler := http.Handler(routes)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, http.Header{"Content-Type": []string{"image/*"}}, rr.HeaderMap)
	bb := bytes.Buffer{}
	sz, err := io.Copy(&bb, rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, int64(21), sz)
	assert.Equal(t, "some picture bin data", bb.String())
}

func TestLocation(t *testing.T) {
	p := AvatarProxy{StorePath: "/tmp/avatars.test"}

	tbl := []struct {
		id  string
		res string
	}{
		{"abc", "/tmp/avatars.test/35"},
		{"xyz", "/tmp/avatars.test/69"},
		{"blah blah", "/tmp/avatars.test/29"},
	}

	for i, tt := range tbl {
		assert.Equal(t, tt.res, p.location(tt.id), "test #%d", i)
	}
}
