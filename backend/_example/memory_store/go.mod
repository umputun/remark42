module github.com/umputun/remark42/memory_store

go 1.14

require (
	github.com/go-pkgz/auth v1.15.0 // indirect
	github.com/go-pkgz/expirable-cache v0.0.3 // indirect
	github.com/go-pkgz/jrpc v0.2.0
	github.com/go-pkgz/lcw v0.8.1 // indirect
	github.com/go-pkgz/lgr v0.10.4
	github.com/go-pkgz/repeater v1.1.3 // indirect
	github.com/go-pkgz/rest v1.9.2 // indirect
	github.com/go-pkgz/syncs v1.1.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/umputun/go-flags v1.5.1
	github.com/umputun/remark42/backend v1.7.1
)

replace github.com/umputun/remark42/backend => ../../
