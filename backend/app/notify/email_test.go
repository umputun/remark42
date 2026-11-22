package notify

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"testing"
	"testing/synctest"
	"time"

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
	e := Email{TokenGenFn: TokenGenFn}

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
	synctest.Test(t, func(t *testing.T) {
		email, err := NewEmail(EmailParams{
			From:                     "from@example.org",
			VerificationTemplatePath: "testdata/verification.html.tmpl",
			MsgTemplatePath:          "testdata/msg.html.tmpl",
		}, ntf.SMTPParams{Port: 99999}) // an invalid port fails the send before any dial, so the bubble has no real I/O
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
	})
}

// the dates in the reply e-mail carry their offset from UTC: a reader in another zone cannot tell
// when a comment was written from the server's local time alone, and the two comments in one message
// can carry different offsets when the server's zone changed between them
// html/template writes the plus sign as &#43;, which a mail client shows as +
func TestEmail_ReplyDatesCarryZone(t *testing.T) {
	email, err := NewEmail(EmailParams{From: "from@example.org"}, ntf.SMTPParams{})
	require.NoError(t, err)
	email.TokenGenFn = TokenGenFn

	req := Request{
		Comment: store.Comment{ID: "2", ParentID: "1", User: store.User{ID: "u2", Name: "replier"},
			Timestamp: time.Date(2026, 11, 21, 15, 30, 0, 0, time.UTC)},
		parent: store.Comment{ID: "1", Text: "question", User: store.User{ID: "u1", Name: "author"},
			Timestamp: time.Date(2026, 11, 21, 18, 5, 0, 0, time.FixedZone("", 3*60*60))},
		Emails: []string{"test@example.org"},
	}
	msg, err := email.buildMessageFromRequest(req, req.Emails[0], false)
	require.NoError(t, err)
	assert.Contains(t, msg.body, "21.11.2026 at 18:05 (UTC&#43;03:00)", "date of the comment replied to")
	assert.Contains(t, msg.body, "21.11.2026 at 15:30 (UTC&#43;00:00)", "date of the reply")
}

