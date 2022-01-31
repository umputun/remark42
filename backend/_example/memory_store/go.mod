module github.com/umputun/remark42/memory_store

go 1.16

require (
	github.com/go-pkgz/jrpc v0.2.0
	github.com/go-pkgz/lgr v0.10.4
	github.com/jessevdk/go-flags v1.5.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/umputun/remark42/backend v1.9.0
)

replace github.com/umputun/remark42/backend => ../../
