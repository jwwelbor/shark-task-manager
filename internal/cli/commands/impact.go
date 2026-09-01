// This file implements `shark impact record <entity-key> --source-kind=<kind>
// --source-key=<key> --source-pointer=<path> --impact-file=<bounded-I-04-json>`
// (T-E34-F05-005, REQ-F-006's ADR-adoption boundary, architecture.md
// "Compatibility and migration"). It is the one parent-owned surface for
// recording an I-04 ChangeImpactSet against an ADR source: workers never
// write this note directly, and the flags — not the impact-file's own
// source_kind/source_key/source_pointer, if present — are the authoritative
// parent-asserted identity (mirroring GateResult's "entity/source status are
// parent-observed, never asserted by worker output" contract).
//
// Persistence reuses internal/gateresult's I-04 validator
// (ValidateChangeImpactSet) and the same bounded "reference" note shape
// gatepersist's coordinator writes for a GateResult's own change_impacts
// entries (record_kind=change_impact) — no second I-04 write path.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// recordKindChangeImpact mirrors gatepersist's unexported
// recordKindImpact constant (internal/gatepersist/operations.go). It is
// duplicated here rather than exported cross-package because gatepersist's
// record_kind constants are an internal implementation detail of its own
// operation-building; this command only needs the one bounded metadata
// value, not gatepersist's run/replay machinery.
const recordKindChangeImpact = "change_impact"

const noteTypeReferenceImpact = "reference"

var (
	impactSourceKind    string
	impactSourceKey     string
	impactSourcePointer string
	impactFile          string
)

// impactNoteWriterOverride lets tests inject a mock gatepersist.NoteWriter
// instead of a real note service — CLI-command tests never use a real
// database (project golden rule). Production callers leave this nil so
// impactNoteWriter falls back to cli.GetNoteService.
var impactNoteWriterOverride gatepersist.NoteWriter

// impactNoteWriter resolves the NoteWriter this command persists through.
// *services.NoteService (returned by cli.GetNoteService) already satisfies
// gatepersist.NoteWriter — the same interface gatepersist.Coordinator uses
// to write its own change_impact reference notes — so this command's note
// shape stays byte-compatible with that path without importing gatepersist's
// run/replay machinery.
func impactNoteWriter(ctx context.Context) (gatepersist.NoteWriter, error) {
	if impactNoteWriterOverride != nil {
		return impactNoteWriterOverride, nil
	}
	return cli.GetNoteService(ctx)
}

var impactCmd = &cobra.Command{
	Use:   "impact",
	Short: "Record parent-owned change-impact (I-04) evidence",
}

var impactRecordCmd = &cobra.Command{
	Use:   "record <entity-key>",
	Short: "Validate and persist an I-04 ChangeImpactSet against an entity",
	Long: `Validate and persist an I-04 ChangeImpactSet (internal/gateresult) as a
bounded reference note on the given entity.

This is the parent-owned ADR-adoption boundary REQ-F-006 requires: --source-kind,
--source-key, and --source-pointer are the authoritative parent-asserted
identity, and always take precedence — the command validates and persists,
it does not let --impact-file silently override what the parent asserted for
those three fields.`,
	Args: cobra.ExactArgs(1),
	RunE: runImpactRecord,
}

func init() {
	impactRecordCmd.Flags().StringVar(&impactSourceKind, "source-kind", "", "Source kind (e.g. adr) — required")
	impactRecordCmd.Flags().StringVar(&impactSourceKey, "source-key", "", "Durable source identity (e.g. ADR-0007) — required")
	impactRecordCmd.Flags().StringVar(&impactSourcePointer, "source-pointer", "", "Authoritative local record path — required")
	impactRecordCmd.Flags().StringVar(&impactFile, "impact-file", "", "Path to a bounded I-04 ChangeImpactSet JSON file — required")
	_ = impactRecordCmd.MarkFlagRequired("source-kind")
	_ = impactRecordCmd.MarkFlagRequired("source-key")
	_ = impactRecordCmd.MarkFlagRequired("source-pointer")
	_ = impactRecordCmd.MarkFlagRequired("impact-file")

	impactCmd.AddCommand(impactRecordCmd)
	cli.RootCmd.AddCommand(impactCmd)
}

