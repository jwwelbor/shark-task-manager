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
	"github.com/jwwelbor/shark-task-manager/internal/services"
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

	// ResolvedVia records the parent entity keys the engine traversed when
	// `shark next` was called on a parent whose status mapped to action
	// "cascade". The harness never receives "cascade" on the wire — the
	// engine recurses internally and returns the eventual dispatch step.
	// resolved_via is the audit trail so the harness can understand what
	// got skipped through. Empty when no cascade happened.
	ResolvedVia []string `json:"resolved_via,omitempty"`

	Error string `json:"error,omitempty"`
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

	resp, err := resolveNext(ctx, entityType, normalizedKey, 0)
	if err != nil {
		return err
	}

	// Post-render guard: any `<token>` surviving every render pass is a bug.
	// Silent pass-through (the 2026-05-11 trial's failure mode) is the
	// scenario we explicitly want to make loud. Exit 3 (invalid state) so
	// orchestrators surface this distinctly from a missing-entity error.
	if tok, found := templates.FirstUnrenderedToken(resp.Prompt); found {
		fmt.Fprintf(os.Stderr, "[shark next] unrendered placeholder %s left in prompt for %s\n", tok, resp.EntityKey)
		os.Exit(3)
	}

	return outputNextJSON(resp)
}

// resolveNext is the recursive core of `shark next`. It produces a single
// dispatch step for (entityType, key) at the given recursion depth, with
// cascade resolution handled inline: if the entity's status maps to action
// "cascade", resolveNext picks the first dispatchable child and calls
// itself on the child's key, prepending the parent key to resolved_via on
// the way back up. The returned NextResponse is always wire-shaped — no
// "cascade" verb ever leaks out of this function.
func resolveNext(ctx context.Context, entityType, normalizedKey string, depth int) (NextResponse, error) {
	if depth > maxCascadeDepth {
		return NextResponse{
			EntityKey:  normalizedKey,
			EntityType: entityType,
			Action:     "error",
			Error:      fmt.Sprintf("cascade depth limit (%d) exceeded — likely a misconfigured workflow", maxCascadeDepth),
		}, nil
	}

	// Step 2: Build the same transitioner + placeholder generator the
	// 'run' loop uses. Reusing these keeps `next` and `run` semantically
	// identical at the per-step level — the only difference is who owns
	// the loop.
	transitioner, err := buildTransitioner(ctx, entityType)
	if err != nil {
		return NextResponse{}, fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}
	placeholderGen := buildPlaceholderGenerator(ctx, entityType)

	// Step 3: Get the action service, narrowed to this entity type.
	// ForEntity ensures status lookups resolve against the entity's own
	// workflow YAML, which is what makes cross-entity status name collisions
	// (e.g. "completed" shared by every entity) unambiguous.
	actionSvcRoot, err := cli.GetActionService(ctx)
	if err != nil {
		return NextResponse{}, fmt.Errorf("failed to initialize action service: %w", err)
	}
	actionSvc := actionSvcRoot.ForEntity(entityType)

	// Step 4: Read current status and detect terminal/archived states.
	nextInfo, err := transitioner.GetNextStatus(ctx, normalizedKey)
	if err != nil {
		return NextResponse{}, fmt.Errorf("failed to read status for %s: %w", normalizedKey, err)
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
		return resp, nil
	}

	// Step 5: Generate placeholders for template rendering. The agent body
	// and instruction template both consume this map; AugmentPlaceholderAliases
	// adds the dash-form and shorthand keys agent files use (e.g. `<task-id>`
	// resolves against `task_id`).
	var vars map[string]string
	if placeholderGen != nil {
		vars, err = placeholderGen.GeneratePlaceholders(ctx, normalizedKey)
		if err != nil {
			return NextResponse{}, fmt.Errorf("failed to generate placeholders for %s: %w", normalizedKey, err)
		}
	}
	if vars == nil {
		vars = map[string]string{}
	}
	templates.AugmentPlaceholderAliases(vars)

	// Step 6: Get the populated action (template rendered + skills inlined
	// in Shark 2.0 layouts via the orchestrator renderer's {{include:}} pass).
	populated, err := actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
	if err != nil {
		return NextResponse{}, fmt.Errorf("failed to populate action for status %q: %w", currentStatus, err)
	}

	// No action defined for this status → pause; harness shows it to user.
	if populated == nil {
		resp.Action = "pause"
		return resp, nil
	}

	// Step 7: Cascade resolution. The YAML's "cascade" verb must never
	// reach the harness — the engine traverses down to the first
	// dispatchable child here, then returns that child's dispatch step.
	internalAction := strings.TrimSpace(populated.Action)
	if internalAction == "cascade" {
		children, err := listDispatchableChildren(ctx, entityType, normalizedKey)
		if err != nil {
			return NextResponse{}, fmt.Errorf("cascade lookup failed for %s: %w", normalizedKey, err)
		}
		for _, child := range children {
			childResp, err := resolveNext(ctx, child.EntityType, child.Key, depth+1)
			if err != nil {
				return NextResponse{}, err
			}
			switch childResp.Action {
			case "spawn_agent":
				// Found something to do — prepend this parent to the trail
				// and propagate the child's response untouched.
				childResp.ResolvedVia = append([]string{normalizedKey}, childResp.ResolvedVia...)
				return childResp, nil
			case "pause", "archive":
				// This child has nothing dispatchable right now (blocked,
				// already done, etc.) — try the next sibling.
				continue
			case "error":
				// Propagate child errors up untouched, with parent in the trail.
				childResp.ResolvedVia = append([]string{normalizedKey}, childResp.ResolvedVia...)
				return childResp, nil
			}
		}
		// All children either non-dispatchable or absent — pause the parent.
		resp.Action = "pause"
		return resp, nil
	}

	// Step 8: Verb normalization. The YAML's internal verb vocabulary
	// (spawn_agent, check_or_resume, advance_status, pause, archive) is
	// richer than the harness wire vocabulary {spawn_agent, pause, archive}.
	// Map onto the wire set here so the harness only ever sees what it
	// knows how to act on.
	wireAction, transitionalTarget := normalizeWireAction(internalAction, populated.AgentType, nextInfo)
	switch wireAction {
	case "advance_and_recurse":
		// Internal verb advance_status with no agent_type: this status is an
		// auto-transition placeholder. The engine performs the advance and
		// recurses on the same key, returning whatever the next status
		// dispatches to. Track the original key in resolved_via so the harness
		// can audit the auto-advance.
		if transitionalTarget == "" {
			// No safe forward transition — leave the entity for human review.
			resp.Action = "pause"
			return resp, nil
		}
		if _, err := transitioner.TransitionStatus(ctx, normalizedKey, transitionalTarget, services.TransitionOptions{}); err != nil {
			return NextResponse{}, fmt.Errorf("auto-advance from %s to %s failed for %s: %w", currentStatus, transitionalTarget, normalizedKey, err)
		}
		recursed, err := resolveNext(ctx, entityType, normalizedKey, depth+1)
		if err != nil {
			return NextResponse{}, err
		}
		// We didn't move to a different entity, only a different status, so
		// resolved_via isn't appropriate (it audits cross-entity hops). The
		// status field on `recursed` already reflects the new state.
		return recursed, nil

	case "error":
		// Unknown verb — surface to the harness as pause with a clear error
		// field so the harness shows the failure to the user instead of
		// silently dropping the entity.
		resp.Action = "pause"
		resp.Error = fmt.Sprintf("unknown internal action verb %q for status %q", internalAction, currentStatus)
		return resp, nil
	}

	resp.Action = wireAction
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
			// Render `<token>` placeholders inside the agent body before
			// concatenating. Agent files use kebab-case tokens like
			// `<task-id>` that the instruction-template engine (Go templates,
			// snake_case) doesn't touch — this dedicated pass closes that gap.
			body = templates.RenderAgentBody(body, vars)
			resp.Prompt = body + "\n\n---\n\n" + resp.Prompt
		}
	}

	return resp, nil
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

