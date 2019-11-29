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
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// EmailParams contain settings for email notifications
type EmailParams struct {
	From                 string        // From email field
	MsgTemplate          string        // request message template
	VerificationSubject  string        // verification message subject
	VerificationTemplate string        // verification message template
	BufferSize           int           // email send buffer size
	FlushDuration        time.Duration // maximum time after which email will me sent, 30s by default
}

// Email implements notify.Destination for email
type Email struct {
	EmailParams
	SmtpParams

	smtp       smtpClientWithCreator
	msgTmpl    *template.Template // parsed request message template
	verifyTmpl *template.Template // parsed verification message template
	submit     chan emailMessage  // unbuffered channel for email sending
	once       sync.Once
}

// SmtpParams contain settings for smtp server connection
type SmtpParams struct {
	Host     string        // SMTP host
	Port     int           // SMTP port
	TLS      bool          // TLS auth
	Username string        // user name
	Password string        // password
	TimeOut  time.Duration // TCP connection timeout
}

// default email client implementation
type emailClient struct{ smtpClientWithCreator }

type smtpClientWithCreator interface {
	smtpClientCreator
	smtpClient
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

// smtpClientCreator interface defines function for creating new smtpClients
type smtpClientCreator interface {
	Create(SmtpParams) (smtpClient, error)
}

type emailMessage struct {
	from    string
	to      string
	message string
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

const (
	defaultVerificationSubject = "Email verification"
	defaultEmailTimeout        = 10 * time.Second
	defaultFlushDuration       = time.Second * 30
	defaultEmailTemplate       = `{{.From}}{{if .To}} → {{.To}}{{end}}

{{.Orig}}

↦ <a href="{{.Link}}">{{if .PostTitle}}{{.PostTitle}}{{else}}original comment{{end}}</a>
`
	defaultEmailVerificationTemplate = `Confirmation for {{.User}} {{.Email}}, site {{.Site}}

Token: {{.Token}}
`
)

//NewEmail makes new Email object, returns it even in case of problems
// (e.MsgTemplate parsing error or error while testing smtp connection by credentials provided in emailParams)
func NewEmail(emailParams EmailParams, smtpParams SmtpParams) (*Email, error) {
	var err error
	// set up Email emailParams
	res := Email{EmailParams: emailParams}
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
	if res.VerificationSubject == "" {
		res.VerificationSubject = defaultVerificationSubject
	}

	// set up SMTP emailParams
	res.smtp = &emailClient{}
	res.SmtpParams = smtpParams
	if res.TimeOut <= 0 {
		res.TimeOut = defaultEmailTimeout
	}

	// unbuffered send channel for sending messages to autoFlush goroutine
	res.submit = make(chan emailMessage)

	log.Printf("[DEBUG] Create new email notifier for server %s with user %s, timeout=%s",
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
	testSmtpClient, err := res.smtp.Create(res.SmtpParams)
	if err != nil {
		return &res, errors.Wrapf(err, "can't establish test connection")
	}
	if err = testSmtpClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", res.Host, res.Port, err)
		if err = testSmtpClient.Close(); err != nil {
			return &res, errors.Wrapf(err, "can't close test smtp connection")
		}
	}
	return &res, err
}

// Send email about reply to Request.Email if it's set, otherwise do nothing and return nil, thread safe
// do not returns sending error, only following:
// 1. (likely impossible) template execution error from email message creation from Request
// 2. message dropped without sending in case of closed ctx
func (e *Email) Send(ctx context.Context, req Request) (err error) {
	if req.Email == "" {
		// this means we can't send this request via Email
		return nil
	}
	var msg string

	if req.Verification.Token != "" {
		log.Printf("[DEBUG] send verification via %s, user %s", e, req.Verification.User)
		msg, err = e.buildVerificationMessage(req.Verification.User, req.Email, req.Verification.Token, req.Verification.Locator.SiteID)
		if err != nil {
			return err
		}
	}

	if req.Comment.ID != "" {
		if req.parent.User == req.Comment.User {
			// don't send anything if if user replied to their own Comment
			return nil
		}
		log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.Comment.ID)
		msg, err = e.buildMessageFromRequest(req, req.Email)
		if err != nil {
			return err
		}
	}

	return e.submitEmailMessage(ctx, emailMessage{from: e.From, to: req.Email, message: msg})
}

// submitEmailMessage submits message to buffered sender and returns error only in case context is closed.
func (e *Email) submitEmailMessage(ctx context.Context, msg emailMessage) error {
	// start auto flush once, as this is the first moment we see the context from caller
	e.once.Do(func() {
		go e.autoFlush(ctx)
	})
	select {
	case e.submit <- msg:
		return nil
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", msg.to)
	}
}

//buildVerificationMessage generates verification email message based on given input
func (e *Email) buildVerificationMessage(user, address, token, site string) (string, error) {
	subject := e.VerificationSubject
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

//buildMessageFromRequest generates email message based on Request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req Request, to string) (string, error) {
	subject := "New comment"
	if req.Comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.Comment.PostTitle)
	}
	msg := bytes.Buffer{}
	err := e.msgTmpl.Execute(&msg, msgTmplData{
		req.Comment.User.Name,
		req.parent.User.Name,
		req.Comment.Orig,
		req.Comment.Locator.URL + uiNav + req.Comment.ID,
		req.Comment.PostTitle,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build message from request")
	}
	return e.buildMessage(subject, msg.String(), to, "text/html"), nil
}

