package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/templates"
)

// EmailParams contain settings for email notifications
type EmailParams struct {
	From                     string   // from email address
	AdminEmails              []string // administrator emails to send copy of comment notification to
	MsgTemplatePath          string   // path to request message template
	VerificationSubject      string   // verification message sub
	VerificationTemplatePath string   // path to verification template
	SubscribeURL             string   // full subscribe handler URL
	UnsubscribeURL           string   // full unsubscribe handler URL

	TokenGenFn func(userID, email, site string) (string, error) // Unsubscribe token generation function
}

// SMTPParams contain settings for smtp server connection
type SMTPParams struct {
	Host     string        // SMTP host
	Port     int           // SMTP port
	TLS      bool          // TLS auth
	Username string        // user name
	Password string        // password
	TimeOut  time.Duration // TCP connection timeout
}

// Email implements notify.Destination for email
type Email struct {
	EmailParams
	SMTPParams

	smtp       smtpClientCreator
	msgTmpl    *template.Template // parsed request message template
	verifyTmpl *template.Template // parsed verification message template
}

// default email client implementation
type emailClient struct{ smtpClientCreator }

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
	Create(SMTPParams) (smtpClient, error)
}

type emailMessage struct {
	from    string
	to      string
	message string
}

// msgTmplData store data for message from request template execution
type msgTmplData struct {
	UserName          string
	UserPicture       string
	CommentText       string
	CommentLink       string
	CommentDate       time.Time
	ParentUserName    string
	ParentUserPicture string
	ParentCommentText string
	ParentCommentLink string
	ParentCommentDate time.Time
	PostTitle         string
	Email             string
	UnsubscribeLink   string
	ForAdmin          bool
}

// verifyTmplData store data for verification message template execution
type verifyTmplData struct {
	User         string
	Token        string
	Email        string
	Site         string
	SubscribeURL string
}

const (
	defaultVerificationSubject           = "Email verification"
	defaultEmailTimeout                  = 10 * time.Second
	defaultEmailTemplatePath             = "email_reply.html.tmpl"
	defaultEmailVerificationTemplatePath = "email_confirmation_subscription.html.tmpl"
)

// NewEmail makes new Email object, returns error in case of e.MsgTemplate or e.VerificationTemplate parsing error
func NewEmail(emailParams EmailParams, smtpParams SMTPParams) (*Email, error) {
	// set up Email emailParams
	res := Email{EmailParams: emailParams}
	res.smtp = &emailClient{}
	res.SMTPParams = smtpParams
	if res.TimeOut <= 0 {
		res.TimeOut = defaultEmailTimeout
	}

	if res.VerificationSubject == "" {
		res.VerificationSubject = defaultVerificationSubject
	}

	// initialize templates
	err := res.setTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "can't set templates")
	}

	log.Printf("[DEBUG] Create new email notifier for server %s with user %s, timeout=%s",
		res.Host, res.Username, res.TimeOut)

	return &res, nil
}

func (e *Email) setTemplates() error {
	var err error
	var msgTmplFile, verifyTmplFile []byte
	fs := templates.NewFS()

	if e.VerificationTemplatePath == "" {
		e.VerificationTemplatePath = defaultEmailVerificationTemplatePath
	}

	if e.MsgTemplatePath == "" {
		e.MsgTemplatePath = defaultEmailTemplatePath
	}

	if msgTmplFile, err = fs.ReadFile(e.MsgTemplatePath); err != nil {
		return errors.Wrapf(err, "can't read message template")
	}
	if verifyTmplFile, err = fs.ReadFile(e.VerificationTemplatePath); err != nil {
		return errors.Wrapf(err, "can't read verification template")
	}
	if e.msgTmpl, err = template.New("msgTmpl").Parse(string(msgTmplFile)); err != nil {
		return errors.Wrapf(err, "can't parse message template")
	}
	if e.verifyTmpl, err = template.New("verifyTmpl").Parse(string(verifyTmplFile)); err != nil {
		return errors.Wrapf(err, "can't parse verification template")
	}

	return nil
}

