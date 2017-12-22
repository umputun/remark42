# remark [![Build Status](http://drone.umputun.com:9080/api/badges/umputun/remark/status.svg)](http://drone.umputun.com:9080/umputun/remark)

Comment engine

## API

### Authorization

- `GET /login/{provider}?from=http://url` - perform "social" login with one of supported providers and redirect to `url`
- `GET /logout` - logout 
- `GET /user` - get user info, _auth required_

```
type User struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Picture string `json:"picture"`
	Profile string `json:"profile"`
	Admin   bool   `json:"admin"`
}
```

_currently supported providers are `google` and `github`_

### Commenting

- `POST /comment` - add a comment. _auth required_

```
type Comment struct {
	ID        int64           `json:"id"`      // read only
	ParentID  int64           `json:"pid"`    
	Text      string          `json:"text"`
	User      User            `json:"user"`    // read only
	Locator   Locator         `json:"locator"`
	Score     int             `json:"score"`   // read only
	Votes     map[string]bool `json:"votes"`   // read only
	Timestamp time.Time       `json:"time"`    // read only
}

type Locator struct {
	SiteID string `json:"site"`
	URL    string `json:"url"`
}
```

- `GET /find?url=post-url` - find all comments for given post returns list of `Comment`
- `GET /last/{max}` - get last `{max}` comments
- `GET /id/{id}` - get comment by `id`
- `PUT /vote/{id}?url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decreases. _auth required_
- `DELETE /comment/{id}` - delete comment by `id`. _auth and admin required_