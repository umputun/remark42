---
worth: later
where: backend/app/rest/api/middleware_test.go:246
added: 2026-08-18
---
# TestRest_securityHeaders deadlocks the whole api suite on macOS

`cd backend/app && go test -timeout=60s -count 1 ./...` fails on macOS with the `app/rest/api` package
timing out. It reproduced 3 times out of 3 in isolation with
`go test -run 'TestRest_securityHeaders$' -timeout=40s`, so it is deterministic rather than flaky, and it
takes the whole package down with it: 14 of 15 packages pass, that one never finishes. CI is unaffected,
the `Tests` job passes on Linux.

The test blocks in `startupT`'s teardown (`rest_test.go:504`), inside `httptest.Server.Close()`, which
waits on the server's `WaitGroup` for handlers still in flight. The handler it is waiting on sits in IO
wait on a socket write:

```
internal/poll.(*FD).Write  ... waitWrite
net/http.(*response).write(..., 0x3c2c13, ...)
api.addFileServer.Timeout.func3.1  vendor/github.com/go-pkgz/rest/timeout.go:67
```

`0x3c2c13` is 3,942,931 bytes. So the file server is writing a ~3.9MB response to a client that has
stopped reading, because the test already moved on to closing the server. On Linux the write fits in the
socket buffers and completes; on macOS the buffers are smaller, the write blocks, and `Close()` waits
forever.

Not caused by the `golang.org/x/image` bump in the same session: that changed only
`golang.org/x/image`, `golang.org/x/sys`, `golang.org/x/text` and `vendor/modules.txt`, while every
package in the stack above (`go-pkgz/rest`, `didip/tollbooth`, `go-pkgz/routegroup`) is untouched.

Worth fixing because it makes the documented local test command unusable on a Mac, and the failure mode
is an eight minute silent hang rather than an error. Likely fix is on the test side: have the client
drain or close the response body before teardown, or stop serving a multi-megabyte asset from that
fixture. `[Unverified]` which of those it is, the handler was not traced past the write.
