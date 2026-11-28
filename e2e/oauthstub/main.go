// Command oauthstub answers as an OAuth2 provider for the e2e suite.
//
// It stands on a site of its own, apart from remark42 and from the page embedding the widget,
// which is where a real provider stands, so the sign-in popup crosses the same boundaries. The
// authorize page shows a sign-in link the test clicks, as a reader clicks through a real provider.
//
// Two cookies on the stub's own origin, set by the test on its browser context, shape a run: "as"
// names the account the page signs in, and "coop" makes the page send
// Cross-Origin-Opener-Policy: same-origin, which cuts the popup off from the frame that opened it,
// as several real providers do. A browser applies that header only on a secure origin, so the
// page the browser reaches is served over TLS as well; remark42 itself calls the token and user
// endpoints over plain http. The page says whether it still has an opener, for the test to check.
package main

import (
	"cmp"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var page = template.Must(template.New("authorize").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Stub provider</title></head>
<body><p>Sign in to remark42 as {{.User}}</p><p id="opener"></p><a id="signin" href="{{.Next}}">Sign in</a>
<script>document.getElementById('opener').textContent = window.opener ? 'opener' : 'no opener';</script></body></html>
`))

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		back, err := url.Parse(q.Get("redirect_uri"))
		if err != nil || back.Scheme == "" {
			http.Error(w, "bad redirect_uri", http.StatusBadRequest)
			return
		}
		user := "oauth-reader"
		if c, cerr := r.Cookie("as"); cerr == nil && c.Value != "" {
			user = c.Value
		}
		back.RawQuery = url.Values{"code": {"c-" + user}, "state": {q.Get("state")}}.Encode()
		if c, cerr := r.Cookie("coop"); cerr == nil && c.Value == "1" {
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, struct{ User, Next string }{User: user, Next: back.String()})
	})

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, ok := strings.CutPrefix(r.PostForm.Get("code"), "c-")
		if !ok || user == "" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at-" + user, "token_type": "bearer"})
	})

	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		user, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer at-")
		if !ok || user == "" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"sub": user, "name": user})
	})

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("pong")) })

	if tlsAddr := os.Getenv("STUB_TLS_ADDR"); tlsAddr != "" {
		go func() {
			//nolint:gosec // a test stub on a loopback-only port
			log.Fatal(http.ListenAndServeTLS(tlsAddr, os.Getenv("STUB_TLS_CERT"), os.Getenv("STUB_TLS_KEY"), mux))
		}()
	}
	addr := cmp.Or(os.Getenv("STUB_ADDR"), ":8092")
	log.Print("oauth stub listening")
	log.Fatal(http.ListenAndServe(addr, mux)) //nolint:gosec // a test stub on a loopback-only port
}