// Send email about comment reply to Request.Emails and Email.AdminEmails
// if they're set.
// Thread safe
func (e *Email) Send(ctx context.Context, req Request) error {
	select {
	case <-ctx.Done():
		return errors.Errorf("sending email messages about comment %q aborted due to canceled context", req.Comment.ID)
	default:
	}

	result := new(multierror.Error)

	for _, email := range req.Emails {
		err := e.buildAndSendMessage(ctx, req, email, false)
		result = multierror.Append(errors.Wrapf(err, "problem sending user email notification to %q", email))
	}

	for _, email := range e.AdminEmails {
		err := e.buildAndSendMessage(ctx, req, email, true)
		result = multierror.Append(errors.Wrapf(err, "problem sending admin email notification to %q", email))
	}

	return result.ErrorOrNil()
}

func (e *Email) buildAndSendMessage(ctx context.Context, req Request, email string, forAdmin bool) error {
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.Comment.ID)
	msg, err := e.buildMessageFromRequest(req, email, forAdmin)
	if err != nil {
		return err
	}

	return repeater.NewDefault(5, time.Millisecond*250).Do(
		ctx,
		func() error {
			return e.sendMessage(emailMessage{from: e.From, to: email, message: msg})
		})
}

// SendVerification email verification VerificationRequest.Email if it's set.
// Thread safe
func (e *Email) SendVerification(ctx context.Context, req VerificationRequest) error {
	if req.Email == "" {
		// this means we can't send this request via Email
		return nil
	}
	select {
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", req.User)
	default:
	}

	log.Printf("[DEBUG] send verification via %s, user %s", e, req.User)
	msg, err := e.buildVerificationMessage(req.User, req.Email, req.Token, req.SiteID)
	if err != nil {
		return err
	}

	return repeater.NewDefault(5, time.Millisecond*250).Do(
		ctx,
		func() error {
			return e.sendMessage(emailMessage{from: e.From, to: req.Email, message: msg})
		})
}

// buildVerificationMessage generates verification email message based on given input
func (e *Email) buildVerificationMessage(user, email, token, site string) (string, error) {
	subject := e.VerificationSubject
	msg := bytes.Buffer{}
	err := e.verifyTmpl.Execute(&msg, verifyTmplData{
		User:         user,
		Token:        token,
		Email:        email,
		Site:         site,
		SubscribeURL: e.SubscribeURL,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build verification message")
	}
	return e.buildMessage(subject, msg.String(), email, "text/html", "")
}

// buildMessageFromRequest generates email message based on Request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req Request, email string, forAdmin bool) (string, error) {
	subject := "New reply to your comment"
	if forAdmin {
		subject = "New comment to your site"
	}
	if req.Comment.PostTitle != "" {
		subject += fmt.Sprintf(" for %q", req.Comment.PostTitle)
	}

	token, err := e.TokenGenFn(req.parent.User.ID, email, req.Comment.Locator.SiteID)
	if err != nil {
		return "", errors.Wrapf(err, "error creating token for unsubscribe link")
	}
	unsubscribeLink := e.UnsubscribeURL + "?site=" + req.Comment.Locator.SiteID + "&tkn=" + token
	if forAdmin {
		unsubscribeLink = ""
	}

	commentURLPrefix := req.Comment.Locator.URL + uiNav
	msg := bytes.Buffer{}
	tmplData := msgTmplData{
		UserName:        req.Comment.User.Name,
		UserPicture:     req.Comment.User.Picture,
		CommentText:     req.Comment.Text,
		CommentLink:     commentURLPrefix + req.Comment.ID,
		CommentDate:     req.Comment.Timestamp,
		PostTitle:       req.Comment.PostTitle,
		Email:           email,
		UnsubscribeLink: unsubscribeLink,
		ForAdmin:        forAdmin,
	}
	// in case of message to admin, parent message might be empty
	if req.Comment.ParentID != "" {
		tmplData.ParentUserName = req.parent.User.Name
		tmplData.ParentUserPicture = req.parent.User.Picture
		tmplData.ParentCommentText = req.parent.Text
		tmplData.ParentCommentLink = commentURLPrefix + req.parent.ID
		tmplData.ParentCommentDate = req.parent.Timestamp
	}
	err = e.msgTmpl.Execute(&msg, tmplData)
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build comment reply message")
	}
	return e.buildMessage(subject, msg.String(), email, "text/html", unsubscribeLink)
}

