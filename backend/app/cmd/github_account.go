package cmd

import (
	"crypto/sha1" //nolint:gosec // the login-based id is the provider's own sha1 hash, kept as it is
	"strings"

	"github.com/go-pkgz/auth/v2/token"
	log "github.com/go-pkgz/lgr"
)

// githubLoginAttr is the user attribute the GitHub provider puts the login into at a sign-in. It is
// removed from the user before the token is made
const githubLoginAttr = "github_login"

// githubLoginID returns the user ID the GitHub provider makes from a login, the ID every GitHub user
// of remark42 has. GitHub gives a login free to anyone once its holder renames or deletes the
// account, so such an ID alone would pass to the next holder of the name
func githubLoginID(login string) string {
	return "github_" + token.HashID(sha1.New(), login) //nolint:gosec // see the import
}

// githubAccountStore keeps, per site, which GitHub account a login-based user ID belongs to
type githubAccountStore interface {
	GetUserGithubAccount(siteID, userID string) (string, error)
	SetUserGithubAccount(siteID, userID, value string) (string, error)
}

// bindGithubAccount gives a GitHub user at a sign-in the login-based ID while the login still
// belongs to the account that had it on this site. The provider makes the user ID from the numeric
// account ID, which GitHub never gives to another account, and puts the login into an attribute.
// The first sign-in of a login-based ID records the account it belongs to and keeps that ID, so
// users keep their comments and rights; a later sign-in of another account under the same login
// keeps the ID made from its own account and gets nothing of the earlier holder's.
//
// When the account can't be checked, because the provider gave no numeric ID or the store fails,
// the user keeps the login-based ID, as before the check existed, and the log says so
func bindGithubAccount(u *token.User, siteID string, st githubAccountStore) {
	login := u.StrAttr(githubLoginAttr)
	if login == "" || !strings.HasPrefix(u.ID, "github_") {
		return // not a GitHub sign-in: a refreshed token has no login attribute
	}
	delete(u.Attributes, githubLoginAttr)

	accountID, loginID := u.ID, githubLoginID(login)
	u.ID = loginID
	if accountID == loginID {
		log.Printf("[WARN] github user info had no numeric account id, user %s signed in without the account check", loginID)
		return
	}
	bound, err := st.GetUserGithubAccount(siteID, loginID)
	if err != nil {
		log.Printf("[WARN] can't read the github account of %s on %s, signed in without the account check: %v", loginID, siteID, err)
		return
	}
	switch bound {
	case "":
		if _, err = st.SetUserGithubAccount(siteID, loginID, accountID); err != nil {
			log.Printf("[WARN] can't record the github account of %s on %s: %v", loginID, siteID, err)
		}
	case accountID:
	default:
		u.ID = accountID
		log.Printf("[INFO] github login of %s on %s belongs to another account now, signed in as %s", loginID, siteID, accountID)
	}
}
