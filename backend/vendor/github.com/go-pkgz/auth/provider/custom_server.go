package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/oauth2"
	goauth2 "gopkg.in/oauth2.v3/server"

	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
)

// CustomHandlerOpt are options to initialize a handler for oauth2 server
type CustomHandlerOpt struct {
	Endpoint  oauth2.Endpoint
	InfoURL   string
	MapUserFn func(UserData, []byte) token.User
	Scopes    []string
}

// CustomServerOpt are options to initialize a custom go-oauth2/oauth2 server
type CustomServerOpt struct {
	logger.L
	URL              string
	WithLoginPage    bool
	LoginPageHandler func(w http.ResponseWriter, r *http.Request)
}

// NewCustomServer is helper function to initiate a customer server and prefill
// options needed for provider registration (see Service.AddCustomProvider)
func NewCustomServer(srv *goauth2.Server, sopts CustomServerOpt) *CustomServer {
	copts := CustomHandlerOpt{
		Endpoint: oauth2.Endpoint{
			AuthURL:  sopts.URL + "/authorize",
			TokenURL: sopts.URL + "/access_token",
		},
		InfoURL:   sopts.URL + "/user",
		MapUserFn: defaultMapUserFn,
	}

	return &CustomServer{
		L:                sopts.L,
		URL:              sopts.URL,
		WithLoginPage:    sopts.WithLoginPage,
		LoginPageHandler: sopts.LoginPageHandler,
		OauthServer:      srv,
		HandlerOpt:       copts,
	}
}

// CustomServer is a wrapper over go-oauth2/oauth2 server running on its own port
type CustomServer struct {
	logger.L
	URL              string                                       // root url for custom oauth2 server
	WithLoginPage    bool                                         // redirect to login html page if true
	LoginPageHandler func(w http.ResponseWriter, r *http.Request) // handler for user-defined login page
	OauthServer      *goauth2.Server                              // an instance of go-oauth2/oauth2 server
	HandlerOpt       CustomHandlerOpt
	httpServer       *http.Server
	lock             sync.Mutex
}

// Run starts serving on port from c.URL
func (c *CustomServer) Run(ctx context.Context) {
	c.Logf("[INFO] run local go-oauth2/oauth2 server on %s", c.URL)
	c.lock.Lock()

	u, err := url.Parse(c.URL)
	if err != nil {
		c.Logf("[ERROR] failed to parse service base URL=%s", c.URL)
		return
	}

	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		c.Logf("[ERROR] failed to get port from URL=%s", c.URL)
		return
	}

	c.httpServer = &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/authorize"):
				c.handleAuthorize(w, r)
			case strings.HasSuffix(r.URL.Path, "/access_token"):
				if err = c.OauthServer.HandleTokenRequest(w, r); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			case strings.HasPrefix(r.URL.Path, "/user"):
				c.handleUserInfo(w, r)
			case strings.HasPrefix(r.URL.Path, "/avatar"):
				c.handleAvatar(w, r)
			default:
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}),
	}
	c.lock.Unlock()

	go func() {
		<-ctx.Done()
		c.Logf("[DEBUG] cancellation via context, %v", ctx.Err())
		c.Shutdown()
	}()

	err = c.httpServer.ListenAndServe()
	c.Logf("[WARN] go-oauth2/oauth2 server terminated, %s", err)
}