func runImpactRecord(cmd *cobra.Command, args []string) error {
	entityKey := strings.TrimSpace(args[0])
	entityType := DetectEntityType(entityKey)
	if entityType == "unknown" {
		return fmt.Errorf("could not determine entity type for key %q", entityKey)
	}

	raw, err := os.ReadFile(impactFile)
	if err != nil {
		return fmt.Errorf("read --impact-file %q: %w", impactFile, err)
	}

	var impact gateresult.ChangeImpactSet
	if err := json.Unmarshal(raw, &impact); err != nil {
		return fmt.Errorf("parse --impact-file %q as an I-04 ChangeImpactSet: %w", impactFile, err)
	}

	// The flags are the parent-asserted identity. A file that claims a
	// DIFFERENT identity for any of the three parent-owned fields is a
	// conflict this command must reject closed, not silently overwrite —
	// this is precisely the failure mode ("recording ADR-1 for a file
	// claiming ADR-9") REQ-NF-001's fail-closed posture exists to prevent.
	// An empty field in the file is filled from the flag.
	if err := reconcileImpactIdentity(&impact); err != nil {
		return err
	}

	if err := gateresult.ValidateChangeImpactSet(impact); err != nil {
		return fmt.Errorf("invalid I-04 ChangeImpactSet: %w", err)
	}

	writer, err := impactNoteWriter(cmd.Context())
	if err != nil {
		return fmt.Errorf("get note service: %w", err)
	}

	content := fmt.Sprintf("[%s/%s] %s (%s)", impact.SourceKind, impact.SourceKey, impact.ChangeSummary, impact.Status)
	metadata := map[string]interface{}{
		"record_kind":    recordKindChangeImpact,
		"source_kind":    impact.SourceKind,
		"source_key":     impact.SourceKey,
		"source_pointer": impact.SourcePointer,
		"status":         impact.Status,
	}
	encodedMeta, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode note metadata: %w", err)
	}

	note, err := writer.AddNoteWithMetadata(
		cmd.Context(), models.EntityType(entityType), entityKey,
		noteTypeReferenceImpact, content, "", string(encodedMeta),
	)
	if err != nil {
		return fmt.Errorf("persist change-impact note on %s %s: %w", entityType, entityKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_key":     entityKey,
			"entity_type":    entityType,
			"source_kind":    impact.SourceKind,
			"source_key":     impact.SourceKey,
			"source_pointer": impact.SourcePointer,
			"status":         impact.Status,
			"note_id":        note.ID,
		})
	}

	fmt.Printf("Recorded change-impact (%s/%s) on %s %s\n", impact.SourceKind, impact.SourceKey, entityType, entityKey)
	return nil
}

// reconcileImpactIdentity applies the flags-are-authoritative,
// reject-on-conflict rule for the three parent-owned identity fields: an
// empty field in the parsed file is filled from the corresponding flag: a
// non-empty field that disagrees with the flag is a hard error.
func reconcileImpactIdentity(impact *gateresult.ChangeImpactSet) error {
	if err := reconcileImpactField("source_kind", impactSourceKind, &impact.SourceKind); err != nil {
		return err
	}
	if err := reconcileImpactField("source_key", impactSourceKey, &impact.SourceKey); err != nil {
		return err
	}
	if err := reconcileImpactField("source_pointer", impactSourcePointer, &impact.SourcePointer); err != nil {
		return err
	}
	return nil
}

func reconcileImpactField(field, flagValue string, fileValue *string) error {
	if strings.TrimSpace(*fileValue) == "" {
		*fileValue = flagValue
		return nil
	}
	if *fileValue != flagValue {
		return fmt.Errorf("--%s=%q conflicts with %s=%q in --impact-file; the flag is the parent-asserted identity and must match", strings.ReplaceAll(field, "_", "-"), flagValue, field, *fileValue)
	}
	return nil
}
