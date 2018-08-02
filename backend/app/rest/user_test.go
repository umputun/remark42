package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/backend/app/store"
)

func TestUser_GetUserInfo(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah.com", nil)
	assert.Nil(t, err)
	_, err = GetUserInfo(r)
	assert.NotNil(t, err, "no user info")

	r = SetUserInfo(r, store.User{Name: "test", ID: "id"})
	u, err := GetUserInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "test", ID: "id"}, u)
}

func TestUSer_MustGetUserInfo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("recovered from panic")
		}
	}()

	r, err := http.NewRequest("GET", "http://blah.com", nil)
	assert.Nil(t, err)
	_ = MustGetUserInfo(r)
	assert.Fail(t, "should panic")

	r = SetUserInfo(r, store.User{Name: "test", ID: "id"})
	u := MustGetUserInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, store.User{Name: "test", ID: "id"}, u)
}
