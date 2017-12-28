# remark42 [![Build Status](http://drone.umputun.com:9080/api/badges/umputun/remark/status.svg)](http://drone.umputun.com:9080/umputun/remark)

Remark42 ia a comment engine, self-hosted, lightweight, simple (but functional) what doesn't spy on users.

- Supports social login via google, facebook and github
- Moderation allowing admins to remove comments and block users
- Voting and pinning system
- Ability to sort comments
- Comment retrieval per post, user and comment's id
- Extractor for recent comments, cross-post
- Multi-level nested comments with both tree and plain presentations
- Export all data to json and automatic backups
- Import from disqus
- No need of external databases, everything embedded in a single data file
- Fully dockerized and can be deployed in a single command
- Nice, lightweight and fully customizable UI
- Multi-site mode to serve comments for multiple sites from a single remark instance
- Integration with automatic ssl (Let’s Encrypt) via [nginx-le](https://github.com/umputun/nginx-le)

## Install

### Backend

- copy provided docker-compose.yml and customize for your needs
- make sure you **don't keep** `DEV=true` for any non-development deployments
- pull and start `docker compose pull && docker compose up`

#### Run modes

- `server` activates regular, server mode
- `import` performs import from external providers (disqus and internal json, see `/api/v1/admin/export`)

#### Register oauth2 providers

TBD

### Frontend

TBD

## API

### Authorization

- `GET /auth/{provider}/login?from=http://url` - perform "social" login with one of supported providers and redirect to `url`
- `GET /auth/{provider}/logout` - logout

```go
type User struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Picture string `json:"picture"`
    Profile string `json:"profile"`
    Admin   bool   `json:"admin"`
}
```

_currently supported providers are `google`, `facebook` and `github`_

### Commenting

- `POST /api/v1/comment` - add a comment. _auth required_

```go
type Comment struct {
    ID        string          `json:"id"`      // comment ID, read only
    ParentID  string          `json:"pid"`     // parent ID
    Text      string          `json:"text"`    // comment text
    User      User            `json:"user"`    // user info, read only
    Locator   Locator         `json:"locator"` // post locator
    Score     int             `json:"score"`   // comment score, read only
    Votes     map[string]bool `json:"votes"`   // comment votes, read only
    Timestamp time.Time       `json:"time"`    // time stamp, read only
    Pin       bool            `json:"pin"`     // pinned status, read only
}

type Locator struct {
    SiteID string `json:"site"`
    URL    string `json:"url"`
}
```

- `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree` - find all comments for given post

This is the primary call used by UI to show comments for given post. It can return comments in two formats - `plain` and `tree`.
In plain format result will be sorted list of `Comment`. In tree format this is going to be tree-like structure with this structure:

```go
type Tree struct {
    Nodes []Node `json:"comments"`
}

type Node struct {
    Comment store.Comment `json:"comment"`
    Replies []Node        `json:"replies,omitempty"`
}
```

Sort can be `time` or `score`. Supported sort order with prefix -/+, i.e. `-time`. For `tree` mode sort will be applied to top-level comments only and all replies always sorted by time.

- `GET /api/v1/last/{max}?site=site-id` - get up to `{max}` last comments
- `GET /api/v1/id/{id}?site=site-id` - get comment by `id`
- `GET /api/v1/comments?site=site-id&user=id` - get comment by `user id`
- `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
- `GET /api/v1/user` - get user info, _auth required_
- `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decreases. _auth required_

### Admin

- `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`. _auth and admin required_
- `PUT /api/v1/admin/user/{userid}?site=site-id&block=1` - block or unblock user. _auth and admin required_
- `GET /api/v1/admin/export?site=side-id&block=1` - export all comments. _auth and admin required_
- `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin or unpin comment. _auth and admin required_
