// Package redact removes known credentials from diagnostic output.
package redact

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

// JSON returns a log-only copy with known Slack credential fields redacted,
// including fields nested inside arrays and objects. Other content is preserved.
// Invalid JSON is omitted because its credential fields cannot be identified.
func JSON(data []byte) string {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "[invalid JSON]"
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return "[invalid JSON]"
	}
	redactFields(value)
	redacted, err := json.Marshal(value)
	if err != nil {
		return "[unavailable JSON]"
	}
	return string(redacted)
}

func redactFields(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			switch strings.ToLower(key) {
			case "token", "bot_access_token", "action_token", "response_url":
				value[key] = "[REDACTED]"
			default:
				redactFields(child)
			}
		}
	case []any:
		for _, child := range value {
			redactFields(child)
		}
	}
}

// URLOrigin retains the scheme and host for connection diagnostics, omitting
// user information, paths, queries, and fragments that could contain credentials.
func URLOrigin(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "[invalid URL]"
	}
	return u.Scheme + "://" + u.Host
}
