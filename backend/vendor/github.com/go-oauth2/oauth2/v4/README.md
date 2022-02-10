# Golang OAuth 2.0 Server

> An open protocol to allow secure authorization in a simple and standard method from web, mobile and desktop applications.

[![Build][build-status-image]][build-status-url] [![Codecov][codecov-image]][codecov-url] [![ReportCard][reportcard-image]][reportcard-url] [![GoDoc][godoc-image]][godoc-url] [![License][license-image]][license-url]

## Protocol Flow

```text
     +--------+                               +---------------+
     |        |--(A)- Authorization Request ->|   Resource    |
     |        |                               |     Owner     |
     |        |<-(B)-- Authorization Grant ---|               |
     |        |                               +---------------+
     |        |
     |        |                               +---------------+
     |        |--(C)-- Authorization Grant -->| Authorization |
     | Client |                               |     Server    |
     |        |<-(D)----- Access Token -------|               |
     |        |                               +---------------+
     |        |
     |        |                               +---------------+
     |        |--(E)----- Access Token ------>|    Resource   |
     |        |                               |     Server    |
     |        |<-(F)--- Protected Resource ---|               |
     +--------+                               +---------------+
```

## Quick Start

### Download and install

```bash
go get -u -v github.com/go-oauth2/oauth2/v4/...
```

### Create file `server.go`

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
)

func main() {
	manager := manage.NewDefaultManager()
	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// client memory store
	clientStore := store.NewClientStore()
	clientStore.Set("000000", &models.Client{
		ID:     "000000",
		Secret: "999999",
		Domain: "http://localhost",
	})
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleTokenRequest(w, r)
	})

	log.Fatal(http.ListenAndServe(":9096", nil))
}

```

### Build and run

```bash
go build server.go

./server
```

### Open in your web browser
**Authorization Request**:
[http://localhost:9096/authorize?client_id=000000&response_type=code](http://localhost:9096/authorize?client_id=000000&response_type=code)

**Grant Token Request**:
[http://localhost:9096/token?grant_type=client_credentials&client_id=000000&client_secret=999999&scope=read](http://localhost:9096/token?grant_type=client_credentials&client_id=000000&client_secret=999999&scope=read)

```json
{
  "access_token": "J86XVRYSNFCFI233KXDL0Q",
  "expires_in": 7200,
  "scope": "read",
  "token_type": "Bearer"
}
```

## Features

- Easy to use
- Based on the [RFC 6749](https://tools.ietf.org/html/rfc6749) implementation
- Token storage support TTL
- Support custom expiration time of the access token
- Support custom extension field
- Support custom scope
- Support jwt to generate access tokens

## Example

> A complete example of simulation authorization code model

Simulation examples of authorization code model, please check [example](/example)

### Use jwt to generate access tokens

```go

import (
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/dgrijalva/jwt-go"
)

// ...
manager.MapAccessGenerate(generates.NewJWTAccessGenerate("", []byte("00000000"), jwt.SigningMethodHS512))

// Parse and verify jwt access token
token, err := jwt.ParseWithClaims(access, &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("parse error")
	}
	return []byte("00000000"), nil
})
if err != nil {
	// panic(err)
}

claims, ok := token.Claims.(*generates.JWTAccessClaims)
if !ok || !token.Valid {
	// panic("invalid token")
}
```

## Store Implements

- [BuntDB](https://github.com/tidwall/buntdb)(default store)
- [Redis](https://github.com/go-oauth2/redis)
- [MongoDB](https://github.com/go-oauth2/mongo)
- [MySQL](https://github.com/go-oauth2/mysql)
- [MySQL (Provides both client and token store)](https://github.com/imrenagi/go-oauth2-mysql)
- [PostgreSQL](https://github.com/vgarvardt/go-oauth2-pg)
- [DynamoDB](https://github.com/contamobi/go-oauth2-dynamodb)
- [XORM](https://github.com/techknowlogick/go-oauth2-xorm)
- [XORM (MySQL, client and token store)](https://github.com/rainlay/go-oauth2-xorm)
- [GORM](https://github.com/techknowlogick/go-oauth2-gorm)
- [Firestore](https://github.com/tslamic/go-oauth2-firestore)

## Handy Utilities

- [OAuth2 Proxy Logger (Debug utility that proxies interfaces and logs)](https://github.com/aubelsb2/oauth2-logger-proxy)

## MIT License

Copyright (c) 2016 Lyric

[build-status-url]: https://travis-ci.org/go-oauth2/oauth2
[build-status-image]: https://travis-ci.org/go-oauth2/oauth2.svg?branch=master
[codecov-url]: https://codecov.io/gh/go-oauth2/oauth2
[codecov-image]: https://codecov.io/gh/go-oauth2/oauth2/branch/master/graph/badge.svg
[reportcard-url]: https://goreportcard.com/report/github.com/go-oauth2/oauth2/v4
[reportcard-image]: https://goreportcard.com/badge/github.com/go-oauth2/oauth2/v4
[godoc-url]: https://godoc.org/github.com/go-oauth2/oauth2/v4
[godoc-image]: https://godoc.org/github.com/go-oauth2/oauth2/v4?status.svg
[license-url]: http://opensource.org/licenses/MIT
[license-image]: https://img.shields.io/npm/l/express.svg
