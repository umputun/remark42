package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	R "github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"

	"github.com/umputun/remark42/backend/app/rest"
)

// A widget framed by a page on another site signs in with OAuth through a popup, and the popup is a
// top-level page of its own: the session cookie its callback sets belongs to remark42's own site,
// and a browser that keeps third-party cookies apart never shows it to the frame. The handoff
// carries that sign-in over to the frame.
//
// The popup opens on the start page instead of going to the provider at once. The page creates a
// transaction bound to this browser by a cookie of its own and to the frame by the S256 challenge of
// a verifier only the frame holds, tells the frame which transaction it is, waits for the frame to
// acknowledge, and only then goes on to the provider's ordinary login, asking it to come back to the
// done page with the transaction id. Whatever goes wrong before that, the page goes to the login
// exactly as the widget would have without it, so a sign-in that works today still does.
//
// The callback is the provider's own, unchanged. The response it gives sets the new session's
// cookie and redirects to the done page, and recordHandoff reads both from that one response, so
// the account the transaction records is the one this provider round signed in, whatever another tab
// does with the browser's session in the meantime. The frame waits on the transaction's status. When
// the session the frame can already see is that account, as it is on the same site and wherever the
// browser still sends remark42's own cookies to the frame, it is done. Otherwise it redeems the
// transaction with its verifier and is given a session of its own, set by embeddedSession from
// inside the frame, so the browser keeps it in the frame's partition.
const (
	handoffRoutePrefix = "/auth/handoff/"
	handoffDonePath    = handoffRoutePrefix + "done"

	// how long a transaction is kept, from the start page. The auth library's own handshake lasts
	// 30 minutes; the widget stops waiting after five
	handoffTTL = 10 * time.Minute
	// transactions kept at once. Past that the start page falls back to the ordinary sign-in
	handoffMaxPending = 10000
	// how long a status request waits for a pending transaction to change
	handoffStatusWait = 20 * time.Second
	// status requests waiting at once. Each holds a slot of the server's concurrency limit, so past
	// this one answers at once and the widget asks again a second later
	handoffMaxWaiters = 100
	// the popup's binding cookie, followed by the transaction id
	handoffBindingCookiePrefix = "remark42-handoff-"
)

// Cookies of a frame's embedded session. They carry the same session the auth library writes to
// JWT and XSRF-TOKEN, under names of their own so that a browser sending the frame both the ordinary
// pair and this one keeps them apart. The marker says the frame chose the embedded session and
// outlives a sign-out, so a frame that signed out stays out of the ordinary session. The __Host-
// prefix makes a browser take them from this host alone, Secure and on the root path, so a page
// on a sibling subdomain or a forged plain-http response cannot plant one and turn remark42's own
// pages to an embedded session.
const (
	embeddedJWTCookie    = "__Host-JWT-EMBEDDED"
	embeddedXSRFCookie   = "__Host-XSRF-TOKEN-EMBEDDED"
	embeddedMarkerCookie = "__Host-REMARK42-EMBEDDED"

	authJWTCookie  = "JWT"
	authXSRFCookie = "XSRF-TOKEN"
	authJWTHeader  = "X-JWT"
)

var handoffChallengeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

type handoffState int

const (
	handoffPending handoffState = iota
	handoffDone
	handoffCanceled
)

func (s handoffState) String() string {
	switch s {
	case handoffDone:
		return "done"
	case handoffCanceled:
		return "canceled"
	default:
		return "pending"
	}
}

// handoff is one sign-in carried over to a frame
type handoff struct {
	challenge string   // S256 of the frame's verifier, base64url without padding
	binding   [32]byte // sha256 of the value of the popup's binding cookie
	site      string
	expires   time.Time
	state     handoffState
	claims    token.Claims  // the session the callback issued, once done
	changed   chan struct{} // closed when state leaves pending
}

// handoffStore keeps transactions in memory. A restart loses the pending ones, which only cancels
// sign-ins under way: the widget gives up on them and the reader signs in again
type handoffStore struct {
	mu   sync.Mutex
	recs map[string]*handoff
	max  int
	ttl  time.Duration
	now  func() time.Time
}

func newHandoffStore() *handoffStore {
	return &handoffStore{recs: map[string]*handoff{}, max: handoffMaxPending, ttl: handoffTTL, now: time.Now}
}