// normalizeWireAction maps an internal YAML verb (the orchestrator_action.action
// field as authored by workflow YAMLs) onto the wire vocabulary the harness
// understands: spawn_agent, pause, archive. Two additional internal returns
// are used by the caller:
//
//   - "advance_and_recurse" — the engine should perform a status transition
//     to autoAdvanceTarget and recurse on the same key. Used for
//     advance_status verbs that have no agent_type (a no-op workflow stage
//     authored as a transient placeholder).
//   - "error" — the verb is not one we know how to handle.
//
// The autoAdvanceTarget return is only meaningful when the wire action is
// "advance_and_recurse"; otherwise it is empty.
//
// Inputs:
//   - internalAction: the verb authored in YAML
//   - agentType: the agent_type populated for this status (may be empty)
//   - nextInfo: the workflow's transition options for the current status
//
// The mapping table (kept in lock-step with the design doc):
//
//	cascade           -> already handled by resolveNext; not reached here
//	spawn_agent       -> spawn_agent
//	check_or_resume   -> spawn_agent (worker decides whether to resume)
//	advance_status w/ agent_type     -> spawn_agent
//	advance_status w/o agent_type    -> advance_and_recurse (auto-transition)
//	pause / ""        -> pause   (or spawn_agent if "" but agent_type present)
//	archive           -> archive
//	other             -> error
func normalizeWireAction(internalAction, agentType string, nextInfo *services.NextStatusInfo) (wireAction, autoAdvanceTarget string) {
	switch internalAction {
	case "spawn_agent", "check_or_resume":
		return "spawn_agent", ""
	case "advance_status":
		if agentType != "" {
			return "spawn_agent", ""
		}
		return "advance_and_recurse", pickAutoAdvanceTarget(nextInfo)
	case "pause":
		return "pause", ""
	case "archive":
		return "archive", ""
	case "":
		// Empty verb falls back on whether an agent_type was authored.
		if agentType != "" {
			return "spawn_agent", ""
		}
		return "pause", ""
	}
	// Anything not in the mapping is an authoring bug — surface as error
	// so the post-render layer can shape a useful response.
	return "error", ""
}

// pickAutoAdvanceTarget returns the natural forward transition for an
// auto-advance status. Workflow authors who use `advance_status` without
// an agent_type implicitly declare the status is a no-op placeholder with
// exactly one productive forward path. We filter out the obviously
// non-productive transitions (back-to-draft, blocked, cancelled, on_hold)
// and take the first remaining option.
//
// Returns "" when there is no safe forward transition — the caller treats
// that as "pause" so a misconfigured workflow is surfaced to the user
// rather than spinning.
func pickAutoAdvanceTarget(nextInfo *services.NextStatusInfo) string {
	if nextInfo == nil {
		return ""
	}
	for _, t := range nextInfo.AvailableTransitions {
		switch t.TargetStatus {
		case "draft", "blocked", "cancelled", "on_hold", "archived":
			continue
		}
		return t.TargetStatus
	}
	return ""
}
