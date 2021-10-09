---
title: Authorization
---

* `GET /auth/{provider}/login?from=http://url&site=site_id&session=1` - perform "social", anonymous or email login with one of [supported providers](#register-oauth2-providers) and redirect to `url`. Presence of `session` (any non-zero value) change the default cookie expiration and makes them session-only
* `GET /auth/logout` - logout

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