var (
	errHandoffNotFound = errors.New("no such sign-in")
	errHandoffDenied   = errors.New("sign-in does not match")
	errHandoffFull     = errors.New("too many sign-ins under way")
	errHandoffState    = errors.New("sign-in is not complete")
	errHandoffSeen     = errors.New("the frame sees this account already")
)

// create starts a transaction and returns its id and the value of its binding cookie
func (s *handoffStore) create(challenge, site string) (id, binding string, err error) {
	idb, bb := make([]byte, 16), make([]byte, 32)
	if _, err = rand.Read(idb); err != nil {
		return "", "", fmt.Errorf("can't make sign-in id: %w", err)
	}
	if _, err = rand.Read(bb); err != nil {
		return "", "", fmt.Errorf("can't make sign-in binding: %w", err)
	}
	id, binding = hex.EncodeToString(idb), base64.RawURLEncoding.EncodeToString(bb)

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.recs) >= s.max {
		s.purgeLocked()
	}
	if len(s.recs) >= s.max {
		return "", "", errHandoffFull
	}
	s.recs[id] = &handoff{challenge: challenge, binding: sha256.Sum256([]byte(binding)), site: site,
		expires: s.now().Add(s.ttl), changed: make(chan struct{})}
	return id, binding, nil
}

// purgeLocked removes expired transactions
func (s *handoffStore) purgeLocked() {
	now := s.now()
	for id, h := range s.recs {
		if now.After(h.expires) {
			s.dropLocked(id, h)
		}
	}
}

// dropLocked removes a transaction and wakes whoever waits on it
func (s *handoffStore) dropLocked(id string, h *handoff) {
	delete(s.recs, id)
	if h.state == handoffPending {
		h.state = handoffCanceled
		close(h.changed)
	}
}

// getLocked returns a live transaction
func (s *handoffStore) getLocked(id string) (*handoff, bool) {
	h, ok := s.recs[id]
	if !ok {
		return nil, false
	}
	if s.now().After(h.expires) {
		s.dropLocked(id, h)
		return nil, false
	}
	return h, true
}

func (h *handoff) verifies(verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	return verifier != "" && subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(h.challenge)) == 1
}

func (h *handoff) boundTo(binding string) bool {
	sum := sha256.Sum256([]byte(binding))
	return binding != "" && subtle.ConstantTimeCompare(sum[:], h.binding[:]) == 1
}

// complete records the session the callback issued, for the transaction the browser holding the
// binding started. Only a pending transaction takes it, so a replayed callback changes nothing
func (s *handoffStore) complete(id, binding string, claims token.Claims) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.getLocked(id)
	if !ok || h.state != handoffPending || !h.boundTo(binding) {
		return false
	}
	h.state, h.claims = handoffDone, claims
	close(h.changed)
	return true
}

// cancel ends a transaction for the frame, which holds the verifier, or for the popup, which holds
// the binding. A pending one is kept as canceled for the frame to see, anything else is removed
func (s *handoffStore) cancel(id, verifier, binding string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.getLocked(id)
	if !ok {
		return errHandoffNotFound
	}
	if !h.verifies(verifier) && !h.boundTo(binding) {
		return errHandoffDenied
	}
	if h.state == handoffPending {
		h.state = handoffCanceled
		close(h.changed)
		return nil
	}
	delete(s.recs, id)
	return nil
}

// status returns the state of a transaction, the account it recorded once done, and a channel closed
// when a pending one changes
func (s *handoffStore) status(id, verifier string) (state handoffState, userID string, changed <-chan struct{}, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.getLocked(id)
	if !ok {
		return 0, "", nil, errHandoffNotFound
	}
	if !h.verifies(verifier) {
		return 0, "", nil, errHandoffDenied
	}
	if h.state == handoffDone && h.claims.User != nil {
		userID = h.claims.User.ID
	}
	return h.state, userID, h.changed, nil
}

// redeem checks the verifier and removes the transaction in one step, so a session is handed out
// once at most. A transaction whose account seen reports the frame already has is kept and refused
func (s *handoffStore) redeem(id, verifier string, seen func(token.Claims) bool) (token.Claims, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.getLocked(id)
	if !ok {
		return token.Claims{}, errHandoffNotFound
	}
	if !h.verifies(verifier) {
		return token.Claims{}, errHandoffDenied
	}
	if h.state != handoffDone {
		return token.Claims{}, errHandoffState
	}
	if seen(h.claims) {
		return token.Claims{}, errHandoffSeen
	}
	delete(s.recs, id)
	return h.claims, nil
}

