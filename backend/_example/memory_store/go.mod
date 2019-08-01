module github.com/umputun/remark/memory_store

go 1.12

require (
	github.com/go-pkgz/jrpc v0.1.0
	github.com/go-pkgz/lgr v0.6.3
	github.com/jessevdk/go-flags v1.4.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/umputun/remark/backend v1.4.0
	golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3 // indirect
)

replace github.com/umputun/remark/backend => ../../

replace gopkg.in/russross/blackfriday.v2 => github.com/russross/blackfriday/v2 v2.0.1
