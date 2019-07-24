package notify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/smtp"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmailNew(t *testing.T) {
	var testSet = map[int]struct {
		params   EmailParams
		template bool
		err      bool
		errText  string
	}{
		1: {EmailParams{}, true, true, ""},
		2: {EmailParams{
			Host:          "test@host",
			Port:          1000,
			TLS:           true,
			From:          "test@from",
			Username:      "test@username",
			Password:      "test@password",
			TimeOut:       time.Second,
			Template:      "{{",
			BufferSize:    10,
			FlushDuration: time.Second,
		},
			false, true, "can't parse message template: template: messageFromRequest:1: unexpected unclosed action in command"},
	}
	for i, d := range testSet {
		email, err := NewEmail(d.params)

		if d.err && d.errText == "" {
			assert.Error(t, err, "error match expected for test set %d", i)
		} else if d.err && d.errText != "" {
			assert.EqualError(t, err, d.errText, "error match expected for test set %d", i)
		} else {
			assert.NoError(t, err, "error match expected for test set %d", i)
		}

		assert.NotNil(t, email, "email returned for test set %d", i)
		assert.Nil(t, email.ctx, "e.ctx is not set during initialisation for test set %d", i)
		assert.NotNil(t, email.submit, "e.submit is created during initialisation for test set %d", i)
		if d.template {
			assert.NotNil(t, email.template, "e.template is set for test set %d", i)
		} else {
			assert.Nil(t, email.template, "e.template is not set for test set %d", i)
		}
		if d.params.Template == "" {
			assert.Equal(t, defaultEmailTemplate, email.EmailParams.Template, "empty params.Template changed to default for test set %d", i)
		} else {
			assert.Equal(t, d.params.Template, email.EmailParams.Template, "params.Template unchanged after creation for test set %d", i)
		}
		if d.params.FlushDuration == 0 {
			assert.Equal(t, defaultFlushDuration, email.EmailParams.FlushDuration, "empty params.FlushDuration changed to default for test set %d", i)
		} else {
			assert.Equal(t, d.params.FlushDuration, email.EmailParams.FlushDuration, "params.FlushDuration unchanged after creation for test set %d", i)
		}
		if d.params.TimeOut == 0 {
			assert.Equal(t, defaultEmailTimeout, email.EmailParams.TimeOut, "empty params.TimeOut changed to default for test set %d", i)
		} else {
			assert.Equal(t, d.params.TimeOut, email.EmailParams.TimeOut, "params.TimOut unchanged after creation for test set %d", i)
		}
		if d.params.BufferSize == 0 {
			assert.Equal(t, 1, email.EmailParams.BufferSize, "empty params.BufferSize changed to default for test set %d", i)
		} else {
			assert.Equal(t, d.params.BufferSize, email.EmailParams.BufferSize, "params.BufferSize unchanged after creation for test set %d", i)
		}
		assert.Equal(t, d.params.From, email.EmailParams.From, "params.From unchanged after creation for test set %d", i)
		assert.Equal(t, d.params.Host, email.EmailParams.Host, "params.Host unchanged after creation for test set %d", i)
		assert.Equal(t, d.params.Username, email.EmailParams.Username, "params.Username unchanged after creation for test set %d", i)
		assert.Equal(t, d.params.Password, email.EmailParams.Password, "params.Password unchanged after creation for test set %d", i)
		assert.Equal(t, d.params.Port, email.EmailParams.Port, "params.Port unchanged after creation for test set %d", i)
		assert.Equal(t, d.params.TLS, email.EmailParams.TLS, "params.TLS unchanged after creation for test set %d", i)
	}
}

func TestEmailSendErrors(t *testing.T) {
	var err error
	e := Email{EmailParams: EmailParams{FlushDuration: time.Second}}
	e.template, err = template.New("test").Parse("{{.Test}}")
	assert.NoError(t, err)
	assert.EqualError(t, e.Send(context.Background(), request{}),
		"error executing template to build message from request: template: test:1:2: executing \"test\" at <.Test>: can't evaluate field Test in type notify.tmplData")
	e.template, err = template.New("test").Parse(defaultEmailTemplate)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.EqualError(t, e.Send(ctx, request{}),
		"canceling sending message to \"test@localhost\" because of canceled context")
}

// TODO: test writing is in process, more tests to come

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
