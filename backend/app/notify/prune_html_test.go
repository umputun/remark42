package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPruneHTML(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		maxLength int
		expected  string
	}{
		{"within limit", "<p>Hello</p>", 20, "<p>Hello</p>"},
		{"exceeds limit", "<p>Hello world, this is a long text</p>", 15, "<p>Hello world,...</p>"},
		{"nested tags", "<div><p>Hello world</p><p>More text</p></div>", 20, "<div><p>Hello world</p><p>More...</p></div>"},
		{"html comment stripped", "<!-- comment --><p>Hello</p>", 20, "<p>Hello</p>"},
		{"self-closing tag", "<p>Hello<br/>World</p>", 8, "<p>Hello<br/>...</p>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pruneHTML(tt.html, tt.maxLength))
		})
	}
}

func TestPruneStringToWord(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{"within limit", "hello world", 15, "hello world"},
		{"cut at word boundary", "hello world and more", 11, "hello world"},
		{"zero length", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pruneStringToWord(tt.text, tt.maxLength))
		})
	}
}