// TestEmail_AdminLinksNameTheirHost checks the links in a message to an admin. The page address of a
// comment comes from the commenter, and an admin is notified about every comment, so the address is
// written again from its parts, the label names the host the link opens, and an address that is not a
// plain web address gets no link.
func TestEmail_AdminLinksNameTheirHost(t *testing.T) {
	email, err := NewEmail(EmailParams{TokenGenFn: func(_, _, _ string) (string, error) { return "token", nil }}, ntf.SMTPParams{})
	require.NoError(t, err)

	build := func(pageURL string, forAdmin bool) string {
		req := Request{
			Comment: store.Comment{ID: "2", ParentID: "1", Text: "<p>reply</p>", User: store.User{ID: "u2", Name: "commenter"},
				Locator: store.Locator{SiteID: "remark", URL: pageURL}},
			parent: store.Comment{ID: "1", Text: "<p>parent</p>", User: store.User{ID: "u1", Name: "author"}},
		}
		msg, e := email.buildMessageFromRequest(req, "to@example.org", forAdmin)
		require.NoError(t, e)
		return msg.body
	}

	linked := []struct{ name, pageURL, link, host string }{
		{name: "plain address", pageURL: "https://evil.example/blog-login", link: "https://evil.example/blog-login", host: "evil.example"},
		{name: "user part, port and fragment", pageURL: "https://blog.example@evil.example:8443/x?a=1#frag",
			link: "https://evil.example:8443/x?a=1", host: "evil.example"},
		{name: "a Cyrillic letter", pageURL: "https://bl\u043eg.example/post", link: "https://xn--blg-ted.example/post", host: "xn--blg-ted.example"},
		{name: "a capital Cyrillic letter", pageURL: "https://BL\u041eG.example/x", link: "https://xn--blg-ted.example/x", host: "xn--blg-ted.example"},
		{name: "letters a browser drops", pageURL: "https://p\u0430ypal" + strings.Repeat("\u00ad", 2400) + "\U000E0100.com/x",
			link: "https://xn--pypal-4ve.com/x", host: "xn--pypal-4ve.com"},
		{name: "a full stop a browser reads as a dot", pageURL: "https://blog.example\u3002evil.example/x",
			link: "https://blog.example.evil.example/x", host: "blog.example.evil.example"},
		{name: "capitals", pageURL: "HTTPS://Blog.Example/Path", link: "https://blog.example/Path", host: "blog.example"},
		{name: "IPv6 address", pageURL: "http://[::1]:8080/x", link: "http://[::1]:8080/x", host: "::1"},
		{name: "IPv4 address", pageURL: "http://127.0.0.1:8080/x", link: "http://127.0.0.1:8080/x", host: "127.0.0.1"},
		{name: "a name that only looks like a number to Go", pageURL: "https://docs.0b10/x", link: "https://docs.0b10/x", host: "docs.0b10"},
		{name: "a question mark with nothing after it", pageURL: "https://blog.example/post?", link: "https://blog.example/post?", host: "blog.example"},
	}
	for _, tt := range linked {
		t.Run(tt.name, func(t *testing.T) {
			body := build(tt.pageURL, true)
			assert.Contains(t, body, `<a href="`+tt.link+`#remark42__comment-2" style="color: #0aa; font-size: 14px;"><b>Reply on `+tt.host+`</b></a>`)
			assert.Contains(t, body, `<a href="`+tt.link+`#remark42__comment-1" style="color: #0aa; font-size: 14px;"><b>Show on `+tt.host+`</b></a>`)
		})
	}

	// what a browser opens as evil.example and Go reads as an address with no host, and what is no web address
	for _, pageURL := range []string{"https:evil.example/x", "https:/evil.example/x", "https:///evil.example/x", "http:evil.example",
		"mailto:x@evil.example", "/post", "https://xn--a-.bl\u043eg.example/x", "https://evil.example\\@blog.example/",
		// a wide "@" and a wide space, which the mapping turns into plain ones, a port out of range, an IPv6 zone
		"https://blog.example\uff20evil.example/x", "https://evil.example\u3000/x", "https://evil.example:65536/x",
		"http://[fe80::1%25evil.example]/x",
		// numbers a browser expands to another address or refuses
		"http://127.1/evil.example", "http://2130706433/evil.example", "http://0x7f.1/evil.example", "https://evil.example.1/x",
		"http://10.0.1.0x/evil.example", "https://evil.example.08/x"} {
		t.Run("no link for "+pageURL, func(t *testing.T) {
			body := build(pageURL, true)
			assert.NotContains(t, body, "Reply")
			assert.NotContains(t, body, "Show")
			assert.NotContains(t, body, "evil.example")
		})
	}

	t.Run("a user's message links the page as it is", func(t *testing.T) {
		body := build("https://blog.example/post", false)
		assert.Contains(t, body, `<a href="https://blog.example/post#remark42__comment-2" style="color: #0aa; font-size: 14px;"><b>Reply</b></a>`)
		assert.Contains(t, body, `<a href="https://blog.example/post#remark42__comment-1" style="color: #0aa; font-size: 14px;"><b>Show</b></a>`)
	})
}

