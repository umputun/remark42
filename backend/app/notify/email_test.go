package notify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/backend/app/store"
)

func TestEmptyEmailServer(t *testing.T) {
	// Test failed start of the server.
	email, err := NewEmail(EmailParams{})
	assert.Error(t, err, "No connection established with empty address and port zero")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Equal(t, email.host, "")
	assert.Equal(t, email.port, 0)
	assert.Equal(t, email.tls, false)
	assert.Equal(t, email.from, "")
	assert.Equal(t, email.username, "")
	assert.Equal(t, email.password, "")
	assert.Equal(t, email.timeOut, 10*time.Second, "default value if TimeOut is not defined is set to 10s")
	assert.Equal(t, "email: ''@'':0", email.String())
}

func TestBuildMessageFromRequest(t *testing.T) {
	email, _ := NewEmail(EmailParams{From: "noreply@example.org"})
	c := store.Comment{Text: "some text"}
	c.User.Name = "@from_user"
	c.Locator.URL = "//example.org"
	c.Orig = "orig"
	req := request{comment: c}
	mgs := email.buildMessageFromRequest(req, "test_address@example.com")
	emptyTitleMessage := `From: noreply@example.org
To: test_address@example.com
Subject: New comment
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

@from_user

orig

↦ <a href="//example.org#remark42__comment-">original comment</a>`
	assert.Equal(t, emptyTitleMessage, mgs)
	c.ParentID = "1"
	c.PostTitle = "post title"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "@to_user"
	req = request{comment: c, parent: cp}
	filledTitleMessage := `From: noreply@example.org
To: test_address@example.com
Subject: New comment for "post title"
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

@from_user → @to_user

orig

↦ <a href="//example.org#remark42__comment-">post title</a>`
	mgs = email.buildMessageFromRequest(req, "test_address@example.com")
	assert.Equal(t, filledTitleMessage, mgs)
}

func TestBuildMessage(t *testing.T) {
	email := Email{from: "from@email"}
	msg := email.BuildMessage("test_subj", "test_body", "recepient@email", "")
	expectedMsg := `From: from@email
To: recepient@email
Subject: test_subj

test_body`
	assert.Equal(t, expectedMsg, msg)
	msg = email.BuildMessage("test_subj", "test_body", "recepient@email", "text/html")
	expectedLines := `MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";`
	assert.Contains(t, msg, expectedLines)
}

func TestConnectErrors(t *testing.T) {
	email := Email{}
	client, err := email.client()
	assert.Nil(t, client)
	assert.Error(t, err, "connection with wrong settings return error")
	email = Email{tls: true}
	client, err = email.client()
	assert.Nil(t, client)
	assert.Error(t, err, "TLS connection with wrong settings return error")
}