// site returns the site a live transaction was started for
func (s *handoffStore) site(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.getLocked(id)
	if !ok {
		return "", false
	}
	return h.site, true
}

// handoffPage is shared by the two popup pages. They carry no script built from a request: the
// script is fixed, allowed by its hash, and reads what it needs from data attributes
var handoffPage = template.Must(template.New("handoff").Parse(`<!doctype html>
<html lang="en"{{range $k, $v := .Data}} data-{{$k}}="{{$v}}"{{end}}>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="referrer" content="no-referrer"><title>Remark42</title>
<style>body{font:16px/1.5 system-ui,sans-serif;margin:2em;color:#333;background:#fff}@media (prefers-color-scheme:dark){body{color:#ddd;background:#222}}</style></head>
<body><p>{{.Text}}</p><script>{{.Script}}</script></body></html>
`))

// handoffStartScript runs in the popup before it goes to the provider. It tells the frame that
// opened it which transaction this is, and goes on with the transaction only once the frame has
// acknowledged it. Without an opener, a reply or a transaction it goes to the ordinary login, and
// cancels the transaction first, so a frame that acknowledged late learns that it was given up
const handoffStartScript = `(function () {
  var d = document.documentElement.dataset, left = false;
  function go(url) { if (!left) { left = true; location.replace(url); } }
  function giveUp() {
    if (left) { return; }
    left = true;
    var leave = function () { location.replace(d.fallback); };
    try {
      fetch('cancel', { method: 'POST', credentials: 'same-origin', keepalive: true,
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ txn: d.txn }) }).then(leave, leave);
    } catch (e) { leave(); }
    setTimeout(leave, 2000);
  }
  if (!window.opener) { giveUp(); return; }
  window.addEventListener('message', function (e) {
    if (e.origin !== location.origin || e.source !== window.opener) { return; }
    if (e.data && e.data.type === 'remark42-handoff-ack' && e.data.txn === d.txn) { go(d.next); }
  });
  try { window.opener.postMessage({ type: 'remark42-handoff', txn: d.txn }, location.origin); } catch (e) { giveUp(); return; }
  setTimeout(giveUp, 3000);
})();`

// handoffDoneScript closes the popup once the provider has brought the reader back. The frame
// learns of the sign-in from the server, so nothing is passed to it here, and closes the popup as
// well once it has. The page asks at once and again a moment later: Safari refused a close asked
// for as the page loaded in the measurements, and took one asked for 400 ms later
const handoffDoneScript = `window.close(); setTimeout(function () { window.close(); }, 400);`

// handoffCSP allows the page's own fixed script, requests to its own origin and nothing else
func handoffCSP(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "default-src 'none'; script-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) +
		"'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
}

func writeHandoffPage(w http.ResponseWriter, text, script string, data map[string]string) {
	w.Header().Set("Content-Security-Policy", handoffCSP(script))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	_ = handoffPage.Execute(w, struct {
		Data   map[string]string
		Text   string
		Script template.JS
	}{Data: data, Text: text, Script: template.JS(script)}) //nolint:gosec // a constant, never request data
}

// handoffs is what the handoff handlers share with the rest of the server
type handoffs struct {
	store      *handoffStore
	tokens     *token.Service
	remarkURL  string
	registered func(provider string) bool
	wait       time.Duration // how long a status request waits for a pending transaction to change
	waiters    chan struct{} // a slot per status request waiting, see handoffMaxWaiters
}

// loginURL is the provider's ordinary login, the one the widget links to
func (h *handoffs) loginURL(provider, from, site string) string {
	return h.remarkURL + authRoutePrefix + url.PathEscape(provider) + "/login?" + url.Values{"from": {from}, "site": {site}}.Encode()
}

func (h *handoffs) doneURL(id string) string {
	return h.remarkURL + handoffDonePath + "?" + url.Values{"txn": {id}}.Encode()
}

func (h *handoffs) secure() bool { return strings.HasPrefix(h.remarkURL, "https://") }

