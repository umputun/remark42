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
	assert.Error(t, err, "no connection established with empty address and port zero")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Nil(t, email.connection, "connection was not created due to error")
	assert.Equal(t, 10*time.Second, email.params.TimeOut, "default value if TimeOut is not defined is set to 10s")
	assert.NotNil(t, email.template, "default template is set")
	assert.Equal(t, email.String(), "email: ''@'':0", "correct empty object string representation")
	// Test broken template
	email, err = NewEmail(EmailParams{Template: "{{"})
	assert.Error(t, err, "error due to parsing improper template")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Nil(t, email.template, "default template is not set due to error")
	assert.Nil(t, email.connection, "connection was not created due to error")

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

↦ <a href="//example.org#remark42__comment-">original comment</a>
`
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

↦ <a href="//example.org#remark42__comment-">post title</a>
`
	mgs = email.buildMessageFromRequest(req, "test_address@example.com")
	assert.Equal(t, filledTitleMessage, mgs)
}

func TestBuildMessage(t *testing.T) {
	email := Email{params: EmailParams{From: "from@email"}}
	msg := email.buildMessage("test_subj", "test_body", "recepient@email", "")
	expectedMsg := `From: from@email
To: recepient@email
Subject: test_subj

test_body`
	assert.Equal(t, expectedMsg, msg)
	msg = email.buildMessage("test_subj", "test_body", "recepient@email", "text/html")
	expectedLines := `MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";`
	assert.Contains(t, msg, expectedLines)
}

func TestConnectErrors(t *testing.T) {
	email := Email{}
	client, err := email.client()
	assert.Nil(t, client)
	assert.Error(t, err, "connection with wrong settings return error")
	email = Email{params: EmailParams{TLS: true}}
	client, err = email.client()
	assert.Nil(t, client)
	assert.Error(t, err, "TLS connection with wrong settings return error")
}
