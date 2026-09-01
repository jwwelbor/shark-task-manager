package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// claimCmd leases an entity for an agent/session (E35-F03).
var claimCmd = &cobra.Command{
	Use:   "claim <key>",
	Short: "Claim an entity (acquire its in-flight lease)",
	Long: `Claim an entity so an agent can work it. Status is a pure phase; the claim
is the lease that prevents two agents from grabbing the same entity.

Expired leases (no heartbeat within the TTL) are reclaimed automatically before
the claim. Use --force to steal a live claim.

The printed session id must be passed to 'shark heartbeat' and 'shark unclaim'
to renew or release the lease safely.

Examples:
  shark claim E07-F01-001                       Claim a task
  shark claim E07-F01-001 --by=dev-agent        Claim as a named actor
  shark claim E07-F01-001 --session=$SID         Claim with an explicit session id
  shark claim E07-F01-001 --force                Steal an existing claim`,
	Args: cobra.ExactArgs(1),
	RunE: runClaim,
}

var unclaimCmd = &cobra.Command{
	Use:   "release <key>",
	Short: "Release an entity's claim (lease)",
	Long: `Release the lease on an entity. With --session the release is session-scoped
(a safe sync-release that will not steal a lease re-issued to another agent);
without --session it is an unconditional administrative release and requires
--force.

Examples:
  shark release E07-F01-001 --force              Administrative release
  shark release E07-F01-001 --session=$SID        Safe session-scoped release
  shark release E07-F01-001 --session=$SID --outcome=pass   Stamp the work session's outcome

Hidden aliases (functional, omitted from help): unclaim, release-claim.`,
	Args: cobra.ExactArgs(1),
	RunE: runUnclaim,
}

// newReleaseAlias builds a hidden alias command for `release`. Hidden commands
// stay fully functional but are omitted from help listings, so `unclaim` and
// `release-claim` keep working without cluttering the CLI help (E35-F03).
func newReleaseAlias(name string) *cobra.Command {
	c := &cobra.Command{
		Use:    name + " <key>",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE:   runUnclaim,
	}
	c.Flags().Bool("force", false, "Administrative release without a session id")
	c.Flags().String("session", "", "Session id for a safe session-scoped release")
	c.Flags().String("outcome", "", "Outcome stamped on the closed work session (default: released)")
	return c
}

var heartbeatCmd = &cobra.Command{
	Use:   "heartbeat <key>",
	Short: "Renew an entity's lease and report progress",
	Long: `Renew the lease on a claimed entity (and optionally record progress/note).
A heartbeat does triple duty: lease renewal, progress reporting, and telemetry.
Requires the session id returned by 'shark claim'.

Examples:
  shark heartbeat E07-F01-001 --session=$SID
  shark heartbeat E07-F01-001 --session=$SID --progress=0.5 --note="tests passing"`,
	Args: cobra.ExactArgs(1),
	RunE: runHeartbeat,
}

var claimsCmd = &cobra.Command{
	Use:   "claims",
	Short: "List active entity claims",
	Long:  "List all current entity claims (expired leases are swept first).",
	Args:  cobra.NoArgs,
	RunE:  runClaims,
}

// claimSvcOverride is non-nil only during tests. It lets tests exercise the
// real *services.ClaimService (validation, defaults, precedence) against a
// mocked ClaimRepository, per the E34-F01 test-plan's Caller-Path Contract:
// mocking ClaimService itself is forbidden.
var claimSvcOverride *services.ClaimService

// getClaimService returns the service to use, preferring the test override.
func getClaimService() *services.ClaimService {
	if claimSvcOverride != nil {
		return claimSvcOverride
	}
	return cli.GetClaimService()
}

func init() {
	claimCmd.Flags().String("by", "", "Actor identity holding the claim (default: $SHARK_ACTOR or 'cli')")
	claimCmd.Flags().String("session", "", "Explicit session id (default: generated)")
	claimCmd.Flags().Bool("force", false, "Steal an existing (even live) claim")
	claimCmd.Flags().String("harness", "", "Harness type (e.g. claude, codex); trimmed and lowercased before persisting")
	claimCmd.Flags().String("harness-version", "", "Harness version string; trimmed only, not normalized")
	claimCmd.Flags().String("harness-model", "", "Harness model string; trimmed only, not normalized")

	unclaimCmd.Flags().Bool("force", false, "Administrative release without a session id")
	unclaimCmd.Flags().String("session", "", "Session id for a safe session-scoped release")
	unclaimCmd.Flags().String("outcome", "", "Outcome stamped on the closed work session (default: released)")

	heartbeatCmd.Flags().String("session", "", "Session id that holds the claim (required)")
	heartbeatCmd.Flags().Float64("progress", 0, "Progress fraction 0.0-1.0 to record")
	heartbeatCmd.Flags().String("note", "", "Progress note to record")

	cli.RootCmd.AddCommand(claimCmd)
	cli.RootCmd.AddCommand(unclaimCmd)
	cli.RootCmd.AddCommand(heartbeatCmd)
	cli.RootCmd.AddCommand(claimsCmd)
	// Hidden aliases for `release` — functional, omitted from help.
	cli.RootCmd.AddCommand(newReleaseAlias("unclaim"))
	cli.RootCmd.AddCommand(newReleaseAlias("release-claim"))
}

