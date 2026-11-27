package store

import (
	"regexp"
	"strings"
)

// logMask replaces the values MaskForLog hides
const logMask = "********"

// reLogEmail matches an email address, plain or with "@" still URL-encoded, as found in a logged URL or request
// body. The domain needs no dot, so an address cut short by body truncation is still caught; a mistyped one with
// several "@" is caught whole; and the local part stops at "/", so a page URL like https://example.com/@handle is
// left alone. Both classes accept "*" on purpose: MaskForLog runs the JWT pass first, and a JWT-like run inside an
// address ("RileyJane.Doe@gmail.com" holds "eyJane.Doe") turns into a mask the email pass has to match across, or
// the rest of the address stays in the log.
//
// The first alternative is a quoted local part, as in "john"@example.com, also with the closing quote escaped the
// way a JSON body carries it. It takes no space, colon, comma, equals sign or bracket, so the quote that closes one
// item or phrase and the quote that opens the next, before a mention, are not read as one. The price is that a
// quoted local part with a space in it is not matched.
var reLogEmail = regexp.MustCompile(`(?:"[^"\\:,={}\[\]<>\s\p{Z}]+\\?"|[^\s\p{Z}"<>()\[\]{},;:/?&=#@\\]+)` +
	`(?:(?:@|%40)+[^\s\p{Z}"'<>()\[\]{},;:/?&=#@\\]+)+`)

// reLogJWT matches a JWT, including one cut short by body truncation. Every JWT starts with "eyJ", the encoded
// `{"` of its header, and its payload may carry an email address, readable by anyone who decodes it.
var reLogJWT = regexp.MustCompile(`eyJ[\w-]+\.[\w-]+(?:\.[\w-]*)?`)

// logLineBreaks maps every character that can start a new line to a space, the same set the request logger of
// go-pkgz/rest collapses. Request data can carry line breaks, also URL-encoded ones that unescaping the logged
// URL turns back into raw ones, and each would start a log record written by whoever sent the request.
var logLineBreaks = strings.NewReplacer("\n", " ", "\r", " ", "\v", " ", "\f", " ", "\u0085", " ", "\u2028", " ", "\u2029", " ")

// MaskForLog makes a log line built from user input safe to write: it hides email addresses and JWTs and keeps
// the line a single line
func MaskForLog(s string) string {
	s = reLogJWT.ReplaceAllLiteralString(s, logMask)
	s = reLogEmail.ReplaceAllLiteralString(s, logMask)
	// after masking, as the masks match some of these characters inside an address
	return logLineBreaks.Replace(s)
}
