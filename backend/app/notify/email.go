package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// EmailParams contain settings for email notifications
type EmailParams struct {
	From                 string // from email address
	MsgTemplate          string // request message template
	VerificationSubject  string // verification message subject
	VerificationTemplate string // verification message template
	UnsubscribeURL       string // full unsubscribe handler URL

	TokenGenFn func(userID, email, site string) (string, error) // Unsubscribe token generation function
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

// Email implements notify.Destination for email
type Email struct {
	EmailParams
	SmtpParams

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
	Create(SmtpParams) (smtpClient, error)
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
	CommentDate       string
	ParentUserName    string
	ParentUserPicture string
	ParentCommentText string
	ParentCommentLink string
	PostTitle         string
	Email             string
	UnsubscribeLink   string
}

// verifyTmplData store data for verification message template execution
type verifyTmplData struct {
	User  string
	Token string
	Email string
	Site  string
}

const (
	defaultVerificationSubject = "Email verification"
	defaultEmailTimeout        = 10 * time.Second
	defaultEmailTemplate       = `<!DOCTYPE html>
<html>
<head>
	<meta name="viewport" content="width=device-width" />
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	<style type="text/css">
		img {
			max-width: 100%;
			max-height: 250px;
			margin: 5px 0;
			display: block;
			color: #000;
		}
		a {
			text-decoration: none;
			color: #0aa;
		}
		p {
			margin: 0 0 12px;
		}
	</style>
</head>
<!-- Some of blocks on this page have color: #000 because GMail can wrap block in his own tags which can change text color -->
<body>
	<div style="font-family: Helvetica, Arial, sans-serif; font-size: 18px; width: 100%; max-width: 640px; margin: auto;">
		<h1 style="text-align: center; position: relative; color: #4fbbd6; margin-top: 10px; margin-bottom: 10px;">Remark42</h1>
		<div style="font-size: 16px; text-align: center; margin-bottom: 10px; color:#000!important;">New reply from {{.UserName}} on your comment{{if .PostTitle}} to «{{.PostTitle}}»{{ end }}</div>
		<div style="background-color: #eee; padding: 15px 20px 20px 20px; border-radius: 3px;">
			<div>
				<div style="margin-bottom: 12px; line-height: 24px;">
					<img src="{{.ParentUserPicture}}" style="width: 24px; height: 24px; float: left; margin: 0 8px 0 0; border-radius: 3px; background-color: #ccc;"/>
					<span style="font-size: 14px; font-weight: bold; color: #777">{{.ParentUserName}}</span>
					<span style="color: #999; font-size: 14px; margin: 0 8px;">{{.CommentDate}}</span>
					<a href="{{.ParentCommentLink}}" style="color: #0aa; font-size: 14px;"><b>Show</b></a>
				</div>
				<div style="font-size: 14px; color:#333!important; padding: 0 14px 0 2px; border-radius: 3px; line-height: 1.4;">
					{{.ParentCommentText}}
				</div>
			</div>
			<div style="padding-left: 20px; border-left: 1px dotted rgba(0,0,0,0.15); margin-top: 15px; padding-top: 5px; line-height: 24px;">
				<div style="margin-bottom: 8px;">
					<img src="{{.UserPicture}}" style="width: 24px; height: 24px; float: left; margin: 0 8px 0 0; border-radius: 3px; background-color: #ccc;"/>
					<div style="float: left; font-size: 14px; font-weight: bold; color: #777">
						{{.UserName}}
					</div>
					<div style="color: #999; font-size: 14px; margin: 0 8px; float: left;">
						{{.CommentDate}}
					</div>
					<a href="{{.CommentLink}}" style="color: #0aa; font-size: 14px;"><b>Reply</b></a>
				</div>
				<div style="font-size: 16px; background-color: #fff; color:#000!important; padding: 14px 14px 2px 14px; border-radius: 3px; line-height: 1.4;">
					{{.CommentText}}
				</div>
			</div>
		</div>
		<div style="text-align: center; font-size: 14px; margin-top: 32px;">
			<i style="color: #000!important;">Sent to <a style="color:inherit; text-decoration: none" href="mailto:{{.Email}}">{{.Email}}</a> for {{.ParentUserName}}</i>
			<div style="margin: auto; width: 150px; border-top: 1px solid rgba(0, 0, 0, 0.15); padding-top: 15px; margin-top: 15px;">
				<a style="color: #0aa;" href="{{.UnsubscribeLink}}">Unsubscribe</a>
				<!-- This is hack for remove collapser in Gmail which can collapse end of the message -->
				<div style="opacity: 0;">[{{.CommentDate}}]</div>
			</div>
		</div>
	</div>
</body>
</html>
`
	defaultEmailVerificationTemplate = `<!DOCTYPE html>
<html>
<head>
	<meta name="viewport" content="width=device-width" />
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head>
<body>
	<!-- Some of blocks on this page have color: #000 because GMail can wrap block in his own tags which can change text color -->
	<div style="text-align: center; font-family: Helvetica, Arial, sans-serif; font-size: 18px;">
		<h1 style="position: relative; color: #4fbbd6; margin-top: 0.2em;">Remark42</h1>
		<p style="position: relative; max-width: 20em; margin: 0 auto 1em auto; line-height: 1.4em; color:#000!important;">Confirmation for <b>{{.User}}</b> on site <b>{{.Site}}</b></p>
		<div style="background-color: #eee; max-width: 20em; margin: 0 auto; border-radius: 0.4em; padding: 0.5em;">
			<p style="position: relative; margin: 0 0 0.5em 0;color:#000!important;">TOKEN</p>
			<p style="position: relative; font-size: 0.7em; opacity: 0.8;"><i style="color:#000!important;">Copy and paste this text into “token” field on comments page</i></p>
			<p style="position: relative; font-family: monospace; background-color: #fff; margin: 0; padding: 0.5em; word-break: break-all; text-align: left; border-radius: 0.2em; -webkit-user-select: all; user-select: all;">{{.Token}}</p>
		</div>
		<p style="position: relative; margin-top: 2em; font-size: 0.8em; opacity: 0.8;"><i style="color:#000!important;">Sent to {{.Email}}</i></p>
	</div>
</body>
</html>
`
)

// NewEmail makes new Email object, returns error in case of e.MsgTemplate or e.VerificationTemplate parsing error
func NewEmail(emailParams EmailParams, smtpParams SmtpParams) (*Email, error) {
	// set up Email emailParams
	res := Email{EmailParams: emailParams}
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

	log.Printf("[DEBUG] Create new email notifier for server %s with user %s, timeout=%s",
		res.Host, res.Username, res.TimeOut)

	// initialise templates
	var err error
	if res.msgTmpl, err = template.New("messageFromRequest").Parse(res.MsgTemplate); err != nil {
		return nil, errors.Wrapf(err, "can't parse message template")
	}
	if res.verifyTmpl, err = template.New("messageFromRequest").Parse(res.VerificationTemplate); err != nil {
		return nil, errors.Wrapf(err, "can't parse verification template")
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
	select {
	case <-ctx.Done():
		return errors.Errorf("sending message to %q aborted due to canceled context", req.Email)
	default:
	}
	var msg string

	if req.Verification.Token != "" {
		log.Printf("[DEBUG] send verification via %s, user %s", e, req.Verification.User)
		msg, err = e.buildVerificationMessage(req.Verification.User, req.Email, req.Verification.Token, req.Verification.SiteID)
		if err != nil {
			return err
		}
	}

	if req.Comment.ID != "" {
		if req.parent.User.ID == req.Comment.User.ID {
			// don't send anything if if user replied to their own comment
			return nil
		}
		log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.Comment.ID)
		msg, err = e.buildMessageFromRequest(req)
		if err != nil {
			return err
		}
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
		User:  user,
		Token: token,
		Email: email,
		Site:  site,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build verification message")
	}
	return e.buildMessage(subject, msg.String(), email, "text/html", "")
}

// buildMessageFromRequest generates email message based on Request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req Request) (string, error) {
	subject := "New reply to your comment"
	if req.Comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.Comment.PostTitle)
	}
	token, err := e.TokenGenFn(req.parent.User.ID, req.Email, req.Comment.Locator.SiteID)
	unsubscribeLink := e.UnsubscribeURL + "?site=" + req.Comment.Locator.SiteID + "&tkn=" + token
	if err != nil {
		return "", errors.Wrapf(err, "error creating token for unsubscribe link")
	}
	commentUrlPrefix := req.Comment.Locator.URL + uiNav
	msg := bytes.Buffer{}
	err = e.msgTmpl.Execute(&msg, msgTmplData{
		UserName:          req.Comment.User.Name,
		UserPicture:       req.Comment.User.Picture,
		CommentText:       req.Comment.Text,
		CommentLink:       commentUrlPrefix + req.Comment.ID,
		CommentDate:       req.Comment.Timestamp.Format("02.01.2006 at 15:04"),
		ParentUserName:    req.parent.User.Name,
		ParentUserPicture: req.parent.User.Picture,
		ParentCommentText: req.parent.Text,
		ParentCommentLink: req.parent.Locator.URL + uiNav + req.parent.ID,
		PostTitle:         req.Comment.PostTitle,
		Email:             req.Email,
		UnsubscribeLink:   unsubscribeLink,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build comment reply message")
	}
	return e.buildMessage(subject, msg.String(), req.Email, "text/html", unsubscribeLink)
}

