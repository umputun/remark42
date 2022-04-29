package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
	"github.com/go-pkgz/repeater"
	"github.com/hashicorp/go-multierror"

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

// Email implements notify.Destination for email
type Email struct {
	*ntf.Email

	EmailParams
	msgTmpl    *template.Template // parsed request message template
	verifyTmpl *template.Template // parsed verification message template
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
func NewEmail(emailParams EmailParams, smtpParams ntf.SMTPParams) (*Email, error) {
	// set up Email emailParams
	if smtpParams.TimeOut <= 0 {
		smtpParams.TimeOut = defaultEmailTimeout
	}

	res := Email{Email: ntf.NewEmail(smtpParams), EmailParams: emailParams}

	if res.VerificationSubject == "" {
		res.VerificationSubject = defaultVerificationSubject
	}

	// initialize templates
	err := res.setTemplates()
	if err != nil {
		return nil, fmt.Errorf("can't set templates: %w", err)
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
		return fmt.Errorf("can't read message template: %w", err)
	}
	if verifyTmplFile, err = fs.ReadFile(e.VerificationTemplatePath); err != nil {
		return fmt.Errorf("can't read verification template: %w", err)
	}
	if e.msgTmpl, err = template.New("msgTmpl").Parse(string(msgTmplFile)); err != nil {
		return fmt.Errorf("can't parse message template: %w", err)
	}
	if e.verifyTmpl, err = template.New("verifyTmpl").Parse(string(verifyTmplFile)); err != nil {
		return fmt.Errorf("can't parse verification template: %w", err)
	}

	return nil
}

// Send email about comment reply to Request.Emails and Email.AdminEmails
// if they're set.
// Thread safe
func (e *Email) Send(ctx context.Context, req Request) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("sending email messages about comment %q aborted due to canceled context", req.Comment.ID)
	default:
	}

	result := new(multierror.Error)

	for _, email := range req.Emails {
		err := e.buildAndSendMessage(ctx, req, email, false)
		if err != nil {
			result = multierror.Append(fmt.Errorf("problem sending user email notification to %q: %w", email, err))
		}
	}

	for _, email := range e.AdminEmails {
		err := e.buildAndSendMessage(ctx, req, email, true)
		if err != nil {
			result = multierror.Append(fmt.Errorf("problem sending admin email notification to %q: %w", email, err))
		}
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
			return e.Email.Send(
				ctx,
				fmt.Sprintf("mailto:%s?from=%s&unsubscribeLink=%s&subject=%s",
					email,
					e.From,
					url.QueryEscape(msg.unsubscribeLink),
					url.QueryEscape(msg.subject),
				),
				msg.body,
			)
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
		return fmt.Errorf("sending message to %q aborted due to canceled context", req.User)
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
			return e.Email.Send(
				ctx,
				fmt.Sprintf("mailto:%s?from=%s&subject=%s",
					req.Email,
					e.From,
					url.QueryEscape(e.VerificationSubject),
				),
				msg,
			)
		})
}

// buildVerificationMessage generates verification email message based on given input
func (e *Email) buildVerificationMessage(user, email, token, site string) (string, error) {
	msg := bytes.Buffer{}
	err := e.verifyTmpl.Execute(&msg, verifyTmplData{
		User:         user,
		Token:        token,
		Email:        email,
		Site:         site,
		SubscribeURL: e.SubscribeURL,
	})
	if err != nil {
		return "", fmt.Errorf("error executing template to build verification message: %w", err)
	}
	return msg.String(), nil
}

type commentMessage struct {
	subject         string
	body            string
	unsubscribeLink string
}

// buildMessageFromRequest generates email message based on Request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req Request, email string, forAdmin bool) (commentMessage, error) {
	subject := "New reply to your comment"
	if forAdmin {
		subject = "New comment to your site"
	}
	if req.Comment.PostTitle != "" {
		subject += fmt.Sprintf(" for %q", req.Comment.PostTitle)
	}

	token, err := e.TokenGenFn(req.parent.User.ID, email, req.Comment.Locator.SiteID)
	if err != nil {
		return commentMessage{}, fmt.Errorf("error creating token for unsubscribe link: %w", err)
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
		return commentMessage{}, fmt.Errorf("error executing template to build comment reply message: %w", err)
	}
	return commentMessage{
		subject:         subject,
		body:            msg.String(),
		unsubscribeLink: unsubscribeLink,
	}, err
}
