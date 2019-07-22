package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/smtp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

func TestBrokenTemplate(t *testing.T) {
	// Test broken template
	email, err := NewEmail(EmailParams{Template: "{{"})
	assert.Error(t, err, "error due to parsing improper template")
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Nil(t, email.template, "default template is not set due to error")
}

func TestBuildAndSendMessageFromRequest(t *testing.T) {
	email, err := NewEmail(EmailParams{From: "noreply@example.org"})
	// test empty connection
	assert.Error(t, err, "no connection established with empty address and port zero")
	fakeSMTP := &fakeTestSMTP{}
	email.smtpClient = fakeSMTP
	assert.NotNil(t, email, "despite the error we got object reference")
	assert.Equal(t, 10*time.Second, email.TimeOut, "default value if TimeOut is not defined is set to 10s")
	assert.NotNil(t, email.template, "default template is set")
	assert.Equal(t, email.String(), "email: noreply@example.org using ''@'':0", "correct string representation of Email")
	// test building message from request
	c := store.Comment{Text: "some text"}
	c.User.Name = "@from_user"
	c.Locator.URL = "//example.org"
	c.Orig = "orig"
	req := request{comment: c}
	msg, err := email.buildMessageFromRequest(req, "test@localhost")
	assert.NoError(t, err)
	emptyTitleMessage := `From: noreply@example.org
To: test@localhost
Subject: New comment
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

@from_user

orig

↦ <a href="//example.org#remark42__comment-">original comment</a>
`
	assert.Equal(t, emptyTitleMessage, msg)
	c.ParentID = "1"
	c.PostTitle = "post title"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "@to_user"
	req = request{comment: c, parent: cp}
	filledTitleMessage := `From: noreply@example.org
To: test@localhost
Subject: New comment for "post title"
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

@from_user → @to_user

orig

↦ <a href="//example.org#remark42__comment-">post title</a>
`
	msg, err = email.buildMessageFromRequest(req, "test@localhost")
	assert.NoError(t, err)
	assert.Equal(t, filledTitleMessage, msg)
	// test sending
	err = email.Send(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "noreply@example.org", fakeSMTP.mail)
	assert.Equal(t, "test@localhost", fakeSMTP.rcpt)
	assert.Equal(t, filledTitleMessage, fakeSMTP.buff.String())
	assert.True(t, fakeSMTP.quit)
	assert.False(t, fakeSMTP.close)
	// test sending failure
	fakeSMTP = &fakeTestSMTP{fail: true}
	email.smtpClient = fakeSMTP
	err = email.Send(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't make email writer: failed")
}

func TestBuildMessage(t *testing.T) {
	email := Email{EmailParams: EmailParams{From: "from@email"}}
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
	email = Email{EmailParams: EmailParams{TLS: true}}
	client, err = email.client()
	assert.Nil(t, client)
	assert.Error(t, err, "TLS connection with wrong settings return error")
}

func TestEmailMultipleSend(t *testing.T) {
	e, _ := NewEmail(EmailParams{BufferSize: 11})
	fakeSMTP := &fakeTestSMTP{}
	e.smtpClient = fakeSMTP
	waitCh := make(chan int)
	var waitGroup sync.WaitGroup
	// accumulate 10 messages in parallel
	for i := 1; i <= 10; i++ {
		waitGroup.Add(1)
		i := i
		go func() {
			// will start once we close the channel
			<-waitCh
			assert.NoError(t, e.Send(context.Background(), request{}), fmt.Sprint(i))
			waitGroup.Done()
		}()
	}
	close(waitCh)
	waitGroup.Wait()
	// make sure they end up in a buffer
	assert.Equal(t, 10, len(e.buffer))
	assert.NoError(t, e.Flush())
	// make sure they are flushed
	assert.Equal(t, 0, len(e.buffer))
	assert.Equal(t, 1, fakeSMTP.quitCount, "10 messages sent reusing same connection, closing it once afterwards")
	assert.True(t, fakeSMTP.quit)
}

type fakeTestSMTP struct {
	fail bool

	buff        bytes.Buffer
	mail, rcpt  string
	auth        bool
	quit, close bool
	quitCount   int
}

func (f *fakeTestSMTP) Mail(m string) error  { f.mail = m; return nil }
func (f *fakeTestSMTP) Auth(smtp.Auth) error { f.auth = true; return nil }
func (f *fakeTestSMTP) Rcpt(r string) error  { f.rcpt = r; return nil }
func (f *fakeTestSMTP) Quit() error          { f.quitCount++; f.quit = true; return nil }
func (f *fakeTestSMTP) Close() error         { f.close = true; return nil }

func (f *fakeTestSMTP) Data() (io.WriteCloser, error) {
	if f.fail {
		return nil, errors.New("failed")
	}
	return nopCloser{&f.buff}, nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
