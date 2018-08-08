package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/keys"
)

func TestDevProvider(t *testing.T) {
	params := Params{RemarkURL: "http://127.0.0.1:8080", Cid: "cid", Csecret: "csecret",
		JwtService:        NewJWT(keys.NewStaticStore("12345"), false, time.Hour, time.Hour*24*31),
		PermissionChecker: &mockUserPermissions{admin: "dev_user"},
	}
	srv := DevAuthServer{Provider: NewDev(params), nonInteractive: true, username: "dev_user"}

	// auth routes for all providers
	router := chi.NewRouter()
	router.Route("/auth", func(r chi.Router) {
		r.Mount("/dev", srv.Provider.Routes()) // mount auth providers as /auth/{name}
	})

	ts := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", 8080), Handler: router}
	go srv.Run()
	go ts.ListenAndServe()
	defer func() {
		srv.Shutdown()
		_ = ts.Shutdown(context.TODO())
	}()

	time.Sleep(100 * time.Millisecond)

	jar, err := cookiejar.New(nil)
	require.Nil(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	// check non-admin, permanent
	resp, err := client.Get("http://127.0.0.1:8080/auth/dev/login?site=remark")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	t.Logf("resp %s", string(body))
	t.Logf("headers: %+v", resp.Header)

	assert.Equal(t, 2, len(resp.Cookies()))
	assert.Equal(t, "JWT", resp.Cookies()[0].Name)
	assert.NotEqual(t, "", resp.Cookies()[0].Value, "jwt set")
	assert.Equal(t, 2678400, resp.Cookies()[0].MaxAge)
	assert.Equal(t, "XSRF-TOKEN", resp.Cookies()[1].Name)
	assert.NotEqual(t, "", resp.Cookies()[1].Value, "xsrf cookie set")

	claims, err := params.JwtService.Parse(resp.Cookies()[0].Value)
	assert.Nil(t, err)

	u := *claims.User
	assert.Equal(t, store.User{Name: "dev_user", ID: "dev_user", Picture: "http://127.0.0.1:8084/avatar?user=dev_user", IP: "",
		Admin: true, Blocked: false, Verified: false}, u)

	// check avatar
	resp, err = client.Get("http://127.0.0.1:8084/avatar?user=dev_user")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 985, len(body))
	t.Logf("headers: %+v", resp.Header)
}
