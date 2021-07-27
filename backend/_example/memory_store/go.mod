module github.com/umputun/remark42/memory_store

go 1.15

require (
	github.com/go-pkgz/jrpc v0.2.0
	github.com/go-pkgz/lgr v0.10.4
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/umputun/go-flags v1.5.1
	github.com/umputun/remark42/backend v1.8.1
)

replace github.com/umputun/remark42/backend => ../../
