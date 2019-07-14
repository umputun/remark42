package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"sync"
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
	EmailParams
	SMTPClient
	template  *template.Template // request message template
	sendMutex sync.Mutex         // Send is synchronious and blocked by this mutex
	count     int                // amount of waiting + running send requests
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

	res := Email{EmailParams: params}
	if res.TimeOut == 0 {
		res.TimeOut = emailConnectionTimeOut
	}
	log.Printf("[DEBUG] create new email notifier for server %s with user %s, timeout=%s",
		res.Host, res.Username, res.TimeOut)

	var err error

	tmpl := msgTemplate
	if params.Template != "" {
		tmpl = params.Template
	}
	res.template, err = template.New("messageFromRequest").Parse(tmpl)
	if err != nil {
		return &res, errors.Wrapf(err, "can't parse message template")
	}
	// establish test connection and
	testSMTPClient, err := res.client()
	if err != nil {
		return &res, errors.Wrapf(err, "can't establish test connection")
	}
	if err = testSMTPClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", res.Host, res.Port, err)
	}
	if err := testSMTPClient.Close(); err != nil {
		log.Printf("[WARN] can't close smtp connection, %v", err)
	}
	return &res, err
}

// Send email from request to address in settings
func (e *Email) Send(ctx context.Context, req request) error {
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	// TODO: decide where to get "to" email from
	to := "test@localhost"
	msg := e.buildMessageFromRequest(req, to)
	err := repeater.NewDefault(5, time.Millisecond*250).Do(ctx, func() error {
		return e.sendEmail(msg, to)
	})
	return err
}

// sendEmail sends message, prepared by Email.buildMessage
func (e *Email) sendEmail(message, to string) error {
	e.count++
	e.sendMutex.Lock()
	if e.SMTPClient == nil { // if client not set make new net/smtp
		c, err := e.client()
		if err != nil {
			e.count--
			e.sendMutex.Unlock()
			return errors.Wrap(err, "failed to make smtp client")
		}
		e.SMTPClient = c
	}

	defer func() {
		// count == 1 means that no one else waiting to send the message at this moment,
		// e.SMTPClient is nil if Quit() call passed because it's closing connection as well
		if e.count == 1 && e.SMTPClient != nil {
			if err := e.SMTPClient.Close(); err != nil {
				log.Printf("[WARN] can't close smtp connection, %v", err)
			}
			e.SMTPClient = nil
		}
		e.count--
		e.sendMutex.Unlock()
	}()

	if err := e.SMTPClient.Mail(e.From); err != nil {
		return errors.Wrapf(err, "bad from address %q", e.From)
	}
	if err := e.SMTPClient.Rcpt(to); err != nil {
		return errors.Wrapf(err, "bad to address %q", to)
	}

	writer, err := e.SMTPClient.Data()
	if err != nil {
		return errors.Wrap(err, "can't make email writer")
	}

	buf := bytes.NewBufferString(message)
	if _, err = buf.WriteTo(writer); err != nil {
		return errors.Wrapf(err, "failed to send email body to %q", to)
	}
	if err = writer.Close(); err != nil {
		log.Printf("[WARN] can't close smtp body writer, %v", err)
	}

	// count == 1 means that no one else waiting to send the message at this moment
	if e.count == 1 {
		if err = e.SMTPClient.Quit(); err != nil {
			log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
		} else {
			e.SMTPClient = nil
		}
	}
	return nil
}

func (e *Email) String() string {
	return fmt.Sprintf("email: %s using '%s'@'%s':%d", e.From, e.Username, e.Host, e.Port)
}

//buildMessageFromRequest generates email message based on request
func (e *Email) buildMessageFromRequest(req request, to string) (message string) {
	subject := "New comment"
	if req.comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.comment.PostTitle)
	}
	msg := bytes.Buffer{}
	// we don't expect valid template to fail
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
	message += fmt.Sprintf("From: %s\n", e.From)
	message += fmt.Sprintf("To: %s\n", to)
	message += fmt.Sprintf("Subject: %s\n", subject)
	if contentType != "" {
		message += fmt.Sprintf("MIME-version: 1.0;\nContent-Type: %s; charset=\"UTF-8\";\n", contentType)
	}
	message += "\n" + msg
	return message
}

func (e *Email) client() (c *smtp.Client, err error) {
	srvAddress := fmt.Sprintf("%s:%d", e.Host, e.Port)
	if e.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         e.Host,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, e.Host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, nil
	}

	conn, err := net.DialTimeout("tcp", srvAddress, e.TimeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	if e.Username != "" && e.Password != "" {
		auth := smtp.PlainAuth("", e.Username, e.Password, e.Host)
		if err := c.Auth(auth); err != nil {
			return nil, errors.Wrapf(err, "failed to auth to smtp %s:%d", e.Host, e.Port)
		}
	}

	return c, nil
}

var msgTemplate = `{{.From}}{{if .To}} → {{.To}}{{end}}

{{.Orig}}

↦ <a href="{{.Link}}">{{if .PostTitle}}{{.PostTitle}}{{else}}original comment{{end}}</a>
`
