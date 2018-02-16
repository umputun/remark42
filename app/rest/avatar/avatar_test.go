package avatar

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

func TestPut(t *testing.T) {
	p := Proxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name", Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png"}
	res, err := p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "/avatar/b3daa77b4c04a9551b8781d03191fe098f325e67.image", res)
	fi, err := os.Stat("/tmp/avatars.test/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(8432), fi.Size())

	u.ID = "user2"
	res, err = p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "/avatar/a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image", res)
	fi, err = os.Stat("/tmp/avatars.test/84/a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(8432), fi.Size())
}

func TestPutDefault(t *testing.T) {
	p := Proxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar", DefaultAvatar: "default.image"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	ioutil.WriteFile("/tmp/avatars.test/default.image", []byte("1234567890"), 0600)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name"}
	res, err := p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "/avatar/default.image", res)
	fi, err := os.Stat("/tmp/avatars.test/default.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), fi.Size())

}
func TestRoutes(t *testing.T) {
	p := Proxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar", DefaultAvatar: "default.image"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name", Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png"}
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
	assert.Equal(t, int64(8432), sz)
}
func TestRoutesDefault(t *testing.T) {
	p := Proxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar", DefaultAvatar: "default.image"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	ioutil.WriteFile("/tmp/avatars.test/default.image", []byte("1234567890"), 0600)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name"}
	_, err := p.Put(u)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/no-such-thing.image", nil)
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
	assert.Equal(t, int64(10), sz)
}
