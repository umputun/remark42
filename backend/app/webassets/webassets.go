// Package webassets holds the files served under /web that the frontend build does not produce,
// embedded into the binary: images, the emoji table, standalone pages, the widget's own document and
// the pages that host the widget (the demo page among them), which are plain files loading the scripts
// and styles the build does produce, and counter.mjs, a script written by hand.
// A file of the same name in the frontend output, on disk under --web-root or embedded at
// app/cmd/web, is served instead, which is how an operator replaces one of these.
// Email and error-page templates are a separate set and live in app/templates.
package webassets

import (
	"embed"
	"io/fs"
)

//go:embed assets
var embedded embed.FS

// FS holds the assets, each named by its path under /web. fs.Sub cannot fail for a constant
// valid path on an embed.FS, so the error is dropped the same way app/cmd/web's is.
var FS, _ = fs.Sub(embedded, "assets")
