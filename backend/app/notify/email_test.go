package notify

import (
	"context"
	"fmt"
	"html/template"
	"testing"

	ntf "github.com/go-pkgz/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestEmailNew(t *testing.T) {
	emailParams := EmailParams{
		From:                     "test@from",
		VerificationTemplatePath: "testdata/verification.html.tmpl",
		MsgTemplatePath:          "testdata/msg.html.tmpl",
	}
	smtpParams := ntf.SMTPParams{
		Host:     "test@host",
		Port:     1000,
		TLS:      true,
		StartTLS: true,
		Username: "test@username",
		Password: "test@password",
	}

	email, err := NewEmail(emailParams, smtpParams)

	assert.NoError(t, err)
	assert.NotNil(t, email, "email returned")

	assert.NotNil(t, email.msgTmpl, "e.template is set")
	assert.Equal(t, emailParams.From, email.From, "emailParams.From unchanged after creation")
	if smtpParams.TimeOut == 0 {
		assert.Equal(t, defaultEmailTimeout, email.TimeOut, "empty emailParams.TimeOut changed to default")
	} else {
		assert.Equal(t, smtpParams.TimeOut, email.TimeOut, "emailParams.TimOut unchanged after creation")
	}
	assert.Equal(t, smtpParams.Host, email.Host, "emailParams.Host unchanged after creation")
	assert.Equal(t, smtpParams.Username, email.Username, "emailParams.Username unchanged after creation")
	assert.Equal(t, smtpParams.Password, email.Password, "emailParams.Password unchanged after creation")
	assert.Equal(t, smtpParams.Port, email.Port, "emailParams.Port unchanged after creation")
	assert.Equal(t, smtpParams.TLS, email.TLS, "emailParams.TLS unchanged after creation")
	assert.Equal(t, smtpParams.StartTLS, email.StartTLS, "emailParams.TLS unchanged after creation")
	assert.Equal(t, "email: with username 'test@username' at server test@host:1000 with TLS with StartTLS", email.String())
}

func Test_initTemplatesErr(t *testing.T) {
	testSet := []struct {
		name        string
		errText     string
		emailParams EmailParams
	}{
		{
			name:    "with wrong path to verification template",
			errText: "notfound.tmpl: file does not exist",
			emailParams: EmailParams{
				VerificationTemplatePath: "notfound.tmpl",
			},
		},
		{
			name:    "with wrong path to message template",
			errText: "notfound.tmpl: file does not exist",
			emailParams: EmailParams{
				MsgTemplatePath: "notfound.tmpl",
			},
		},
		{
			name:    "with error on read verification template",
			errText: "can't parse verification template: template: verifyTmpl",
			emailParams: EmailParams{
				VerificationTemplatePath: "testdata/bad.html.tmpl",
				MsgTemplatePath:          "testdata/msg.html.tmpl",
			},
		},
		{
			name:    "with error on read message template",
			errText: "can't parse message template: template: msgTmpl",
			emailParams: EmailParams{
				VerificationTemplatePath: "testdata/verification.html.tmpl",
				MsgTemplatePath:          "testdata/bad.html.tmpl",
			},
		},
	}

	for _, d := range testSet {
		t.Run(d.name, func(t *testing.T) {
			e, err := NewEmail(d.emailParams, ntf.SMTPParams{})
			require.Error(t, err)
			require.Nil(t, e)
			assert.Contains(t, err.Error(), d.errText)
		})
	}
}

