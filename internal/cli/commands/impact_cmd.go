package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// impactSvcOverride is non-nil only during tests. It lets tests wire a real
// *services.ImpactService — backed by mocked repositories, never a mocked
// ImpactService or NoteService — into the registered Cobra command, mirroring
// the override pattern other thin dispatch commands use (e.g. bugSvcOverride
// in bug.go) but holding the concrete production type per test-plan.md's
// Caller-Path Contract for TC-007..TC-009 (mock the entity-type repository
// and NoteEntityNoteRepository seams, never ImpactService/NoteService).
var impactSvcOverride *services.ImpactService

// getImpactService returns the test override when set, otherwise the global
// production ImpactService.
func getImpactService(ctx context.Context) (*services.ImpactService, error) {
	if impactSvcOverride != nil {
		return impactSvcOverride, nil
	}
	return cli.GetImpactService(ctx)
}

// impactCmd is the parent command group for change-impact operations.
var impactCmd = &cobra.Command{
	Use:   "impact",
	Short: "Manage change-impact records",
	Long:  "Commands for recording change-impact sets (I-04) against Shark entities.",
}

// impactRecordCmd implements `shark impact record <entity-key> <content-or-@file>`.
var impactRecordCmd = &cobra.Command{
	Use:   "record <entity-key> <content-or-@file>",
	Short: "Record a change-impact set (I-04) on an entity as a reference note",
	Long: `Record a change-impact set (I-04) on an entity as a reference note.

content-or-@file is either an inline JSON string matching the I-04 shape, or
a file path prefixed with @ (e.g. @impact.json) whose contents are read as
the note content.

Minimal validation (not full schema enforcement): source_kind, source_key,
and a non-empty affected_artifacts array must be present.

Examples:
  shark impact record E07-F01-001 '{"source_kind":"tech_debt","source_key":"TD-042","affected_artifacts":["spec.md"]}'
  shark impact record E07-F01 @impact.json`,
	Args: cobra.ExactArgs(2),
	RunE: runImpactRecord,
}

func init() {
	impactCmd.AddCommand(impactRecordCmd)
	cli.RootCmd.AddCommand(impactCmd)
}

func runImpactRecord(cmd *cobra.Command, args []string) error {
	key := args[0]
	contentArg := args[1]

	content := contentArg
	if strings.HasPrefix(contentArg, "@") {
		path := strings.TrimPrefix(contentArg, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read impact content file %q: %w", path, err)
		}
		content = string(data)
	}

	entityType, entityName, err := resolveEntityFromKey(key)
	if err != nil {
		return err
	}

	impactSvc, err := getImpactService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get impact service: %w", err)
	}

	note, err := impactSvc.RecordImpact(cmd.Context(), entityType, key, content, "")
	if err != nil {
		return fmt.Errorf("failed to record impact on %s %s: %w", entityName, key, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	cli.Success(fmt.Sprintf("Impact recorded on %s %s", entityName, key))
	return nil
}
