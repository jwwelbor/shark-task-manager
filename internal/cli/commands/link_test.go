package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

func TestMapDetectedTypeToEntityType(t *testing.T) {
	tests := []struct {
		name     string
		detected string
		want     models.EntityType
		wantErr  bool
	}{
		{
			name:     "epic",
			detected: "epic",
			want:     models.EntityTypeEpic,
		},
		{
			name:     "feature",
			detected: "feature",
			want:     models.EntityTypeFeature,
		},
		{
			name:     "task",
			detected: "task",
			want:     models.EntityTypeTask,
		},
		{
			name:     "bug",
			detected: "bug",
			want:     models.EntityTypeBug,
		},
		{
			name:     "change",
			detected: "change",
			want:     models.EntityTypeChange,
		},
		{
			name:     "change_card",
			detected: "change_card",
			want:     models.EntityTypeChange,
		},
		{
			name:     "question",
			detected: "question",
			want:     models.EntityTypeQuestion,
		},
		{
			name:     "tech debt",
			detected: "tech_debt",
			want:     models.EntityTypeTechDebt,
		},
		{
			name:     "unknown type returns error",
			detected: "unknown",
			wantErr:  true,
		},
		{
			name:     "empty string returns error",
			detected: "",
			wantErr:  true,
		},
		{
			name:     "idea returns error (not a relationship entity)",
			detected: "idea",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapDetectedTypeToEntityType(tt.detected)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapDetectedTypeToEntityType(%q) error = %v, wantErr %v", tt.detected, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("mapDetectedTypeToEntityType(%q) = %q, want %q", tt.detected, got, tt.want)
			}
		})
	}
}

func TestMapDetectedTypeToEntityType_AllValidEntityTypes(t *testing.T) {
	// Ensure all entity types that have registered repositories can be mapped.
	// This is a smoke test to catch if new entity types are added without updating the mapping.
	requiredMappings := map[string]models.EntityType{
		"epic":      models.EntityTypeEpic,
		"feature":   models.EntityTypeFeature,
		"task":      models.EntityTypeTask,
		"bug":       models.EntityTypeBug,
		"change":    models.EntityTypeChange,
		"tech_debt": models.EntityTypeTechDebt,
		"question":  models.EntityTypeQuestion,
	}

	for detected, want := range requiredMappings {
		got, err := mapDetectedTypeToEntityType(detected)
		if err != nil {
			t.Errorf("mapDetectedTypeToEntityType(%q) returned unexpected error: %v", detected, err)
			continue
		}
		if got != want {
			t.Errorf("mapDetectedTypeToEntityType(%q) = %q, want %q", detected, got, want)
		}
	}
}

func TestLinkCommandRelationshipTypeValidation(t *testing.T) {
	// Verify that all valid relationship types are accepted
	for relType := range models.ValidEntityRelationshipTypeSet {
		if !models.ValidEntityRelationshipTypeSet[relType] {
			t.Errorf("expected relationship type %q to be valid", relType)
		}
	}

	// Verify that an invalid type is rejected
	invalid := models.EntityRelationshipType("invalid_type")
	if models.ValidEntityRelationshipTypeSet[invalid] {
		t.Error("expected 'invalid_type' to be rejected as invalid relationship type")
	}
}

// TC-302 / UAT-001: the generic link surface must advertise the directed
// Question gate rather than requiring users to discover an undocumented type.
func TestLinkHelpAdvertisesDirectedQuestionBlocks(t *testing.T) {
	for _, help := range []string{linkCmd.Long, linkCmd.Flag("type").Usage} {
		if !strings.Contains(help, "question_blocks") {
			t.Errorf("link help %q does not advertise question_blocks", help)
		}
	}
	if !strings.Contains(linkCmd.Long, "Question") || !strings.Contains(linkCmd.Long, "eligible") {
		t.Errorf("link long help does not describe Question's directed eligible-target rule: %q", linkCmd.Long)
	}
	if !strings.Contains(unlinkCmd.Long, "question_blocks") {
		t.Errorf("unlink long help does not advertise removal of a Question gate: %q", unlinkCmd.Long)
	}
	if !strings.Contains(linkCmd.Long, "from-key is blocked by to-key") ||
		!strings.Contains(linkCmd.Long, "from-key blocks to-key") {
		t.Errorf("link long help does not explain directional dependency syntax: %q", linkCmd.Long)
	}
}

