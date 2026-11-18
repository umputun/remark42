package store

import "regexp"

// logMask replaces the values MaskForLog hides
const logMask = "********"

// reLogEmail matches an email address, plain or with "@" still URL-encoded, as found in a logged URL or request
// body. The domain needs no dot, so an address cut short by body truncation is still caught; a mistyped one with
// several "@" is caught whole; and the local part stops at "/", so a page URL like https://example.com/@handle is
// left alone.
var reLogEmail = regexp.MustCompile(`[^\s\p{Z}"<>()\[\]{},;:/?&=#@\\]+(?:(?:@|%40)+[^\s\p{Z}"'<>()\[\]{},;:/?&=#@\\]+)+`)

// reLogJWT matches a JWT, including one cut short by body truncation. Every JWT starts with "eyJ", the encoded
// `{"` of its header, and its payload may carry an email address, readable by anyone who decodes it.
var reLogJWT = regexp.MustCompile(`eyJ[\w-]+\.[\w-]+(?:\.[\w-]*)?`)

// MaskForLog hides email addresses and JWTs in a log line built from user input
func MaskForLog(s string) string {
	s = reLogJWT.ReplaceAllLiteralString(s, logMask)
	return reLogEmail.ReplaceAllLiteralString(s, logMask)
}
