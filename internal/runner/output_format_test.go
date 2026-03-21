package runner

import (
	"strings"
	"testing"
)

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "short string no truncation",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length no truncation",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "over length gets truncated with ellipsis",
			input:  "hello world",
			maxLen: 8,
			want:   "hello w…",
		},
		{
			name:   "single newline replaced with space",
			input:  "hello\nworld",
			maxLen: 20,
			want:   "hello world",
		},
		{
			name:   "multiple newlines replaced with spaces",
			input:  "line1\nline2\nline3",
			maxLen: 30,
			want:   "line1 line2 line3",
		},
		{
			name:   "multiline over length truncated after flattening",
			input:  "line1\nline2\nline3",
			maxLen: 10,
			want:   "line1 lin…",
		},
		{
			name:   "carriage return stripped",
			input:  "hello\r\nworld",
			maxLen: 20,
			want:   "hello world",
		},
		{
			name:   "maxLen of 1 returns single ellipsis for long string",
			input:  "hello",
			maxLen: 1,
			want:   "…",
		},
		{
			name:   "maxLen below 1 clamped to 1",
			input:  "hello",
			maxLen: 0,
			want:   "…",
		},
		{
			name:   "unicode runes counted correctly",
			input:  "café",
			maxLen: 3,
			want:   "ca…",
		},
		{
			name:   "typical agent output line short enough",
			input:  "Task completed successfully.",
			maxLen: 120,
			want:   "Task completed successfully.",
		},
		{
			name:   "long agent output truncated to 120",
			input:  strings.Repeat("x", 200),
			maxLen: 120,
			want:   strings.Repeat("x", 119) + "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateOutput(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateOutput(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
