module github.com/umputun/remark/backend/_example/memory_store

go 1.12

require (
	github.com/go-pkgz/jrpc v0.1.0
	github.com/go-pkgz/lgr v0.6.3
	github.com/jessevdk/go-flags v0.0.0-20180331124232-1c38ed7ad0cc
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/umputun/remark/backend v1.4.0
)

replace github.com/umputun/remark/backend => ../../

replace gopkg.in/russross/blackfriday.v2 => github.com/russross/blackfriday/v2 v2.0.1