// bindingName is the name of the popup's cookie for a transaction. Over https it carries the
// __Host- prefix, which a browser accepts only from this host, Secure and on the root path, so a
// page on a sibling subdomain or a forged plain-http response cannot plant one
func (h *handoffs) bindingName(id string) string {
	if h.secure() {
		return "__Host-" + handoffBindingCookiePrefix + id
	}
	return handoffBindingCookiePrefix + id
}

// bindingCookie is the popup's cookie for a transaction, sent back to the provider callback and to
// cancel. Over https it is on the root path, which its prefix requires, and so reaches the
// callback wherever remark42 is served. Over http it is on the auth routes as the browser sees
// them, under REMARK_URL's path when remark42 is served below one
func (h *handoffs) bindingCookie(id, value string, maxAge int) *http.Cookie {
	path := "/"
	if !h.secure() {
		prefix := ""
		if u, err := url.Parse(h.remarkURL); err == nil {
			prefix = strings.TrimSuffix(u.EscapedPath(), "/")
		}
		path = prefix + authRoutePrefix
	}
	return &http.Cookie{Name: h.bindingName(id), Value: value, Path: path, MaxAge: maxAge, //nolint:gosec // Secure and SameSite as the auth library's own cookies
		HttpOnly: true, Secure: h.secure(), SameSite: h.tokens.SameSite}
}

// startCtrl is the page the widget opens the sign-in popup on.
// GET /auth/handoff/start?provider=github&site=remark42&from=<url>&challenge=<S256>
func (h *handoffs) startCtrl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	provider, site, from, challenge := q.Get("provider"), q.Get("site"), q.Get("from"), q.Get("challenge")
	if !h.registered(provider) {
		http.NotFound(w, r)
		return
	}
	fallback := h.loginURL(provider, from, site)
	// only the popup's own navigation gets a transaction. A page read any other way, by a fetch from
	// another site, say, would hand that site the transaction id this page exists to keep to the
	// frame. A browser that sends no fetch metadata gets the ordinary login
	if r.Header.Get("Sec-Fetch-Mode") != "navigate" || r.Header.Get("Sec-Fetch-Dest") != "document" {
		http.Redirect(w, r, fallback, http.StatusFound) //nolint:gosec // remark42's own login, which checks "from" itself
		return
	}
	if !handoffChallengeRe.MatchString(challenge) {
		http.Redirect(w, r, fallback, http.StatusFound) //nolint:gosec // remark42's own login, which checks "from" itself
		return
	}
	id, binding, err := h.store.create(challenge, site)
	if err != nil {
		http.Redirect(w, r, fallback, http.StatusFound) //nolint:gosec // remark42's own login, which checks "from" itself
		return
	}
	http.SetCookie(w, h.bindingCookie(id, binding, int(handoffTTL.Seconds())))
	writeHandoffPage(w, "Signing in…", handoffStartScript, map[string]string{
		"txn": id, "next": h.loginURL(provider, h.doneURL(id), site), "fallback": fallback,
	})
}

// doneCtrl is where the provider's callback sends the reader back to. GET /auth/handoff/done
func (h *handoffs) doneCtrl(w http.ResponseWriter, _ *http.Request) {
	writeHandoffPage(w, "Signed in. You can close this window.", handoffDoneScript, nil)
}

type handoffRequest struct {
	Txn      string `json:"txn"`
	Verifier string `json:"verifier"`
}

func (h *handoffs) readHandoffRequest(w http.ResponseWriter, r *http.Request) (handoffRequest, bool) {
	var req handoffRequest
	// the frame and the start page call these on their own origin with a JSON body. A form another
	// site submits as a top-level POST can do neither, and a redeem answered there would put a
	// session in remark42's own partition. A browser without fetch metadata still names the origin
	// of a cross-origin fetch
	// Fetch metadata is the browser's own word on it. Only without it is the Origin compared, since
	// REMARK_URL may say http behind a proxy that serves the page over https
	site, origin := r.Header.Get("Sec-Fetch-Site"), r.Header.Get("Origin")
	crossSite := site != "" && site != "same-origin"
	foreign := site == "" && origin != "" && !strings.EqualFold(origin, ownOrigin(h.remarkURL))
	if crossSite || foreign || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		rest.SendErrorJSON(w, r, http.StatusForbidden, errors.New("not a same-origin json request"), "sign-in request refused", rest.ErrNoAccess)
		return req, false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't decode sign-in request", rest.ErrDecode)
		return req, false
	}
	if req.Txn == "" {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, errors.New("no txn"), "can't decode sign-in request", rest.ErrDecode)
		return req, false
	}
	return req, true
}

