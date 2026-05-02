package commands

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestHealthLevelFromCounts_Fallback(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   healthLevel
	}{
		{"empty is healthy", map[string]int{}, healthHealthy},
		{"3 blocked → at risk", map[string]int{"blocked": 3}, healthAtRisk},
		{"4 blocked → at risk", map[string]int{"blocked": 4}, healthAtRisk},
		{"1 blocked → warning", map[string]int{"blocked": 1}, healthWarning},
		{"only ready_for_approval → warning", map[string]int{"ready_for_approval": 2}, healthWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := healthLevelFromCounts(tt.counts, nil); got != tt.want {
				t.Errorf("healthLevelFromCounts(%v, nil) = %v, want %v", tt.counts, got, tt.want)
			}
		})
	}
}

func TestHealthLevelFromCounts_ConfigDriven(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"blocked":          {BlocksFeature: true},
			"ready_for_review": {BlocksFeature: true},
			"in_progress":      {BlocksFeature: false},
			"completed":        {BlocksFeature: false},
		},
	}

	tests := []struct {
		name   string
		counts map[string]int
		want   healthLevel
	}{
		{"all non-blocking → healthy", map[string]int{"in_progress": 5, "completed": 3}, healthHealthy},
		{"1 blocking → warning", map[string]int{"in_progress": 5, "ready_for_review": 1}, healthWarning},
		{"2 blocking across statuses → warning", map[string]int{"blocked": 1, "ready_for_review": 1}, healthWarning},
		{"3 blocking → at risk", map[string]int{"blocked": 3}, healthAtRisk},
		{"4 blocking across statuses → at risk", map[string]int{"blocked": 2, "ready_for_review": 2}, healthAtRisk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := healthLevelFromCounts(tt.counts, cfg); got != tt.want {
				t.Errorf("healthLevelFromCounts(%v, cfg) = %v, want %v", tt.counts, got, tt.want)
			}
		})
	}
}

func TestHealthSuffix(t *testing.T) {
	tests := []struct {
		level healthLevel
		want  string
	}{
		{healthHealthy, ""},
		{healthWarning, "!"},
		{healthAtRisk, "!!"},
	}
	for _, tt := range tests {
		if got := healthSuffix(tt.level); got != tt.want {
			t.Errorf("healthSuffix(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestColorByHealthLevel_NoColorReturnsRaw(t *testing.T) {
	for _, lvl := range []healthLevel{healthHealthy, healthWarning, healthAtRisk} {
		got := colorByHealthLevel("active", lvl, false)
		if got != "active" {
			t.Errorf("colorByHealthLevel(active, %v, false) = %q, want raw text", lvl, got)
		}
	}
}

func TestColorByHealthLevel_AppliesAnsiColor(t *testing.T) {
	// Palette: bright blue (healthy) / yellow (attention) / red (at-risk).
	// Picked for red-green colorblind distinguishability — blue and red
	// remain distinct where green would collapse against red.
	tests := []struct {
		level    healthLevel
		wantCode string
	}{
		{healthHealthy, cli.ColorBrightBlue},
		{healthWarning, cli.ColorYellow},
		{healthAtRisk, cli.ColorRed},
	}
	for _, tt := range tests {
		got := colorByHealthLevel("active", tt.level, true)
		if !strings.HasPrefix(got, tt.wantCode) {
			t.Errorf("colorByHealthLevel(active, %v, true) = %q, want prefix %q", tt.level, got, tt.wantCode)
		}
		if !strings.HasSuffix(got, cli.ColorReset) {
			t.Errorf("colorByHealthLevel(active, %v, true) = %q, want suffix ColorReset", tt.level, got)
		}
		if !strings.Contains(got, "active") {
			t.Errorf("colorByHealthLevel(active, %v, true) = %q, want to contain text", tt.level, got)
		}
	}
}

func TestHealthLegend_NoColor(t *testing.T) {
	got := healthLegend(false)
	for _, want := range []string{"!", "!!", "attention", "at-risk", "key"} {
		if !strings.Contains(got, want) {
			t.Errorf("healthLegend(false) missing %q; got %q", want, got)
		}
	}
}

func TestHealthLegend_ColorWrapsMarkers(t *testing.T) {
	got := healthLegend(true)
	if !strings.Contains(got, cli.ColorYellow) {
		t.Errorf("healthLegend(true) should color the warning marker; got %q", got)
	}
	if !strings.Contains(got, cli.ColorRed) {
		t.Errorf("healthLegend(true) should color the at-risk marker; got %q", got)
	}
}

// TestCalculateHealthIndicator_PreservesEmojiAPI is a regression guard:
// the JSON output uses calculateHealthIndicator and external consumers
// depend on the emoji string format.
func TestCalculateHealthIndicator_PreservesEmojiAPI(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   string
	}{
		{"healthy", map[string]int{"in_progress": 1}, "🟢"},
		{"warning", map[string]int{"blocked": 1}, "🟡"},
		{"at risk", map[string]int{"blocked": 3}, "🔴"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateHealthIndicator(tt.counts, nil); got != tt.want {
				t.Errorf("calculateHealthIndicator(%v) = %q, want %q", tt.counts, got, tt.want)
			}
		})
	}
}
