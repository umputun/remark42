module github.com/umputun/remark/memory_store

go 1.12

require (
	github.com/go-pkgz/jrpc v0.1.0
	github.com/go-pkgz/lgr v0.6.3
	github.com/jessevdk/go-flags v1.4.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/umputun/remark/backend v1.4.0
)

replace github.com/umputun/remark/backend => ../../
