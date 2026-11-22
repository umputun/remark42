---
title: API
---

## Authentication

A request is authenticated in one of three ways:

- the `JWT` cookie set at login, with the `X-XSRF-TOKEN` header carrying the value of the `XSRF-TOKEN` cookie. An expired cookie token is renewed;
- the token in the `X-JWT` header or in the `jwt` query parameter, with no XSRF header. Such a token must not be expired, and it takes precedence over the cookie, so a request carrying one does not renew the cookie session. With `AUTH_SEND_JWT_HEADER` set, a response that issues or renews a token carries it in the `X-JWT` header, which the built-in CORS policy does not expose to cross-origin scripts;
- Basic auth as user `admin` with the `ADMIN_PASSWD` password, which makes the request an admin of every site.

Calls marked _auth required_ and all admin calls need `site` in the query, and it has to be the site the user signed in to; otherwise the answer is 403. Basic-auth requests are exempt from this check. Under `/api/v1`, a Basic-auth request that the browser marks as cross-site, or that carries an `Origin` other than the `REMARK_URL` origin, is rejected with 403.

Anonymous users get 403 from the email and Telegram subscription calls, `POST /api/v1/deleteme` and `POST /api/v1/picture`, and from `PUT /api/v1/vote/{id}` unless both `ANON_VOTE` and `VOTES_IP` are set.

With `SUBSCRIBERS_ONLY` set, creating, editing and previewing comments, voting, `POST /api/v1/deleteme` and the email and Telegram subscription calls get 403 for anyone who is not a paid Patreon subscriber, the Basic-auth admin included.

## Errors and limits

Requests are rate-limited per client address and URL path, so that, for example, each comment ID has its own counter. The limits in requests per second are:

- 2 under `/auth/`;
- 10 on the open `/api/v1` calls, on `GET /api/v1/user` and `GET /api/v1/userdata`, on `/api/v1/img`, `/api/v1/picture/{user}/{id}` and `/api/v1/qr/telegram`, and on the admin calls;
- `UPDATE_LIMIT` (0.5 by default, 10 when it is 0) on the other calls that need auth: comment changes, preview, votes, picture upload, email and Telegram subscription, and `POST /api/v1/deleteme`;
- 100 on avatars, 50 on `/robots.txt` and `/email/unsubscribe.html`, and 20 on `/web/`.

A request over the limit gets 429 with a plain-text body.

Request bodies are limited to 64 KiB for comment creation, editing and preview and for email subscription and confirmation, to 128 KiB for `POST /api/v1/counts`, to 32 MiB for a picture upload and to 256 MiB for `POST /api/v1/admin/import/form`.

Most `/api/v1` handlers report an error with an HTTP status and a JSON body, `{"error": "error text", "details": "what failed", "code": 4}`. Avatar errors carry only `error`, and a `GET /api/v1/admin/wait` timeout carries `status` and `site_id`. Rejections by the authentication, admin, site and anonymous-user checks have a plain-text body, and a request that runs out of its time limit gets 504 with an empty body. The `code` values are:

```go
const (
    ErrInternal             = 0  // any internal error
    ErrCommentNotFound      = 1  // can't find comment
    ErrDecode               = 2  // failed to unmarshal incoming request
    ErrNoAccess             = 3  // rejected by auth
    ErrCommentValidation    = 4  // validation failed
    ErrPostNotFound         = 5  // can't find post
    ErrSiteNotFound         = 6  // can't find site
    ErrUserBlocked          = 7  // user blocked
    ErrReadOnly             = 8  // write failed on read only
    ErrCommentRejected      = 9  // general error on rejected comment change
    ErrCommentEditExpired   = 10 // too late for edit
    ErrCommentEditChanged   = 11 // parent comment cannot be changed
    ErrVoteRejected         = 12 // general error on vote rejected
    ErrVoteSelf             = 13 // vote for own comment
    ErrVoteDbl              = 14 // already voted for the comment
    ErrVoteMax              = 15 // too many votes for the comment
    ErrVoteMinScore         = 16 // min score reached for the comment
    ErrActionRejected       = 17 // general error for rejected actions
    ErrAssetNotFound        = 18 // requested file not found
    ErrCommentRestrictWords = 19 // restricted words in a comment
    ErrImgNotFound          = 20 // posted image not found in the storage
)
```

