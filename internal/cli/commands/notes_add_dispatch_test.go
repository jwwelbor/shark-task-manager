package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestNotesAddDispatch_ResolvesEntityType verifies the verb-first
// `shark notes add <KEY>` dispatch picks the correct EntityType for every
// supported key shape (epic, feature, task, bug, change card, tech-debt, idea).
func TestNotesAddDispatch_ResolvesEntityType(t *testing.T) {
	cases := []struct {
		name     string
		key      string
		wantType models.EntityType
		wantName string
	}{
		{"epic numeric", "E07", models.EntityTypeEpic, "epic"},
		{"epic slugged", "E07-user-management", models.EntityTypeEpic, "epic"},
		{"feature combined", "E07-F01", models.EntityTypeFeature, "feature"},
		{"task short", "E07-F01-001", models.EntityTypeTask, "task"},
		{"task full", "T-E07-F01-001", models.EntityTypeTask, "task"},
		{"bug", "B042", models.EntityTypeBug, "bug"},
		{"change card", "CC-007", models.EntityTypeChange, "change"},
		{"tech-debt", "TD-003", models.EntityTypeTechDebt, "tech-debt"},
		{"idea", "I-2026-05-02-01", models.EntityTypeIdea, "idea"},
		{"case-insensitive task", "e07-f01-001", models.EntityTypeTask, "task"},
		{"case-insensitive bug", "b042", models.EntityTypeBug, "bug"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotName, err := resolveNoteEntityType(tc.key)
			if err != nil {
				t.Fatalf("resolveNoteEntityType(%q) returned error: %v", tc.key, err)
			}
			if gotType != tc.wantType {
				t.Errorf("entity type for %q: got %q, want %q", tc.key, gotType, tc.wantType)
			}
			if gotName != tc.wantName {
				t.Errorf("entity name for %q: got %q, want %q", tc.key, gotName, tc.wantName)
			}
		})
	}
}

// TestNotesAddDispatch_RejectsUnknownKey verifies that the dispatch returns a
// helpful error when the key shape doesn't match any known entity, instead of
// silently picking the wrong service.
func TestNotesAddDispatch_RejectsUnknownKey(t *testing.T) {
	_, _, err := resolveNoteEntityType("garbage-key-123")
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
}

// TestNotesAddDispatch_FlagsRegistered asserts the verb-first
// `shark notes add` command registers the `--type` (required) and
// `--created-by` flags with the same shape as the entity-first
// `shark <entity> note add` commands. Mirrors TestUpdateDispatch_*FlagRegistered.
func TestNotesAddDispatch_FlagsRegistered(t *testing.T) {
	typeFlag := notesAddCmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("--type flag not registered on notesAddCmd")
	}
	if typeFlag.Value.Type() != "string" {
		t.Errorf("expected --type to be string, got %s", typeFlag.Value.Type())
	}

	createdByFlag := notesAddCmd.Flags().Lookup("created-by")
	if createdByFlag == nil {
		t.Fatal("--created-by flag not registered on notesAddCmd")
	}
	if createdByFlag.Value.Type() != "string" {
		t.Errorf("expected --created-by to be string, got %s", createdByFlag.Value.Type())
	}

	// Cobra exposes required flags via the BashCompOneRequiredFlag annotation.
	if anns := typeFlag.Annotations["cobra_annotation_bash_completion_one_required_flag"]; len(anns) == 0 {
		t.Error("expected --type to be marked required")
	}
}

// TestNotesAddDispatch_RegisteredOnNotesParent confirms the dispatch is
// reachable as `shark notes add` (subcommand of the existing notesCmd, not a
// standalone top-level verb).
func TestNotesAddDispatch_RegisteredOnNotesParent(t *testing.T) {
	found := false
	for _, sub := range notesCmd.Commands() {
		if sub.Name() == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("`add` is not a subcommand of notesCmd — verb-first dispatch is not wired")
	}
}
