package commands

import "testing"

func TestAbbreviateStatus_Known(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		// Universal terminal / pause
		{"completed", "DONE"},
		{"cancelled", "CANC"},
		{"blocked", "BLOK"},
		{"on_hold", "HOLD"},
		{"draft", "DRFT"},
		{"todo", "TODO"},

		// in_*
		{"in_progress", "PROG"},
		{"in_development", "DEV"},
		{"in_code_review", "CR"},
		{"in_qa", "QA"},

		// ready_for_*
		{"ready_for_development", "R-DEV"},
		{"ready_for_code_review", "R-CR"},
		{"ready_for_qa", "R-QA"},
		{"ready_for_approval", "R-APR"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := abbreviateStatus(tt.in)
			if got != tt.want {
				t.Errorf("abbreviateStatus(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAbbreviateStatus_AllAbbreviationsAreShort(t *testing.T) {
	// Guardrail: every hand-tuned abbreviation must fit a 5-char column.
	const maxLen = 5
	for status, abbrev := range knownStatusAbbreviations {
		if len(abbrev) > maxLen {
			t.Errorf("abbreviation for %q is %q (len %d), want ≤%d",
				status, abbrev, len(abbrev), maxLen)
		}
		if abbrev == "" {
			t.Errorf("abbreviation for %q is empty", status)
		}
	}
}

func TestAbbreviateStatus_FallbackUnknown(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		// Unknown in_* status — strip prefix, take 4 chars
		{"in_something_new", "SOME"},
		// Unknown ready_for_* — prepend R-, take 4 chars of payload
		{"ready_for_something", "R-SOME"},
		// No prefix — take 4 chars
		{"unknownstatus", "UNKN"},
		// Empty input
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := abbreviateStatus(tt.in)
			if got != tt.want {
				t.Errorf("abbreviateStatus(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAbbreviateStatus_NoCollisionsAcrossKnownInStates(t *testing.T) {
	// Verify hand-tuned in_* and ready_for_* statuses never produce the
	// same abbreviation — collisions defeat the purpose.
	seen := map[string]string{}
	for status, abbrev := range knownStatusAbbreviations {
		if prev, dup := seen[abbrev]; dup {
			t.Errorf("collision: %q and %q both abbreviate to %q",
				prev, status, abbrev)
		}
		seen[abbrev] = status
	}
}
