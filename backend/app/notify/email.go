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
	Host                 string        // SMTP host
	Port                 int           // SMTP port
	TLS                  bool          // TLS auth
	From                 string        // From email field
	Username             string        // user name
	Password             string        // password
	TimeOut              time.Duration // TLS connection timeout
	MsgTemplate          string        // request message template
	VerificationTemplate string        // verification message template
	BufferSize           int           // email send buffer size
	FlushDuration        time.Duration // maximum time after which email will me sent, 30s by default
}

// Email implements notify.Destination for email
type Email struct {
	EmailParams
	smtpClient // embedded into Email for test purposes: not set in production code, used in autoFlush initialisation

	msgTmpl    *template.Template // parsed request message template
	verifyTmpl *template.Template // parsed verification message template
	submit     chan emailMessage  // unbuffered channel for email sending
	once       sync.Once
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

// msgTmplData store data for message from request template execution
type msgTmplData struct {
	From      string
	To        string
	Orig      string
	Link      string
	PostTitle string
}

// verifyTmplData store data for verification message template execution
type verifyTmplData struct {
	User  string
	Email string
	Token string
	Site  string
}

const defaultEmailTimeout = 10 * time.Second
const defaultFlushDuration = time.Second * 30
const defaultEmailTemplate = `{{.From}}{{if .To}} → {{.To}}{{end}}

{{.Orig}}

↦ <a href="{{.Link}}">{{if .PostTitle}}{{.PostTitle}}{{else}}original comment{{end}}</a>
`

const defaultEmailVerificationTemplate = `Confirmation for {{.User}} {{.Email}}, site {{.Site}}

Token: {{.Token}}
`

//NewEmail makes new Email object, returns it even in case of problems
// (e.MsgTemplate parsing error or error while testing smtp connection by credentials provided in params)
func NewEmail(params EmailParams) (*Email, error) {
	var err error
	res := Email{EmailParams: params}
	if res.TimeOut <= 0 {
		res.TimeOut = defaultEmailTimeout
	}
	if res.FlushDuration <= 0 {
		res.FlushDuration = defaultFlushDuration
	}
	if res.BufferSize <= 0 {
		res.BufferSize = 1
	}
	if res.MsgTemplate == "" {
		res.MsgTemplate = defaultEmailTemplate
	}
	if res.VerificationTemplate == "" {
		res.VerificationTemplate = defaultEmailVerificationTemplate
	}
	// unbuffered send channel for sending messages to autoFlush goroutine
	res.submit = make(chan emailMessage)

	log.Printf("[DEBUG] create new email notifier for server %s with user %s, timeout=%s",
		res.Host, res.Username, res.TimeOut)

	// initialise templates
	res.msgTmpl, err = template.New("messageFromRequest").Parse(res.MsgTemplate)
	if err != nil {
		return &res, errors.Wrapf(err, "can't parse message template")
	}
	res.verifyTmpl, err = template.New("messageFromRequest").Parse(res.VerificationTemplate)
	if err != nil {
		return &res, errors.Wrapf(err, "can't parse verification template")
	}

	// establish test connection
	testSMTPClient, err := res.client()
	if err != nil {
		return &res, errors.Wrapf(err, "can't establish test connection")
	}
	if err = testSMTPClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", res.Host, res.Port, err)
	}
	if err = testSMTPClient.Close(); err != nil {
		log.Printf("[WARN] can't close smtp connection, %v", err)
	}
	return &res, err
}

// Send email from request to request.parentUserEmail if it's set, otherwise do nothing and return nil, thread safe
// do not returns sending error, except following cases:
// 1. (likely impossible) template execution error from email message creation from request
// 2. message dropped without sending in case of closed ctx
func (e *Email) Send(ctx context.Context, req request) error {
	if req.parentUserEmail == "" || req.parent.User == req.comment.User {
		// don't send anything if there is no email to send information to
		// or if user replied to his own comment
		return nil
	}
	// start auto flush once, as this is the first moment we see the context from caller
	e.once.Do(func() {
		// e.smtpClient initialised only in tests
		go e.autoFlush(ctx, e.smtpClient)
	})
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.comment.ID)
	msg, err := e.buildMessageFromRequest(req, req.parentUserEmail)
	if err != nil {
		return err
	}
	select {
	case e.submit <- emailMessage{msg, req.parentUserEmail}:
		return nil
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", req.parentUserEmail)
	}
}

// SendVerification sends email verification
// do not returns sending error, except following cases:
// 1. (likely impossible) template execution error from verify message template execution
// 2. message dropped without sending in case of closed ctx
func (e *Email) SendVerification(ctx context.Context, req VerificationRequest) error {
	// start auto flush once, as this is the first moment we see the context from caller
	e.once.Do(func() {
		// e.smtpClient initialised only in tests
		go e.autoFlush(ctx, e.smtpClient)
	})
	log.Printf("[DEBUG] send verification via %s, user %s", e, req.User)
	msg, err := e.buildVerificationMessage(req.User, req.Email, req.Token, req.Locator.SiteID)
	if err != nil {
		return err
	}

	select {
	case e.submit <- emailMessage{msg, req.Email}:
		return nil
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", req.Email)
	}
}

