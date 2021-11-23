package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/store"
)

func TestUser_GetUserInfo(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah.com", http.NoBody)
	assert.NoError(t, err)
	_, err = GetUserInfo(r)
	assert.Error(t, err, "no user info")

	r = SetUserInfo(r, store.User{Name: "test", ID: "id", SiteID: "test"})
	u, err := GetUserInfo(r)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "test", ID: "id", SiteID: "test"}, u)
}

func TestUser_MustGetUserInfo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("recovered from panic")
		}
	}()

	r, err := http.NewRequest("GET", "http://blah.com", http.NoBody)
	assert.NoError(t, err)
	_ = MustGetUserInfo(r)
	assert.Fail(t, "should panic")

	r = SetUserInfo(r, store.User{Name: "test", ID: "id"})
	u := MustGetUserInfo(r)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "test", ID: "id"}, u)
}