// buildMessage generates email message to send using net/smtp.Data()
func (e *Email) buildMessage(subject, body, to, contentType, unsubscribeLink string) (message string, err error) {
	addHeader := func(msg, h, v string) string {
		msg += fmt.Sprintf("%s: %s\n", h, v)
		return msg
	}
	message = addHeader(message, "From", e.From)
	message = addHeader(message, "To", to)
	message = addHeader(message, "Subject", mime.BEncoding.Encode("utf-8", subject))
	message = addHeader(message, "Content-Transfer-Encoding", "quoted-printable")

	if contentType != "" {
		message = addHeader(message, "MIME-version", "1.0")
		message = addHeader(message, "Content-Type", contentType+`; charset="UTF-8"`)
	}

	if unsubscribeLink != "" {
		// https://support.google.com/mail/answer/81126 -> "Include option to unsubscribe"
		message = addHeader(message, "List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
		message = addHeader(message, "List-Unsubscribe", "<"+unsubscribeLink+">")
	}

	message = addHeader(message, "Date", time.Now().Format(time.RFC1123Z))

	buff := &bytes.Buffer{}
	qp := quotedprintable.NewWriter(buff)
	if _, err := qp.Write([]byte(body)); err != nil {
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

// sendMessage sends messages to server in a new connection, closing the connection after finishing.
// Thread safe.
func (e *Email) sendMessage(m emailMessage) error {
	if e.smtp == nil {
		return errors.New("sendMessage called without client set")
	}
	client, err := e.smtp.Create(e.SMTPParams)
	if err != nil {
		return errors.Wrap(err, "failed to make smtp Create")
	}

	defer func() {
		if err = client.Quit(); err != nil {
			log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
			if err = client.Close(); err != nil {
				log.Printf("[WARN] can't close smtp connection, %v", err)
			}
		}
	}()

	if err = client.Mail(m.from); err != nil {
		return errors.Wrapf(err, "bad from address %q", m.from)
	}
	if err = client.Rcpt(m.to); err != nil {
		return errors.Wrapf(err, "bad to address %q", m.to)
	}

	writer, err := client.Data()
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

// String representation of Email object
func (e *Email) String() string {
	return fmt.Sprintf("email: from %q with username '%s' at server %s:%d", e.From, e.Username, e.Host, e.Port)
}

// Create establish SMTP connection with server using credentials in smtpClientWithCreator.SMTPParams
// and returns pointer to it. Thread safe.
func (s *emailClient) Create(params SMTPParams) (smtpClient, error) {
	authenticate := func(c *smtp.Client) error {
		if params.Username == "" || params.Password == "" {
			return nil
		}
		auth := smtp.PlainAuth("", params.Username, params.Password, params.Host)
		if err := c.Auth(auth); err != nil {
			return errors.Wrapf(err, "failed to auth to smtp %s:%d", params.Host, params.Port)
		}
		return nil
	}

	var c *smtp.Client
	srvAddress := fmt.Sprintf("%s:%d", params.Host, params.Port)
	if params.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         params.Host,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err := tls.Dial("tcp", srvAddress, tlsConf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial smtp tls to %s", srvAddress)
		}
		if c, err = smtp.NewClient(conn, params.Host); err != nil {
			return nil, errors.Wrapf(err, "failed to make smtp client for %s", srvAddress)
		}
		return c, authenticate(c)
	}

	conn, err := net.DialTimeout("tcp", srvAddress, params.TimeOut)
	if err != nil {
		return nil, errors.Wrapf(err, "timeout connecting to %s", srvAddress)
	}

	c, err = smtp.NewClient(conn, params.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	return c, authenticate(c)
}
