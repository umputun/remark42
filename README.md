# remark42 [![Build Status](https://travis-ci.org/umputun/remark.svg?branch=master)](https://travis-ci.org/umputun/remark) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/remark)](https://goreportcard.com/report/github.com/umputun/remark)

Remark42 is a self-hosted, lightweight, and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles or any other place where readers add comments.

* Social login via Google, Facebook and Github
* Multi-level nested comments with both tree and plain presentations
* Import from disqus
* Markdown support
* Moderator can remove comments and block users
* Voting and pinning system
* Sortable comments
* Extractor for recent comments, cross-post
* RSS for all comments and each post
* Export data to json with automatic backups
* No external databases, everything embedded in a single data file
* Fully dockerized and can be deployed in a single command
* Clean, lightweight and fully customizable UI
* Multi-site mode from a single instance
* Integration with automatic ssl via [nginx-le](https://github.com/umputun/nginx-le)

## Install

### Backend

* copy provided `docker-compose.yml` and customize for your needs
* prepare user id for container `` export USER=`id -u $USER` ``
* make sure you **don't keep** `DEV_PASSWD=something...` for any non-development deployments
* pull prepared images from docker hub and start - `docker-compose pull && docker-compose up -d`
* alternatively compile from sources - `docker-compose build && docker-compose up -d`

#### Parameters

| Command line      | Environment          | Default                | Multi            | Description                             |
| ----------------- | -------------------- | ---------------------- | ---------------- | --------------------------------------- |
| --url             | REMARK_URL           | `https://remark42.com` | no               | url to remark server                    |
| --bolt            | BOLTDB_PATH          | `/tmp`                 | no               | path to data directory                  |
| --site            | SITE                 | `remark`               | yes              | site name(s)                            |
| --admin           | ADMIN                |                        | yes              | admin names (list of user ids)          |
| --backup          | BACKUP_PATH          | `/tmp`                 | no               | backups location                        |
| --max-back        | MAX_BACKUP_FILES     | `10`                   | no               | max backup files to keep                |
| --max-cache-items | MAX_CACHE_ITEMS      | `1000`                 | no               | max number of cached items, 0-unlimited |
| --max-cache-value | MAX_CACHE_VALUE      | `65536`                | no               | max size of cached value, o-unlimited   |
| --secret          | SECRET               |                        | no               | secret key, required                    |
| --max-comment     | MAX_COMMENT_SIZE     | 2048                   | no               | comment's size limit                    |
| --google-cid      | REMARK_GOOGLE_CID    |                        | no               | Google OAuth client ID                  |
| --google-csec     | REMARK_GOOGLE_CSEC   |                        | no               | Google OAuth client secret              |
| --facebook-cid    | REMARK_FACEBOOK_CID  |                        | no               | Facebook OAuth client ID                |
| --facebook-csec   | REMARK_FACEBOOK_CSEC |                        | no               | Facebook OAuth client secret            |
| --github-cid      | REMARK_GITHUB_CID    |                        | no               | Github OAuth client ID                  |
| --github-csec     | REMARK_GITHUB_CSEC   |                        | no               | Github OAuth client secret              |
| --low-score       | LOW_SCORE            | `-5`                   | no               | Low score threshold                     |
| --critical-score  | CRITICAL_SCORE       | `-10`                  | no               | Critical score threshold                |
| --img-proxy       | IMG_PROXY            | `false`                | no               | Enable http->https proxy for images     |
| --dbg             | DEBUG                | `false`                | no               | debug mode                              |
| --dev-passwd      | DEV_PASSWD           |                        | no               | password for `dev` user                 |


**user has to provide secret key, can be any long and hard-to-guess string.**

_all multi parameters separated by `,` in environment or repeated with command line key, like `--site=s1 --site=s2 ...`_

#### Register oauth2 providers

Authentication handled by external providers. You should setup oauth2 for all (or some) of them to allow users to make comments. It is not mandatory to have all of them, but at least one should be correctly configured.

##### Google Auth Provider

1.  Create a new project: https://console.developers.google.com/project
1.  Choose the new project from the top right project dropdown (only if another project is selected)
1.  In the project Dashboard center pane, choose **"API Manager"**
1.  In the left Nav pane, choose **"Credentials"**
1.  In the center pane, choose **"OAuth consent screen"** tab. Fill in **"Product name shown to users"** and hit save.
1.  In the center pane, choose **"Credentials"** tab.
    * Open the **"New credentials"** drop down
    * Choose **"OAuth client ID"**
    * Choose **"Web application"**
    * Application name is freeform, choose something appropriate
    * Authorized origins is your domain ex: `https://remark42.mysite.com`
    * Authorized redirect URIs is the location of oauth2/callback constructed as domain + `/auth/google/callback`, ex: `https://remark42.mysite.com/auth/google/callback`
    * Choose **"Create"**
1.  Take note of the **Client ID** and **Client Secret**

_instructions for google oauth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

##### GitHub Auth Provider

1.  Create a new **"OAuth App"**: https://github.com/settings/developers
1.  Fill **"Application Name"** and **"Homepage URL"** for your site
1.  Under **"Authorization callback URL"** enter the correct url constructed as domain + `/auth/github/callback`. ie `https://remark42.mysite.com/auth/github/callback`
1.  Take note of the **Client ID** and **Client Secret**

##### Facebook Auth Provider

1.  From https://developers.facebook.com select **"My Apps"** / **"Add a new App"**
1.  Set **"Display Name"** and **"Contact email"**
1.  Choose **"Facebook Login"** and then **"Web"**
1.  Set "Site URL" to your domain, ex: `https://remark42.mysite.com`
1.  Under **"Facebook login"** / **"Settings"** fill "Valid OAuth redirect URIs" with your callback url constructed as domain + `/auth/facebook/callback`
1.  Select **"App Review"** and turn public flag on. This step may ask you to provide a link to your privacy policy.

#### Initial import from Disqus

1.  Disqus provides an export of all comments on your site in a g-zipped file. This is found in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready. Direct link to export will be something like `https://<siteud>.disqus.com/admin/discussions/export/`. See [importing-exporting](https://help.disqus.com/customer/portal/articles/1104797-importing-exporting) for more details.
2.  Move this file to your remark42 host within `.var` and unzip, i.e. `gunzip <disqus-export-name>.xml.gz`.
3.  Run import command - `docker-compose exec remark /srv/import-disqus.sh <disqus-export-name>.xml <your site id>`


#### Admin users

Admins/moderators should be defined in `docker-compose.yml` as a list of user IDs or passed in the command line. 

```
    environment:
        - ADMIN=github_ef0f706a79cc24b17bbbb374cd234a691a034128,github_dae9983158e9e5e127ef2b87a411ef13c891e9e5
```

To get user id just login and click on your username or any other user you want to promote to admins. 
It will expand login info and show full user ID.


### Frontend

Frontend part is building automatically along with backend if you use `docker-compose`.

For manual building:

* install [Node.js 8](https://nodejs.org/en/) or higher;
* run `npm install` inside `./web`;
* run `npm run build` there;
* result files will be saved in `./web/public`.

For development mode use `npm start` instead of `npm run build`.
In this case `webpack` will serve files using `webpack-dev-server` on `localhost:8080`.

URLs for development:

* `localhost:8080` — page with embedded script from `REMARK_URL` (default: `https://demo.remark42.com`);
* `localhost:8080/dev.html` — page with embedded script from local folder;
* `localhost:8080/last-comments.html` — page with embedded script for last comments;
* `localhost:8080/counter.html` — page with embedded script for counter with examples.

#### Usage

##### Comments

It's a main widget which renders list of comments. 

Add this snippet to the bottom of web page:

```html
<script>
  var remark_config = {
    site_id: 'YOUR_SITE_ID',
    url: 'PAGE_URL', // optional param; if url isn't defined window.location.href will be used 
  };

  (function() {
    var d = document, s = d.createElement('script');
    s.src = '/web/embed.js'; // prepend this address with domain where remark42 is placed
    (d.head || d.body).appendChild(s);
  })();
</script>
```

And then add this node in the place where you want to see Remark42 widget:

```html
<div id="remark42"></div>
``` 

After that widget will be rendered inside this node.

##### Last comments

It's a widget which renders list of last comments from your site.

Add this snippet to the bottom of web page:

```html
<script>
  var remark_config = {
    site_id: 'YOUR_SITE_ID', 
  };

  (function() {
    var d = document, s = d.createElement('script');
    s.src = '/web/last-comments.js'; // prepend this address with domain where remark42 is placed
    (d.head || d.body).appendChild(s);
  })();
</script>
```

And then add this node in the place where you want to see last comments widget:

```html
<div class="remark42__last-comments" data-max="50"></div>
```

`data-max` sets the max amount of comments (default: `15`).

##### Counter

It's a widget which renders a number of comments for the specified page.

Add this snippet to the bottom of web page:

```html
<script>
  var remark_config = {
    site_id: 'YOUR_SITE_ID', 
  };

  (function() {
    var d = document, s = d.createElement('script');
    s.src = '/web/counter.js'; // prepend this address with domain where remark42 is placed
    (d.head || d.body).appendChild(s);
  })();
</script>
```

And then add a node like this in the place where you want to see a number of comments:

```html
<span class="remark42__counter" data-url="https://domain.com/path/to/article/"></span>
```

You can use as many nodes like this as you need to. 
The script will found all them by the class `remark__counter`, 
and it will use `data-url` attribute to define the page with comments.

Also script can uses `url` property from `remark_config` object, or `window.location.href` if nothing else is defined. 

## API

### Authorization

* `GET /auth/{provider}/login?from=http://url` - perform "social" login with one of supported providers and redirect to `url`
* `GET /auth/logout` - logout

```go
type User struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Picture string `json:"picture"`
    Admin   bool   `json:"admin"`
    Blocked bool   `json:"block"`
}
```

_currently supported providers are `google`, `facebook` and `github`_

### Commenting

* `POST /api/v1/comment` - add a comment. _auth required_

```go
type Comment struct {
    ID        string          `json:"id"`      // comment ID, read only
    ParentID  string          `json:"pid"`     // parent ID
    Text      string          `json:"text"`    // comment text, after md processing
    Orig      string          `json:"orig"`    // original comment text
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

* `POST /api/v1/preview` - preview comment in html. Body is `Comment` to render

* `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree|plain` - find all comments for given post

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

Sort can be `time`, `active` or `score`. Supported sort order with prefix -/+, i.e. `-time`. For `tree` mode sort will be applied to top-level comments only and all replies always sorted by time.

* `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit comment, allowed once in 5min since creation

```json
  Content-Type: application/json

  {
    "text": "edit comment blah http://radio-t.com 12345",
    "summary": "fix blah"
  }
```

* `GET /api/v1/last/{max}?site=site-id` - get up to `{max}` last comments
* `GET /api/v1/id/{id}?site=site-id` - get comment by `comment id`
* `GET /api/v1/comments?site=site-id&user=id&limit=N` - get comment by `user id`, returns `response` object
  ```go
  type response struct {
      Comments []store.Comment  `json:"comments"`
      Count    int              `json:"count"`
  }{}
  ```
* `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
* `GET /api/v1/list?site=site-id&limit=5&skip=2` - list commented posts, returns array or `PostInfo`, limit=0 will return all posts
  ```go
  type PostInfo struct {
      URL   string `json:"url"`
      Count int    `json:"count"`
  }
  ```
* `GET /api/v1/user` - get user info, _auth required_
* `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decrease. _auth required_
* `GET /api/v1/config?site=site-id` - returns configuration (parameters) for given site

  ```go
  type config struct {
      Version       string   `json:"version"`
      EditDuration  int      `json:"edit_duration"` // seconds
      Admins        []string `json:"admins"`
      Auth          []string `json:"auth_providers"`
      LowScore      int      `json:"low_score"`
      CriticalScore int      `json:"critical_score"`
  }
  ``` 
  
### RSS feeds
  
* `GET /api/v1/rss/post?site=site-id&url=post-url` - rss feed for a post
* `GET /api/v1/rss/site?site=site-id` - rss feed for given site

### Admin

* `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`.
* `PUT /api/v1/admin/user/{userid}?site=site-id&block=1` - block or unblock user.
* `GET api/v1/admin/blocked&site=site-id` - list of blocked user ids.
  ```go
  type BlockedUser struct {
      ID        string    `json:"id"`
      Name      string    `json:"name"`
      Timestamp time.Time `json:"time"`
  }
  ```
* `GET /api/v1/admin/export?site=side-id&mode=[stream|file]` - export all comments to json stream or gz file.
* `POST /api/v1/admin/import?site=side-id` - import comments from the backup.
* `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin or unpin comment.

_all admin calls require auth and admin privilege_

## Technical details

* Data stored in [boltdb](https://github.com/boltdb/bolt) (embedded key/value database) files under `BOLTDB_PATH`
* Each site stored in a separate boltbd file.
* In order to migrate/move remark42 to another host boltbd files should be transferred.
* Automatic backup process runs every 24h and exports all content in json-like format to `backup-remark-YYYYMMDD.gz`.
* Sessions implemented with [gorilla/sessions](https://github.com/gorilla/sessions) and file-system store under `SESSION_STORE` path. It uses HttpOnly, secure cookies.
* All heavy REST calls cached internally, default expiration 4h
* User's activity throttled globally (up to 1000 simultaneous requests) and limited locally (per user, up to 10 req/sec)
* Request timeout set to 60sec
* Development mode (`--dev-password` set) allows to test remark42 without social login and with admin privileges. Adds basic-auth for username: `dev`, password: `${DEV_PASSWD}`. **should not be used in production deployment**
* User can vote for the comment multiple times but only to change his/her vote. Double-voting not allowed.
* User can edit comments in 5 mins window after creation.
* User ID hashed and prefixed by oauth provider name to avoid collisions and potential abuse.
* All avatars cached locally to prevent rate limiters from google/github/facebook.
* Docker build uses [publicly available](https://github.com/umputun/baseimage) base images.
