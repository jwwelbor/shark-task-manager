package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// integrationCmd groups epic integration-run maintenance subcommands
// (E34-F08 REQ-F-004).
var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "Manage epic integration-run state",
}

// integrationBackfillCmd is a thin wrapper over integration.Backfill
// (T-E34-F08-007): it parses/validates CLI input, performs the claim/session
// authorization check that only this layer owns (integration.Backfill's own
// signature has no session parameter), and delegates the actual
// capture-then-append sequence to Backfill unchanged.
var integrationBackfillCmd = &cobra.Command{
	Use:   "backfill <epic-key>",
	Short: "Register an integration run for an epic already active before this feature shipped",
	Long: `Register an IntegrationRun/IntegrationEvent/IntegrationCandidate set for an
epic that was already active before the integration-event log existed, so its
history is available to the integration_review gate. This performs the
identical capture-then-append sequence steady-state capture performs
(integration.Backfill), so there is one correctness argument for
atomicity/CAS, not two.

Requires an active claim on <epic-key> whose session matches --session — a
check this CLI layer owns, since integration.Backfill has no session
parameter of its own.

--dry-run performs every validation and reports what would be written
without writing anything. Any invalid input (including a malformed
--events-file) is rejected before the first write.

Examples:
  shark integration backfill E07 --epic-run-id=run-1 --base=<sha> \
    --events-file=events.json --session=$SID --dry-run
  shark integration backfill E07 --epic-run-id=run-1 --base=<sha> \
    --events-file=events.json --session=$SID`,
	Args: cobra.ExactArgs(1),
	RunE: runIntegrationBackfill,
}

func init() {
	integrationBackfillCmd.Flags().String("epic-run-id", "", "Epic run ID to register (required)")
	integrationBackfillCmd.Flags().String("base", "", "Base commit for the epic's pre-existing history (required)")
	integrationBackfillCmd.Flags().String("events-file", "", "Path to a JSON array of IntegrationEvent-shaped entries (required)")
	integrationBackfillCmd.Flags().String("session", "", "Session id of the active claim on <epic-key> (required)")
	integrationBackfillCmd.Flags().Bool("dry-run", false, "Validate and report without writing anything")

	integrationCmd.AddCommand(integrationBackfillCmd)
	cli.RootCmd.AddCommand(integrationCmd)
}

// integrationClaimLookup is the claim-lookup seam runIntegrationBackfill
// calls to authorize a backfill attempt: an active claim on <epic-key>
// whose session matches --session (spec.md REQ-F-004 — a check this CLI
// layer owns since integration.Backfill's own signature has no session
// parameter). A package variable, not a hardwired call, so a test can
// substitute a fake without a database (test-plan.md TC-010's Caller-Path
// Contract: mock only the claim-lookup seam), mirroring next.go's
// fanoutDescribeCandidateEdges test seam.
var integrationClaimLookup = func(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	return cli.GetClaimService().Get(ctx, entityType, entityKey)
}

// integrationNoteRecorder resolves the integration.NoteRecorder Backfill
// writes its epic reference note through (the CLI's *services.NoteService,
// which structurally satisfies integration.NoteRecorder). A package
// variable — like integrationClaimLookup above — so a test can substitute a
// fake recorder without a database, matching the CLI-tests-never-touch-the-
// database convention while still exercising the real, unmocked
// integration.Backfill call.
var integrationNoteRecorder = func(ctx context.Context) (integration.NoteRecorder, error) {
	return cli.GetNoteService(ctx)
}

// runIntegrationBackfill implements `shark integration backfill <epic-key>`.
func runIntegrationBackfill(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	epicKey := strings.ToUpper(strings.TrimSpace(args[0]))
	if DetectEntityType(epicKey) != "epic" {
		err := fmt.Errorf("%q is not a valid epic key", args[0])
		cli.Error(err.Error())
		return err
	}

	epicRunID, _ := cmd.Flags().GetString("epic-run-id")
	base, _ := cmd.Flags().GetString("base")
	eventsFile, _ := cmd.Flags().GetString("events-file")
	session, _ := cmd.Flags().GetString("session")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	epicRunID = strings.TrimSpace(epicRunID)
	base = strings.TrimSpace(base)
	eventsFile = strings.TrimSpace(eventsFile)
	session = strings.TrimSpace(session)

	if epicRunID == "" || base == "" || eventsFile == "" || session == "" {
		err := fmt.Errorf("--epic-run-id, --base, --events-file, and --session are all required")
		cli.Error(err.Error())
		return err
	}

	// Claim/session authorization — this layer's own check. A rejection here
	// leaves every sidecar and note untouched: integration.Backfill is never
	// called on this path.
	claim, err := integrationClaimLookup(ctx, "epic", epicKey)
	if err != nil {
		err = fmt.Errorf("look up claim for %s: %w", epicKey, err)
		cli.Error(err.Error())
		return err
	}
	if claim == nil {
		err := fmt.Errorf("no active claim on %s; backfill requires an active claim matching --session", epicKey)
		cli.Error(err.Error())
		return err
	}
	if claim.SessionID != session {
		err := fmt.Errorf("--session does not match the active claim's session on %s", epicKey)
		cli.Error(err.Error())
		return err
	}

	// Parse --events-file ourselves: integration.Backfill's own signature
	// (spec.md) accepts only an already-parsed []IntegrationEvent, so
	// "--events-file is not valid JSON" must be rejected here, before
	// Backfill is ever invoked (zero mutation on this path).
	data, err := os.ReadFile(eventsFile)
	if err != nil {
		err = fmt.Errorf("read --events-file %q: %w", eventsFile, err)
		cli.Error(err.Error())
		return err
	}
	var events []integration.IntegrationEvent
	if err := json.Unmarshal(data, &events); err != nil {
		err = fmt.Errorf("--events-file %q is not valid JSON: %w", eventsFile, err)
		cli.Error(err.Error())
		return err
	}

	recorder, err := integrationNoteRecorder(ctx)
	if err != nil {
		cli.Error(err.Error())
		return err
	}

	candidate, err := integration.Backfill(ctx, recorder, epicKey, epicRunID, base, events, dryRun, backfillActor())
	if err != nil {
		cli.Error(err.Error())
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]any{
			"dry_run":     dryRun,
			"epic_key":    epicKey,
			"epic_run_id": epicRunID,
			"candidate":   candidate,
		})
	}
	if dryRun {
		cli.Info(fmt.Sprintf("Dry run: would register integration run %s for %s (base %s, %d event(s))", epicRunID, epicKey, base, len(events)))
	} else {
		cli.Success(fmt.Sprintf("Backfilled integration run %s for %s (base %s, %d event(s))", epicRunID, epicKey, base, len(events)))
	}
	return nil
}

// backfillActor returns the actor identity recorded as the backfill's
// created-by, reading $SHARK_ACTOR and defaulting to "cli" (matching
// status_group.go's advanceActor convention).
func backfillActor() string {
	if actor := os.Getenv("SHARK_ACTOR"); actor != "" {
		return actor
	}
	return "cli"
}
