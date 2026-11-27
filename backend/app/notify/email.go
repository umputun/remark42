package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
	"github.com/go-pkgz/repeater/v2"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/idna"

	"github.com/umputun/remark42/backend/app/store"
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
	AvatarURLs               []string // URL prefixes of avatars stored by this instance, other user pictures are left out

	TokenGenFn func(userID, email, site string) (string, error) // unsubscribe token generation function
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
	CommentText       template.HTML
	CommentLink       string
	CommentDate       time.Time
	ParentUserName    string
	ParentUserPicture string
	ParentCommentText template.HTML
	ParentCommentLink string
	ParentCommentDate time.Time
	PostTitle         string
	Email             string
	UnsubscribeLink   string
	ForAdmin          bool
	LinkHost          string // the host CommentLink and ParentCommentLink lead to, set in admin notifications only
}

// emailCommentPolicy sanitizes comment HTML for inclusion in notification emails.
// It is intentionally stricter than the store-level UGC policy used for web rendering:
// links (<a>) and images (<img>) are dropped so a comment can't smuggle phishing links
// or remote tracking pixels into an email sent from the legitimate remark42 address,
// while basic inline and block text formatting is preserved.
// Telegram notifications pass comment HTML through it too, see telegramCommentHTML.
var emailCommentPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "hr", "div", "span",
		"b", "strong", "i", "em", "u", "s", "strike", "del", "ins", "sub", "sup", "mark", "small",
		"blockquote", "q", "cite",
		"code", "pre", "kbd", "samp", "var",
		"ul", "ol", "li", "dl", "dt", "dd",
		"h1", "h2", "h3", "h4", "h5", "h6",
	)
	return p
}()

// emailSafeHTML strips links and images from pre-rendered comment HTML and returns
// it as template.HTML so html/template renders the remaining safe formatting as-is.
// The sanitizer filters tags one by one and doesn't match them, so the result is balanced
// to keep a stray end tag from closing elements of the message template around the comment.
func emailSafeHTML(commentHTML string) template.HTML {
	clean := emailCommentPolicy.Sanitize(commentHTML)
	balanced, err := store.BalanceHTML(clean)
	if err != nil {
		// nested too deep or misnested beyond repair: keep the text only
		balanced = bluemonday.StrictPolicy().Sanitize(clean)
	}
	return template.HTML(balanced) //nolint:gosec // sanitized above: <a>/<img> dropped, only balanced formatting tags survive
}

// reNotVerificationName matches a run of characters a user name in the verification message may not have
var reNotVerificationName = regexp.MustCompile(`[^\p{L}\p{M}\d_]+`)