func sendHandoffError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errHandoffNotFound):
		rest.SendErrorJSON(w, r, http.StatusNotFound, err, "sign-in not found", rest.ErrNoAccess)
	case errors.Is(err, errHandoffDenied):
		rest.SendErrorJSON(w, r, http.StatusForbidden, err, "sign-in rejected", rest.ErrNoAccess)
	case errors.Is(err, errHandoffSeen):
		rest.SendErrorJSON(w, r, http.StatusConflict, err, "signed in already", rest.ErrNoAccess)
	default:
		rest.SendErrorJSON(w, r, http.StatusConflict, err, "sign-in not complete", rest.ErrNoAccess)
	}
}

// statusCtrl answers with the state of a transaction, waiting a while for a pending one to change.
// POST /auth/handoff/status {"txn": "...", "verifier": "..."}
func (h *handoffs) statusCtrl(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readHandoffRequest(w, r)
	if !ok {
		return
	}
	state, user, changed, err := h.store.status(req.Txn, req.Verifier)
	if err != nil {
		sendHandoffError(w, r, err)
		return
	}
	if state == handoffPending && h.wait > 0 {
		select {
		case h.waiters <- struct{}{}:
			timer := time.NewTimer(h.wait)
			select {
			case <-changed:
			case <-timer.C:
			case <-r.Context().Done():
			}
			timer.Stop()
			<-h.waiters
		default: // too many waiting already: answer now
		}
		if state, user, _, err = h.store.status(req.Txn, req.Verifier); err != nil {
			sendHandoffError(w, r, err)
			return
		}
	}
	resp := struct {
		Status string `json:"status"`
		User   string `json:"user,omitempty"`
	}{Status: state.String(), User: user}
	R.RenderJSON(w, resp)
}

// redeemCtrl gives the frame a session of its own for the account the transaction recorded, and
// removes the transaction. POST /auth/handoff/redeem {"txn": "...", "verifier": "..."}
func (h *handoffs) redeemCtrl(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readHandoffRequest(w, r)
	if !ok {
		return
	}
	if !h.secure() {
		// the embedded pair is Partitioned, which a browser takes only over https
		rest.SendErrorJSON(w, r, http.StatusBadRequest, errors.New("not https"), "embedded session needs https", rest.ErrNoAccess)
		return
	}
	// a frame that sends remark42's own session for this account sees it, and is on the same site
	// or allowed the site's cookies. Its embedded pair would land in the site's own partition and
	// turn the site's pages to it, so it is refused, and the widget takes the session it sees. A
	// frame already on its embedded session cannot take that one, which its requests never read, so
	// it is given its own whatever else it sends
	ordinary, embedded := cookieValues(r, authJWTCookie), isEmbedded(r)
	seen := func(c token.Claims) bool {
		if embedded || c.User == nil {
			return false
		}
		for _, tkn := range ordinary {
			got, perr := h.tokens.Parse(tkn)
			if perr == nil && got.Handshake == nil && got.User != nil && got.User.ID == c.User.ID &&
				slices.Equal(got.Audience, c.Audience) {
				return true
			}
		}
		return false
	}
	recorded, err := h.store.redeem(req.Txn, req.Verifier, seen)
	if err != nil {
		sendHandoffError(w, r, err)
		return
	}
	cid := make([]byte, 20)
	if _, err = rand.Read(cid); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make claim's id", rest.ErrInternal)
		return
	}
	// a session of its own: the same account, issued now, with an id, and so an xsrf token, the
	// session the popup holds does not share
	claims := token.Claims{
		User: recorded.User,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   recorded.Issuer,
			ID:       hex.EncodeToString(cid),
			Audience: recorded.Audience,
		},
		SessionOnly:  recorded.SessionOnly,
		NoAva:        recorded.NoAva,
		AuthProvider: recorded.AuthProvider,
	}
	setEmbedded(r)
	if _, err = h.tokens.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't set token", rest.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// cancelCtrl ends a transaction. The popup gives one up with its binding cookie when the frame did
