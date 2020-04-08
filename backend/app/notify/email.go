package notify

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
	"github.com/umputun/remark/backend/app/emailprovider"
)

// EmailParams contain settings for email notifications
type EmailParams struct {
	From                 string // from email address
	MsgTemplate          string // request message template
	VerificationSubject  string // verification message subject
	VerificationTemplate string // verification message template
	SubscribeURL         string // full subscribe handler URL
	UnsubscribeURL       string // full unsubscribe handler URL

	TokenGenFn func(userID, email, site string) (string, error) // Unsubscribe token generation function
}

// Email implements notify.Destination for email
type Email struct {
	EmailParams

	sender     emailprovider.EmailSender
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
	defaultVerificationSubject = "Email verification"
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
		blockquote {
			margin: 10px 0;
			padding: 12px 12px 1px 12px;
			background: rgba(255,255,255,.5)
		}
	</style>
</head>
<!-- Some of blocks on this page have color: #000 because GMail can wrap block in his own tags which can change text color -->
<body>
	<div style="font-family: Helvetica, Arial, sans-serif; font-size: 18px; width: 100%; max-width: 640px; margin: auto;">
		<h1 style="text-align: center; position: relative; color: #4fbbd6; margin-top: 10px; margin-bottom: 10px;">Remark42</h1>
        {{- if .ForAdmin}}
		<div style="font-size: 16px; text-align: center; margin-bottom: 10px; color:#000!important;">New comment from {{.UserName}} on your site {{if .PostTitle}} to «{{.PostTitle}}»{{ end }}</div>
        {{- else }}
		<div style="font-size: 16px; text-align: center; margin-bottom: 10px; color:#000!important;">New reply from {{.UserName}} on your comment{{if .PostTitle}} to «{{.PostTitle}}»{{ end }}</div>
        {{- end }}
		<div style="background-color: #eee; padding: 15px 20px 20px 20px; border-radius: 3px;">
            {{- if .ParentCommentText}}
			<div style="margin-bottom: 12px; line-height: 24px; word-break: break-all;">
				<img src="{{.ParentUserPicture}}" style="width: 24px; height: 24px; display: inline; vertical-align: middle; margin: 0 8px 0 0; border-radius: 3px; background-color: #ccc;"/>
				<span style="font-size: 14px; font-weight: bold; color: #777">{{.ParentUserName}}</span>
				<span style="color: #999; font-size: 14px; margin: 0 8px;">{{.ParentCommentDate.Format "02.01.2006 at 15:04"}}</span>
				<a href="{{.ParentCommentLink}}" style="color: #0aa; font-size: 14px;"><b>Show</b></a>
			</div>
			<div style="font-size: 14px; color:#333!important; padding: 0 14px 0 2px; border-radius: 3px; line-height: 1.4;">
				{{.ParentCommentText}}
			</div>
            {{- end }}
			<div style="padding-left: 20px; border-left: 1px dotted rgba(0,0,0,0.15); margin-top: 15px; padding-top: 5px;">
				<div style="margin-bottom: 12px;" line-height: 24px;word-break: break-all;>
					<img src="{{.UserPicture}}" style="width: 24px; height: 24px; display:inline; vertical-align:middle; margin: 0 8px 0 0; border-radius: 3px; background-color: #ccc;"/>
					<span style="font-size: 14px; font-weight: bold; color: #777">{{.UserName}}</span>
					<span style="color: #999; font-size: 14px; margin: 0 8px;">{{.CommentDate.Format "02.01.2006 at 15:04"}}</span>
					<a href="{{.CommentLink}}" style="color: #0aa; font-size: 14px;"><b>Reply</b></a>
				</div>
				<div style="font-size: 16px; background-color: #fff; color:#000!important; padding: 14px 14px 2px 14px; border-radius: 3px; line-height: 1.4;">{{.CommentText}}</div>
			</div>
		</div>
		<div style="text-align: center; font-size: 14px; margin-top: 32px;">
			<i style="color: #000!important;">Sent to <a style="color:inherit; text-decoration: none" href="mailto:{{.Email}}">{{.Email}}</a>{{if not .ForAdmin}} for {{.ParentUserName}}{{ end }}</i>
			<div style="margin: auto; width: 150px; border-top: 1px solid rgba(0, 0, 0, 0.15); padding-top: 15px; margin-top: 15px;"></div>
			{{- if .UnsubscribeLink}}
			<a style="color: #0aa;" href="{{.UnsubscribeLink}}">Unsubscribe</a>
			{{- end }}
			<!-- This is hack for remove collapser in Gmail which can collapse end of the message -->
			<div style="opacity: 0;font-size: 1;">[{{.CommentDate.Format "02.01.2006 at 15:04"}}]</div>
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
		{{- if .SubscribeURL}}
		<p style="position: relative; margin: 0 0 0.5em 0;color:#000!important;"><a href="{{.SubscribeURL}}{{.Token}}">Click here to subscribe to email notifications</a></p>
		<p style="position: relative; margin: 0 0 0.5em 0;color:#000!important;">Alternatively, you can use code below for subscription.</p>
		{{- end }}
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
func NewEmail(emailParams EmailParams, sender emailprovider.EmailSender) (*Email, error) {
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

	sender.SetFrom(emailParams.From)
	// set up client
	res.sender = sender

	log.Printf("[DEBUG] Create new email notifier for sender: %s",
		res.sender)

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

// Send email about comment reply to Request.Email if it's set,
// also sends email to site administrator if appropriate option is set.
// Thread safe
func (e *Email) Send(ctx context.Context, req Request) (err error) {
	if e.sender == nil {
		return fmt.Errorf("Email.Send() called without valid sender set")
	}
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
		if req.parent.User.ID == req.Comment.User.ID && !req.ForAdmin {
			// don't send anything if if user replied to their own comment
			return nil
		}
		log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.Comment.ID)
		msg, err = e.buildMessageFromRequest(req, req.ForAdmin)
		if err != nil {
			return err
		}
	}

	return repeater.NewDefault(5, time.Millisecond*250).Do(
		ctx,
		func() error {
			return e.sender.Send(req.Email, msg)
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
	e.sender.SetSubject(subject)
	return msg.String(), nil
}

// buildMessageFromRequest generates email message based on Request using e.MsgTemplate
func (e *Email) buildMessageFromRequest(req Request, forAdmin bool) (string, error) {
	subject := "New reply to your comment"
	if forAdmin {
		subject = "New comment to your site"
	}
	if req.Comment.PostTitle != "" {
		subject += fmt.Sprintf(" for \"%s\"", req.Comment.PostTitle)
	}

	token, err := e.TokenGenFn(req.parent.User.ID, req.Email, req.Comment.Locator.SiteID)
	if err != nil {
		return "", errors.Wrapf(err, "error creating token for unsubscribe link")
	}
	unsubscribeLink := e.UnsubscribeURL + "?site=" + req.Comment.Locator.SiteID + "&tkn=" + token
	if forAdmin {
		unsubscribeLink = ""
	}

	commentUrlPrefix := req.Comment.Locator.URL + uiNav
	msg := bytes.Buffer{}
	tmplData := msgTmplData{
		UserName:        req.Comment.User.Name,
		UserPicture:     req.Comment.User.Picture,
		CommentText:     req.Comment.Text,
		CommentLink:     commentUrlPrefix + req.Comment.ID,
		CommentDate:     req.Comment.Timestamp,
		PostTitle:       req.Comment.PostTitle,
		Email:           req.Email,
		UnsubscribeLink: unsubscribeLink,
		ForAdmin:        forAdmin,
	}
	// in case of message to admin, parent message might be empty
	if req.Comment.ParentID != "" {
		tmplData.ParentUserName = req.parent.User.Name
		tmplData.ParentUserPicture = req.parent.User.Picture
		tmplData.ParentCommentText = req.parent.Text
		tmplData.ParentCommentLink = commentUrlPrefix + req.parent.ID
		tmplData.ParentCommentDate = req.parent.Timestamp
	}
	err = e.msgTmpl.Execute(&msg, tmplData)
	if err != nil {
		return "", errors.Wrapf(err, "error executing template to build comment reply message")
	}
	e.sender.SetSubject(subject)
	if unsubscribeLink != "" {
		e.sender.AddHeader("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
		e.sender.AddHeader("List-Unsubscribe", "<"+unsubscribeLink+">")
	} else {
		e.sender.ResetHeaders()
	}
	return msg.String(), nil
}

// String representation of Email object
func (e *Email) String() string {
	return e.sender.String()
}