// verificationUserName reduces a display name to letters with their combining marks, digits, underscores
// and single spaces, at most 64 characters. The verification message goes to an address nobody has confirmed
// yet, and most login providers let users choose a free-text name, which as it is could carry a message and
// a link of the user's choice to anyone. Dots and hyphens, which a name typed at sign-in may have, are
// dropped too, so no domain name survives for a mail client to turn into a link
func verificationUserName(name string) string {
	name = strings.TrimSpace(reNotVerificationName.ReplaceAllString(name, " "))
	if runes := []rune(name); len(runes) > 64 {
		name = strings.TrimSpace(string(runes[:64]))
	}
	return name
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

	if e.VerificationTemplatePath == "" {
		e.VerificationTemplatePath = defaultEmailVerificationTemplatePath
	}

	if e.MsgTemplatePath == "" {
		e.MsgTemplatePath = defaultEmailTemplatePath
	}

	if msgTmplFile, err = templates.Read(e.MsgTemplatePath); err != nil {
		return fmt.Errorf("can't read message template: %w", err)
	}
	if verifyTmplFile, err = templates.Read(e.VerificationTemplatePath); err != nil {
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

	var errs []error

	for _, email := range req.Emails {
		err := e.buildAndSendMessage(ctx, req, email, false)
		if err != nil {
			errs = append(errs, fmt.Errorf("problem sending user email notification to %q: %w", email, err))
		}
	}

	for _, email := range e.AdminEmails {
		err := e.buildAndSendMessage(ctx, req, email, true)
		if err != nil {
			errs = append(errs, fmt.Errorf("problem sending admin email notification to %q: %w", email, err))
		}
	}

	return errors.Join(errs...)
}

func (e *Email) buildAndSendMessage(ctx context.Context, req Request, email string, forAdmin bool) error {
	// an admin's address comes from the server's options and goes in as it is written there
	recipient := email
	if !forAdmin {
		var err error
		if recipient, err = mailtoRecipient(email); err != nil {
			return err
		}
	}
	log.Printf("[DEBUG] send notification via %s, comment id %s", e, req.Comment.ID)
	msg, err := e.buildMessageFromRequest(req, email, forAdmin)
	if err != nil {
		return err
	}

	return repeater.NewFixed(5, time.Millisecond*250).Do(
		ctx,
		func() error {
			return e.Email.Send(
				ctx,
				fmt.Sprintf("mailto:%s?from=%s&unsubscribeLink=%s&subject=%s",
					recipient,
					url.QueryEscape(e.From),
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

	recipient, err := mailtoRecipient(req.Email)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] send verification via %s, user %s", e, req.User)
	msg, err := e.buildVerificationMessage(req.User, req.Email, req.Token, req.SiteID)
	if err != nil {
		return err
	}

	return repeater.NewFixed(5, time.Millisecond*250).Do(
		ctx,
		func() error {
			return e.Email.Send(
				ctx,
				fmt.Sprintf("mailto:%s?from=%s&subject=%s",
					recipient,
					url.QueryEscape(e.From),
					url.QueryEscape(e.VerificationSubject),
				),
				msg,
			)
		})
}

// errMailtoAddress is the refusal of a recipient the mailto destination can't carry
var errMailtoAddress = errors.New("recipient is not one plain email address")

// mailtoRecipient returns the address to put into the mailto destination of a mail to a user, or
// an error for one that would change that mail. The notify library takes a mail's recipients,
// sender and subject as one mailto URL and reads the recipients from it as they stand, so the
// address goes in unescaped. There a comma adds a recipient, and a "?" or a "#" ends the
// recipients, with what follows read as the query: a sender and a subject of the address's own,
// ahead of ours. A quoted local part is refused too, the mail library sends to it with the quotes
// taken off, and what they held can be a list.
//
// The address of a subscriber is whatever was stored as typed, so it is checked here, where the
// destination is made. One written with a display name or with spaces around it is mailed at the
// bare address, as it was before.
func mailtoRecipient(address string) (string, error) {
	addr, err := mail.ParseAddress(address)
	if err != nil {
		return "", errMailtoAddress
	}
	// the bare address has to read as itself: with a quoted local part it does not
	bare, err := mail.ParseAddress(addr.Address)
	if err != nil || bare.Address != addr.Address || strings.ContainsAny(addr.Address, "?#") {
		return "", errMailtoAddress
	}
	return addr.Address, nil
}

// buildVerificationMessage generates verification email message based on given input
func (e *Email) buildVerificationMessage(user, email, token, site string) (string, error) {
	msg := bytes.Buffer{}
	err := e.verifyTmpl.Execute(&msg, verifyTmplData{
		User:         verificationUserName(user),
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

// adminLinkIDNA turns a host name into its ASCII form the way browsers do before they open it: letters
// are mapped and normalised, characters a browser drops are dropped, and a name it would refuse is an error
var adminLinkIDNA = idna.New(idna.MapForLookup(), idna.Transitional(false), idna.StrictDomainName(false),
	idna.CheckHyphens(false), idna.BidiRule(), idna.CheckJoiners(true))

// adminPageLink returns the page address of a comment for a link in a message to an admin, and the host
// to name next to that link. The address comes from the commenter and is not checked against the site, and
// an admin is notified about every comment, so the link must not lead anywhere the message does not say.
// Go and browsers read some addresses differently: "https:evil.example" has no host for Go and opens
// evil.example in a browser, and a host can hold letters that are invisible or look like others. So the
// address is written again from its parts, with the host in ASCII, and that is both what the link opens
// and what the label shows. An address that is not a plain web address with a host gets no link.
func adminPageLink(pageURL string) (link, host string) {
	u, err := url.Parse(pageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Opaque != "" || u.Host == "" {
		return "", ""
	}
	host = u.Hostname()
	if net.ParseIP(host) == nil {
		// the mapping turns wide forms of "@", "/" or a space into the plain ones, which no host name
		// may hold; an IPv6 address with a zone lands here as well and has a "%" and colons
		if host, err = adminLinkIDNA.ToASCII(host); err != nil || host == "" || strings.ContainsAny(host, " #%/:<>?@[\\]^|") {
			return "", ""
		}
		// a browser reads a name whose last label is a number, all digits or "0x" and hexadecimal
		// digits, as an IPv4 address in a short spelling and opens the address it stands for,
		// "127.1" as 127.0.0.1 and "10.0.1.0x" as 10.0.1.0, or refuses it
		last := strings.TrimSuffix(host, ".")
		last = last[strings.LastIndexByte(last, '.')+1:]
		if last == "" || strings.Trim(last, "0123456789") == "" ||
			(strings.HasPrefix(last, "0x") && strings.Trim(last[2:], "0123456789abcdef") == "") {
			return "", ""
		}
	}
	hostPort := host
	switch {
	case u.Port() != "":
		if _, err = strconv.ParseUint(u.Port(), 10, 16); err != nil {
			return "", ""
		}
		hostPort = net.JoinHostPort(host, u.Port())
	case strings.Contains(host, ":"):
		hostPort = "[" + host + "]"
	}
	// ForceQuery keeps a "?" with nothing after it: comments are stored under the exact page address
	clean := url.URL{Scheme: u.Scheme, Host: hostPort, Path: u.Path, RawPath: u.RawPath, RawQuery: u.RawQuery, ForceQuery: u.ForceQuery}
	return clean.String(), host
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

	// a reply notifies the authors of every comment above it, so the link names the user the address
	// belongs to; the unsubscribe handler accepts a link only with the address stored for the user in it.
	// A request not made by Service.Submit has no such users and names the author of the parent comment
	userID := req.parent.User.ID
	if id, ok := req.emailUsers[email]; ok {
		userID = id
	}
	token, err := e.TokenGenFn(userID, email, req.Comment.Locator.SiteID)
	if err != nil {
		return commentMessage{}, fmt.Errorf("error creating token for unsubscribe link: %w", err)
	}
	unsubscribeLink := e.UnsubscribeURL + "?site=" + req.Comment.Locator.SiteID + "&tkn=" + token
	if forAdmin {
		unsubscribeLink = ""
	}

	// a user is notified only about a page that user commented on, so the address is linked as it is;
	// an admin gets a message for a comment on any address the commenter named
	commentURLPrefix, linkHost := req.Comment.Locator.URL+uiNav, ""
	if forAdmin {
		commentURLPrefix, linkHost = adminPageLink(req.Comment.Locator.URL)
		if commentURLPrefix != "" {
			commentURLPrefix += uiNav
		}
	}
	commentLink := func(id string) string {
		if commentURLPrefix == "" {
			return ""
		}
		return commentURLPrefix + id
	}
	msg := bytes.Buffer{}
	tmplData := msgTmplData{
		UserName:        req.Comment.User.Name,
		UserPicture:     e.storedAvatar(req.Comment.User.Picture),
		CommentText:     emailSafeHTML(req.Comment.Text),
		CommentLink:     commentLink(req.Comment.ID),
		LinkHost:        linkHost,
		CommentDate:     req.Comment.Timestamp,
		PostTitle:       req.Comment.PostTitle,
		Email:           email,
		UnsubscribeLink: unsubscribeLink,
		ForAdmin:        forAdmin,
	}
	// in case of message to admin, parent message might be empty
	if req.Comment.ParentID != "" {
		tmplData.ParentUserName = req.parent.User.Name
		tmplData.ParentUserPicture = e.storedAvatar(req.parent.User.Picture)
		tmplData.ParentCommentText = emailSafeHTML(req.parent.Text)
		tmplData.ParentCommentLink = commentLink(req.parent.ID)
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

// storedAvatar returns the user picture when it is an avatar stored by this instance, and an empty string
// otherwise. A user who logs in gets such a picture from the avatar proxy, but an imported or restored user
// keeps the URL from the file, which may point to any host and would put the same remote tracking image
// into the email that emailCommentPolicy removes from the comment text.
func (e *Email) storedAvatar(picture string) string {
	for _, prefix := range e.AvatarURLs {
		avatarID, ok := strings.CutPrefix(picture, prefix)
		// the avatar id is a single path segment, anything else may lead to another endpoint of the same host
		if prefix != "" && ok && avatarID != "" && avatarID != "." && avatarID != ".." && !strings.ContainsAny(avatarID, `/\?#%`) {
			return picture
		}
	}
	return ""
}