// not acknowledge it, the frame with its verifier when it no longer waits.
// POST /auth/handoff/cancel {"txn": "...", "verifier": "..."}
func (h *handoffs) cancelCtrl(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readHandoffRequest(w, r)
	if !ok {
		return
	}
	binding := ""
	if c, err := r.Cookie(h.bindingName(req.Txn)); err == nil {
		binding = c.Value
	}
	if err := h.store.cancel(req.Txn, req.Verifier, binding); err != nil {
		sendHandoffError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recordHandoff wraps the auth handler. On a provider callback that redirects to the done page of a
// transaction this browser started, it records the session that same response sets
func (h *handoffs) recordHandoff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/callback") {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&handoffRecorder{ResponseWriter: w, r: r, h: h}, r)
	})
}

type handoffRecorder struct {
	http.ResponseWriter
	r    *http.Request
	h    *handoffs
	done bool
}

func (c *handoffRecorder) WriteHeader(code int) {
	c.record(code)
	c.ResponseWriter.WriteHeader(code)
}

func (c *handoffRecorder) Write(b []byte) (int, error) {
	c.record(http.StatusOK)
	return c.ResponseWriter.Write(b)
}

func (c *handoffRecorder) Flush() {
	c.record(http.StatusOK)
	_ = http.NewResponseController(c.ResponseWriter).Flush()
}

func (c *handoffRecorder) Unwrap() http.ResponseWriter { return c.ResponseWriter }

func (c *handoffRecorder) record(code int) {
	if c.done {
		return
	}
	c.done = true
	if code < 300 || code >= 400 {
		return
	}
	id := c.h.doneTxn(c.Header().Get("Location"))
	if id == "" {
		return
	}
	bc, err := c.r.Cookie(c.h.bindingName(id))
	if err != nil {
		return
	}
	site, ok := c.h.store.site(id)
	if !ok {
		return
	}
	claims, ok := c.h.issuedSession(c.Header(), site)
	if !ok {
		return
	}
	if c.h.store.complete(id, bc.Value, claims) {
		http.SetCookie(c.ResponseWriter, c.h.bindingCookie(id, "", -1))
	}
}

// doneTxn returns the transaction a redirect to the done page carries
func (h *handoffs) doneTxn(location string) string {
	u, err := url.Parse(location)
	if err != nil {
		return ""
	}
	done, err := url.Parse(h.doneURL(""))
	if err != nil || u.Scheme != done.Scheme || u.Host != done.Host || u.Path != done.Path {
		return ""
	}
	return u.Query().Get("txn")
}

// issuedSession returns the user session a response sets in the JWT cookie, for the site given
func (h *handoffs) issuedSession(hdr http.Header, site string) (token.Claims, bool) {
	var claims token.Claims
	found := false
	for _, line := range hdr.Values("Set-Cookie") {
		c, err := http.ParseSetCookie(line)
		if err != nil || c.Name != authJWTCookie || c.Value == "" || c.MaxAge < 0 {
			continue
		}
		parsed, err := h.tokens.Parse(c.Value)
		if err != nil {
			continue
		}
		claims, found = parsed, true
	}
	if !found || claims.User == nil || claims.User.ID == "" || claims.Handshake != nil ||
		len(claims.Audience) != 1 || claims.Audience[0] != site {
		return token.Claims{}, false
	}
	return claims, true
}

// embeddedKey is the context key of the flag that turns the embedded session on for a request
type embeddedKey struct{}

// cookieValues returns the value of every cookie of that name r came with. A frame can hold more
// than one JWT: the site's own and a partitioned copy from header mode
func cookieValues(r *http.Request, name string) []string {
	var out []string
	for _, c := range r.Cookies() {
		if c.Name == name {
			out = append(out, c.Value)
		}
	}
	return out
}

// isEmbedded reports whether r came on the embedded session
func isEmbedded(r *http.Request) bool {
	on, ok := r.Context().Value(embeddedKey{}).(*bool)
	return ok && *on
}