func (c *CustomServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// called for first time, ask for username
	if c.WithLoginPage || c.LoginPageHandler != nil {
		if r.ParseForm() != nil || r.Form.Get("username") == "" {
			// show default template if user-defined function not specified
			if c.LoginPageHandler != nil {
				c.LoginPageHandler(w, r)
				return
			}
			userLoginTmpl, err := template.New("page").Parse(defaultLoginTmpl)
			if err != nil {
				c.Logf("[ERROR] can't parse user login template, %s", err)
				return
			}

			formData := struct{ Query string }{Query: r.URL.RawQuery}

			if err := userLoginTmpl.Execute(w, formData); err != nil {
				c.Logf("[WARN] can't write, %s", err)
			}
			return
		}
	}

	err := c.OauthServer.HandleAuthorizeRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (c *CustomServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	ti, err := c.OauthServer.ValidationBearerToken(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	userID := ti.GetUserID()

	p := bluemonday.UGCPolicy()
	ava := p.Sanitize(fmt.Sprintf(c.URL+"/avatar?user=%s", userID))
	res := fmt.Sprintf(`{
					"id": "%s",
					"name":"%s",
					"picture":"%s"
					}`, userID, userID, ava)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write([]byte(res)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (c *CustomServer) handleAvatar(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	b, err := avatar.GenerateAvatar(user)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if _, err = w.Write(b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Shutdown go-oauth2/oauth2 server
func (c *CustomServer) Shutdown() {
	c.Logf("[WARN] shutdown go-oauth2/oauth2 server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.lock.Lock()
	if c.httpServer != nil {
		if err := c.httpServer.Shutdown(ctx); err != nil {
			c.Logf("[DEBUG] go-oauth2/oauth2 shutdown error, %s", err)
		}
	}
	c.Logf("[DEBUG] shutdown go-oauth2/oauth2 server completed")
	c.lock.Unlock()
}

// NewCustom creates a handler for go-oauth2/oauth2 server
func NewCustom(name string, p Params, copts CustomHandlerOpt) Oauth2Handler {
	return initOauth2Handler(p, Oauth2Handler{
		name:     name,
		endpoint: copts.Endpoint,
		scopes:   copts.Scopes,
		infoURL:  copts.InfoURL,
		mapUser:  copts.MapUserFn,
	})
}

func defaultMapUserFn(data UserData, _ []byte) token.User {
	userInfo := token.User{
		ID:      data.Value("id"),
		Name:    data.Value("name"),
		Picture: data.Value("picture"),
	}
	return userInfo
}

var defaultLoginTmpl = `
<html>
	<head>
		<title>Dev OAuth</title>
		<style>
			body {
				text-align: center;
			}

			a {
				color: hsl(200, 50%, 50%);
				text-decoration-color: hsla(200, 50%, 50%, 0.5);
			}

			a:hover {
				color: hsl(200, 50%, 70%);
				text-decoration-color: hsla(200, 50%, 70%, 0.5);
			}
			
			form {
				font-family: Helvetica, Arial, sans-serif;
				margin: 100px auto;
				display: inline-block;
				padding: 1em;
				box-shadow: 0 0 0.1rem rgba(0, 0, 0, 0.2), 0 0 0.4rem rgba(0, 0, 0, 0.1);
			}

			.form-header {
				text-align: center;
			}

			.form-header h1 {
				margin: 0;
			}

			.form-header h1 a:not(:hover) {
				text-decoration: none;
			}

			.form-header p {
				opacity: 0.6;
				margin-top: 0;
				margin-bottom: 2rem;
			}

			.username-label {
				opacity: 0.6;
				font-size: 0.8em;
			}

			.username-input {
				font-size: inherit;
				margin: 0;
				width: 100%;
				text-align: inherit;
			}

			.form-submit {
				border: none;
				background: hsl(200, 50%, 50%);
				color: white;
				font: inherit;
				padding: 0.4em 0.8em 0.3em 0.8em;
				border-radius: 0.2em;
				width: 100%;
			}

			.form-submit:hover,
			.form-submit:focus {
				background-color: hsl(200, 50%, 70%);
			}

			.form-submit:active {
				background-color: hsl(200, 80%, 70%);
			}

			.username-label,
			.username-input,
			.form-submit {
				display: block;
				margin-bottom: 0.4rem;
			}

			.notice {
				margin: 0;
				margin-top: 2rem;
				font-size: 0.8em;
				opacity: 0.6;
			}
		</style>
	</head>
	<body>
		<form action="/login/oauth/authorize?{{.Query}}" method="POST">
			<header class="form-header">
				<h1><a href="https://github.com/go-oauth2/oauth2">go-oauth2/oauth2</a></h1>
				<p>Golang OAuth 2.0 Server</p>
			</header>
			<label>
				<span class="username-label">Username</span>
				<input
					class="username-input"
					type="text"
					name="username"
					value=""
					autofocus
				/>
			</label>

			<label>
			<span class="username-label">Password</span>
			<input
				class="username-input"
				type="password"
				name="password"
				value=""
				autofocus
			/>
			</label>

			<input type="submit" class="form-submit" value="Authorize" />
			<p class="notice"></p>
		</form>
	</body>
	<script>
		var input = document.querySelector(".username-input");
		input.focus();
		input.setSelectionRange(0, input.value.length)
	</script>
</html>
`