// a reply notifies the authors of every comment above it by email; the unsubscribe link in each message
// must name the user the address belongs to, not the author of the comment replied to
func TestEmail_UnsubscribeLinkPerRecipient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// thread p0 <- p1 <- p2 <- p3 <- p4 <- c, c by u5 who also wrote p0
		dataStore := &mockStore{data: map[string]store.Comment{}, userDetails: map[string]string{
			"u1": "pair@example.org", "u2": "pair@example.org", // one address for two users
			"u3": "u3@example.org", "u4": "u4@example.org", "u5": "u5@example.org",
		}}
		dataStore.data["p0"] = store.Comment{ID: "p0", User: store.User{ID: "u5"}}
		dataStore.data["p1"] = store.Comment{ID: "p1", ParentID: "p0", User: store.User{ID: "u1"}}
		dataStore.data["p2"] = store.Comment{ID: "p2", ParentID: "p1", User: store.User{ID: "u2"}}
		dataStore.data["p3"] = store.Comment{ID: "p3", ParentID: "p2", User: store.User{ID: "u3"}}
		dataStore.data["p4"] = store.Comment{ID: "p4", ParentID: "p3", User: store.User{ID: "u4"}}
		dest := &MockDest{id: 1}
		s := NewService(dataStore, 1, dest)
		s.Submit(Request{Comment: store.Comment{ID: "c", ParentID: "p4", User: store.User{ID: "u5"},
			Locator: store.Locator{SiteID: "remark"}}})
		synctest.Wait()
		s.Close()
		require.Len(t, dest.Get(), 1)
		req := dest.Get()[0]
		assert.Equal(t, []string{"pair@example.org", "u3@example.org", "u4@example.org"}, req.Emails,
			"one message per address, none to u5 who wrote the reply")

		email, err := NewEmail(EmailParams{MsgTemplatePath: "testdata/msg.html.tmpl",
			UnsubscribeURL: "https://example.org/email/unsubscribe.html"}, ntf.SMTPParams{})
		require.NoError(t, err)
		email.TokenGenFn = func(userID, address, _ string) (string, error) { return userID + "::" + address, nil }

		for address, user := range map[string]string{
			"u4@example.org":   "u4", // the author of the comment replied to
			"u3@example.org":   "u3",
			"pair@example.org": "u2", // the user nearest to the reply of the two with this address
		} {
			msg, err := email.buildMessageFromRequest(req, address, false)
			require.NoError(t, err)
			link := "https://example.org/email/unsubscribe.html?site=remark&tkn=" + user + "::" + address
			assert.Equal(t, link, msg.unsubscribeLink, "link in the header of the message to %s", address)
			assert.Contains(t, msg.body, "Unsubscribe link: "+template.HTMLEscapeString(link),
				"link in the text of the message to %s", address)
		}
	})
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

// TestEmail_UserPictureFromThisInstanceOnly covers the pictures of the comment's and the parent comment's
// authors: the email shows them only when they are avatars stored by this instance, at either avatar route.
// An imported or restored user keeps the picture URL of the file, which may point to any host and would be
// the same remote tracking image the email policy drops from the comment text.
func TestEmail_UserPictureFromThisInstanceOnly(t *testing.T) {
	const apiAvatars, rootAvatars = "https://remark42.example.org/api/v1/avatar/", "https://remark42.example.org/avatar/"
	email, err := NewEmail(EmailParams{From: "from@example.org", AvatarURLs: []string{apiAvatars, rootAvatars}},
		ntf.SMTPParams{}) // the default template, which shows both pictures
	require.NoError(t, err)
	email.TokenGenFn = TokenGenFn

	build := func(e *Email, picture, parentPicture string) string {
		req := Request{
			Comment: store.Comment{ID: "999", User: store.User{ID: "1", Name: "test_user", Picture: picture}, ParentID: "1", Text: "reply"},
			parent:  store.Comment{ID: "1", User: store.User{ID: "2", Name: "parent_user", Picture: parentPicture}, Text: "parent"},
		}
		msg, e1 := e.buildMessageFromRequest(req, "admin@example.org", true)
		require.NoError(t, e1)
		return msg.body
	}

	for _, stored := range []string{apiAvatars + "aabbcc.image", rootAvatars + "aabbcc.image"} {
		body := build(email, stored, apiAvatars+"ddeeff.image")
		assert.Contains(t, body, `<img src="`+stored+`"`, "picture of the user")
		body = build(email, apiAvatars+"ddeeff.image", stored)
		assert.Contains(t, body, `<img src="`+stored+`"`, "picture of the parent's user")
	}

	const stored = apiAvatars + "ddeeff.image"
	for _, external := range []string{
		"https://attacker.example/track.png",
		"https://remark42.example.org.attacker.example/api/v1/avatar/aabbcc.image",
		"https://remark42.example.org/api/v1/img?src=aHR0cHM6Ly9hdHRhY2tlci5leGFtcGxlL3RyYWNrLnBuZw",
		apiAvatars + "../img?src=aHR0cHM6Ly9hdHRhY2tlci5leGFtcGxlL3RyYWNrLnBuZw",
		rootAvatars + "../api/v1/img?src=aHR0cHM6Ly9hdHRhY2tlci5leGFtcGxlL3RyYWNrLnBuZw",
		apiAvatars + "..%2Fimg%3Fsrc=aHR0cHM6Ly9hdHRhY2tlci5leGFtcGxlL3RyYWNrLnBuZw",
		apiAvatars + "../picture/user/aabbcc.png",
		apiAvatars + `..\img`,
		apiAvatars + "aabbcc.image?attacker.example",
		apiAvatars + "aabbcc.image#attacker.example",
		apiAvatars,
		apiAvatars + "..",
		rootAvatars + ".",
		"/api/v1/avatar/aabbcc.image",
	} {
		body := build(email, external, stored)
		assert.NotContains(t, body, `src="`+external+`"`, "picture of the user %q", external)
		assert.Equal(t, 1, strings.Count(body, "<img"), "only the parent's picture is left, %q", external)
		assert.Contains(t, body, `<img src="`+stored+`"`)

		body = build(email, stored, external)
		assert.NotContains(t, body, `src="`+external+`"`, "picture of the parent's user %q", external)
		assert.Equal(t, 1, strings.Count(body, "<img"), "only the user's picture is left, %q", external)
		assert.Contains(t, body, `<img src="`+stored+`"`)
	}

	assert.NotContains(t, build(email, "", ""), "<img", "an empty picture is left out")

	// pictures are left out when the location of the avatars is not known
	email, err = NewEmail(EmailParams{From: "from@example.org"}, ntf.SMTPParams{})
	require.NoError(t, err)
	email.TokenGenFn = TokenGenFn
	assert.NotContains(t, build(email, stored, "https://attacker.example/track.png"), "<img")

	email.AvatarURLs = []string{""}
	assert.NotContains(t, build(email, "aabbcc.image", "track.png"), "<img", "an empty prefix matches nothing")
}

