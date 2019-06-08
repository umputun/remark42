// Package engine defines interfaces each supported storage should implement.
// Includes default implementation with boltdb
package engine

import (
	"sort"
	"strings"
	"time"

	"github.com/umputun/remark/backend/app/store"
)

// NOTE: mockery works from linked to go-path and with GOFLAGS='-mod=vendor' go generate
//go:generate sh -c "mockery -inpkg -name Interface -print > /tmp/engine-mock.tmp && mv /tmp/engine-mock.tmp engine_mock.go"

// Interface defines methods provided by low-level storage engine
type Interface interface {
	Create(comment store.Comment) (commentID string, err error)                  // create new comment, avoid dups by id
	Get(locator store.Locator, commentID string) (store.Comment, error)          // get comment by id
	Put(locator store.Locator, comment store.Comment) error                      // update comment, mutable parts only
	Find(locator store.Locator, sort string) ([]store.Comment, error)            // find comments for locator
	Last(siteID string, limit int, since time.Time) ([]store.Comment, error)     // last comments for given site, sorted by time
	User(siteID, userID string, limit, skip int) ([]store.Comment, error)        // comments by user, sorted by time
	UserCount(siteID, userID string) (int, error)                                // comments count by user
	Count(locator store.Locator) (int, error)                                    // number of comments for the post
	List(siteID string, limit int, skip int) ([]store.PostInfo, error)           // list of commented posts
	Info(locator store.Locator, readonlyAge int) (store.PostInfo, error)         // get post info
	Delete(locator store.Locator, commentID string, mode store.DeleteMode) error // delete comment by id
	DeleteAll(siteID string) error                                               // delete all data from site
	DeleteUser(siteID string, userID string) error                               // remove all comments from user
	SetBlock(siteID string, userID string, status bool, ttl time.Duration) error // block or unblock user with TTL (0-permanent)
	IsBlocked(siteID string, userID string) bool                                 // check if user blocked
	Blocked(siteID string) ([]store.BlockedUser, error)                          // get list of blocked users
	SetReadOnly(locator store.Locator, status bool) error                        // set/reset read-only flag
	IsReadOnly(locator store.Locator) bool                                       // check if post read-only
	SetVerified(siteID string, userID string, status bool) error                 // set/reset verified flag
	IsVerified(siteID string, userID string) bool                                // check verified status
	Verified(siteID string) ([]string, error)                                    // list of verified user ids
	Close() error                                                                // close/stop engine
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
