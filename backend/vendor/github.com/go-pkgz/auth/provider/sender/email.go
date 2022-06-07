// Package sender provides email sender
package sender

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"time"

	"github.com/go-pkgz/auth/logger"
)

// Email implements sender interface for VerifyHandler
// Uses common subject line and "from" for all messages
type Email struct {
	logger.L
	SMTPClient
	EmailParams
}

// EmailParams  with all needed to make new Email client with smtp
type EmailParams struct {
	Host        string // SMTP host
	Port        int    // SMTP port
	From        string // From email field
	Subject     string // Email subject
	ContentType string // Content type, optional. Will trigger MIME and Content-Type headers

	TLS          bool   // TLS auth
	SMTPUserName string // user name
	SMTPPassword string // password
	TimeOut      time.Duration
}

// SMTPClient interface defines subset of net/smtp used by email client
type SMTPClient interface {
	Mail(string) error
	Auth(smtp.Auth) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

// NewEmailClient creates email client with prepared smtp
func NewEmailClient(p EmailParams, l logger.L) *Email {
	return &Email{EmailParams: p, L: l, SMTPClient: nil}
}

// Send email with given text
// If SMTPClient defined in Email struct it will be used, if not - new smtp.Client on each send.
// Always closes client on completion or failure.
func (em *Email) Send(to, text string) error {
	em.Logf("[DEBUG] send %q to %s", text, to)
	client := em.SMTPClient
	if client == nil { // if client not set make new net/smtp
		c, err := em.client()
		if err != nil {
			return fmt.Errorf("failed to make smtp client: %w", err)
		}
		client = c
	}

	var quit bool
	defer func() {
		if quit { // quit set if Quit() call passed because it's closing connection as well.
			return
		}
		if err := client.Close(); err != nil {
			em.Logf("[WARN] can't close smtp connection, %v", err)
		}
	}()

	if em.SMTPUserName != "" && em.SMTPPassword != "" {
		auth := smtp.PlainAuth("", em.SMTPUserName, em.SMTPPassword, em.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to auth to smtp %s:%d: %w", em.Host, em.Port, err)
		}
	}

	if err := client.Mail(em.From); err != nil {
		return fmt.Errorf("bad from address %q: %w", em.From, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("bad to address %q: %w", to, err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("can't make email writer: %w", err)
	}

	msg, err := em.buildMessage(text, to)
	if err != nil {
		return fmt.Errorf("can't make email message: %w", err)
	}
	buf := bytes.NewBufferString(msg)
	if _, err = buf.WriteTo(writer); err != nil {
		return fmt.Errorf("failed to send email body to %q: %w", to, err)
	}
	if err = writer.Close(); err != nil {
		em.Logf("[WARN] can't close smtp body writer, %v", err)
	}

	if err = client.Quit(); err != nil {
		em.Logf("[WARN] failed to send quit command to %s:%d, %v", em.Host, em.Port, err)
	} else {
		quit = true
	}
	return nil
}

func (em *Email) client() (c *smtp.Client, err error) {
	srvAddress := fmt.Sprintf("%s:%d", em.Host, em.Port)
	if em.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         em.Host,
			MinVersion:         tls.VersionTLS12,
		}
		conn, e := tls.Dial("tcp", srvAddress, tlsConf)
		if e != nil {
			return nil, fmt.Errorf("failed to dial smtp tls to %s: %w", srvAddress, e)
		}
		if c, err = smtp.NewClient(conn, em.Host); err != nil {
			return nil, fmt.Errorf("failed to make smtp client for %s: %w", srvAddress, err)
		}
		return c, nil
	}

	conn, err := net.DialTimeout("tcp", srvAddress, em.TimeOut)
	if err != nil {
		return nil, fmt.Errorf("timeout connecting to %s: %w", srvAddress, err)
	}

	c, err = smtp.NewClient(conn, em.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return c, nil
}

func (em *Email) buildMessage(msg, to string) (message string, err error) {
	addHeader := func(msg, h, v string) string {
		msg += fmt.Sprintf("%s: %s\n", h, v)
		return msg
	}
	message = addHeader(message, "From", em.From)
	message = addHeader(message, "To", to)
	message = addHeader(message, "Subject", em.Subject)
	message = addHeader(message, "Content-Transfer-Encoding", "quoted-printable")

	if em.ContentType != "" {
		message = addHeader(message, "MIME-version", "1.0")
		message = addHeader(message, "Content-Type", em.ContentType+`; charset="UTF-8"`)
	}
	message = addHeader(message, "Date", time.Now().Format(time.RFC1123Z))

	buff := &bytes.Buffer{}
	qp := quotedprintable.NewWriter(buff)
	if _, err := qp.Write([]byte(msg)); err != nil {
		return "", err
	}
	// flush now, must NOT use defer, for small body, defer may cause buff.String() got empty body
	if err := qp.Close(); err != nil {
		return "", fmt.Errorf("quotedprintable Write failed: %w", err)
	}
	m := buff.String()
	message += "\n" + m
	return message, nil
}
