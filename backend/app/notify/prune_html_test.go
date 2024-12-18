package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringArr_Push(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		push     string
		expected []string
		expLen   int
	}{
		{
			name:     "push to empty array",
			initial:  []string{},
			push:     "hello",
			expected: []string{"hello"},
			expLen:   5,
		},
		{
			name:     "push to non-empty array",
			initial:  []string{"hello"},
			push:     "world",
			expected: []string{"hello", "world"},
			expLen:   10,
		},
		{
			name:     "push empty string",
			initial:  []string{"hello"},
			push:     "",
			expected: []string{"hello", ""},
			expLen:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stringArr{data: tt.initial, len: 0}
			for _, str := range tt.initial {
				s.len += len(str)
			}

			s.Push(tt.push)

			assert.Equal(t, tt.expected, s.data)
			assert.Equal(t, tt.expLen, s.len)
		})
	}
}

func TestStringArr_Pop(t *testing.T) {
	tests := []struct {
		name        string
		initial     []string
		expectedPop string
		remaining   []string
		expLen      int
	}{
		{
			name:        "pop from array with multiple elements",
			initial:     []string{"hello", "world"},
			expectedPop: "world",
			remaining:   []string{"hello"},
			expLen:      5,
		},
		{
			name:        "pop from array with one element",
			initial:     []string{"hello"},
			expectedPop: "hello",
			remaining:   []string{},
			expLen:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stringArr{data: tt.initial, len: 0}
			for _, str := range tt.initial {
				s.len += len(str)
			}

			popped := s.Pop()

			assert.Equal(t, tt.expectedPop, popped)
			assert.Equal(t, tt.remaining, s.data)
			assert.Equal(t, tt.expLen, s.len)
		})
	}
}

func TestStringArr_Unshift(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		unshift  string
		expected []string
		expLen   int
	}{
		{
			name:     "unshift to empty array",
			initial:  []string{},
			unshift:  "hello",
			expected: []string{"hello"},
			expLen:   5,
		},
		{
			name:     "unshift to non-empty array",
			initial:  []string{"world"},
			unshift:  "hello",
			expected: []string{"hello", "world"},
			expLen:   10,
		},
		{
			name:     "unshift empty string",
			initial:  []string{"world"},
			unshift:  "",
			expected: []string{"", "world"},
			expLen:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stringArr{data: tt.initial, len: 0}
			for _, str := range tt.initial {
				s.len += len(str)
			}

			s.Unshift(tt.unshift)

			assert.Equal(t, tt.expected, s.data)
			assert.Equal(t, tt.expLen, s.len)
		})
	}
}

func TestStringArr_Shift(t *testing.T) {
	tests := []struct {
		name          string
		initial       []string
		expectedShift string
		remaining     []string
		expLen        int
	}{
		{
			name:          "shift from array with multiple elements",
			initial:       []string{"hello", "world"},
			expectedShift: "hello",
			remaining:     []string{"world"},
			expLen:        5,
		},
		{
			name:          "shift from array with one element",
			initial:       []string{"hello"},
			expectedShift: "hello",
			remaining:     []string{},
			expLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stringArr{data: tt.initial, len: 0}
			for _, str := range tt.initial {
				s.len += len(str)
			}

			shifted := s.Shift()

			assert.Equal(t, tt.expectedShift, shifted)
			assert.Equal(t, tt.remaining, s.data)
			assert.Equal(t, tt.expLen, s.len)
		})
	}
}

func TestStringArr_String(t *testing.T) {
	tests := []struct {
		name     string
		data     []string
		expected string
	}{
		{
			name:     "empty array",
			data:     []string{},
			expected: "",
		},
		{
			name:     "single element",
			data:     []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple elements",
			data:     []string{"hello", " ", "world"},
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stringArr{data: tt.data}
			assert.Equal(t, tt.expected, s.String())
		})
	}
}

func TestPruneHTML(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		maxLength int
		expected  string
	}{
		{
			name:      "simple text within limit",
			html:      "<p>Hello</p>",
			maxLength: 20,
			expected:  "<p>Hello</p>",
		},
		{
			name:      "text exceeding limit",
			html:      "<p>Hello world, this is a long text</p>",
			maxLength: 15,
			expected:  "<p>...</p>",
		},
		{
			name:      "nested tags within limit",
			html:      "<div><p>Hello</p></div>",
			maxLength: 30,
			expected:  "<div><p>Hello</p></div>",
		},
		{
			name:      "nested tags exceeding limit",
			html:      "<div><p>Hello world</p><p>More text</p></div>",
			maxLength: 20,
			expected:  "<div>...</div>",
		},
		{
			name:      "with comment",
			html:      "<!-- comment --><p>Hello</p>",
			maxLength: 20,
			expected:  "<p>Hello</p>",
		},
		{
			name:      "self-closing tag",
			html:      "<p>Hello<br/>World</p>",
			maxLength: 20,
			expected:  "<p>Hello<br/>...</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pruneHTML(tt.html, tt.maxLength)
			assert.Equal(t, tt.expected, result)
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
		{
			name:      "within limit",
			text:      "hello world",
			maxLength: 15,
			expected:  "hello world",
		},
		{
			name:      "exact limit",
			text:      "hello world",
			maxLength: 11,
			expected:  "hello",
		},
		{
			name:      "cut at word boundary",
			text:      "hello world and more",
			maxLength: 11,
			expected:  "hello",
		},
		{
			name:      "zero length",
			text:      "hello",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "negative length",
			text:      "hello",
			maxLength: -1,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pruneStringToWord(tt.text, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}