## Authorization

- `GET /auth/{provider}/login?from=http://url&site=site_id&session=1` - perform "social" login with one of [supported providers](https://remark42.com/docs/configuration/authorization/#oauth-providers) and redirect to `url`. The host of `url` has to be the `REMARK_URL` host (any port) or one of `ALLOWED_HOSTS`, where entries with `*` do not count; for another host, the login completes without the redirect. A non-empty `session` value other than `0` makes the cookies session-only instead of using the default expiration
- `GET /auth/email/login?site=site-id&user=name&address=user@example.org` - send a sign-in token to the address, with `AUTH_EMAIL_ENABLE` set; see [email authentication](https://remark42.com/docs/configuration/email/). Returns `{"user": "name", "address": "user@example.org"}`. `GET /auth/email/login?token=token` then signs the user in, within 30 minutes; a token used once gets 403
- `GET /auth/telegram/login?site=site-id` - start Telegram sign-in, with `AUTH_TELEGRAM` set. Returns `{"token": "token", "bot": "bot-name"}`; the user sends `/start token` to the bot. `GET /auth/telegram/login?site=site-id&token=token` then signs the user in. Both steps have to finish within 10 minutes; the call answers 404 while the bot has no message, and for an unknown, expired or already used token
- `GET /auth/anonymous/login?user=name&aud=site-id` - sign in anonymously, with `AUTH_ANON` set. The [authorization page](https://remark42.com/docs/configuration/authorization/#anonymous) describes the accepted names
- `GET /auth/user` - the signed-in user as stored in the token, 401 without a valid token
- `GET /auth/status` - returns `{"status": "logged in", "user": "name"}` or `{"status": "not logged in"}`
- `GET /auth/list` - list of enabled providers
- `GET /auth/logout` - logout

## Commenting

- `POST /api/v1/comment?site=site-id` - add a comment, _auth required_. A reply sets `pid` to the ID of the comment it answers, which must be a comment on the same post; otherwise the request is rejected with status 400. The body carries Markdown in `text`, the post's `locator`, and optionally `pid` and `title`; the server sets the other fields itself, `orig` included. Except for the Basic-auth admin, `locator.site` has to be the user's site. Returns status 201 with the stored `Comment`. A blocked user or a read-only post gets 403. The text gets 400 when it is empty, when its length in Unicode code points is outside `MIN_COMMENT_SIZE` and `MAX_COMMENT_SIZE` (2048 when that is 0 or less), or when a non-empty Markdown link or image destination does not start with `http://`, `https://` or `mailto:`

```go
type Comment struct {
    ID          string    `json:"id"`                    // comment ID, read only
    ParentID    string    `json:"pid"`                   // parent ID
    Text        string    `json:"text"`                  // comment text, after md processing
    Orig        string    `json:"orig,omitempty"`        // original comment text in Markdown, should never be rendered as HTML as-is!
    User        User      `json:"user"`                  // user info, read only
    Locator     Locator   `json:"locator"`               // post locator
    Score       int       `json:"score"`                 // comment score, read only
    Vote        int       `json:"vote"`                  // vote for the current user, -1/1/0
    Controversy float64   `json:"controversy,omitempty"` // comment controversy, read only
    Timestamp   time.Time `json:"time"`                  // time stamp, read only
    Edit        *Edit     `json:"edit,omitempty"`        // pointer to have empty default in JSON response
    Pin         bool      `json:"pin,omitempty"`         // pinned status, read only
    Deleted     bool      `json:"delete,omitempty"`      // delete status, read only
    Imported    bool      `json:"imported,omitempty"`    // imported from a backup, read only
    PostTitle   string    `json:"title,omitempty"`       // post title
}

type Locator struct {
    SiteID string `json:"site,omitempty"` // site ID
    URL    string `json:"url"`            // post URL
}

type Edit struct {
    Timestamp time.Time `json:"time"`
    Summary   string    `json:"summary"`
}
```

- `POST /api/v1/preview?site=site-id` - preview comment in HTML, _auth required_. Body is `Comment` to render. The text gets the length, link and stored-picture checks of a new comment, but not the restricted-words check, and the answer is the sanitised HTML
- `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree|plain` - find all comments for given post

This call returns the comments of a post. The plain format returns `{"comments": [...], "info": {...}}`, the sorted `Comment` list and the post's `PostInfo`. With `format=tree`, it returns this structure:

```go
type treeWithInfo struct {
    *service.Tree
    Info store.PostInfo `json:"info"`
}

type Tree struct {
    Nodes []*Node `json:"comments"`
}

type Node struct {
    Comment store.Comment `json:"comment"`
    Replies []*Node       `json:"replies,omitempty"`
}
```

Sort can be `time`, `active`, `score` or `controversy`. Supported sort order with prefix -/+, i.e., `-time`. For `tree` mode, the sort will be applied to top-level comments only, and all replies are always sorted by time.

The call takes more optional parameters:

- `view=user` fills only `id` and `user` of each comment that is not deleted; the other fields keep zero values, so in the `tree` format every comment becomes a top-level node.
- `since` (epoch time, milliseconds) returns only newer comments, and `info.count` then counts the matching comments that are not deleted, before paging. The `tree` format ignores it.
- `limit`, when positive, returns the first `limit` comments; the `tree` format returns whole top-level threads, and always at least one. `info.count_left` is the number of comments after the page, deleted placeholders the result keeps included, and `info.last_comment` is the ID of the last returned comment, top-level in the `tree` format.
- `offset_id` returns the comments after the one with this ID, so `info.last_comment` passed as `offset_id` gives the next page. An ID not in the result starts from the beginning, and the `tree` format matches top-level comments only.

In the `tree` format, the first page of a request with `limit` (no `offset_id`) also carries `pinned`, the post's pinned comments that are not deleted, oldest first, wherever their pages are.

Without `url`, the call returns up to 1000 latest comments of the site that are not deleted.

- `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit or delete own comment, _auth required_, any number of times within `EDIT_TIME` since creation (a duration, `5m` by default; `0` removes the limit) and only while the comment has no reply that is not deleted; the check looks at the latest 1000 comments of the site. With `ADMIN_EDIT` set, an admin edits own comments without these limits. Another user's comment gets 403, and a deleted comment cannot be edited. Body is `EditRequest` JSON; `delete` set deletes the comment. Returns the updated `Comment`. The new text is checked as a new comment's is: it must not be empty, must fit `MIN_COMMENT_SIZE` and `MAX_COMMENT_SIZE`, and the destinations of its Markdown links and images must start with `http://`, `https://` or `mailto:`. A request with `delete` set skips this check, even when it carries text.

```go
type EditRequest struct {
    Text    string `json:"text"`    // updated text
    Summary string `json:"summary"` // optional, summary of the edit
    Delete  bool   `json:"delete"`  // delete flag
}
```

- `GET /api/v1/last/{max}?site=site-id&since=ts-msec` - get up to `{max}` last comments, newest first, `since` (epoch time, milliseconds) is optional. `{max}` is 1000 at most, and `0` means 1000. Deleted comments are left out
- `GET /api/v1/id/{id}?site=site-id&url=post-url` - get comment by `comment id`. `url` is required: the comment is looked up among the post's comments, and without `url` the answer is 400
- `GET /api/v1/comments?site=site-id&user=id&limit=N&skip=M` - get comments by `user id`, newest first, returns `userComments` object. `limit` is 500 at most, and `0` means 500; `skip` skips that many comments. Deleted comments are left out of the list and of `count`.

**Important**: original comment text in Markdown in the `orig` field should never be rendered as HTML as-is, only `text` containing HTML is sanitized and safe for render.

```go
type userComments struct {
    Comments []store.Comment `json:"comments"`
    Count    int             `json:"count"`
}
```

- `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`, returns `{"count": N, "locator": {"site": "site-id", "url": "post-url"}}`
- `POST /api/v1/counts?site=site-id` - get number of comments for posts from post body, a JSON array of post URLs of 128 KB at most. Returns an array of `PostInfo` with `url` and `count`; a URL that cannot be counted is left out
- `GET /api/v1/list?site=site-id&limit=5&skip=2` - list commented posts, returns array of `PostInfo`, limit=0 will return all posts

```go
type PostInfo struct {
    URL         string    `json:"url,omitempty"`
    Count       int       `json:"count"`
    CountLeft   int       `json:"count_left"`             // used only with returning search results limited by number, otherwise zero
    LastComment string    `json:"last_comment,omitempty"` // used only with returning search results limited by number
    ReadOnly    bool      `json:"read_only,omitempty"`
    FirstTS     time.Time `json:"first_time"`
    LastTS      time.Time `json:"last_time"`
}
```

- `GET /api/v1/user?site=site-id` - get user info, _auth required_

```go
type User struct {
    Name              string `json:"name"`
    ID                string `json:"id"`
    Picture           string `json:"picture"`
    IP                string `json:"ip,omitempty"` // hashed client IP, in comments shown to site admins only
    Admin             bool   `json:"admin"`
    Blocked           bool   `json:"block,omitempty"`
    Verified          bool   `json:"verified,omitempty"`
    EmailSubscription bool   `json:"email_subscription,omitempty"`
    SiteID            string `json:"site_id,omitempty"`
    PaidSub           bool   `json:"paid_sub,omitempty"` // is paid Patreon subscriber
}
```

- `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment, _auth required_. `vote=1` increases the score, and any other value, a missing one included, decreases it. A vote opposite to the user's earlier one cancels it, and a second one casts the opposite vote. Returns `{"id": "comment-id", "score": N}`. A vote for one's own comment, a repeated vote in the same direction, and a vote refused by `MAX_VOTES`, `POSITIVE_SCORE` or `VOTES_IP` get 400; a blocked user or a read-only post gets 403
- `GET /api/v1/userdata?site=site-id` - export user data to gz stream, _auth required_. The document is `{"info": {...}, "details": {...}, "comments": [...]}`: user info, notification email and Telegram (`details` is left out when neither is stored), and up to 10,000 of the user's comments, newest first. Deleted comments are left out
- `POST /api/v1/deleteme?site=site-id` - request deletion of user data, _auth required_. The call deletes nothing itself. It returns `{"site": "site-id", "user_id": "user-id", "token": "token", "link": "url"}`, and the data is deleted when an admin opens `link` within three months
- `GET /api/v1/config?site=site-id` - returns configuration (parameters) for given site. `edit_duration` is in seconds

```go
type Config struct {
    Version               string   `json:"version"`
    EditDuration          int      `json:"edit_duration"`
    AdminEdit             bool     `json:"admin_edit"`
    MinCommentSize        int      `json:"min_comment_size"`
    MaxCommentSize        int      `json:"max_comment_size"`
    Admins                []string `json:"admins"`
    AdminEmail            string   `json:"admin_email"`
    Auth                  []string `json:"auth_providers"`
    AnonVote              bool     `json:"anon_vote"`
    LowScore              int      `json:"low_score"`
    CriticalScore         int      `json:"critical_score"`
    PositiveScore         bool     `json:"positive_score"`
    ReadOnlyAge           int      `json:"readonly_age"`
    MaxImageSize          int      `json:"max_image_size"`
    EmailNotifications    bool     `json:"email_notifications"`
    TelegramNotifications bool     `json:"telegram_notifications"`
    EmojiEnabled          bool     `json:"emoji_enabled"`
    SimpleView            bool     `json:"simple_view"`
    SendJWTHeader         bool     `json:"send_jwt_header"`
    SubscribersOnly       bool     `json:"subscribers_only"` // enable commenting only for Patreon subscribers
}
```

- `GET /api/v1/info?site=site-id&url=post-url` - returns `PostInfo` for site and URL

## Streaming API

<details><summary>Not available</summary>
Streaming API supposed to provide server-sent events for post updates as well as a site update:

- `GET /api/v1/stream/info?site=site-id&url=post-url&since=unix_ts_msec` - returns stream (`event: info`) with `PostInfo` records for the site and URL. `since` is optional
- `GET /api/v1/stream/last?site=site-id&since=unix_ts_msec` - returns updates stream (`event: last`) with comments for the site, `since` is optional

It was removed in https://github.com/umputun/remark42/pull/826 due to not being used and affecting tests flakiness and could be returned if there will be a developer who would be willing to write frontend support for it.

Response example of `/stream/info`:

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

- `GET /api/v1/picture/{user}/{id}` - load stored image. A malformed or unknown ID gets 400, and stored content that is not an image gets 415
- `POST /api/v1/picture?site=site-id` - upload and store image, uses post form with `FormFile("file")`. Returns `{"id": "user/imgid"}`, _auth required_. Accepts GIF, JPEG, PNG and WebP. A file larger than `IMAGE_MAX_SIZE` bytes, reported as `max_image_size` in `/api/v1/config`, a file in another format, a malformed image or one over 16,777,216 pixels is rejected with 400; a form that cannot be parsed, a request over 32 MiB included, gets 500

_returned ID should be appended to load image URL on the caller side_

- `GET /api/v1/img?src=encoded-url` - load an external image through the server; `src` is the image URL in URL-safe base64 with padding
- `GET /api/v1/avatar/{id}` - load a user's avatar, also served as `/avatar/{id}`

## Email Subscription

- `GET /api/v1/email?site=site-id` - get user's email, _auth required_
- `POST /api/v1/email/subscribe?site=site-id` - makes confirmation token and sends it to the user over email, _auth required_. Body is `{"address": "user@example.org"}`. The token is valid for 30 minutes. A user signed in by email with the same address is subscribed at once, and the answer carries `"updated": true`. The older form sends an empty body and `address` in the query, and always sends the token

  Trying to subscribe to the same email a second time will return response code `409 Conflict` with explaining error message

- `POST /api/v1/email/confirm?site=site-id` - uses the token from the body, `{"token": "token"}`, to set email for the user, _auth required_. The older form sends an empty body and `tkn` in the query

  With email notifications configured, a user with an email set is notified about every reply in the threads under their comments, at any depth, except their own replies

- `DELETE /api/v1/email?site=site-id` - removes user's email, _auth required_
- `GET /email/unsubscribe.html?site=site-id&tkn=token` - unsubscribe from the link in a notification email, answers with an HTML page. `POST` with the same query does the same

## Telegram Subscription

- `GET /api/v1/telegram/subscribe?site=site-id` - start a subscription to notifications over Telegram, _auth required_. Returns `{"token": "token", "bot": "bot-name"}`; the user sends `/start token` to the bot. A user who already has a subscription gets 409, and a server without Telegram notifications answers 500
- `GET /api/v1/telegram/subscribe?site=site-id&tkn=token` - complete the subscription, _auth required_, as the user who started it and within 10 minutes of the first call. Returns `{"updated": true, "address": "telegram-id"}`. 404 means the bot has no message yet, the token is unknown, expired or already used, or another user sent the request
- `DELETE /api/v1/telegram?site=site-id` - removes user's Telegram subscription, _auth required_
- `GET /api/v1/qr/telegram?url=https://t.me/...` - PNG with the QR code of a `https://t.me/` link, used for Telegram sign-in and subscription

## Admin

- `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`
- `PUT /api/v1/admin/user/{userid}?site=site-id&block=1&ttl=168h` - block (`block=1`) or unblock (any other value) a user. `ttl` is optional and uses Go duration syntax with units up to hours (`30m`, `24h`, `168h` for a week); there is no day unit. Without `ttl`, or with `ttl=0`, the block is permanent, and a permanent block also deletes all comments of the user. A `ttl` that does not parse, such as `7d`, or is negative is rejected with status 400 and changes nothing. Returns `{"user_id": "...", "site_id": "...", "block": true}`.
- `GET /api/v1/admin/blocked?site=site-id` - list of blocked users

```go
type BlockedUser struct {
    ID    string    `json:"id"`
    Name  string    `json:"name"`
    Until time.Time `json:"time"`
}
```

- `GET /api/v1/admin/export?site=site-id&mode=[stream|file]` - export all comments to JSON stream or gz file
- `POST /api/v1/admin/import?site=site-id&provider=native` - import comments from the backup, uses post body. `provider` is `native` (the default when omitted), `disqus`, `wordpress` or `commento`; any other value is rejected with 400. `remark` is accepted as `native`. Returns 202 once the upload is saved and runs the import in the background; while another import or remap runs on the site, the answer is 409
- `POST /api/v1/admin/import/form?site=site-id&provider=native` - import comments from the backup sent as `multipart/form-data` in the part named `file`, `provider` and responses as above
- `POST /api/v1/admin/remap?site=site-id` - remap comments to different URLs. Expect a list of "from-url new-url" pairs separated by \n. From-url and new-url parts are separated by space. If URLs end with an asterisk (\*), it means matching the prefix. Remap procedure based on export/import chain so make the backup first

```
http://oldsite.com* https://newsite.com*
http://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1
```

- `GET /api/v1/admin/wait?site=site-id&timeout=15m` - wait for completion for any async migration ops (import or remap). `timeout` is a Go duration, `15m` by default; when it runs out, the answer is 504. `"status": "completed"` means the operation has stopped, not that it succeeded; failures are reported in the server log
- `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin (`pin=1`) or unpin (any other value) comment
- `GET /api/v1/admin/user/{userid}?site=site-id` - get user's info, taken from the user's latest comment that is not deleted; a user without one gets 400
- `DELETE /api/v1/admin/user/{userid}?site=site-id` - delete the user's comments and stored details; succeeds even if the user has no comments or is already absent. The comments stay as deleted placeholders without the text and the user's name. The pictures the user uploaded that the user's comments show are removed too, except those that another user's comment on the same posts shows. The avatar is removed when a comment that is not deleted gives its URL
- `PUT /api/v1/admin/readonly?site=site-id&url=post-url&ro=1` - set (`ro=1`) or clear (any other value) read-only status. A post that is read-only by `READONLY_AGE` cannot be cleared and gets 403
- `PUT /api/v1/admin/verify/{userid}?site=site-id&verified=1` - set (`verified=1`) or clear (any other value) verified status
- `PUT /api/v1/admin/title/{id}?site=site-id&url=post-url` - set the comment's `title` to the title of the post page
- `GET /api/v1/admin/deleteme?site=site-id&token=token` - process a user's deleteme request; a token older than three months gets 403, and already-deleted or dataless users return success (idempotent). Removes the comments, stored details and pictures as `DELETE /api/v1/admin/user/{userid}` does, and the avatar named in the token

_all admin calls require auth and admin privilege_

## Other

- `GET /ping` - returns `pong`
- `GET /robots.txt` - rules for crawlers
