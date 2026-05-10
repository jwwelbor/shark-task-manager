// Package commands provides CLI command implementations.
//
// This file implements the `shark next <entity-key>` command. Unlike `shark
// run`, which executes the full dispatch loop in-process, `shark next`
// returns a single dispatch step's metadata and the fully-rendered prompt as
// JSON, then exits. The harness owns the loop:
//
//	while true:
//	  resp = shark next <key> --json
//	  case resp.action of
//	    spawn_agent -> spawn(resp.agent_type, resp.prompt); shark advance <key>
//	    pause       -> stop, surface to user
//	    archive     -> stop, surface to user
//
// Spec: F2/E3 of E02 (Shark 2.0 — Single-Artifact Consolidation). The output
// JSON shape is the contract between shark and any harness — see NextResponse.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// NextResponse is the JSON contract returned by `shark next`. The shape is
// stable; harnesses dispatch on `action` to decide what to do next.
//
// `action` values:
//   - "spawn_agent" — host should spawn an agent of `agent_type` with `prompt`,
//     then call `shark advance <key>` on completion.
//   - "pause" — entity is in a terminal/blocked/wait state for the harness;
//     stop the loop and surface to the user. `prompt` and `agent_type` may be
//     empty.
//   - "archive" — entity has reached an archived state; stop and report.
//   - "error" — see `error` field for context. Reserved for non-fatal advisory
//     errors that the harness should surface; fatal errors come back as a
//     non-zero exit code.
type NextResponse struct {
	EntityKey  string `json:"entity_key"`
	EntityType string `json:"entity_type"`
	Status     string `json:"status"`     // current status of the entity
	Action     string `json:"action"`     // dispatch action (see above)
	AgentType  string `json:"agent_type"` // agent type to spawn ("" when action != spawn_agent)
	Provider   string `json:"provider"`   // AI provider (e.g., "anthropic", "openai")
	Model      string `json:"model"`      // model override
	Prompt     string `json:"prompt"`     // fully-rendered, skill-inlined prompt
	Error      string `json:"error,omitempty"`
}

var nextPreview bool

var nextCmd = &cobra.Command{
	Use:   "next <entity-key>",
	Short: "Get the next dispatch step for an entity as JSON",
	Long: `Return the next dispatch step for an entity as JSON, then exit.

Unlike 'shark run', which executes the full dispatch loop in-process,
'shark next' returns a single step and lets the harness drive the loop.
This is the canonical entry point for harness-side dispatch in Shark 2.0.

Output JSON shape:
  {
    "entity_key":   "<key>",
    "entity_type":  "task" | "feature" | "epic" | "bug" | "change" | "tech_debt",
    "status":       "<current status>",
    "action":       "spawn_agent" | "pause" | "archive",
    "agent_type":   "<agent type if action=spawn_agent>",
    "provider":     "<provider>",
    "model":        "<model override>",
    "prompt":       "<fully-rendered, skill-inlined prompt>"
  }

Examples:
  shark next E07-F01-001              # JSON dispatch step for a task
  shark next E07-F01                  # Feature
  shark next E07                      # Epic
  shark next E07-F01-001 --preview    # Same output but with --preview the
                                       # caller signals it does not intend to
                                       # spawn — useful for harness debugging.

Errors:
  - Unknown entity key  → exit 1
  - Entity in a state with no orchestrator action → action="pause" (not error)
  - Internal failure rendering the prompt → exit 2 with stderr context`,
	Args: cobra.ExactArgs(1),
	RunE: runNext,
}

func init() {
	nextCmd.Flags().BoolVar(&nextPreview, "preview", false, "Caller advisory: signal that the harness will not actually spawn (no semantics change)")
	cli.RootCmd.AddCommand(nextCmd)
}

