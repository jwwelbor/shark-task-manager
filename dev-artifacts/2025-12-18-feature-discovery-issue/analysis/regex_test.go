package main

import (
	"fmt"
	"regexp"
)

func main() {
	// Feature patterns from config
	pattern1 := `^E(?P<epic_num>\d{2})-F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`
	pattern2 := `^F(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)$`

	testCases := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{
			name:     "Feature with F prefix and full name",
			pattern:  pattern2,
			input:    "F01-content-upload-security-implementation",
			expected: true,
		},
		{
			name:     "Intermediate folder",
			pattern:  pattern2,
			input:    "01-foundation",
			expected: false,
		},
		{
			name:     "Full epic+feature format",
			pattern:  pattern1,
			input:    "E01-F01-content-upload-security",
			expected: true,
		},
		{
			name:     "Just F prefix format against pattern2",
			pattern:  pattern2,
			input:    "F01-content-upload-security-implementation",
			expected: true,
		},
	}

	for _, tc := range testCases {
		re, err := regexp.Compile(tc.pattern)
		if err != nil {
			fmt.Printf("Error compiling regex: %v\n", err)
			continue
		}

		matches := re.FindStringSubmatch(tc.input)
		matched := matches != nil

		status := "✓"
		if matched != tc.expected {
			status = "✗"
		}

		fmt.Printf("%s Test: %s\n", status, tc.name)
		fmt.Printf("  Pattern: %s\n", tc.pattern)
		fmt.Printf("  Input:   %s\n", tc.input)
		fmt.Printf("  Expected: %v, Got: %v\n", tc.expected, matched)
		if matched {
			fmt.Printf("  Matches: %v\n", matches)
		}
		fmt.Println()
	}
}
