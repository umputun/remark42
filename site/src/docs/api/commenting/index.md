---
title: Commenting
---


* `POST /api/v1/comment` - add a comment, _auth required_

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

* `POST /api/v1/preview` - preview comment in HTML. Body is `Comment` to render
* `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree|plain` - find all comments for given post

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

* `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit comment, allowed once in `EDIT_TIME` minutes since creation. Body is `EditRequest` JSON

```go
type EditRequest struct {
    Text    string `json:"text"`    // updated text
    Summary string `json:"summary"` // optional, summary of the edit
    Delete  bool   `json:"delete"`  // delete flag
}{}
```

* `GET /api/v1/last/{max}?site=site-id&since=ts-msec` - get up to `{max}` last comments, `since` (epoch time, milliseconds) is optional
* `GET /api/v1/id/{id}?site=site-id` - get comment by `comment id`
* `GET /api/v1/comments?site=site-id&user=id&limit=N` - get comment by `user id`, returns `response` object

```go
type response struct {
    Comments []store.Comment `json:"comments"`
    Count    int             `json:"count"`
}{}
```

* `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
* `POST /api/v1/count?site=siteID` - get number of comments for posts from post body (list of post IDs)
* `GET /api/v1/list?site=site-id&limit=5&skip=2` - list commented posts, returns array or `PostInfo`, limit=0 will return all posts

```go
type PostInfo struct {
    URL      string    `json:"url"`
    Count    int       `json:"count"`
    ReadOnly bool      `json:"read_only,omitempty"`
    FirstTS  time.Time `json:"first_time,omitempty"`
    LastTS   time.Time `json:"last_time,omitempty"`
}
```

* `GET /api/v1/user` - get user info, _auth required_
* `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decrease, _auth required_
* `GET /api/v1/userdata?site=site-id` - export all user data to gz stream, _auth required_
* `POST /api/v1/deleteme?site=site-id` - request deletion of user data, _auth required_
* `GET /api/v1/config?site=site-id` - returns configuration (parameters) for given site

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

* `GET /api/v1/info?site=site-idd&url=post-url` - returns `PostInfo` for site and URL
