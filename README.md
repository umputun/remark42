# remark42 [![Build Status](https://drone.umputun.com/api/badges/umputun/remark/status.svg)](https://drone.umputun.com/umputun/remark)

Remark42 is a self-hosted, lightweight, and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles or any other place where readers add comments.

- Social login via Google, Facebook and Github
- Multi-level nested comments with both tree and plain presentations
- Import from disqus
- Markdown support
- Moderator can remove comments and block users
- Voting and pinning system
- Sortable comments
- Extractor for recent comments, cross-post
- Export data to json with automatic backups
- No external databases, everything embedded in a single data file
- Fully dockerized and can be deployed in a single command
- Clean, lightweight and fully customizable UI
- Multi-site mode from a single instance
- Integration with automatic ssl via [nginx-le](https://github.com/umputun/nginx-le)

## Install

### Backend

- copy provided `docker-compose.yml` and customize for your needs
- make sure you **don't keep** `DEV=true` for any non-development deployments
- pull prepared images from docker hub and start - `docker-compose pull && docker compose up -d`
- alternatively compile from sources - `docker-compose build`

#### Parameters

| Command line    | Environment          | Default                | Multi | Scope  | Description                     |
| --------------- | -------------------- | ---------------------- | ----- | ------ | ------------------------------- |
| --url           | REMARK_URL           | `https://remark42.com` | no    | all    | url to remark server            |
| --bolt          | BOLTDB_PATH          | `/tmp`                 | no    | all    | path to data directory          |
| --dbg           | DEBUG                | `false`                | no    | all    | debug mode                      |
| --dev           | DEV                  | `false`                | no    | all    | development mode, no auth!      |
| --site          | SITE                 | `remark`               | yes   | server | site name(s)                    |
| --admin         | ADMIN                |                        | yes   | server | admin(s) names (user id)        |
| --backup        | BACKUP_PATH          | `/tmp`                 | no    | server | backups location                |
| --max-back      | MAX_BACKUP_FILES     | `10`                   | no    | server | max backup files to keep        |
| --session       | SESSION_STORE        | `/tmp`                 | no    | server | path to session store directory |
| --store-key     | STORE_KEY            | `secure-store-key`     | no    | server | session store encryption key    |
| --google-cid    | REMARK_GOOGLE_CID    |                        | no    | server | Google OAuth client ID          |
| --google-csec   | REMARK_GOOGLE_CSEC   |                        | no    | server | Google OAuth client secret      |
| --facebook-cid  | REMARK_FACEBOOK_CID  |                        | no    | server | Facebook OAuth client ID        |
| --facebook-csec | REMARK_FACEBOOK_CSEC |                        | no    | server | Facebook OAuth client secret    |
| --github-cid    | REMARK_GITHUB_CID    |                        | no    | server | Github OAuth client ID          |
| --github-csec   | REMARK_GITHUB_CSEC   |                        | no    | server | Github OAuth client secret      |
| --provider      |                      | `disqus`               | no    | import | provider type for import        |
| --site          |                      | `remark`               | no    | import | site ID                         |
| --file          |                      | `disqus.xml`           | no    | import | import file                     |

#### Run modes

- `server` activates regular, server mode
- `import` performs import from external providers (disqus and internal json, see `/api/v1/admin/export`)

#### Register oauth2 providers

Authentication handled by external providers. You should setup oauth2 for all (or some) of them in order to allow users to make comments. It is not mandatory to have all of them, but at least one should be property configured.

##### Google Auth Provider

1. Create a new project: https://console.developers.google.com/project
1. Choose the new project from the top right project dropdown (only if another project is selected)
1. In the project Dashboard center pane, choose **"API Manager"**
1. In the left Nav pane, choose **"Credentials"**
1. In the center pane, choose **"OAuth consent screen"** tab. Fill in **"Product name shown to users"** and hit save.
1. In the center pane, choose **"Credentials"** tab.
   - Open the **"New credentials"** drop down
   - Choose **"OAuth client ID"**
   - Choose **"Web application"**
   - Application name is freeform, choose something appropriate
   - Authorized origins is your domain ex: `https://remark42.mysite.com`
   - Authorized redirect URIs is the location of oauth2/callback constructed as domain + `/auth/google/callback`, ex: `https://remark42.mysite.com/auth/google/callback`
   - Choose **"Create"**
1. Take note of the **Client ID** and **Client Secret**

_instructions for google oauth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

##### GitHub Auth Provider

1. Create a new **"OAuth App"**: https://github.com/settings/developers 
1. Fill **"Application Name"** and **"Homepage URL"** for your site
1. Under **"Authorization callback URL"** enter the correct url constructed as domain + `/auth/github/callback`. ie `https://remark42.mysite.com/auth/github/callback`
1. Take note of the **Client ID** and **Client Secret**

##### Facebook Auth Provider

1. From https://developers.facebook.com select **"My Apps"** / **"Add a new App"**
1. Set **"Display Name"** and **"Contact email"**
1. Choose **"Facebook Login"** and then **"Web"**
1. Set "Site URL" to your domain, ex: `https://remark42.mysite.com`
1. Under **"Facebook login"** / **"Settings"** fill "Valid OAuth redirect URIs" with your callback url constructed as domain + `/auth/facebook/callback`
1. Select **"App Review"** and turn public flag on. This step may ask you to provide a link to your privacy policy.

#### Initial import from disqus

1. Disqus provides an export of all comments on your site in a g-zipped file. This is found in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready.
2. Move this file to your remark42 host and unzip.
3. Stop remark42 containers if started, i.e. `docker-compose stop` 
4. Run import command - `docker-compose run --rm /srv/remark import --file=<disqus-export-xml> --site=<your site id>`
5. Start remark42 containers `docker-compose up -d`

### Frontend

TBD

## API

### Authorization

- `GET /auth/{provider}/login?from=http://url` - perform "social" login with one of supported providers and redirect to `url`
- `GET /auth/logout` - logout

```go
type User struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Picture string `json:"picture"`
    Profile string `json:"profile"`
    Admin   bool   `json:"admin"`
    Blocked bool   `json:"block"`
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
    Delete    bool            `json:"delete"`  // delete status, read only
}

type Locator struct {
    SiteID string `json:"site"`     // site id
    URL    string `json:"url"`      // post url
}
```

- `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree` - find all comments for given post

This is the primary call used by UI to show comments for given post. It can return comments in two formats - `plain` and `tree`.
In plain format result will be sorted list of `Comment`. In tree format this is going to be tree-like object with this structure:

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

- `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit comment, allowed once in 5min since creation
  ```
  Content-Type: application/json
  
  {
    "text": "edit comment blah http://radio-t.com 12345",
    "summary": "fix blah"
  }
  ```

- `GET /api/v1/last/{max}?site=site-id` - get up to `{max}` last comments
- `GET /api/v1/id/{id}?site=site-id` - get comment by `comment id`
- `GET /api/v1/comments?site=site-id&user=id` - get comment by `user id`, returns `response` object
    ```go
    type response struct {
        Comments []store.Comment  `json:"comments"`
        Count    int              `json:"count"`
    }{}
    ```
- `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
- `GET /api/v1/list?site=site-id` - list commented posts, returns array or `PostInfo`
    ```go
    type PostInfo struct {
        URL   string `json:"url"`
        Count int    `json:"count"`
    }
    ```
- `GET /api/v1/user` - get user info, _auth required_
- `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decrease. _auth required_
- `GET /api/v1/config?site=siteID` - returns configuration (parameters) for given site
    ```go
    type config struct {
        Version      string   `json:"version"`
        EditDuration int      `json:"edit_duration"` // seconds
        Admins       []string `json:"admins"`
        Auth         []string `json:"auth_providers"`
    }
    ```
- `PUT /api/v1/notify?site=site-id&url=post-url&action=1` - subscribe(action=1)/unsubscribe(0) comment notifications. _auth required_

### Admin

- `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`.
- `PUT /api/v1/admin/user/{userid}?site=site-id&block=1` - block or unblock user.
- `GET api/v1/admin/blocked&site=site-id` - list of blocked user ids.
    ```go
    type BlockedUser struct {
        ID        string    `json:"id"`
        Timestamp time.Time `json:"time"`
    }
    ```
- `GET /api/v1/admin/export?site=side-id&mode=[stream|file]` - export all comments to json stream or gz file.
- `POST /api/v1/admin/import?site=side-id` - import comments from the backup.
- `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin or unpin comment.

_all calls require auth and admin_

## Technical details

- Data stored in [boltdb](https://github.com/boltdb/bolt) (embedded key/value database) files under `BOLTDB_PATH`
- Each site stored in a separate boltbd file.
- In order to migrate/move remark42 to another host boltbd files should be transferred.
- Automatic backup process runs every 24h and exports all content in json-like format to `backup-remark-YYYYMMDD.gz`.
- Sessions implemented with [gorilla/sessions](https://github.com/gorilla/sessions) and file-system store under `SESSION_STORE` path. It uses HttpOnly, secure cookies.
- All heavy REST calls cached internally, default expiration 4h
- Users activity throttled globally (up to 1000) and limited locally (per user, up to 10 req/sec)
- Request timeout set to 60sec
- Development mode (`--dev`) allows to test remark42 without login and with admin privileges. **should not be used in production deployment**
- User can vote for the comment multiple times but only to change his/her vote. Double-voting not allowed.
- User can edit comments in 5 mins window after creation.
- User ID prefixed by oauth provider name in order to avoid collisions and potential abuse.
- All avatars cached locally in order to prevent rate limiters from google/github/facebook.
