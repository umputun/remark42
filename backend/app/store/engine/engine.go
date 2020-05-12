package engine

// Package engine defines interfaces each supported storage should implement.
// Includes default implementation with boltdb

import (
	"sort"
	"strings"
	"time"

	"github.com/umputun/remark42/backend/app/store"
)

// NOTE: mockery works from linked to go-path and with GOFLAGS='-mod=vendor' go generate
//go:generate sh -c "mockery -inpkg -name Interface -print > /tmp/engine-mock.tmp && mv /tmp/engine-mock.tmp engine_mock.go"

// Interface defines methods provided by low-level storage engine
type Interface interface {
	Create(comment store.Comment) (commentID string, err error) // create new comment, avoid dups by id
	Update(comment store.Comment) error                         // update comment, mutable parts only
	Get(req GetRequest) (store.Comment, error)                  // get comment by id
	Find(req FindRequest) ([]store.Comment, error)              // find comments for locator or site
	Info(req InfoRequest) ([]store.PostInfo, error)             // get post(s) meta info
	Count(req FindRequest) (int, error)                         // get count for post or user
	Delete(req DeleteRequest) error                             // Delete post(s), user, comment, user details, or everything
	Flag(req FlagRequest) (bool, error)                         // set and get flags
	ListFlags(req FlagRequest) ([]interface{}, error)           // get list of flagged keys, like blocked & verified user

	// UserDetail sets or gets single detail value, or gets all details for requested site
	// Returns list even for single entry request is a compromise in order to have both single detail getting and setting
	// and all site's details listing under the same function (and not to extend interface by two separate functions)
	UserDetail(req UserDetailRequest) ([]UserDetailEntry, error)

	Close() error // close storage engine
}

// GetRequest is the input for Get func
type GetRequest struct {
	Locator   store.Locator `json:"locator"`
	CommentID string        `json:"comment_id"`
}

// FindRequest is the input for all find operations
type FindRequest struct {
	Locator store.Locator `json:"locator"`           // lack of URL means site operation
	UserID  string        `json:"user_id,omitempty"` // presence of UserID treated as user-related find
	Sort    string        `json:"sort,omitempty"`    // sort order with +/-field syntax
	Since   time.Time     `json:"since,omitempty"`   // time limit for found results
	Limit   int           `json:"limit,omitempty"`
	Skip    int           `json:"skip,omitempty"`
}

// InfoRequest is the input of Info operation used to get meta data about posts
type InfoRequest struct {
	Locator     store.Locator `json:"locator"`
	Limit       int           `json:"limit,omitempty"`
	Skip        int           `json:"skip,omitempty"`
	ReadOnlyAge int           `json:"ro_age,omitempty"`
}

// DeleteRequest is the input for all delete operations (comments, sites, users)
type DeleteRequest struct {
	Locator    store.Locator    `json:"locator"` // lack of URL means site operation
	CommentID  string           `json:"comment_id,omitempty"`
	UserID     string           `json:"user_id,omitempty"`
	UserDetail UserDetail       `json:"user_detail,omitempty"`
	DeleteMode store.DeleteMode `json:"del_mode"`
}

// Flag defines type of binary attribute
type Flag string

// FlagStatus represents values of the flag update
type FlagStatus int

// enum of update values
const (
	FlagNonSet FlagStatus = 0
	FlagTrue   FlagStatus = 1
	FlagFalse  FlagStatus = -1
)

// Enum of all flags
const (
	ReadOnly = Flag("readonly")
	Verified = Flag("verified")
	Blocked  = Flag("blocked")
)

// All possible user details
const (
	// UserEmail is a user email
	UserEmail = UserDetail("email")
	// AllUserDetails used for listing and deletion requests
	AllUserDetails = UserDetail("all")
)

// FlagRequest is the input for both get/set for flags, like blocked, verified and so on
type FlagRequest struct {
	Flag    Flag          `json:"flag"`              // flag type
	Locator store.Locator `json:"locator"`           // post locator
	UserID  string        `json:"user_id,omitempty"` // for flags setting user status
	Update  FlagStatus    `json:"update,omitempty"`  // if FlagNonSet it will be get op, if set will set the value
	TTL     time.Duration `json:"ttl,omitempty"`     // ttl for time-sensitive flags only, like blocked for some period
}

// UserDetail defines name of the user detail
type UserDetail string

// UserDetailEntry contains single user details entry
type UserDetailEntry struct {
	UserID string `json:"user_id"`         // duplicate user's id to use this structure not only embedded but separately
	Email  string `json:"email,omitempty"` // UserEmail
}

// UserDetailRequest is the input for both get/set for details, like email
type UserDetailRequest struct {
	Detail  UserDetail    `json:"detail"`           // detail name
	Locator store.Locator `json:"locator"`          // post locator
	UserID  string        `json:"user_id"`          // user id for get\set
	Update  string        `json:"update,omitempty"` // update value
}

const (
	// limits
	lastLimit = 1000
	userLimit = 500
)

// SortComments is for engines can't sort data internally
func SortComments(comments []store.Comment, sortFld string) []store.Comment {
	sort.Slice(comments, func(i, j int) bool {
		switch sortFld {
		case "+time", "-time", "time", "+active", "-active", "active":
			if strings.HasPrefix(sortFld, "-") {
				return comments[i].Timestamp.After(comments[j].Timestamp)
			}
			return comments[i].Timestamp.Before(comments[j].Timestamp)

		case "+score", "-score", "score":
			if strings.HasPrefix(sortFld, "-") {
				if comments[i].Score == comments[j].Score {
					return comments[i].Timestamp.Before(comments[j].Timestamp)
				}
				return comments[i].Score > comments[j].Score
			}
			if comments[i].Score == comments[j].Score {
				return comments[i].Timestamp.Before(comments[j].Timestamp)
			}
			return comments[i].Score < comments[j].Score

		case "+controversy", "-controversy", "controversy":
			if strings.HasPrefix(sortFld, "-") {
				if comments[i].Controversy == comments[j].Controversy {
					return comments[i].Timestamp.Before(comments[j].Timestamp)
				}
				return comments[i].Controversy > comments[j].Controversy
			}
			if comments[i].Controversy == comments[j].Controversy {
				return comments[i].Timestamp.Before(comments[j].Timestamp)
			}
			return comments[i].Controversy < comments[j].Controversy

		default:
			return comments[i].Timestamp.Before(comments[j].Timestamp)
		}
	})
	return comments
}
