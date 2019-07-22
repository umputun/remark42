package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// EmailParams contain settings for email set up
type EmailParams struct {
	Host          string        // SMTP host
	Port          int           // SMTP port
	TLS           bool          // TLS auth
	From          string        // From email field
	Username      string        // user name
	Password      string        // password
	TimeOut       time.Duration // TLS connection timeout
	Template      string        // request message template
	BufferSize    int           // email send buffer size
	FlushDuration time.Duration // maximum time after which email will me sent
}

// Email implements notify.Destination for email
type Email struct {
	EmailParams
	smtpClient
	template *template.Template // request message template

	ctx context.Context

	buffer        []emailMessage
	lock          sync.Mutex
	lastWriteTime time.Time
	once          sync.Once
}

// smtpClient interface defines subset of net/smtp used by email client
type smtpClient interface {
	Mail(string) error
	Auth(smtp.Auth) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

type emailMessage struct {
	message string
	to      string
}

// tmplData store data for message from request template execution
type tmplData struct {
	From      string
	To        string
	Orig      string
	Link      string
	PostTitle string
}

const defaultEmailTimeout = 10 * time.Second

//NewEmail makes email object for notifications and run sending daemon
func NewEmail(params EmailParams) (*Email, error) {

	res := Email{EmailParams: params}
	if res.TimeOut == 0 {
		res.TimeOut = defaultEmailTimeout
	}
	if res.BufferSize == 0 {
		res.BufferSize = 1
	}
	res.buffer = make([]emailMessage, 0, res.BufferSize+1)
	log.Printf("[DEBUG] create new email notifier for server %s with user %s, timeout=%s",
		res.Host, res.Username, res.TimeOut)

	var err error

	tmpl := defaultEmailTemplate
	if params.Template != "" {
		tmpl = params.Template
	}
	res.template, err = template.New("messageFromRequest").Parse(tmpl)
	if err != nil {
		return &res, errors.Wrapf(err, "can't parse message template")
	}
	// establish test connection
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

// Send email from request to address in settings via buffer, thread-safe
func (e *Email) Send(ctx context.Context, req request) error {
	// initialise context and start auto flush once,
	// as this is the first moment we see the context from caller
	e.once.Do(func() {
		e.ctx = ctx
		e.startAutoFlush()
	})
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	// TODO: decide where to get "to" email from
	to := "test@localhost"
	msg, err := e.buildMessageFromRequest(req, to)
	if err != nil {
		return err
	}
	return e.synced(func() error {
		e.lastWriteTime = time.Now()
		e.buffer = append(e.buffer, emailMessage{msg, to})
		if len(e.buffer) >= e.BufferSize {
			if err := e.sendBuffer(); err != nil {
				log.Printf("[WARN] , %s", err)
				return errors.Wrapf(err, "email flush failed")
			}
			e.buffer = e.buffer[0:0]
		}
		return nil
	})
}

// sendEmail sends message prepared by Email.buildMessage to prepared Email.smtpClient, thread unsafe
func (e *Email) sendEmail(m emailMessage) error {
	if e.smtpClient == nil {
		return errors.New("sendEmail called without smtpClient set")
	}
	if err := e.smtpClient.Mail(e.From); err != nil {
		return errors.Wrapf(err, "bad from address %q", e.From)
	}
	if err := e.smtpClient.Rcpt(m.to); err != nil {
		return errors.Wrapf(err, "bad to address %q", m.to)
	}

	writer, err := e.smtpClient.Data()
	if err != nil {
		return errors.Wrap(err, "can't make email writer")
	}

	buf := bytes.NewBufferString(m.message)
	if _, err = buf.WriteTo(writer); err != nil {
		return errors.Wrapf(err, "failed to send email body to %q", m.to)
	}
	if err = writer.Close(); err != nil {
		log.Printf("[WARN] can't close smtp body writer, %v", err)
	}
	return nil
}

// sendBuffer sends all collected messages to server, thread unsafe
func (e *Email) sendBuffer() (err error) {
	if len(e.buffer) == 0 {
		return nil
	}

	if e.smtpClient == nil { // if client not set make new net/smtp
		c, err := e.client()
		if err != nil {
			return errors.Wrap(err, "failed to make smtp client")
		}
		e.smtpClient = c
	}

	var cumulativeErrors []string

	for _, m := range e.buffer {
		err := repeater.NewDefault(5, time.Millisecond*250).Do(e.ctx, func() error { return e.sendEmail(m) })
		if err != nil {
			cumulativeErrors = append(cumulativeErrors, errors.Wrapf(err, "can't send message to %s", m.to).Error())
		}
	}

	if err := e.smtpClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
		if err := e.smtpClient.Close(); err != nil {
			log.Printf("[WARN] can't close smtp connection, %v", err)
			cumulativeErrors = append(cumulativeErrors, err.Error())
		}
	}
	// forget smtpClient to make next run establish new connection
	e.smtpClient = nil
	if len(cumulativeErrors) > 0 {
		err = fmt.Errorf(strings.Join(cumulativeErrors, "\n"))
	}
	return errors.Wrapf(err, "problems with sending messages")
}

// startAutoFlush sets auto flush duration from e.FlushDuration if it's not zero
// and flushes all in-fly records in case of e.ctx closure
func (e *Email) startAutoFlush() {
	if e.FlushDuration > 0 { // activate background auto-flush
		ticker := time.NewTicker(e.FlushDuration)
		go func() {
			for {
				select {
				case <-ticker.C:
					var shouldFlush bool
					_ = e.synced(func() error {
						shouldFlush = time.Now().After(e.lastWriteTime.Add(e.FlushDuration)) && len(e.buffer) > 0
						return nil
					})
					if shouldFlush {
						if err := e.Flush(); err != nil {
							log.Printf("[WARN] email flush failed, %s", err)
						}
					}
				case <-e.ctx.Done():
					err := e.synced(func() error { return e.sendBuffer() })
					log.Printf("[WARN] email flush failed, %s", err)
					return
				}
			}
		}()

	}
}

// Flush sends everything left in buffer to server
func (e *Email) Flush() error {
	err := e.synced(func() error {
		err := e.sendBuffer()
		e.buffer = e.buffer[0:0]
		return err
	})
	return errors.Wrapf(err, "failed to flush emails")
}

func (e *Email) String() string {
	return fmt.Sprintf("email: %s using '%s'@'%s':%d", e.From, e.Username, e.Host, e.Port)
}

//buildMessageFromRequest generates email message based on request
func (e *Email) buildMessageFromRequest(req request, to string) (string, error) {
	subject := "New comment"
	if req.comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.comment.PostTitle)
	}
	msg := bytes.Buffer{}
	err := e.template.Execute(&msg, tmplData{
		req.comment.User.Name,
		req.parent.User.Name,
		req.comment.Orig,
		req.comment.Locator.URL + uiNav + req.comment.ID,
		req.comment.PostTitle,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build message from request")
	}
	return e.buildMessage(subject, msg.String(), to, "text/html"), nil
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

func (e *Email) synced(fn func() error) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	return fn()
}

var defaultEmailTemplate = `{{.From}}{{if .To}} → {{.To}}{{end}}

{{.Orig}}

↦ <a href="{{.Link}}">{{if .PostTitle}}{{.PostTitle}}{{else}}original comment{{end}}</a>
`
