package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/app/store"
)

func TestProviders_NewGoogle(t *testing.T) {
	r := NewGoogle(Params{RemarkURL: "http://demo.remark42.com", Cid: "cid", Csecret: "cs"})
	assert.Equal(t, "google", r.Name)

	udata := userData{"sub": "1234567890", "name": "test user", "picture": "http://demo.remark42.com/blah.png"}
	user := r.MapUser(udata, nil)
	assert.Equal(t, store.User{Name: "test user", ID: "google_01b307acba4f54f55aafc33bb06bbbf6ca803e9a",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)

	// no name in data
	udata = userData{"sub": "1234567890", "picture": "http://demo.remark42.com/blah.png"}
	user = r.MapUser(udata, nil)
	assert.Equal(t, store.User{Name: "noname_1b30", ID: "google_01b307acba4f54f55aafc33bb06bbbf6ca803e9a",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)
}

func TestProviders_NewGithub(t *testing.T) {
	r := NewGithub(Params{RemarkURL: "http://demo.remark42.com", Cid: "cid", Csecret: "cs"})
	assert.Equal(t, "github", r.Name)

	udata := userData{"login": "lll", "name": "test user", "avatar_url": "http://demo.remark42.com/blah.png"}
	user := r.MapUser(udata, nil)
	assert.Equal(t, store.User{Name: "test user", ID: "github_e80b2d2608711cbb3312db7c4727a46fbad9601a",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)

	// nil name in data (json response contains `"name": null`); using login, it's always required
	udata = userData{"login": "lll", "name": nil, "avatar_url": "http://demo.remark42.com/blah.png"}
	user = r.MapUser(udata, nil)
	assert.Equal(t, store.User{Name: "lll", ID: "github_e80b2d2608711cbb3312db7c4727a46fbad9601a",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)
}

func TestProviders_NewFacebook(t *testing.T) {
	r := NewFacebook(Params{RemarkURL: "http://demo.remark42.com", Cid: "cid", Csecret: "cs"})
	assert.Equal(t, "facebook", r.Name)

	udata := userData{"id": "myid", "name": "test user"}
	user := r.MapUser(udata, []byte(`{"picture": {"data": {"url": "http://demo.remark42.com/blah.png"} }}`))
	assert.Equal(t, store.User{Name: "test user", ID: "facebook_6e34471f84557e1713012d64a7477c71bfdac631",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)

	udata = userData{"id": "myid", "name": ""}
	user = r.MapUser(udata, []byte(`{"picture": {"data": {"url": "http://demo.remark42.com/blah.png"} }}`))
	assert.Equal(t, store.User{Name: "facebook_6e34471", ID: "facebook_6e34471f84557e1713012d64a7477c71bfdac631",
		Picture: "http://demo.remark42.com/blah.png", Admin: false, Blocked: false, IP: ""}, user, "got %+v", user)

}
