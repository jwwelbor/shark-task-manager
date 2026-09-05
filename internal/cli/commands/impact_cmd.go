package commands

import (
	"context"
	"encoding/json"
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

// Flags for `shark impact record`. Package-level so both the registered
// production command and isolated per-test commands (built fresh so tests
// don't mutate impactRecordCmd's shared, root-wired flag state) can bind the
// same variables, mirroring change_test.go's changeLinkKey-style pattern.
var (
	impactSourceKind    string
	impactSourceKey     string
	impactSourcePointer string
	impactFile          string
)

// impactRecordCmd implements the parent-owned ADR-adoption boundary declared
// in architecture.md's "Compatibility and migration" section:
//
//	shark impact record <entity-key> --source-kind=adr --source-key=<ADR-ID> \
//	  --source-pointer=<path> --impact-file=<bounded-I-04-json>
//
// This replaces the earlier positional `<content-or-@file>` form entirely —
// architecture.md documents only the flag form for this boundary.
var impactRecordCmd = &cobra.Command{
	Use:   "record <entity-key> --source-kind=<kind> --source-key=<key> --source-pointer=<path> --impact-file=<path>",
	Short: "Record a change-impact set (I-04) on an entity as a reference note",
	Long: `Record a change-impact set (I-04) on an entity as a reference note.

--impact-file points to a JSON file containing the bounded I-04 content (at
minimum a non-empty affected_artifacts array; see
architecture.md#i-04-changeimpactset-v1 for the full field table).
--source-kind, --source-key, and --source-pointer are the caller-supplied
identity of the change source (e.g. an ADR) and are merged into the
impact-file's JSON as the authoritative source_kind/source_key/source_pointer
fields — they override any value already present in the file.

Minimal validation (not full schema enforcement): after merging, source_kind,
source_key, and a non-empty affected_artifacts array must be present.

Example:
  shark impact record E07-F01-001 \
    --source-kind=adr --source-key=ADR-014 --source-pointer=docs/adr/ADR-014.md \
    --impact-file=impact.json`,
	Args: cobra.ExactArgs(1),
	RunE: runImpactRecord,
}

func init() {
	impactRecordCmd.Flags().StringVar(&impactSourceKind, "source-kind", "", "Source kind (question, tech_debt, change_card, adr, state_change, design_divergence)")
	impactRecordCmd.Flags().StringVar(&impactSourceKey, "source-key", "", "Durable Shark or ADR identity")
	impactRecordCmd.Flags().StringVar(&impactSourcePointer, "source-pointer", "", "Authoritative local record (e.g. path to the ADR file)")
	impactRecordCmd.Flags().StringVar(&impactFile, "impact-file", "", "Path to a JSON file containing the bounded I-04 content")
	_ = impactRecordCmd.MarkFlagRequired("source-kind")
	_ = impactRecordCmd.MarkFlagRequired("source-key")
	_ = impactRecordCmd.MarkFlagRequired("source-pointer")
	_ = impactRecordCmd.MarkFlagRequired("impact-file")

	impactCmd.AddCommand(impactRecordCmd)
	cli.RootCmd.AddCommand(impactCmd)
}

func runImpactRecord(cmd *cobra.Command, args []string) error {
	key := args[0]

	sourceKind := strings.TrimSpace(mustGetStringFlag(cmd, "source-kind"))
	sourceKey := strings.TrimSpace(mustGetStringFlag(cmd, "source-key"))
	sourcePointer := strings.TrimSpace(mustGetStringFlag(cmd, "source-pointer"))
	filePath := strings.TrimSpace(mustGetStringFlag(cmd, "impact-file"))

	if sourceKind == "" {
		return fmt.Errorf("--source-kind must not be empty")
	}
	if sourceKey == "" {
		return fmt.Errorf("--source-key must not be empty")
	}
	if sourcePointer == "" {
		return fmt.Errorf("--source-pointer must not be empty")
	}
	if filePath == "" {
		return fmt.Errorf("--impact-file must not be empty")
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read impact file %q: %w", filePath, err)
	}

	content := mergeImpactSourceFields(fileBytes, sourceKind, sourceKey, sourcePointer)

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

// mustGetStringFlag reads a string flag's value, ignoring the (always-nil
// for a StringVar-backed flag) lookup error — the flag is guaranteed to
// exist because it is registered in this same file's init().
func mustGetStringFlag(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// mergeImpactSourceFields sets source_kind, source_key, and source_pointer
// on the impact-file's JSON object, overriding any value already present —
// the CLI-supplied identity is the parent-owned authority for the
// ADR-adoption boundary (architecture.md's I-04 field table lists all three
// as top-level I-04 fields, not mere CLI bookkeeping).
//
// If impactFileJSON does not parse as a JSON object, it is returned
// unchanged so ImpactService.RecordImpact's existing shape validator
// produces its canonical malformed-content error — this function never
// shadows that error with a CLI-layer parse failure.
func mergeImpactSourceFields(impactFileJSON []byte, sourceKind, sourceKey, sourcePointer string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(impactFileJSON, &payload); err != nil {
		return string(impactFileJSON)
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["source_kind"] = sourceKind
	payload["source_key"] = sourceKey
	payload["source_pointer"] = sourcePointer

	merged, err := json.Marshal(payload)
	if err != nil {
		// Extremely unlikely immediately after a successful Unmarshal into
		// the same shape; fall back to the raw file bytes rather than lose
		// the content entirely.
		return string(impactFileJSON)
	}
	return string(merged)
}
