
[![Build status][travis-img]][travis-url]
[![License][license-img]][license-url]
[![GoDoc][doc-img]][doc-url]

### rewrite

golang URL rewriting

### Usage

```go
import "github.com/pkg4go/rewrite"

// ...

handler := rewrite.NewHandler(map[string]string{
  "/a": "/b",
  "/api/(.*)", "/api/v1/$1",
  "/api/(.*)/actions/(.*)", "/api/v1/$1/actions/$2",
  "/from/:one/to/:two", "/from/:two/to/:one",
})

// ...
```

### License
MIT

[travis-img]: https://img.shields.io/travis/pkg4go/rewrite.svg?style=flat-square
[travis-url]: https://travis-ci.org/pkg4go/rewrite
[license-img]: https://img.shields.io/badge/license-MIT-green.svg?style=flat-square
[license-url]: http://opensource.org/licenses/MIT
[doc-img]: https://img.shields.io/badge/GoDoc-reference-blue.svg?style=flat-square
[doc-url]: http://godoc.org/github.com/pkg4go/rewrite
