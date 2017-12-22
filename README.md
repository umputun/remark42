# remark [![Build Status](http://drone.umputun.com:9080/api/badges/umputun/remark/status.svg)](http://drone.umputun.com:9080/umputun/remark)

Comment engine

## API

### Authorization

- `GET /login/{provider}?from=http://url` - login with one of supported providers and redirects to `url`
- `GET /logout` - logout 
- `GET /user` - returns user info, auth required

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

- `POST /comment` - adds a comment. auth required

```
type Comment struct {
	ID        int64     `json:"id"`     // read only
	ParentID  int64     `json:"pid"`    
	Text      string    `json:"text"`
	User      User      `json:"user"`   // read only
	Locator   Locator   `json:"locator"`
	Score     int       `json:"score"`  // read only
	Timestamp time.Time `json:"time"`   // read only
}
```

- `GET /find?url=post-url` - find all comments for given post return list of `Comment`
- `GET /last/{max}` - get last `{max}` comments
- `GET /id/{id}` - get comment by `id`

- `DELETE /comment/{id}` - delete comment by `id`. auth and admin required