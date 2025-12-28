package utils

import (
	"fmt"
	"math"
	"time"
)

// FormatRelativeTime formats a timestamp as a relative time string (e.g., "2 hours ago")
func FormatRelativeTime(t time.Time) string {
	return FormatRelativeTimeFrom(t, time.Now())
}

// FormatRelativeTimeFrom formats a timestamp relative to a given reference time
// This is useful for testing with a fixed reference time
func FormatRelativeTimeFrom(t time.Time, from time.Time) string {
	duration := from.Sub(t)

	// Future times
	if duration < 0 {
		duration = -duration
		formatted := formatDuration(duration)
		if formatted == "just now" {
			return "just now"
		}
		return formatted + " from now"
	}

	formatted := formatDuration(duration)
	if formatted == "just now" {
		return "just now"
	}
	return formatted + " ago"
}

// formatDuration converts a duration to a human-readable string
func formatDuration(d time.Duration) string {
	seconds := d.Seconds()

	// Less than a minute
	if seconds < 60 {
		if seconds < 10 {
			return "just now"
		}
		return fmt.Sprintf("%d seconds", int(seconds))
	}

	// Less than an hour
	minutes := seconds / 60
	if minutes < 60 {
		if minutes < 2 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", int(minutes))
	}

	// Less than a day
	hours := minutes / 60
	if hours < 24 {
		if hours < 2 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", int(hours))
	}

	// Less than a week
	days := hours / 24
	if days < 7 {
		if days < 2 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", int(days))
	}

	// Less than a month
	weeks := days / 7
	if weeks < 4 {
		if weeks < 2 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", int(weeks))
	}

	// Less than a year
	months := days / 30
	if months < 12 {
		if months < 2 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", int(months))
	}

	// Years
	years := math.Floor(days / 365)
	if years < 2 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", int(years))
}