// emailSafeHTML drops links/images while keeping inline/block formatting and escaping nothing extra.
func TestEmailSafeHTML(t *testing.T) {
	tbl := []struct{ name, in, want string }{
		{"strips anchor keeps text", `<a href="http://evil">x</a>`, "x"},
		{"strips image entirely", `a<img src="http://evil/t.png">b`, "ab"},
		{"keeps bold/italic/code", `<b>b</b><i>i</i><code>c</code>`, `<b>b</b><i>i</i><code>c</code>`},
		{"keeps blockquote and lists", `<blockquote>q</blockquote><ul><li>x</li></ul>`, `<blockquote>q</blockquote><ul><li>x</li></ul>`},
		{"drops onclick handlers", `<span onclick="alert(1)">s</span>`, `<span>s</span>`},
		{"drops stray end tags", `<p>ok </div></div><h1>h</h1></p>`, `<p>ok </p><h1>h</h1><p></p>`},
		{"drops stray end tags left by removed elements", `<p>ok <svg><title><xmp></title></div></div><h1>h</h1></xmp></title></svg></p>`,
			`<p>ok </p><h1>h</h1><p></p>`},
		{"closes unclosed tags", `<div><h1><s>x`, `<div><h1><s>x</s></h1></div>`},
		{"keeps misnested tags from closing each other", `<h1><div><h2><b><h3>x</b></div></div>`,
			`<h1><div><h2><b></b></h2><h3><b>x</b></h3></div></h1>`},
		{"keeps text only if nested too deep", `</div><h1>h</h1>` + strings.Repeat("<b>", 600) + `x &amp; y`, `hx &amp; y`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(emailSafeHTML(tt.in)))
			assert.Equal(t, tt.want, string(emailSafeHTML(tt.want)), "parsed to the same html again")
		})
	}
}