func TestLinkRejectsBlockedByWithDirectionalGuidance(t *testing.T) {
	previousRelType := linkRelType
	linkRelType = "blocked_by"
	t.Cleanup(func() { linkRelType = previousRelType })

	err := runLink(&cobra.Command{}, []string{"from-key", "to-key"})
	if err == nil {
		t.Fatal("runLink() returned nil for unsupported blocked_by relationship type")
	}

	message := err.Error()
	for _, expected := range []string{
		`"blocked_by" is not a relationship type`,
		`--type=depends_on`,
		`from-key is blocked by to-key`,
		`--type=blocks`,
		`from-key blocks to-key`,
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("runLink() error = %q, want guidance containing %q", message, expected)
		}
	}
}

func TestUnlinkRejectsBlockedByWithDirectionalGuidance(t *testing.T) {
	previousRelType := linkRelType
	linkRelType = "blocked_by"
	t.Cleanup(func() { linkRelType = previousRelType })

	err := runUnlink(&cobra.Command{}, []string{"from-key", "to-key"})
	if err == nil {
		t.Fatal("runUnlink() returned nil for unsupported blocked_by relationship type")
	}
	if !strings.Contains(err.Error(), `"blocked_by" is not a relationship type`) ||
		!strings.Contains(err.Error(), "--type=depends_on") ||
		!strings.Contains(err.Error(), "--type=blocks") {
		t.Fatalf("runUnlink() error = %q, want directional blocked_by guidance", err)
	}
}

func TestB073WorkflowGuideUsesCurrentRelationshipSyntax(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	documents := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: filepath.Join("docs", "WORKFLOW_GUIDE.md"),
			required: []string{
				"shark link A B --type=depends_on",
				"shark link A B --type=blocks",
				"shark task unlink <source-task> --depends-on <target-task>",
			},
			forbidden: []string{"shark task unlink <source> <target>"},
		},
		{
			path: filepath.Join("skills", "shark-rider", "context", "entity-crud.md"),
			required: []string{
				"shark task link E01-F02-001 --depends-on E01-F02-002",
				"shark task unlink E01-F02-001 --depends-on E01-F02-002",
			},
		},
		{
			path: filepath.Join(".claude", "rules", "quickref.md"),
			required: []string{
				"shark task link E07-F01-001 --depends-on E07-F01-002",
				"shark task unlink E07-F01-001 --depends-on E07-F01-002",
			},
			forbidden: []string{
				"shark task link E07-F01-001 E07-F01-002 --type=depends_on",
				"shark task unlink E07-F01-001 E07-F01-002",
			},
		},
		{
			path: filepath.Join(".claude", "rules", "cli", "commands.md"),
			required: []string{
				"shark task link <task-key> --depends-on <target-task>",
				"shark task unlink <task-key> --depends-on <target-task>",
			},
			forbidden: []string{
				"shark task link <key1> <key2> --type=TYPE",
				"shark task unlink <key1> <key2>",
			},
		},
	}
	for _, document := range documents {
		content, err := os.ReadFile(filepath.Join(projectRoot, document.path))
		if err != nil {
			t.Fatalf("read %s: %v", document.path, err)
		}
		text := string(content)
		for _, forbidden := range []string{"--type=depends-on", "--type=relates-to", "--type=blocked_by"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains obsolete relationship syntax %q", document.path, forbidden)
			}
		}
		for _, forbidden := range document.forbidden {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains obsolete task relationship syntax %q", document.path, forbidden)
			}
		}
		for _, required := range document.required {
			if !strings.Contains(text, required) {
				t.Errorf("%s does not contain current relationship syntax %q", document.path, required)
			}
		}
	}
}
