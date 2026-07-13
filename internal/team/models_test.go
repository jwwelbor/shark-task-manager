package team

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestValidateEntityIdentity_RejectsDeclaredTypeMismatches(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want models.EntityType
	}{
		{name: "epic", key: "E38", want: models.EntityTypeEpic},
		{name: "feature", key: "E38-F01", want: models.EntityTypeFeature},
		{name: "task", key: "T-E38-F01-001", want: models.EntityTypeTask},
		{name: "bug", key: "B001", want: models.EntityTypeBug},
		{name: "change", key: "CC-001", want: models.EntityTypeChange},
		{name: "sprint", key: "S001", want: models.EntityTypeSprint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for declared := range models.ValidEntityTypes {
				if declared == tt.want {
					continue
				}
				if err := validateEntityIdentity(tt.key, declared); err == nil {
					t.Errorf("validateEntityIdentity(%q, %q) accepted key owned by %q", tt.key, declared, tt.want)
				}
			}
			if err := validateEntityIdentity(tt.key, tt.want); err != nil {
				t.Errorf("validateEntityIdentity(%q, %q) rejected matching identity: %v", tt.key, tt.want, err)
			}
		})
	}
}

func TestLedgerOutput_RejectsSensitiveContent_TC009(t *testing.T) {
	base := ItemResultUpdate{
		RunID:        1,
		ItemID:       1,
		Attempt:      0,
		Status:       ItemStatusCompleted,
		Outcome:      "passed",
		Evidence:     "completed the requested work",
		ArtifactRefs: []string{"artifacts/run-1/result.md"},
	}

	tests := []struct {
		name   string
		mutate func(*ItemResultUpdate)
		cause  error
	}{
		{name: "evidence over bound", mutate: func(update *ItemResultUpdate) {
			update.Evidence = strings.Repeat("x", MaxEvidenceBytes+1)
		}, cause: ErrEvidenceTooLarge},
		{name: "bearer token", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "Bearer sk-test-123"
		}, cause: ErrSensitiveEvidence},
		{name: "private key", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "-----BEGIN PRIVATE KEY-----"
		}, cause: ErrSensitiveEvidence},
		{name: "rendered prompt", mutate: func(update *ItemResultUpdate) {
			update.Evidence = "rendered prompt: do the work"
		}, cause: ErrSensitiveEvidence},
		{name: "absolute artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"/tmp/secret"}
		}, cause: ErrInvalidArtifactPath},
		{name: "traversal artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"../secret"}
		}, cause: ErrInvalidArtifactPath},
		{name: "duplicate artifact", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"artifacts/result.md", "artifacts/result.md"}
		}, cause: ErrInvalidArtifactPath},
		{name: "foreign drive path", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{`C:\\outside\\result.md`}
		}, cause: ErrInvalidArtifactPath},
		{name: "home path", mutate: func(update *ItemResultUpdate) {
			update.ArtifactRefs = []string{"~/outside/result.md"}
		}, cause: ErrInvalidArtifactPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := base
			tt.mutate(&update)
			if err := update.Validate(); !errors.Is(err, tt.cause) {
				t.Fatalf("ItemResultUpdate.Validate() error = %v, want %v", err, tt.cause)
			}
		})
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
}

func TestItemResultUpdate_ArtifactRefsStayWithinAllowedProjectBase_TC009(t *testing.T) {
	projectRoot := t.TempDir()
	valid := ItemResultUpdate{RunID: 1, ItemID: 1, Attempt: 0, Status: ItemStatusCompleted, ArtifactRefs: []string{"artifacts/./result.md"}}

	normalized, err := valid.NormalizeArtifactRefs(projectRoot)
	if err != nil {
		t.Fatalf("valid project-relative artifact rejected: %v", err)
	}
	want := filepath.ToSlash(filepath.Join("artifacts", "result.md"))
	if len(normalized) != 1 || normalized[0] != want {
		t.Fatalf("normalized refs = %v, want [%q]", normalized, want)
	}

	for _, ref := range []string{
		filepath.Join(projectRoot, "outside.md"),
		"../../outside.md",
		`C:\outside\result.md`,
		`\\server\share\result.md`,
		"~/outside/result.md",
	} {
		update := valid
		update.ArtifactRefs = []string{ref}
		if _, err := update.NormalizeArtifactRefs(projectRoot); !errors.Is(err, ErrInvalidArtifactPath) {
			t.Errorf("NormalizeArtifactRefs(%q) error = %v, want ErrInvalidArtifactPath", ref, err)
		}
	}
}

func TestTeamRunResult_ContainsCompleteSafeShape_TC014(t *testing.T) {
	run := &TeamRun{
		ID:               7,
		RootKey:          "E38-F01",
		RootType:         "feature",
		Status:           RunStatusCompleted,
		ExecutionMode:    ExecutionModeParallel,
		ConcurrencyLimit: 2,
		PlanHash:         strings.Repeat("a", 64),
	}
	items := []*TeamRunItem{{
		ID:             9,
		TeamRunID:      7,
		ChildKey:       "T-E38-F01-001",
		ChildType:      "task",
		Wave:           1,
		ItemStatus:     ItemStatusCompleted,
		Attempt:        1,
		Outcome:        stringPtr("passed"),
		Evidence:       "bounded summary",
		ArtifactRefs:   []string{"artifacts/result.md"},
		ClaimSessionID: stringPtr("claim-1"),
	}}

	result, err := NewTeamRunResult(run, items)
	if err != nil {
		t.Fatalf("NewTeamRunResult() error = %v", err)
	}
	if result.RunID != 7 || result.RootKey != "E38-F01" || len(result.Items) != 1 {
		t.Fatalf("incomplete result: %+v", result)
	}
	if result.Items[0].Attempt != 1 || result.Items[0].ClaimSessionID == nil {
		t.Fatalf("item result lost lifecycle fields: %+v", result.Items[0])
	}
}

func stringPtr(value string) *string { return &value }
