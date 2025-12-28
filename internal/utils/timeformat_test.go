package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatRelativeTime(t *testing.T) {
	// Use a fixed time to avoid timing issues in tests
	now := time.Date(2025, 12, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now (5 seconds ago)",
			time:     now.Add(-5 * time.Second),
			expected: "just now",
		},
		{
			name:     "30 seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "30 seconds ago",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-90 * time.Second),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-90 * time.Minute),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-30 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "1 week ago",
			time:     now.Add(-10 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "2 weeks ago",
			time:     now.Add(-14 * 24 * time.Hour),
			expected: "2 weeks ago",
		},
		{
			name:     "1 month ago",
			time:     now.Add(-35 * 24 * time.Hour),
			expected: "1 month ago",
		},
		{
			name:     "3 months ago",
			time:     now.Add(-90 * 24 * time.Hour),
			expected: "3 months ago",
		},
		{
			name:     "1 year ago",
			time:     now.Add(-400 * 24 * time.Hour),
			expected: "1 year ago",
		},
		{
			name:     "2 years ago",
			time:     now.Add(-800 * 24 * time.Hour),
			expected: "2 years ago",
		},
		{
			name:     "future time (2 hours from now)",
			time:     now.Add(2 * time.Hour),
			expected: "2 hours from now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRelativeTimeFrom(tt.time, now)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "5 seconds",
			duration: 5 * time.Second,
			expected: "just now",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "30 seconds",
		},
		{
			name:     "90 seconds (1.5 minutes)",
			duration: 90 * time.Second,
			expected: "1 minute",
		},
		{
			name:     "5 minutes",
			duration: 5 * time.Minute,
			expected: "5 minutes",
		},
		{
			name:     "90 minutes (1.5 hours)",
			duration: 90 * time.Minute,
			expected: "1 hour",
		},
		{
			name:     "5 hours",
			duration: 5 * time.Hour,
			expected: "5 hours",
		},
		{
			name:     "30 hours (1.25 days)",
			duration: 30 * time.Hour,
			expected: "1 day",
		},
		{
			name:     "3 days",
			duration: 72 * time.Hour,
			expected: "3 days",
		},
		{
			name:     "10 days (1.4 weeks)",
			duration: 240 * time.Hour,
			expected: "1 week",
		},
		{
			name:     "14 days (2 weeks)",
			duration: 336 * time.Hour,
			expected: "2 weeks",
		},
		{
			name:     "35 days (1.16 months)",
			duration: 840 * time.Hour,
			expected: "1 month",
		},
		{
			name:     "90 days (3 months)",
			duration: 2160 * time.Hour,
			expected: "3 months",
		},
		{
			name:     "400 days (1.09 years)",
			duration: 9600 * time.Hour,
			expected: "1 year",
		},
		{
			name:     "800 days (2.19 years)",
			duration: 19200 * time.Hour,
			expected: "2 years",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