// comment text with end tags of its own must not close the blocks of the message template it is placed in
func TestEmail_BuildMessageKeepsCommentInsideItsBlock(t *testing.T) {
	email, err := NewEmail(EmailParams{
		TokenGenFn: func(_, _, _ string) (string, error) { return "token", nil },
	}, ntf.SMTPParams{})
	require.NoError(t, err)

	build := func(text string) string {
		req := Request{
			Comment: store.Comment{ID: "1", ParentID: "2", Text: text, User: store.User{ID: "u1", Name: "attacker"},
				Locator: store.Locator{SiteID: "remark", URL: "https://example.org/post"}},
			parent: store.Comment{ID: "2", Text: text, User: store.User{ID: "u2", Name: "victim"}},
			Emails: []string{"victim@example.org"},
		}
		msg, e := email.buildMessageFromRequest(req, req.Emails[0], false)
		require.NoError(t, e)
		return msg.body
	}

	plain := build("<p>text</p>")
	body := build("<p>text </div></div></div></div><h1>Your account is suspended</h1></p>")
	assert.Contains(t, body, "<h1>Your account is suspended</h1>", "formatting is preserved")
	assert.Equal(t, strings.Count(plain, "<div"), strings.Count(body, "<div"), "no block opened by the comment")
	assert.Equal(t, strings.Count(plain, "</div>"), strings.Count(body, "</div>"), "no block closed by the comment")
}

func TestEmail_SendVerification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		email, err := NewEmail(EmailParams{
			From:                     "from@example.org",
			VerificationTemplatePath: "testdata/verification.html.tmpl",
			MsgTemplatePath:          "testdata/msg.html.tmpl",
		}, ntf.SMTPParams{Port: 99999}) // an invalid port fails the send before any dial, so the bubble has no real I/O
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
	})
}

// the default template carries the confirmation link, with the token box kept below it as the fallback
func TestEmail_DefaultVerificationTemplateLink(t *testing.T) {
	email, err := NewEmail(EmailParams{From: "from@example.org", SubscribeURL: "https://remark.example.org/email/subscribe.html?tkn="},
		ntf.SMTPParams{})
	require.NoError(t, err)
	res, err := email.buildVerificationMessage("test_username", "test@example.org", "aaa.bbb-ccc_ddd", "remark")
	require.NoError(t, err)
	assert.Contains(t, res, `<a href="https://remark.example.org/email/subscribe.html?tkn=aaa.bbb-ccc_ddd">`)
	assert.Contains(t, res, `user-select: all;">aaa.bbb-ccc_ddd</p>`, "the token to paste stays")

	email.SubscribeURL = ""
	res, err = email.buildVerificationMessage("test_username", "test@example.org", "aaa.bbb-ccc_ddd", "remark")
	require.NoError(t, err)
	assert.NotContains(t, res, "<a href", "no link without a subscribe URL")
}

func TokenGenFn(user, _, _ string) (string, error) {
	if user == "error" {
		return "", fmt.Errorf("token generation error")
	}
	return "token", nil
}

// the verification message goes to an address nobody has confirmed yet, so a free-text display name
// can't carry a message and a link of the user's choice there
func TestEmail_VerificationUserNameReduced(t *testing.T) {
	email, err := NewEmail(EmailParams{From: "from@example.org"}, ntf.SMTPParams{})
	require.NoError(t, err)

	res, err := email.buildVerificationMessage(
		"your prize! Claim it within 24 hours: https://evil.example/claim?id=42 - link expires",
		"test@example.org", "secret_", "remark")
	require.NoError(t, err)
	assert.Contains(t, res, "Confirmation for <b>your prize Claim it within 24 hours https evil example claim id</b> on site <b>remark</b>")
	assert.NotContains(t, res, "evil.example")
}

func TestVerificationUserName(t *testing.T) {
	tbl := []struct{ name, in, want string }{
		{"plain name", "developer one", "developer one"},
		{"underscore and digits", "some_user 42", "some_user 42"},
		{"non-latin letters", "Пользователь Ёж", "Пользователь Ёж"},
		{"combining marks", "हिन्दी Cafe\u0301", "हिन्दी Cafe\u0301"},
		{"empty", "", ""},
		{"punctuation becomes a space", "  J. O'Neil-Smith, Jr.  ", "J O Neil Smith Jr"},
		{"nothing left", "!!! ???", ""},
		{"control and invisible characters", "dev\r\n\tone\u00a0\u202etwo\xffthree", "dev one two three"},
		{"markup", `<a href="https://example.org">one</a>`, "a href https example org one a"},
		{"cut to 64 characters", strings.Repeat("я", 100), strings.Repeat("я", 64)},
		{"no trailing space after the cut", strings.Repeat("a", 63) + " " + strings.Repeat("b", 10), strings.Repeat("a", 63)},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, verificationUserName(tt.in))
		})
	}
}