func runClaim(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	key := strings.ToUpper(strings.TrimSpace(args[0]))
	entityType := DetectEntityType(key)
	if entityType == "" {
		cli.Error(fmt.Sprintf("Could not detect entity type from key %q", key))
		return fmt.Errorf("unknown key format: %s", key)
	}
	by, _ := cmd.Flags().GetString("by")
	session, _ := cmd.Flags().GetString("session")
	force, _ := cmd.Flags().GetBool("force")

	// Harness identity (spec.md REQ-F-001): --harness is trimmed and
	// lowercased before persisting; --harness-version/--harness-model are
	// trimmed opaque strings only (never lowercased) per
	// .claude/rules/go/input-sanitization.md.
	harness, _ := cmd.Flags().GetString("harness")
	harness = strings.ToLower(strings.TrimSpace(harness))
	harnessVersion, _ := cmd.Flags().GetString("harness-version")
	harnessVersion = strings.TrimSpace(harnessVersion)
	harnessModel, _ := cmd.Flags().GetString("harness-model")
	harnessModel = strings.TrimSpace(harnessModel)

	svc := getClaimService()
	claimed, err := svc.Claim(ctx, services.ClaimInput{
		EntityType:     entityType,
		EntityKey:      key,
		ClaimedBy:      by,
		SessionID:      session,
		Force:          force,
		Harness:        harness,
		HarnessVersion: harnessVersion,
		HarnessModel:   harnessModel,
	})
	if err != nil {
		cli.Error(err.Error())
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(claimed)
	}
	cli.Success(fmt.Sprintf("Claimed %s %s (session %s, by %s)", entityType, claimed.EntityKey, claimed.SessionID, claimed.ClaimedBy))
	cli.Info(fmt.Sprintf("Heartbeat with: shark heartbeat %s --session=%s", key, claimed.SessionID))
	return nil
}

func runUnclaim(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	key := strings.ToUpper(strings.TrimSpace(args[0]))
	entityType := DetectEntityType(key)
	if entityType == "" {
		return fmt.Errorf("unknown key format: %s", key)
	}
	session, _ := cmd.Flags().GetString("session")
	outcome, _ := cmd.Flags().GetString("outcome")
	force, _ := cmd.Flags().GetBool("force")

	svc := getClaimService()
	released, err := svc.Release(ctx, entityType, key, session, outcome, force)
	if err != nil {
		cli.Error(err.Error())
		return err
	}
	if !released {
		cli.Warning(fmt.Sprintf("No matching claim to release on %s %s", entityType, key))
		return nil
	}
	cli.Success(fmt.Sprintf("Released claim on %s %s", entityType, key))
	return nil
}

func runHeartbeat(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	key := strings.ToUpper(strings.TrimSpace(args[0]))
	entityType := DetectEntityType(key)
	if entityType == "" {
		return fmt.Errorf("unknown key format: %s", key)
	}
	session, _ := cmd.Flags().GetString("session")
	if strings.TrimSpace(session) == "" {
		cli.Error("--session is required for heartbeat")
		return fmt.Errorf("missing --session")
	}
	note, _ := cmd.Flags().GetString("note")
	var progress *float64
	if cmd.Flags().Changed("progress") {
		p, _ := cmd.Flags().GetFloat64("progress")
		if p < 0.0 || p > 1.0 {
			cli.Error("--progress must be between 0.0 and 1.0")
			return fmt.Errorf("progress out of range: %v", p)
		}
		progress = &p
	}

	svc := getClaimService()
	if err := svc.Heartbeat(ctx, entityType, key, session, progress, note); err != nil {
		cli.Error(err.Error())
		return err
	}
	cli.Success(fmt.Sprintf("Renewed lease on %s %s", entityType, key))
	return nil
}

func runClaims(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	svc := getClaimService()
	claims, err := svc.List(ctx)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(claims)
	}
	if len(claims) == 0 {
		cli.Info("No active claims")
		return nil
	}
	headers := []string{"Entity", "Key", "Claimed By", "Session", "Claimed At", "Progress"}
	rows := make([][]string, 0, len(claims))
	for _, c := range claims {
		prog := "-"
		if c.Progress != nil {
			prog = fmt.Sprintf("%.0f%%", *c.Progress*100)
		}
		rows = append(rows, []string{
			c.EntityType, c.EntityKey, c.ClaimedBy, c.SessionID,
			c.ClaimedAt.Format(time.RFC3339), prog,
		})
	}
	cli.OutputTable(headers, rows)
	return nil
}
