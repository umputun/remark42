---
title: API
parent: Contributing
order: 400
---

## Authorization

- `GET /auth/{provider}/login?from=http://url&site=site_id&session=1` - perform "social" login with one of [supported providers](#register-oauth2-providers) and redirect to `url`. Presence of `session` (any non-zero value) change the default cookie expiration and makes them session-only
- `GET /auth/logout` - logout

```go
type User struct {
    Name     string `json:"name"`
    ID       string `json:"id"`
    Picture  string `json:"picture"`
    Admin    bool   `json:"admin"`
    Blocked  bool   `json:"block"`
    Verified bool   `json:"verified"`
}
```

## Commenting

- `POST /api/v1/comment` - add a comment, _auth required_

```go
type Comment struct {
    ID          string    `json:"id"`      // comment ID, read only
    ParentID    string    `json:"pid"`     // parent ID
    Text        string    `json:"text"`    // comment text, after md processing
    Orig        string    `json:"orig"`    // original comment text
    User        User      `json:"user"`    // user info, read only
    Locator     Locator   `json:"locator"` // post locator
    Score       int       `json:"score"`   // comment score, read only
    Vote        int       `json:"vote"`    // vote for the current user, -1/1/0
    Controversy float64   `json:"controversy,omitempty"` // comment controversy, read only
    Timestamp   time.Time `json:"time"`    // time stamp, read only
    Edit        *Edit     `json:"edit,omitempty" bson:"edit,omitempty"` // pointer to have empty default in JSON response
    Pin         bool      `json:"pin"`     // pinned status, read only
    Delete      bool      `json:"delete"`  // delete status, read only
    PostTitle   string    `json:"title"`   // post title
}

type Locator struct {
    SiteID string `json:"site"` // site ID
    URL    string `json:"url"`  // post URL
}

type Edit struct {
    Timestamp time.Time `json:"time" bson:"time"`
    Summary   string    `json:"summary"`
}
```

- `POST /api/v1/preview` - preview comment in HTML. Body is `Comment` to render
- `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree|plain` - find all comments for given post

This is the primary call used by UI to show comments for the given post. It can return comments in two formats - `plain` and `tree`. In plain format result will be sorted list of `Comment`. In tree format this is going to be tree-like object with this structure:

```go
type Tree struct {
    Nodes []Node         `json:"comments"`
    Info  store.PostInfo `json:"info,omitempty"`
}

type Node struct {
    Comment store.Comment `json:"comment"`
    Replies []Node        `json:"replies,omitempty"`
}
```

Sort can be `time`, `active` or `score`. Supported sort order with prefix -/+, i.e. `-time`. For `tree` mode sort will be applied to top-level comments only and all replies are always sorted by time.

- `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit comment, allowed once in `EDIT_TIME` minutes since creation. Body is `EditRequest` JSON

```go
type EditRequest struct {
    Text    string `json:"text"`    // updated text
    Summary string `json:"summary"` // optional, summary of the edit
    Delete  bool   `json:"delete"`  // delete flag
}{}
```

- `GET /api/v1/last/{max}?site=site-id&since=ts-msec` - get up to `{max}` last comments, `since` (epoch time, milliseconds) is optional
- `GET /api/v1/id/{id}?site=site-id` - get comment by `comment id`
- `GET /api/v1/comments?site=site-id&user=id&limit=N` - get comment by `user id`, returns `response` object

```go
type response struct {
    Comments []store.Comment `json:"comments"`
    Count    int             `json:"count"`
}{}
```

- `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
- `POST /api/v1/count?site=siteID` - get number of comments for posts from post body (list of post IDs)
- `GET /api/v1/list?site=site-id&limit=5&skip=2` - list commented posts, returns array or `PostInfo`, limit=0 will return all posts

```go
type PostInfo struct {
    URL      string    `json:"url"`
    Count    int       `json:"count"`
    ReadOnly bool      `json:"read_only,omitempty"`
    FirstTS  time.Time `json:"first_time,omitempty"`
    LastTS   time.Time `json:"last_time,omitempty"`
}
```

- `GET /api/v1/user` - get user info, _auth required_
- `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decrease, _auth required_
- `GET /api/v1/userdata?site=site-id` - export all user data to gz stream, _auth required_
- `POST /api/v1/deleteme?site=site-id` - request deletion of user data, _auth required_
- `GET /api/v1/config?site=site-id` - returns configuration (parameters) for given site

```go
type Config struct {
    Version        string   `json:"version"`
    EditDuration   int      `json:"edit_duration"`
    MaxCommentSize int      `json:"max_comment_size"`
    Admins         []string `json:"admins"`
    AdminEmail     string   `json:"admin_email"`
    Auth           []string `json:"auth_providers"`
    LowScore       int      `json:"low_score"`
    CriticalScore  int      `json:"critical_score"`
    PositiveScore  bool     `json:"positive_score"`
    ReadOnlyAge    int      `json:"readonly_age"`
    MaxImageSize   int      `json:"max_image_size"`
    EmojiEnabled   bool     `json:"emoji_enabled"`
}
```

- `GET /api/v1/info?site=site-idd&url=post-url` - returns `PostInfo` for site and URL

## Streaming API

Streaming API provides server-sent events for post updates as well as a site update:

- `GET /api/v1/stream/info?site=site-idd&url=post-url&since=unix_ts_msec` - returns stream (`event: info`) with `PostInfo` records for the site and URL. `since` is optional
- `GET /api/v1/stream/last?site=site-id&since=unix_ts_msec` - returns updates stream (`event: last`) with comments for the site, `since` is optional

<details><summary>Response example</summary>

```
data: {"url":"https://radio-t.com/blah1","count":2,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.142872-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":3,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.157709-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":4,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.172991-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":5,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.188429-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":6,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.204742-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":7,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.220692-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":8,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.23817-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":9,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.254669-05:00"}
```

</details>

## RSS Feeds

- `GET /api/v1/rss/post?site=site-id&url=post-url` - RSS feed for a post
- `GET /api/v1/rss/site?site=site-id` - RSS feed for given site
- `GET /api/v1/rss/reply?site=site-id&user=user-id` - RSS feed for replies to user's comments

## Images Management

- `GET /api/v1/picture/{user}/{id}` - load stored image
- `POST /api/v1/picture` - upload and store image, uses post form with `FormFile("file")`. Returns `{"id": user/imgid}`, _auth required_

_returned ID should be appended to load image URL on caller side_

## Email Subscription

- `GET /api/v1/email?site=site-id` - get user's email, _auth required_
- `POST /api/v1/email/subscribe?site=site-id&address=user@example.org` - makes confirmation token and sends it to user over email, _auth required_

  Trying to subscribe to the same email a second time will return response code `409 Conflict` and explaining error message

- `POST /api/v1/email/confirm?site=site-id&tkn=token` - uses provided token parameter to set email for the user, _auth required_

  Setting email subscribe user for all first-level replies to his messages

- `DELETE /api/v1/email?site=siteID` - removes user's email, _auth required_

## Admin

- `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`
- `PUT /api/v1/admin/user/{userid}?site=site-id&block=1&ttl=7d` - block or unblock user with optional TTL (default=permanent)
- `GET api/v1/admin/blocked&site=site-id` - list of blocked user IDs

```go
type BlockedUser struct {
    ID    string    `json:"id"`
    Name  string    `json:"name"`
    Until time.Time `json:"time"`
}
```

- `GET /api/v1/admin/export?site=site-id&mode=[stream|file]` - export all comments to JSON stream or gz file
- `POST /api/v1/admin/import?site=site-id` - import comments from the backup, uses post body
- `POST /api/v1/admin/import/form?site=site-id` - import comments from the backup, user post form
- `POST /api/v1/admin/remap?site=site-id` - remap comments to different URLs. Expect list of "from-url new-url" pairs separated by \n. From-url and new-url parts are separated by space. If URLs end with an asterisk (\*) it means matching by the prefix. Remap procedure based on export/import chain so make the backup first

```
http://oldsite.com* https://newsite.com*
http://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1
```

- `GET /api/v1/admin/wait?site=site-id` - wait for completion for any async migration ops (import or remap)
- `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin or unpin comment
- `GET /api/v1/admin/user/{userid}?site=site-id` - get user's info
- `DELETE /api/v1/admin/user/{userid}?site=site-id` - delete all user's comments
- `PUT /api/v1/admin/readonly?site=site-id&url=post-url&ro=1` - set read-only status
- `PUT /api/v1/admin/verify/{userid}?site=site-id&verified=1` - set verified status
- `GET /api/v1/admin/deleteme?token=token` - process deleteme user's request

_all admin calls require auth and admin privilege_