// autoFlush flushes all in-fly records in case of:
// 1. buffer of size e.BufferSize + 1 is filled
// 2. there are no new messages for e.FlushDuration and buffer is not empty
// 3. ctx is closed
// This function is blocking and should be running in goroutine.
// Killed by canceling the provided context.
func (e *Email) autoFlush(ctx context.Context) {
	lastWriteTime := time.Time{}
	msgBuffer := make([]emailMessage, 0, e.BufferSize+1)
	ticker := time.NewTicker(e.FlushDuration)
	for {
		select {
		case m := <-e.submit:
			lastWriteTime = time.Now()
			msgBuffer = append(msgBuffer, m)
			if len(msgBuffer) >= e.BufferSize {
				if err := e.sendMessages(ctx, msgBuffer); err != nil {
					log.Printf("[WARN] notification email(s) send failed, %s", err)
				}
				msgBuffer = msgBuffer[0:0]
			}
		case <-ticker.C:
			shouldFlush := time.Now().After(lastWriteTime.Add(e.FlushDuration)) && len(msgBuffer) > 0
			if shouldFlush {
				if err := e.sendMessages(ctx, msgBuffer); err != nil {
					log.Printf("[WARN] notification email(s) send failed, %s", err)
				}
				msgBuffer = msgBuffer[0:0]
			}
		case <-ctx.Done():
			// e.sendMessages is context-aware and won't send messages, but will produce meaningful error message
			if err := e.sendMessages(ctx, msgBuffer); err != nil {
				log.Printf("[WARN] notification email(s) send failed, %s", err)
			}
			return
		}
	}
}

// sendMessages sends messages to server in a new connection, closing the connection after finishing.
// Thread safe.
func (e *Email) sendMessages(ctx context.Context, messages []emailMessage) error {
	if e.smtp == nil {
		return errors.New("sendMessages called without smtpClient set")
	}
	if len(messages) == 0 {
		return nil
	}

	smtpClient, err := e.smtp.Create(e.SmtpParams)
	if err != nil {
		return errors.Wrap(err, "failed to make smtp Create")
	}

	errs := new(multierror.Error)

	for _, m := range messages {
		err := repeater.NewDefault(5, time.Millisecond*250).Do(ctx, func() error { return sendEmail(m, smtpClient) })
		if err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "can't send message to %s", m.to))
		}
	}

	if err := smtpClient.Quit(); err != nil {
		log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
		if err := smtpClient.Close(); err != nil {
			log.Printf("[WARN] can't close smtp connection, %v", err)
			errs = multierror.Append(errs, err)
		}
	}
	return errors.Wrapf(errs.ErrorOrNil(), "problems with sending messages")
}

// String representation of Email object
func (e *Email) String() string {
	return fmt.Sprintf("email: from %q using '%s'@'%s':%d", e.From, e.Username, e.Host, e.Port)
}

// Create establish SMTP connection with server using credentials in smtpClientWithCreator.SmtpParams
// and returns pointer to it. Thread safe.
func (s *emailClient) Create(params SmtpParams) (smtpClient, error) {
	var c *smtp.Client
	srvAddress := fmt.Sprintf("%s:%d", params.Host, params.Port)
	if params.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         params.Host,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, params.Host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, nil
	}

	conn, err := net.DialTimeout("tcp", srvAddress, params.TimeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	if params.Username != "" && params.Password != "" {
		auth := smtp.PlainAuth("", params.Username, params.Password, params.Host)
		if err := c.Auth(auth); err != nil {
			return nil, errors.Wrapf(err, "failed to auth to smtp %s:%d", params.Host, params.Port)
		}
	}

	return c, nil
}

// sendEmail to smtpClient by already established connection.
// Thread safe.
func sendEmail(m emailMessage, smtpClient smtpClient) error {
	if smtpClient == nil {
		return errors.New("send called without smtpClient set")
	}
	if err := smtpClient.Mail(m.from); err != nil {
		return errors.Wrapf(err, "bad from address %q", m.from)
	}
	if err := smtpClient.Rcpt(m.to); err != nil {
		return errors.Wrapf(err, "bad to address %q", m.to)
	}

	writer, err := smtpClient.Data()
	if err != nil {
		return errors.Wrap(err, "can't make email writer")
	}
	defer func() {
		if err = writer.Close(); err != nil {
			log.Printf("[WARN] can't close smtp body writer, %v", err)
		}
	}()

	buf := bytes.NewBufferString(m.message)
	if _, err = buf.WriteTo(writer); err != nil {
		return errors.Wrapf(err, "failed to send email body to %q", m.to)
	}
	return nil
}