// noCORS keeps the handoff pages and calls to remark42's own pages: the root middleware lets any
// origin read a response with credentials, and the start page names a transaction only the frame
// may learn. A preflight still gets its answer, and the request after it no permission to read
func noCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k := range w.Header() {
			if strings.HasPrefix(k, "Access-Control-") {
				w.Header().Del(k)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// setEmbedded makes the response to r set the embedded session instead of the ordinary one
func setEmbedded(r *http.Request) {
	if on, ok := r.Context().Value(embeddedKey{}).(*bool); ok {
		*on = true
	}
}

// embeddedSession serves a frame that chose its embedded session. Such a request carries the
// marker or the embedded JWT. Its ordinary pair and any JWT header are taken off it, the embedded
// pair is handed to the auth library under the ordinary names, and every auth cookie the response
// sets is renamed back and made Partitioned, so the library reads, renews and expires the embedded
// session alone. A missing or broken embedded session is then simply no session, never the
// ordinary one. The JWT header is left out of the response too, so no token reaches script.
//
// The redeem handler turns it on for the response that hands the session out, through setEmbedded.
// Over http it is not mounted at all: a Partitioned cookie has to be Secure.
func embeddedSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		on := hasCookie(r, embeddedMarkerCookie) || hasCookie(r, embeddedJWTCookie)
		if on {
			r = asEmbedded(r)
		}
		r = r.WithContext(context.WithValue(r.Context(), embeddedKey{}, &on))
		next.ServeHTTP(&embeddingWriter{ResponseWriter: w, on: &on}, r)
	})
}

func hasCookie(r *http.Request, name string) bool {
	_, err := r.Cookie(name)
	return err == nil
}

// asEmbedded returns r with the embedded pair under the ordinary names and nothing else to
// authenticate with
func asEmbedded(r *http.Request) *http.Request {
	r = r.Clone(r.Context())
	kept := make([]string, 0, 4)
	for _, c := range r.Cookies() {
		switch c.Name {
		case authJWTCookie, authXSRFCookie:
			continue
		case embeddedJWTCookie:
			c.Name = authJWTCookie
		case embeddedXSRFCookie:
			c.Name = authXSRFCookie
		}
		kept = append(kept, c.Name+"="+c.Value)
	}
	r.Header.Del("Cookie")
	if len(kept) > 0 {
		r.Header.Set("Cookie", strings.Join(kept, "; "))
	}
	r.Header.Del(authJWTHeader)
	return r
}

// embeddingWriter renames the auth cookies of an embedded session's response just before the
// headers are sent
type embeddingWriter struct {
	http.ResponseWriter
	on   *bool
	done bool
}

func (e *embeddingWriter) WriteHeader(code int) {
	e.rename()
	e.ResponseWriter.WriteHeader(code)
}

func (e *embeddingWriter) Write(b []byte) (int, error) {
	e.rename()
	return e.ResponseWriter.Write(b)
}

func (e *embeddingWriter) Flush() {
	e.rename()
	_ = http.NewResponseController(e.ResponseWriter).Flush()
}

func (e *embeddingWriter) Unwrap() http.ResponseWriter { return e.ResponseWriter }

func (e *embeddingWriter) rename() {
	if e.done || !*e.on {
		return
	}
	e.done = true
	hdr := e.Header()
	hdr.Del(authJWTHeader)
	lines := hdr.Values("Set-Cookie")
	if len(lines) == 0 {
		return
	}
	out := make([]string, 0, len(lines)+1)
	var marker *http.Cookie //nolint:gosec // the widget reads it, it carries no credential
	for _, line := range lines {
		c, err := http.ParseSetCookie(line)
		if err != nil || (c.Name != authJWTCookie && c.Name != authXSRFCookie) {
			out = append(out, line)
			continue
		}
		live := c.Value != "" && c.MaxAge >= 0
		if c.Name == authJWTCookie {
			c.Name = embeddedJWTCookie
			if live {
				marker = &http.Cookie{Name: embeddedMarkerCookie, Value: "1", Path: "/", MaxAge: c.MaxAge} //nolint:gosec // read by the widget, Secure and SameSite set below
			}
		} else {
			c.Name = embeddedXSRFCookie
		}
		c.Domain = ""
		c.Partitioned, c.Secure, c.SameSite = true, true, http.SameSiteNoneMode
		out = append(out, c.String())
	}
	if marker != nil {
		// renewed with the session it marks, and left alone when the session is expired
		marker.Partitioned, marker.Secure, marker.SameSite = true, true, http.SameSiteNoneMode
		out = append(out, marker.String())
	}
	hdr["Set-Cookie"] = out
}
