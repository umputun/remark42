package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/umputun/remark/backend/app/store"
)

const devAuthPort = 8084

// DevAuthServer is a fake oauth server for development
// it provides stand-alone server running on its own port and pretending to be the real oauth2. It also provides
// Dev Provider the same way as normal providers di, i.e. github, google and others.
// can run in interractive and non-interactive mode. In interactive mode login attempts will show login form to select
// desired user name.
type DevAuthServer struct {
	Provider Provider

	username       string // unsafe, but fine for dev
	nonInteractive bool

	httpServer *http.Server
	lock       sync.Mutex
}

// Run oauth2 dev server on port devAuthPort
func (d *DevAuthServer) Run() {
	log.Printf("[INFO] run local oauth2 dev server on %d", devAuthPort)
	d.lock.Lock()
	d.httpServer = &http.Server{
		Addr: fmt.Sprintf(":%d", devAuthPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[DEBUG] dev oauth request %s %s %+v", r.Method, r.URL, r.Header)
			switch {

			case strings.HasPrefix(r.URL.Path, "/login/oauth/authorize"):
				// first time it will be called without usernam and will ask for onw
				if !d.nonInteractive && (r.ParseForm() != nil || r.Form.Get("username") == "") {
					if _, err := w.Write([]byte(fmt.Sprintf(devUserForm, r.URL.RawQuery))); err != nil {
						log.Printf("[WARN] can't write, %s", err)
					}
					return
				}

				if !d.nonInteractive {
					d.username = r.Form.Get("username")
				}

				state := r.URL.Query().Get("state")
				callbackURL := fmt.Sprintf("%s?code=g0ZGZmNjVmOWI&state=%s", d.Provider.RedirectURL, state)
				log.Printf("[DEBUG] callback url=%s", callbackURL)
				w.Header().Add("Location", callbackURL)
				w.WriteHeader(http.StatusFound)

			case strings.HasPrefix(r.URL.Path, "/login/oauth/access_token"):
				res := `{
					"access_token":"MTQ0NjJkZmQ5OTM2NDE1ZTZjNGZmZjI3",
					"token_type":"bearer",
					"expires_in":3600,
					"refresh_token":"IwOGYzYTlmM2YxOTQ5MGE3YmNmMDFkNTVk",
					"scope":"create",
					"state":"12345678"
					}`
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if _, err := w.Write([]byte(res)); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/user"):
				res := fmt.Sprintf(`{
					"id": "%s",
					"name":"%s"
					}`, d.username, d.username)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if _, err := w.Write([]byte(res)); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		}),
	}
	d.lock.Unlock()

	err := d.httpServer.ListenAndServe()
	log.Printf("[WARN] dev oauth2 server terminated, %s", err)
}

// Shutdown oauth2 dev server
func (d *DevAuthServer) Shutdown() {
	log.Print("[WARN] shutdown oauth2 dev server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.lock.Lock()
	if d.httpServer != nil {
		if err := d.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] oauth2 dev shutdown error, %s", err)
		}
	}
	log.Print("[DEBUG] shutdown dev oauth2 server completed")
	d.lock.Unlock()
}

// NewDev makes dev oauth2 provider for admin user
func NewDev(p Params) Provider {
	return initProvider(p, Provider{
		Name: "dev",
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://127.0.0.1:%d/login/oauth/authorize", devAuthPort),
			TokenURL: fmt.Sprintf("http://127.0.0.1:%d/login/oauth/access_token", devAuthPort),
		},
		RedirectURL: "http://127.0.0.1:8080/auth/dev/callback",
		Scopes:      []string{"user:email"},
		InfoURL:     fmt.Sprintf("http://127.0.0.1:%d/user", devAuthPort),
		MapUser: func(data userData, _ []byte) store.User {
			userInfo := store.User{
				ID:      data.value("id"),
				Name:    data.value("name"),
				Picture: "",
			}
			return userInfo
		},
	})
}

var devUserForm = `
<html>
    <head>
	<title>Remark42 Dev User</title>
	<style>
		form {
			margin: 100 auto;
			width: 300px;
			padding: 1em;
			border: 1px solid #CCC;
		}
	</style>
    </head>
	<body>
		<form action="/login/oauth/authorize?%s" method="post">
			username: <input type="text" name="username" value="dev_user">
			<input type="submit" value="Login">
		</form>
    </body>
</html>
`