// autoFlush flushes all in-fly records in case of:
// 1. buffer of size e.BufferSize + 1 is filled
// 2. there are no new messages for e.FlushDuration and buffer is not empty
// 3. ctx is closed
func (e *Email) autoFlush(ctx context.Context, smtpClient smtpClient) {
	lastWriteTime := time.Time{}
	msgBuffer := make([]emailMessage, 0, e.BufferSize+1)
	ticker := time.NewTicker(e.FlushDuration)
	for {
		select {
		case m := <-e.submit:
			lastWriteTime = time.Now()
			msgBuffer = append(msgBuffer, m)
			if len(msgBuffer) >= e.BufferSize {
				if err := e.sendBuffer(ctx, smtpClient, msgBuffer); err != nil {
					log.Printf("[WARN] notification email(s) send failed, %s", err)
				}
				msgBuffer = msgBuffer[0:0]
			}
		case <-ticker.C:
			shouldFlush := time.Now().After(lastWriteTime.Add(e.FlushDuration)) && len(msgBuffer) > 0
			if shouldFlush {
				if err := e.sendBuffer(ctx, smtpClient, msgBuffer); err != nil {
					log.Printf("[WARN] notification email(s) send failed, %s", err)
				}
				msgBuffer = msgBuffer[0:0]
			}
		case <-ctx.Done():
			// e.sendBuffer is context-aware and won't send messages, but will produce meaningful error message
			if err := e.sendBuffer(ctx, smtpClient, msgBuffer); err != nil {
				log.Printf("[WARN] notification email(s) send failed, %s", err)
			}
			return
		}
	}
}

// sendBuffer sends all collected messages to server, closing the connection after finishing
// in case smtpClient is not provided, establish connection using e.client()
// thread safe in case smtpClient is unique per thread or nil
func (e *Email) sendBuffer(ctx context.Context, smtpClient smtpClient, sendBuffer []emailMessage) (err error) {
	if len(sendBuffer) == 0 {
		return nil
	}

	if smtpClient == nil {
		smtpClient, err = e.client()
		if err != nil {
			return errors.Wrap(err, "failed to make smtp client")
		}
	}

	var cumulativeErrors []string

	for _, m := range sendBuffer {
		err := repeater.NewDefault(5, time.Millisecond*250).Do(ctx, func() error { return e.sendEmail(smtpClient, m) })
		if err != nil {
			cumulativeErrors = append(cumulativeErrors, errors.Wrapf(err, "can't send message to %s", m.to).Error())
		}
	}

	if err := smtpClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
		if err := smtpClient.Close(); err != nil {
			log.Printf("[WARN] can't close smtp connection, %v", err)
			cumulativeErrors = append(cumulativeErrors, err.Error())
		}
	}
	if len(cumulativeErrors) > 0 {
		err = fmt.Errorf(strings.Join(cumulativeErrors, "\n"))
	}
	return errors.Wrapf(err, "problems with sending messages")
}

// sendEmail sends message prepared by e.buildMessage to net/smtp.Client with established connection, thread unsafe
func (e *Email) sendEmail(smtpClient smtpClient, m emailMessage) error {
	if smtpClient == nil {
		return errors.New("sendEmail called without smtpClient set")
	}
	if err := smtpClient.Mail(e.From); err != nil {
		return errors.Wrapf(err, "bad from address %q", e.From)
	}
	if err := smtpClient.Rcpt(m.to); err != nil {
		return errors.Wrapf(err, "bad to address %q", m.to)
	}

	writer, err := smtpClient.Data()
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

//buildMessageFromRequest generates email message based on request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req request, to string) (string, error) {
	subject := "New comment"
	if req.comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.comment.PostTitle)
	}
	msg := bytes.Buffer{}
	err := e.msgTmpl.Execute(&msg, msgTmplData{
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

//buildVerificationMessage generates verification email message based on given input
func (e *Email) buildVerificationMessage(user, address, token, site string) (string, error) {
	subject := "Email verification"
	msg := bytes.Buffer{}
	err := e.verifyTmpl.Execute(&msg, verifyTmplData{user, address, token, site})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build verifying message from request")
	}
	return e.buildMessage(subject, msg.String(), address, "text/html"), nil
}

//buildMessage generates email message to send with using net/smtp.Data()
func (e *Email) buildMessage(subject, body, to, contentType string) (message string) {
	message += fmt.Sprintf("From: %s\n", e.From)
	message += fmt.Sprintf("To: %s\n", to)
	message += fmt.Sprintf("Subject: %s\n", subject)
	if contentType != "" {
		message += fmt.Sprintf("MIME-version: 1.0;\nContent-Type: %s; charset=\"UTF-8\";\n", contentType)
	}
	message += "\n" + body
	return message
}

// client establish connection with smtp server using credentials in e.EmailParams
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

// String representation of Email object
func (e *Email) String() string {
	return fmt.Sprintf("email: %s using '%s'@'%s':%d", e.From, e.Username, e.Host, e.Port)
}