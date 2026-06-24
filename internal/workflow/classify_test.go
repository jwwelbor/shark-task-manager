package workflow

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// customNamedRouteService builds a route-based Service whose step names are all
// non-standard. This proves the classification helpers read step metadata (the
// Parking/Terminal flags and the derived phase) rather than matching hardcoded
// status names like "blocked"/"on_hold"/"cancelled" (B028 / WS1-A).
func customNamedRouteService() *Service {
	cfg := &config.WorkflowConfig{
		Version: "1.0",
		Start:   "kickoff",
		Steps: map[string]*config.Step{
			"kickoff": {
				Phase:      "planning",
				IsPlanning: true,
				Action:     "advance_status",
				Outcomes: map[string]string{
					"pass":    "build",
					"fail":    "kickoff",
					"blocked": "stuck",
				},
			},
			"build": {
				Phase:  "development",
				Action: "spawn_agent",
				Agent:  "dev",
				Outcomes: map[string]string{
					"pass":    "shipped",
					"fail":    "kickoff",
					"blocked": "stuck",
				},
			},
			// Parking step in the "blocked" phase, named with a non-standard
			// label and carrying the legacy "blocked" name as an alias.
			"stuck": {
				Phase:   "blocked",
				Parking: true,
				Aliases: []string{"blocked"},
			},
			// Parking step that is NOT in the blocked phase.
			"shelved": {
				Phase:   "paused",
				Parking: true,
			},
			"shipped":  {Phase: "done", Terminal: true},
			"scrapped": {Phase: "done", Terminal: true},
		},
	}
	// Populate the derived legacy maps so getStatusPhase (used by
	// IsBlockedStatus) can read each step's phase.
	cfg.DeriveLegacy()
	return &Service{workflow: cfg, level: LevelTask}
}

func TestIsParkingStatus_RouteBased(t *testing.T) {
	s := customNamedRouteService()
	tests := []struct {
		status string
		want   bool
	}{
		{"stuck", true},
		{"shelved", true},
		{"build", false},
		{"kickoff", false},
		{"shipped", false},
		// The literal name "on_hold" is NOT special — this workflow has no such
		// step, so it must not be classified as parking.
		{"on_hold", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := s.IsParkingStatus(tt.status); got != tt.want {
			t.Errorf("IsParkingStatus(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestIsParkingStatus_ResolvesAlias(t *testing.T) {
	s := customNamedRouteService()
	// "blocked" is an alias of the parking step "stuck"; alias resolution must
	// run so pre-migration entities classify correctly.
	if !s.IsParkingStatus("blocked") {
		t.Error("IsParkingStatus(\"blocked\") should resolve alias to parking step \"stuck\"")
	}
}

func TestIsBlockedStatus_RouteBased(t *testing.T) {
	s := customNamedRouteService()
	tests := []struct {
		status string
		want   bool
	}{
		{"stuck", true},     // phase == "blocked"
		{"shelved", false},  // parking but phase == "paused"
		{"build", false},    // development phase
		{"shipped", false},  // done phase
		{"blocked", true},   // alias of "stuck" (blocked phase)
		{"nonsense", false}, // unknown status
	}
	for _, tt := range tests {
		if got := s.IsBlockedStatus(tt.status); got != tt.want {
			t.Errorf("IsBlockedStatus(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestIsTerminalStatus_CustomNames(t *testing.T) {
	s := customNamedRouteService()
	for _, term := range []string{"shipped", "scrapped"} {
		if !s.IsTerminalStatus(term) {
			t.Errorf("IsTerminalStatus(%q) = false, want true", term)
		}
	}
	for _, nonTerm := range []string{"kickoff", "build", "stuck", "completed", "cancelled"} {
		// "completed"/"cancelled" are the *standard* terminal names; this
		// workflow does not use them, so they must NOT be treated as terminal.
		if s.IsTerminalStatus(nonTerm) {
			t.Errorf("IsTerminalStatus(%q) = true, want false", nonTerm)
		}
	}
}

func TestClassifyHelpers_NilWorkflowGraceful(t *testing.T) {
	s := &Service{workflow: nil, level: LevelTask}
	if s.IsParkingStatus("anything") {
		t.Error("IsParkingStatus on nil workflow should be false")
	}
	if s.IsBlockedStatus("anything") {
		t.Error("IsBlockedStatus on nil workflow should be false")
	}
}
