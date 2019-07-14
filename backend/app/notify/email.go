package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
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
	Template string        // request message template
}

// Email implements notify.Destination for email
type Email struct {
	connection *smtp.Client
	params     EmailParams
	template   *template.Template // request message template
}

// tmplData store data for message from request template execution
type tmplData struct {
	From      string
	To        string
	Orig      string
	Link      string
	PostTitle string
}

const emailConnectionTimeOut = 10 * time.Second

//NewEmail makes email object for notifications and run sending daemon
func NewEmail(params EmailParams) (*Email, error) {

	res := Email{params: params}
	if res.params.TimeOut == 0 {
		res.params.TimeOut = emailConnectionTimeOut
	}
	log.Printf("[DEBUG] create new email notifier for server %s with user %s, timeout=%s",
		res.params.Host, res.params.Username, res.params.TimeOut)

	var err error

	tmpl := msgTemplate
	if params.Template != "" {
		tmpl = params.Template
	}
	res.template, err = template.New("messageFromRequest").Parse(tmpl)
	if err != nil {
		return &res, errors.Wrapf(err, "can't parse message template")
	}
	res.connection, err = res.client()

	return &res, err
}

// Send email from request to address in settings
func (e *Email) Send(ctx context.Context, req request) error {
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	// TODO: decide where to get "to" email from
	msg := e.buildMessageFromRequest(req, "test@localhost")
	err := repeater.NewDefault(5, time.Millisecond*250).Do(ctx, func() error {
		return e.sendEmail(msg)
	})
	return err
}

// sendEmail sends prepared message
func (e *Email) sendEmail(message string) error {
	// TODO: write send logic reusing e.connection
	return nil
}

func (e *Email) String() string {
	return fmt.Sprintf("email: '%s'@'%s':%d", e.params.Username, e.params.Host, e.params.Port)
}

//buildMessageFromRequest generates email message based on request
func (e *Email) buildMessageFromRequest(req request, to string) (message string) {
	subject := "New comment"
	if req.comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.comment.PostTitle)
	}
	msg := bytes.Buffer{}
	_ = e.template.Execute(&msg, tmplData{
		req.comment.User.Name,
		req.parent.User.Name,
		req.comment.Orig,
		req.comment.Locator.URL + uiNav + req.comment.ID,
		req.comment.PostTitle,
	})
	return e.buildMessage(subject, msg.String(), to, "text/html")
}

//buildMessage generates email message
func (e *Email) buildMessage(subject, msg, to, contentType string) (message string) {
	message += fmt.Sprintf("From: %s\n", e.params.From)
	message += fmt.Sprintf("To: %s\n", to)
	message += fmt.Sprintf("Subject: %s\n", subject)
	if contentType != "" {
		message += fmt.Sprintf("MIME-version: 1.0;\nContent-Type: %s; charset=\"UTF-8\";\n", contentType)
	}
	message += "\n" + msg
	return message
}

func (e *Email) client() (c *smtp.Client, err error) {
	srvAddress := fmt.Sprintf("%s:%d", e.params.Host, e.params.Port)
	if e.params.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         e.params.Host,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, e.params.Host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, nil
	}

	conn, err := net.DialTimeout("tcp", srvAddress, e.params.TimeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}
	return c, nil
}

var msgTemplate = `{{.From}}{{if .To}} → {{.To}}{{end}}

{{.Orig}}

↦ <a href="{{.Link}}">{{if .PostTitle}}{{.PostTitle}}{{else}}original comment{{end}}</a>
`
