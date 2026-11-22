// Package enginetest is the test set every engine.Interface implementation runs: the bolt engine and the
// memory engine in the example both do, and an author of another engine can run it the same way:
//
//	func TestMyEngine(t *testing.T) {
//		enginetest.Run(t, func(t *testing.T) engine.Interface {
//			e := NewMyEngine(enginetest.SiteID, enginetest.OtherSiteID)
//			t.Cleanup(func() { _ = e.Close() })
//			return e
//		})
//	}
//
// The cases check what the rest of remark42 relies on. Where engines may differ, in how they report an unknown
// site or user and in the wording of most errors, the cases only check that an error is returned and leave the
// rest to each engine's own tests.
package enginetest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

const (
	// SiteID is the site the cases write to
	SiteID = "radio-t"
	// OtherSiteID is a second site, used to check that sites are kept apart
	OtherSiteID = "radio-t-2"
)

// postURL is the post holding the two comments every case starts with
const postURL = "https://radio-t.com"

// Run runs every case as a subtest, each on its own engine made by newEngine. newEngine must return a new, empty
// engine that serves SiteID and OtherSiteID and keeps them apart, and close it when the test ends, for example
// with t.Cleanup; the Close case closes it twice first, so a further Close must not fail or panic. No case
// waits for time to pass or moves a clock, so a block that runs out is for each engine's own tests to check.
func Run(t *testing.T, newEngine func(t *testing.T) engine.Interface) {
	s := suite{newEngine: newEngine}
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"CreateAndFind", s.createAndFind},
		{"CreateFailedReadOnly", s.createFailedReadOnly},
		{"Get", s.get},
		{"Update", s.update},
		{"UpdateKeepsFields", s.updateKeepsFields},
		{"UpdateDeleted", s.updateDeleted},
		{"FindLast", s.findLast},
		{"FindLastSince", s.findLastSince},
		{"FindInPostSince", s.findInPostSince},
		{"FindForUser", s.findForUser},
		{"FindForUserPagination", s.findForUserPagination},
		{"CountPost", s.countPost},
		{"CountUser", s.countUser},
		{"InfoPost", s.infoPost},
		{"InfoPostCreatedOutOfOrder", s.infoPostCreatedOutOfOrder},
		{"InfoList", s.infoList},
		{"FlagBlockedUser", s.flagBlockedUser},
		{"FlagReadOnlyPost", s.flagReadOnlyPost},
		{"FlagVerified", s.flagVerified},
		{"FlagListVerified", s.flagListVerified},
		{"FlagListBlocked", s.flagListBlocked},
		{"FlagListsSortedWithNames", s.flagListsSortedWithNames},
		{"FlagsKeepUserMeta", s.flagsKeepUserMeta},
		{"UserMetaPerSite", s.userMetaPerSite},
		{"UserDetail", s.userDetail},
		{"UserDetailAllEmpty", s.userDetailAllEmpty},
		{"SetUserDetailReturnsEntry", s.setUserDetailReturnsEntry},
		{"ListDetailsKeepsEntriesSeparate", s.listDetailsKeepsEntriesSeparate},
		{"UserDetailsListOnlyStored", s.userDetailsListOnlyStored},
		{"DeleteUserDetail", s.deleteUserDetail},
		{"DeleteComment", s.deleteComment},
		{"DeleteHard", s.deleteHard},
		{"DeleteAll", s.deleteAll},
		{"DeleteAllDropsUserDetails", s.deleteAllDropsUserDetails},
		{"DeleteUserHard", s.deleteUserHard},
		{"DeleteUserHardNoComments", s.deleteUserHardNoComments},
		{"DeleteUserSoft", s.deleteUserSoft},
		{"DeleteUserSoftNoComments", s.deleteUserSoftNoComments},
		{"Close", s.close},
	}
	for _, c := range cases {
		t.Run(c.name, c.fn)
	}
}

type suite struct {
	newEngine func(t *testing.T) engine.Interface
}

// prep returns a new engine holding two comments of user1 on postURL, at 15:18:22 and 15:18:23
func (s suite) prep(t *testing.T) engine.Interface {
	e := s.newEngine(t)
	for _, c := range []store.Comment{
		{ID: "id-1", Text: `some text, <a href="http://radio-t.com">link</a>`, Timestamp: ts(22), Locator: loc(postURL),
			User: store.User{ID: "user1", Name: "user name"}},
		{ID: "id-2", Text: "some text2", Timestamp: ts(23), Locator: loc(postURL), User: store.User{ID: "user1", Name: "user name"}},
	} {
		_, err := e.Create(c)
		require.NoError(t, err)
	}
	return e
}

// ts returns 2017-12-20 15:18 UTC with the given seconds
func ts(sec int) time.Time { return time.Date(2017, 12, 20, 15, 18, sec, 0, time.UTC) }

// loc returns the locator of a post on SiteID
func loc(url string) store.Locator { return store.Locator{SiteID: SiteID, URL: url} }

// site is the locator of SiteID itself
var site = store.Locator{SiteID: SiteID}

func getReq(locator store.Locator, commentID string) engine.GetRequest {
	return engine.GetRequest{Locator: locator, CommentID: commentID}
}

func commentIDs(comments []store.Comment) []string {
	ids := make([]string, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.ID)
	}
	return ids
}
