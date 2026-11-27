package api

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

// Premoderation strategy defined in the system as part of the initialization.
// Unapproved messages are only visible to mods who can decide to approve it.
type Premoderation string

const (
	// PremoderationNone means that all messages approved by default
	PremoderationNone Premoderation = "none"
	// PremoderationFirst means that once a message from a user is approved, all previous messages from the same user are also approved
	PremoderationFirst Premoderation = "first"
	// PremoderationAll requires all messages to be approved
	PremoderationAll Premoderation = "all"
	// PremoderationLinks requires all messages that contain links are not from admin or verified users need to be approved
	PremoderationLinks Premoderation = "links"
)

// NewPremoderationStrategy matches the provided string with available premoderations strategies.
// Defaults to PremoderationLinks
func newPremoderationStrategy(str string) (Premoderation, error) {
	switch strings.ToLower(strings.TrimSpace(str)) {
	case string(PremoderationLinks), "":
		return PremoderationLinks, nil
	case string(PremoderationFirst):
		return PremoderationFirst, nil
	case string(PremoderationAll):
		return PremoderationAll, nil
	case string(PremoderationNone):
		return PremoderationNone, nil
	default:
		return "", fmt.Errorf("unkwonwn value selected for premoderation: %s", str)
	}
}

type PremoderationStore interface {
	IsVerified(siteID, userID string) bool
	User(siteID, userID string, limit, skip int, user store.User) ([]store.Comment, error)
	ApproveComments(locator store.Locator, id string, approveStatus, approvePrevious bool) (store.Comment, error)
}

type Premoderator struct {
	strategy Premoderation
	store    PremoderationStore
}

func NewPremoderator(strategy string, store PremoderationStore) (Premoderator, error) {
	strat, err := newPremoderationStrategy(strategy)
	if err != nil {
		return Premoderator{}, err
	}
	return Premoderator{strategy: strat, store: store}, nil
}

func MaskComment(c *store.Comment) {
	c.Text = ""
	c.Orig = ""
}

func (p *Premoderator) premoderateComment(comment store.Comment, user store.User) (store.Comment, error) {
	// admin messages are approved by default
	if user.Admin {
		comment.PendingApproval = false
		return comment, nil
	}

	// if previous comments were approved and the strategy is "first" approve other comments by default
	switch p.strategy {
	case PremoderationLinks:
		if comment.User.Admin || p.store.IsVerified(comment.Locator.SiteID, comment.User.ID) {
			comment.PendingApproval = false
			return comment, nil
		}

		hasLink, err := comment.ContainsLinks()
		if err != nil {
			return store.Comment{}, err
		}

		comment.PendingApproval = hasLink
		return comment, nil

	case PremoderationFirst:
		prevComments, err := p.store.User(user.SiteID, user.ID, 100, 0, user) // ! todo: check the usage of limit/skip
		if err != nil {
			if errors.Is(err, engine.ErrNoUserComments) { // this is the first message for this user
				comment.PendingApproval = true
				return comment, nil
			}
			return store.Comment{}, fmt.Errorf("error accessing user's comments: %s", err.Error())
		}
		// if there is a previously approved comment then approve the new comment
		for _, pc := range prevComments {
			if !pc.PendingApproval && !pc.Deleted {
				comment.PendingApproval = false
				return comment, nil
			}
		}
	case PremoderationAll: // we need to approve every single comment
		comment.PendingApproval = true
	default: // if no premoderation is applied we mark new comments as approved
		comment.PendingApproval = false
	}
	return comment, nil
}

func (p Premoderator) Active() bool {
	return p.strategy != PremoderationNone
}

func (p Premoderator) VisibleTo(c store.Comment, user store.User) bool {
	if !c.PendingApproval || !p.Active() {
		return true
	}
	return user.Admin || c.User.ID == user.ID
}

func (p Premoderator) FilterVisibleTo(comments []store.Comment, user store.User) []store.Comment {
	if !p.Active() {
		return comments
	}
	return slices.DeleteFunc(slices.Clone(comments), func(c store.Comment) bool {
		return !p.VisibleTo(c, user)
	})
}

func (p Premoderator) ApproveComment(locator store.Locator, id string, approveStatus bool) (store.Comment, error) {
	var comment store.Comment
	var err error
	switch p.strategy {
	// if we use the "ApproveFirst" strategy we need to approve all pending comments once one is approved
	case PremoderationFirst:
		comment, err = p.store.ApproveComments(locator, id, approveStatus, true)
	default:
		comment, err = p.store.ApproveComments(locator, id, approveStatus, false)
	}
	return comment, err
}