func runNext(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Step 1: Parse and detect entity type.
	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid entity key %q: %w", args[0], err)
	}

	// Step 2: Build the same transitioner + placeholder generator the
	// 'run' loop uses. Reusing these keeps `next` and `run` semantically
	// identical at the per-step level — the only difference is who owns
	// the loop.
	transitioner, err := buildTransitioner(ctx, entityType)
	if err != nil {
		return fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}
	placeholderGen := buildPlaceholderGenerator(ctx, entityType)

	// Step 3: Get the action service.
	actionSvc, err := cli.GetActionService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize action service: %w", err)
	}

	// Step 4: Read current status and detect terminal/archived states.
	nextInfo, err := transitioner.GetNextStatus(ctx, normalizedKey)
	if err != nil {
		return fmt.Errorf("failed to read status for %s: %w", normalizedKey, err)
	}
	currentStatus := nextInfo.CurrentStatus

	resp := NextResponse{
		EntityKey:  normalizedKey,
		EntityType: entityType,
		Status:     currentStatus,
	}

	// Terminal status: nothing to dispatch.
	if nextInfo.IsTerminal || isArchivedStatus(currentStatus) {
		if isArchivedStatus(currentStatus) {
			resp.Action = "archive"
		} else {
			resp.Action = "pause"
		}
		return outputNextJSON(resp)
	}

	// Step 5: Generate placeholders for template rendering.
	var vars map[string]string
	if placeholderGen != nil {
		vars, err = placeholderGen.GeneratePlaceholders(ctx, normalizedKey)
		if err != nil {
			return fmt.Errorf("failed to generate placeholders for %s: %w", normalizedKey, err)
		}
	}
	if vars == nil {
		vars = map[string]string{}
	}

	// Step 6: Get the populated action (template rendered + skills inlined
	// in Shark 2.0 layouts via the orchestrator renderer's {{include:}} pass).
	populated, err := actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
	if err != nil {
		return fmt.Errorf("failed to populate action for status %q: %w", currentStatus, err)
	}

	// No action defined for this status → pause; harness shows it to user.
	if populated == nil {
		resp.Action = "pause"
		return outputNextJSON(resp)
	}

	// Map PopulatedAction onto the wire shape. The "action" field on
	// PopulatedAction can be a domain-specific verb (e.g., "spawn_agent",
	// "wait_for_human", "skip"); Shark 2.0 uses "spawn_agent" by default
	// when an agent_type is present.
	resp.Action = strings.TrimSpace(populated.Action)
	if resp.Action == "" {
		if populated.AgentType != "" {
			resp.Action = "spawn_agent"
		} else {
			resp.Action = "pause"
		}
	}
	resp.AgentType = populated.AgentType
	resp.Provider = populated.Provider
	resp.Model = populated.Model
	resp.Prompt = populated.Instruction

	// Auto-inline the agent body. Per the 2026-05-10 rendering-model decision,
	// the rendered response from `shark next` should contain the agent persona /
	// config inline rather than requiring the harness to resolve agent files
	// from its own filesystem. We prepend the agent body (with frontmatter
	// stripped) above the action prompt, separated by a horizontal rule.
	//
	// If the data root is unknown (legacy shark-templates/ mode) or the agent
	// file doesn't exist, we proceed without inlining — the harness can still
	// spawn the agent by type if it has a local copy. This is graceful
	// degradation, not a hard requirement.
	if resp.AgentType != "" {
		root := templates.GetOrchestratorEngine().IncludeRoot()
		if body, ok := LoadAgentBodyForInline(root, resp.AgentType); ok {
			resp.Prompt = body + "\n\n---\n\n" + resp.Prompt
		}
	}

	return outputNextJSON(resp)
}

// LoadAgentBodyForInline resolves <root>/agents/<type>.md (with overrides/
// preference) via the engine's IncludeResolver and returns its content with
// any YAML frontmatter stripped. Returns (content, true) on success;
// ("", false) when root is empty (legacy mode), the file is missing, or any
// resolution error occurs.
//
// Exported for testability — callers in non-test code use the
// GetOrchestratorEngine().IncludeRoot() value as the root.
func LoadAgentBodyForInline(root, agentType string) (string, bool) {
	if root == "" || agentType == "" {
		return "", false
	}
	resolver := templates.NewIncludeResolver(root)
	// Construct a synthetic include directive and let the resolver do the
	// path / override / frontmatter-strip work. Reusing the resolver keeps
	// behavior identical to {{include: agents/<type>.md}} when an author
	// writes it explicitly in a prompt template.
	directive := fmt.Sprintf("{{include: agents/%s.md}}", agentType)
	resolved, err := resolver.Resolve(directive)
	if err != nil {
		// Non-fatal: agent file may legitimately not exist (e.g., legacy
		// projects mid-migration). Log to stderr for diagnostics but do
		// not fail the dispatch step.
		fmt.Fprintf(os.Stderr, "[shark next] agent body inline skipped for %q: %v\n", agentType, err)
		return "", false
	}
	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return "", false
	}
	return resolved, true
}

// outputNextJSON marshals the response. `shark next` always emits JSON to
// stdout — the global --json flag is implicit. We do not honor a non-JSON
// output mode because the entire purpose of this command is the JSON contract.
func outputNextJSON(resp NextResponse) error {
	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal next response: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

// isArchivedStatus reports whether a status name conventionally indicates an
// archived entity. The exact set varies by entity type / workflow profile;
// we accept a loose suffix match so different vocabularies route consistently.
func isArchivedStatus(status string) bool {
	s := strings.ToLower(status)
	switch s {
	case "archived", "completed", "cancelled", "done":
		return true
	}
	return strings.HasSuffix(s, "_archived")
}

// Compile-time check that buildTransitioner / buildPlaceholderGenerator
// remain in scope. These are defined in run.go; we reuse them.
var _ = context.Background