func TestEmailSendErrors(t *testing.T) {
	var err error
	e := Email{}
	e.TokenGenFn = TokenGenFn

	e.verifyTmpl, err = template.New("test").Parse("{{.Test}}")
	assert.NoError(t, err)
	assert.EqualError(t, e.SendVerification(context.Background(), VerificationRequest{Email: "bad@example.org", Token: "some"}),
		"error executing template to build verification message: template: test:1:2: executing \"test\" at <.Test>: can't evaluate field Test in type notify.verifyTmplData")

	e.msgTmpl, err = template.New("test").Parse("{{.Test}}")
	assert.NoError(t, err)
	assert.EqualError(t, e.Send(context.Background(), Request{Comment: store.Comment{ID: "999"}, parent: store.Comment{User: store.User{ID: "test"}}, Emails: []string{"bad@example.org"}}),
		"problem sending user email notification to \"bad@example.org\": "+
			"error executing template to build comment reply message: "+
			"template: test:1:2: executing \"test\" at <.Test>: "+
			"can't evaluate field Test in type notify.msgTmplData")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.EqualError(t, e.Send(ctx, Request{Comment: store.Comment{ID: "999"}, parent: store.Comment{User: store.User{ID: "test"}}, Emails: []string{"bad@example.org"}}),
		"sending email messages about comment \"999\" aborted due to canceled context")

	assert.EqualError(t, e.Send(context.Background(), Request{Comment: store.Comment{ID: "999"}, parent: store.Comment{User: store.User{ID: "error"}}, Emails: []string{"bad@example.org"}}),
		"problem sending user email notification to \"bad@example.org\":"+
			" error creating token for unsubscribe link: token generation error")

	// errors for all failed recipients are reported, not just the last one
	assert.EqualError(t, e.Send(context.Background(),
		Request{Comment: store.Comment{ID: "999"}, parent: store.Comment{User: store.User{ID: "error"}}, Emails: []string{"bad1@example.org", "bad2@example.org"}}),
		"problem sending user email notification to \"bad1@example.org\": error creating token for unsubscribe link: token generation error\n"+
			"problem sending user email notification to \"bad2@example.org\": error creating token for unsubscribe link: token generation error")
}

func TestEmailSend_ExitConditions(t *testing.T) {
	email, err := NewEmail(EmailParams{
		VerificationTemplatePath: "testdata/verification.html.tmpl",
		MsgTemplatePath:          "testdata/msg.html.tmpl",
	}, ntf.SMTPParams{})
	assert.NoError(t, err)
	assert.NotNil(t, email, "expecting email returned")
	// prevent triggering e.autoFlush creation
	emptyRequest := Request{Comment: store.Comment{ID: "999"}}
	assert.NoError(t, email.Send(context.Background(), emptyRequest),
		"Message without Emails and AdminEmails is not sent and returns nil")
}

func TestEmail_Send(t *testing.T) {
	email, err := NewEmail(EmailParams{
		From:                     "from@example.org",
		VerificationTemplatePath: "testdata/verification.html.tmpl",
		MsgTemplatePath:          "testdata/msg.html.tmpl",
	}, ntf.SMTPParams{})
	assert.NoError(t, err)
	assert.NotNil(t, email)
	email.TokenGenFn = TokenGenFn
	email.UnsubscribeURL = "https://remark42.com/api/v1/email/unsubscribe"
	req := Request{
		Comment: store.Comment{ID: "999", User: store.User{ID: "1", Name: "test_user"}, ParentID: "1", PostTitle: "test_title"},
		parent:  store.Comment{ID: "1", User: store.User{ID: "999", Name: "parent_user"}},
		Emails:  []string{"test@example.org"},
	}
	assert.Contains(t, email.Send(context.Background(), req).Error(), "problem sending user email notification to \"test@example.org\"")
	// test buildMessageFromRequest separately for message text
	msg, err := email.buildMessageFromRequest(req, req.Emails[0], false)
	assert.NoError(t, err)
	assert.Equal(t, `
	New reply from test_user on your comment to «test_title»

User: test_user
01.01.0001 at 00:00
Comment: 
test@example.org  for parent_user
Unsubscribe link: https://remark42.com/api/v1/email/unsubscribe?site=&amp;tkn=token
`, msg.body)
	assert.Equal(t, "https://remark42.com/api/v1/email/unsubscribe?site=&tkn=token", msg.unsubscribeLink)
	assert.Equal(t, `New reply to your comment for "test_title"`, msg.subject)

	// send email to both user and admin, without parent set
	email.AdminEmails = []string{"admin@example.org"}
	req = Request{
		Comment: store.Comment{ID: "999", User: store.User{ID: "1", Name: "test_user"}, PostTitle: "test_title"},
		Emails:  []string{"test@example.org"},
	}
	assert.Error(t, email.Send(context.Background(), req))
	msg, err = email.buildMessageFromRequest(req, email.AdminEmails[0], true)
	assert.NoError(t, err)
	assert.Equal(t, `
New comment from test_user on your site  to «test_title»

User: test_user
01.01.0001 at 00:00
Comment: 
admin@example.org 
`, msg.body)
	assert.Equal(t, `New comment to your site for "test_title"`, msg.subject)
	assert.Empty(t, msg.unsubscribeLink)
}