// buildMessage generates email message to send using net/smtp.Data()
func (e *Email) buildMessage(subject, body, to, contentType, unsubscribeLink string) (message string, err error) {
	addHeader := func(msg, h, v string) string {
		msg += fmt.Sprintf("%s: %s\n", h, v)
		return msg
	}
	message = addHeader(message, "From", e.From)
	message = addHeader(message, "To", to)
	message = addHeader(message, "Subject", subject)
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
	defer qp.Close()
	m := buff.String()
	message += "\n" + m
	return message, nil
}

// sendMessage sends messages to server in a new connection, closing the connection after finishing.
// Thread safe.
func (e *Email) sendMessage(m emailMessage) error {
	if e.smtp == nil {
		return errors.New("sendMessage called without smtpClient set")
	}
	smtpClient, err := e.smtp.Create(e.SmtpParams)
	if err != nil {
		return errors.Wrap(err, "failed to make smtp Create")
	}

	defer func() {
		if err := smtpClient.Quit(); err != nil {
			log.Printf("[WARN] failed to send quit command to %s:%d, %v", e.Host, e.Port, err)
			if err := smtpClient.Close(); err != nil {
				log.Printf("[WARN] can't close smtp connection, %v", err)
			}
		}
	}()

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

// String representation of Email object
func (e *Email) String() string {
	return fmt.Sprintf("email: from %q with username '%s' at server %s:%d", e.From, e.Username, e.Host, e.Port)
}

// Create establish SMTP connection with server using credentials in smtpClientWithCreator.SmtpParams
// and returns pointer to it. Thread safe.
func (s *emailClient) Create(params SmtpParams) (smtpClient, error) {
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

	c, err = smtp.NewClient(conn, srvAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	return c, authenticate(c)
}
