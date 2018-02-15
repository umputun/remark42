package avatar

import (
	"bytes"
	"io"
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
	assert.Equal(t, "/avatar/user1.image", res)
	fi, err := os.Stat("/tmp/avatars.test/20/user1.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(8432), fi.Size())

	u.ID = "user2"
	res, err = p.Put(u)
	assert.NoError(t, err)
	assert.Equal(t, "/avatar/user2.image", res)
	fi, err = os.Stat("/tmp/avatars.test/92/user2.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(8432), fi.Size())
}

func TestRoutes(t *testing.T) {
	p := Proxy{StorePath: "/tmp/avatars.test", RoutePath: "/avatar"}
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	u := store.User{ID: "user1", Name: "user1 name", Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png"}
	_, err := p.Put(u)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/user1.image", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.Handler(p.Routes())
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, http.Header{"Content-Type": []string{"image/*"}}, rr.HeaderMap)
	bb := bytes.Buffer{}
	sz, err := io.Copy(&bb, rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, int64(8432), sz)
}
