package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// EmailParams contain settings for email set up
type EmailParams struct {
	Host     string        // SMTP host
	Port     int           // SMTP port
	TLS      bool          // TLS auth
	From     string        // From email field
	Username string        // user name
	Password string        // password
	TimeOut  time.Duration // TLS connection timeout
}

// Email implements notify.Destination for email
type Email struct {
	connection *smtp.Client
	host       string        // SMTP host
	port       int           // SMTP port
	tls        bool          // TLS auth
	from       string        // From email field
	username   string        // user name
	password   string        // password
	timeOut    time.Duration // TLS connection timeout
}

const emailConnectionTimeOut = 10 * time.Second

//NewEmail makes email object for notifications and run sending daemon
func NewEmail(params EmailParams) (*Email, error) {

	res := Email{
		host:     params.Host,
		port:     params.Port,
		tls:      params.TLS,
		from:     params.From,
		username: params.Username,
		password: params.Password,
		timeOut:  params.TimeOut,
	}
	if res.timeOut == 0 {
		res.timeOut = emailConnectionTimeOut
	}
	log.Printf("[DEBUG] create new email notifier for server %s with user %s, timeout=%s",
		res.host, res.username, res.timeOut)

	var err error
	res.connection, err = res.client()

	return &res, err
}

// Send email from request to address in settings
func (e *Email) Send(ctx context.Context, req request) error {
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	msg := e.buildMessageFromRequest(req, "test@localhost")
	return e.SendEmail(msg)
}

// SendEmail sends prepared message
func (e *Email) SendEmail(message string) error {
	// TODO: write send logic reusing e.connection
	return nil
}

func (e *Email) String() string {
	return fmt.Sprintf("email: '%s'@'%s':%d", e.username, e.host, e.port)
}

//buildMessageFromRequest generates email message based on request
func (e *Email) buildMessageFromRequest(req request, to string) (message string) {
	subject := "New comment"
	link := fmt.Sprintf("↦ <a href=\"%s\">original comment</a>", req.comment.Locator.URL+uiNav+req.comment.ID)
	if req.comment.PostTitle != "" {
		link = fmt.Sprintf("↦ <a href=\"%s\">%s</a>", req.comment.Locator.URL+uiNav+req.comment.ID, req.comment.PostTitle)
		subject += fmt.Sprintf(" for \"%s\"", req.comment.PostTitle)
	}
	from := req.comment.User.Name
	if req.comment.ParentID != "" {
		from += " → " + req.parent.User.Name
	}
	// TODO: message looks bad, review it
	msg := fmt.Sprintf("%s\n\n%s\n\n%s", from, req.comment.Orig, link)
	return e.BuildMessage(subject, msg, to, "text/html")
}

//BuildMessage generates email message
func (e *Email) BuildMessage(subject, msg, to, contentType string) (message string) {
	message += fmt.Sprintf("From: %s\n", e.from)
	message += fmt.Sprintf("To: %s\n", to)
	message += fmt.Sprintf("Subject: %s\n", subject)
	if contentType != "" {
		message += fmt.Sprintf("MIME-version: 1.0;\nContent-Type: %s; charset=\"UTF-8\";\n", contentType)
	}
	message += "\n" + msg
	return message
}

func (e *Email) client() (c *smtp.Client, err error) {
	srvAddress := fmt.Sprintf("%s:%d", e.host, e.port)
	if e.tls {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         e.host,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, e.host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, nil
	}

	conn, err := net.DialTimeout("tcp", srvAddress, e.timeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}
	return c, nil
}
