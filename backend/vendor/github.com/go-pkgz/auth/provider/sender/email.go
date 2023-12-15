// Package sender provides email sender
package sender

import (
	"time"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/email"
)

// Email implements sender interface for VerifyHandler
// Uses common subject line and "from" for all messages
type Email struct {
	EmailParams
	logger.L
	sender *email.Sender
}

// EmailParams  with all needed to make new Email client with smtp
type EmailParams struct {
	Host        string // SMTP host
	Port        int    // SMTP port
	From        string // From email field
	Subject     string // Email subject
	ContentType string // Content type

	TLS                bool          // TLS auth
	StartTLS           bool          // StartTLS auth
	InsecureSkipVerify bool          // Skip certificate verification
	Charset            string        // Character set
	LoginAuth          bool          // LOGIN auth method instead of default PLAIN, needed for Office 365 and outlook.com
	SMTPUserName       string        // username
	SMTPPassword       string        // password
	TimeOut            time.Duration // TCP connection timeout
}

// NewEmailClient creates email client
func NewEmailClient(emailParams EmailParams, l logger.L) *Email {
	var opts []email.Option

	if emailParams.SMTPUserName != "" {
		opts = append(opts, email.Auth(emailParams.SMTPUserName, emailParams.SMTPPassword))
	}

	if emailParams.ContentType != "" {
		opts = append(opts, email.ContentType(emailParams.ContentType))
	}

	if emailParams.Charset != "" {
		opts = append(opts, email.Charset(emailParams.Charset))
	}

	if emailParams.LoginAuth {
		opts = append(opts, email.LoginAuth())
	}

	if emailParams.Port != 0 {
		opts = append(opts, email.Port(emailParams.Port))
	}

	if emailParams.TimeOut != 0 {
		opts = append(opts, email.TimeOut(emailParams.TimeOut))
	}

	if emailParams.TLS {
		opts = append(opts, email.TLS(true))
	}

	if emailParams.StartTLS {
		opts = append(opts, email.STARTTLS(true))
	}

	if emailParams.InsecureSkipVerify {
		opts = append(opts, email.InsecureSkipVerify(true))
	}

	sender := email.NewSender(emailParams.Host, opts...)

	return &Email{EmailParams: emailParams, L: l, sender: sender}
}

// Send email with given text
func (e *Email) Send(to, text string) error {
	e.Logf("[DEBUG] send %q to %s", text, to)
	return e.sender.Send(text, email.Params{
		From:    e.From,
		To:      []string{to},
		Subject: e.Subject,
	})
}
