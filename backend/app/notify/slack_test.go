package notify

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestSlack_New(t *testing.T) {

	ts := newMockSlackServer()
	defer ts.Close()

	tb, err := ts.newClient("general")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "C12345678", tb.channelID)

	_, err = ts.newClient("unknown-channel")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such channel")

}

func TestSlack_Send(t *testing.T) {

	ts := newMockSlackServer()
	defer ts.Close()

	tb, err := ts.newClient("general")
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	c := store.Comment{Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "to"

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	c.PostTitle = "test title"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	c.PostTitle = "[test title]"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	tb, err = ts.newClient("general")
	assert.NoError(t, err)
	ts.isServerDown = true
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack server error", "send on broken client")

}

func TestSlack_Name(t *testing.T) {
	ts := newMockSlackServer()
	defer ts.Close()

	tb, err := ts.newClient("general")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "slack: general (C12345678)", tb.String())
}

func TestSlack_SendVerification(t *testing.T) {
	ts := newMockSlackServer()
	defer ts.Close()

	tb, err := ts.newClient("general")
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	err = tb.SendVerification(context.TODO(), VerificationRequest{})
	assert.NoError(t, err)
}

type mockSlackServer struct {
	*httptest.Server
	isServerDown bool
}

func (ts *mockSlackServer) newClient(channelName string) (*Slack, error) {
	return NewSlack("any-token", channelName, slack.OptionAPIURL(ts.URL+"/"))
}

func newMockSlackServer() *mockSlackServer {

	mockServer := mockSlackServer{}
	router := chi.NewRouter()
	router.Post("/conversations.list", func(w http.ResponseWriter, r *http.Request) {
		s := `{
		    "ok": true,
		    "channels": [
		        {
		            "id": "C12345678",
		            "name": "general",
		            "is_channel": true,
		            "is_group": false,
		            "is_im": false,
		            "created": 1503888888,
		            "is_archived": false,
		            "is_general": false,
		            "unlinked": 0,
		            "name_normalized": "random",
		            "is_shared": false,
		            "parent_conversation": null,
		            "creator": "U12345678",
		            "is_ext_shared": false,
		            "is_org_shared": false,
		            "pending_shared": [],
		            "pending_connected_team_ids": [],
		            "is_pending_ext_shared": false,
		            "is_member": false,
		            "is_private": false,
		            "is_mpim": false,
		            "previous_names": [],
		            "num_members": 1
		        }
		    ],
		    "response_metadata": {
		        "next_cursor": ""
		    }
		}`
		_, _ = w.Write([]byte(s))
	})

	router.Post("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {

		if mockServer.isServerDown {
			w.WriteHeader(500)

		} else {
			s := `{
			    "ok": true,
			    "channel": "C12345678",
			    "ts": "1617008342.000100",
			    "message": {
			        "type": "message",
			        "subtype": "bot_message",
			        "text": "wowo",
			        "ts": "1617008342.000100",
			        "username": "slackbot",
			        "bot_id": "B12345678"
			    }
			}`
			_, _ = w.Write([]byte(s))
		}
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("..... 404 for %s .....\n", r.URL)
	})

	mockServer.Server = httptest.NewServer(router)
	return &mockServer
}
