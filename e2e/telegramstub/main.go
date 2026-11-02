// Command telegramstub answers as the Telegram bot API for the e2e suite.
//
// The provider talks to whatever base url it is given, so pointing remark42 at this instead of
// api.telegram.org is the whole of what makes the Telegram auth flow reachable from a browser test.
// It implements the four calls that flow uses and nothing else: getMe once at startup, getUpdates
// on a poll, sendMessage when a login is confirmed, and getUserProfilePhotos for the avatar.
//
// The test drives it through /control/send, which enqueues an update as if a reader had sent the
// bot a message, and reads /control/sent to match the bot's replies. Those are the steps no browser
// can perform, since they happen inside Telegram.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// update is the subset of Telegram's Update the provider reads
type update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		From struct {
			ID           int    `json:"id"`
			Username     string `json:"username"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		// the provider skips any update whose chat is not private, and takes the reader's
		// display name from the chat, not the sender
		Chat struct {
			ID   int    `json:"id"`
			Name string `json:"first_name"`
			Type string `json:"type"`
		} `json:"chat"`
	} `json:"message"`
}

type stub struct {
	mu      sync.Mutex
	queued  []update
	nextID  int
	botName string
	token   string

	// every message the provider asked to send, so a test can assert the reader was told
	sent []string
}

func main() {
	addr := os.Getenv("STUB_ADDR")
	if addr == "" {
		addr = ":8091"
	}
	botName := os.Getenv("STUB_BOT_USERNAME")
	if botName == "" {
		botName = "remark42_e2e_bot"
	}
	// the token the caller has to present. Checking it is what makes these tests witness the
	// <base>/bot<token>/<method> construction: without it a build that dropped, truncated or
	// double-escaped the token would be served exactly as a correct one and every case would pass
	token := os.Getenv("STUB_BOT_TOKEN")
	if token == "" {
		token = "stub-token"
	}

	s := &stub{nextID: 1, botName: botName, token: token}

	mux := http.NewServeMux()
	// the provider builds every call as <base>/bot<token>/<method>, so the token segment is part
	// of the path and the method is its last element
	mux.HandleFunc("/", s.route)
	mux.HandleFunc("/control/send", s.send)
	mux.HandleFunc("/control/sent", s.listSent)

	log.Print("[INFO] telegram stub started")
	srv := &http.Server{Addr: addr, Handler: mux} //nolint:gosec // test double, no timeouts wanted
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}

// route dispatches a bot call. The provider builds every one as <base>/bot<token>/<method>, so both
// halves of the path are checked: a request whose token segment is not the one this stub was given
// is refused with 401, matching Telegram's response
func (s *stub) route(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) > 1 && segments[0] == "file" {
		segments = segments[1:]
	}
	if len(segments) < 2 || segments[0] != "bot"+s.token {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": false, "error_code": 401, "description": "Unauthorized: bot token is wrong or missing",
		})
		return
	}

	method := segments[len(segments)-1]
	switch method {
	case "getMe":
		s.reply(w, map[string]any{"id": 1, "is_bot": true, "username": s.botName, "first_name": "remark42 e2e"})
	case "getUpdates":
		s.reply(w, s.drain())
	case "sendMessage":
		text := r.URL.Query().Get("text")
		if text == "" && r.Body != nil {
			var message telegramMsg
			if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
				http.Error(w, `{"ok":false,"description":"invalid message"}`, http.StatusBadRequest)
				return
			}
			text = message.Text
		}
		s.record(text)
		s.reply(w, map[string]any{"message_id": 1})
	case "getUserProfilePhotos":
		// no avatar: the flow has to work for a reader who never set one, and an empty list is
		// what Telegram returns for that
		s.reply(w, map[string]any{"total_count": 0, "photos": []any{}})
	default:
		http.Error(w, `{"ok":false,"description":"unsupported method"}`, http.StatusNotFound)
	}
}

type telegramMsg struct {
	Text string `json:"text"`
}

// send enqueues an update as if the reader had sent the bot a message. text and id are the two
// things a test decides; the rest is filled in so the provider has a complete user to build from
func (s *stub) send(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, err := strconv.Atoi(q.Get("id"))
	if err != nil {
		// never defaulted: a substituted id signs the test in as somebody else and says nothing
		// about it, which is how a case asserting "a distinct user per run" can be neither
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	var u update
	u.Message.Text = q.Get("text")
	u.Message.From.ID = id
	u.Message.From.Username = q.Get("username")
	u.Message.From.FirstName = q.Get("first_name")
	u.Message.Chat.ID = id
	u.Message.Chat.Name = q.Get("first_name")
	u.Message.Chat.Type = "private"

	s.mu.Lock()
	u.UpdateID = s.nextID
	s.nextID++
	s.queued = append(s.queued, u)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *stub) listSent(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.sent)
}

// drain returns what is queued and clears it. The provider polls in a loop and tracks its own
// offset, so handing the same update back twice would have it processed twice
func (s *stub) drain() []update {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := s.queued
	s.queued = nil
	if out == nil {
		out = []update{}
	}
	return out
}

func (s *stub) record(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, text)
}

func (s *stub) reply(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result}); err != nil {
		log.Printf("[WARN] %v", err)
	}
}