func TestEmail_CommentTextSanitizedForEmail(t *testing.T) {
	// comment HTML reaching the email path is sanitized by the store-level UGC policy,
	// which permits <a> and <img>. The email must drop both so a comment can't inject
	// phishing links or remote tracking pixels into a notification (GHSA-74pc-3r2m-ppx3).
	email, err := NewEmail(EmailParams{
		From:            "from@example.org",
		MsgTemplatePath: "testdata/msg.html.tmpl",
	}, ntf.SMTPParams{})
	require.NoError(t, err)
	email.TokenGenFn = TokenGenFn

	malicious := `hello <a href="https://phishing.example/verify">click to verify</a>` +
		` <img src="https://attacker.example/track.png" width="1" height="1"> <b>kept</b>`
	req := Request{
		Comment: store.Comment{ID: "999", User: store.User{ID: "1", Name: "test_user"}, PostTitle: "test_title", Text: malicious},
		Emails:  []string{"test@example.org"},
	}
	msg, err := email.buildMessageFromRequest(req, req.Emails[0], false)
	require.NoError(t, err)

	assert.NotContains(t, msg.body, "phishing.example", "phishing link must be stripped")
	assert.NotContains(t, msg.body, "attacker.example", "tracking pixel must be stripped")
	assert.NotContains(t, msg.body, "<img", "no image tags in email body")
	assert.NotContains(t, msg.body, "<a ", "no anchor tags in email body")
	assert.Contains(t, msg.body, "click to verify", "anchor text is preserved, only the link is dropped")
	assert.Contains(t, msg.body, "<b>kept</b>", "basic formatting is preserved")
}

// emailSafeHTML drops links/images while keeping inline/block formatting and escaping nothing extra.
func TestEmailSafeHTML(t *testing.T) {
	tbl := []struct{ name, in, want string }{
		{"strips anchor keeps text", `<a href="http://evil">x</a>`, "x"},
		{"strips image entirely", `a<img src="http://evil/t.png">b`, "ab"},
		{"keeps bold/italic/code", `<b>b</b><i>i</i><code>c</code>`, `<b>b</b><i>i</i><code>c</code>`},
		{"keeps blockquote and lists", `<blockquote>q</blockquote><ul><li>x</li></ul>`, `<blockquote>q</blockquote><ul><li>x</li></ul>`},
		{"drops onclick handlers", `<span onclick="alert(1)">s</span>`, `<span>s</span>`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(emailSafeHTML(tt.in)))
		})
	}
}

func TestEmail_SendVerification(t *testing.T) {
	email, err := NewEmail(EmailParams{
		From:                     "from@example.org",
		VerificationTemplatePath: "testdata/verification.html.tmpl",
		MsgTemplatePath:          "testdata/msg.html.tmpl",
	}, ntf.SMTPParams{})
	assert.NoError(t, err)
	assert.NotNil(t, email)
	email.TokenGenFn = TokenGenFn
	// proper VerificationRequest without email
	req := VerificationRequest{
		SiteID: "remark",
		User:   "test_username",
		Token:  "secret_",
	}
	assert.NoError(t, email.SendVerification(context.Background(), req))

	// proper VerificationRequest with email
	req.Email = "test@example.org"
	assert.Error(t, email.SendVerification(context.Background(), req), "failed to make smtp client")

	// VerificationRequest with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.EqualError(t, email.SendVerification(ctx, req), "sending message to \"test_username\" aborted due to canceled context")

	// test buildVerificationMessage separately for message text
	res, err := email.buildVerificationMessage(req.User, req.Email, req.Token, req.SiteID)
	assert.NoError(t, err)
	assert.Equal(t, res, `Confirmation for test_username on site remark
Token:secret_
Sent to test@example.org

`)
	assert.Contains(t, res, `secret_`)
	assert.NotContains(t, res, `https://example.org/`)
	email.SubscribeURL = "https://example.org/subscribe.html?token="
	res, err = email.buildVerificationMessage(req.User, req.Email, req.Token, req.SiteID)
	assert.NoError(t, err)
	assert.Equal(t, res, `Confirmation for test_username on site remark
Subscribe url: https://example.org/subscribe.html?token=secret_
Token:secret_
Sent to test@example.org

`)
}

func TokenGenFn(user, _, _ string) (string, error) {
	if user == "error" {
		return "", fmt.Errorf("token generation error")
	}
	return "token", nil
}
