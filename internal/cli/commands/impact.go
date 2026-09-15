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
	"strings"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

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

// readBoundedImpactFile reads --impact-file through gaterun's shared
// no-follow-open + fstat-regular-file-check + size-bound helper (code-review
// round-6 finding: the sibling gap this rework closes). It mirrors
// run_apply_result.go's readBoundedEnvelopeFile — same helper, same
// workercontrol.MaxEnvelopeBytes bound, since internal/gateresult defines
// only per-field text bounds for ChangeImpactSet (SummaryMaxBytes,
// PointerMaxBytes, IdentityMaxBytes), not a bound on the serialized file as a
// whole. A plain os.ReadFile here would buffer the entire file into memory
// before ValidateChangeImpactSet's field-level bounds are ever evaluated,
// and would silently follow a symlink target or hang on a FIFO with no
// writer connected.
func readBoundedImpactFile(path string) ([]byte, error) {
	data, err := gaterun.ReadBoundedRegularFile(path, workercontrol.MaxEnvelopeBytes)
	if err != nil {
		if strings.Contains(err.Error(), "byte bound") {
			return nil, fmt.Errorf("--impact-file %q exceeds the maximum size of %d bytes", path, workercontrol.MaxEnvelopeBytes)
		}
		return nil, fmt.Errorf("read --impact-file %q: %w", path, err)
	}
	return data, nil
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

	raw, err := readBoundedImpactFile(impactFile)
	if err != nil {
		return err
	}

	var impact gateresult.ChangeImpactSet
	if err := json.Unmarshal(raw, &impact); err != nil {
		return fmt.Errorf("parse --impact-file %q as an I-04 ChangeImpactSet: %w", impactFile, err)
	}

	writer, err := impactNoteWriter(cmd.Context())
	if err != nil {
		return fmt.Errorf("get note service: %w", err)
	}

	service, err := services.NewImpactService(writer)
	if err != nil {
		return err
	}
	record, err := service.Record(cmd.Context(), services.RecordImpactInput{
		EntityType: models.EntityType(entityType), EntityKey: entityKey,
		SourceKind: impactSourceKind, SourceKey: impactSourceKey, SourcePointer: impactSourcePointer, Impact: impact,
	})
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"entity_key":     entityKey,
			"entity_type":    entityType,
			"source_kind":    record.SourceKind,
			"source_key":     record.SourceKey,
			"source_pointer": record.SourcePointer,
			"status":         record.Status,
			"note_id":        record.NoteID,
		})
	}

	fmt.Printf("Recorded change-impact (%s/%s) on %s %s\n", record.SourceKind, record.SourceKey, entityType, entityKey)
	return nil
}
