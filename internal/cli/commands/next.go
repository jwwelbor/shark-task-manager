// Package commands provides CLI command implementations.
//
// This file implements keyed `shark next <entity-key>` dispatch. It returns a
// single dispatch step's metadata and fully-rendered prompt as JSON, then
// exits. Bare `shark next` (no entity key) is invalid — work selection lives
// in `shark plan` (see plan.go). The harness owns the keyed dispatch loop:
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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	configworkflow "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// nextAdapters bundles the three per-entity-type adapters resolveNext needs
// (transitioner, placeholder generator, narrowed action service). Constructing
// these is cheap individually — the underlying DB / workflow / root action
// service are singletons — but in a cascade chain (epic → feature → task)
// resolveNext is called several times and was rebuilding the same adapters on
// each hop. We cache per-invocation so each entity type gets built at most
// once per top-level `shark next` call (TD-020).
type nextAdapters struct {
	transitioner runner.EntityTransitioner
	generator    runner.PlaceholderGenerator
	actionSvc    action.ActionService
}

// nextAdapterCache is the per-invocation cache hoisted into runNext and passed
// through resolveNext recursion. Lookup is keyed by entity type; entries are
// populated lazily on first use and reused for the remainder of the call.
// Reset between top-level `shark next` calls (no cross-invocation caching).
type nextAdapterCache struct {
	entries       map[string]*nextAdapters
	actionSvcRoot action.ActionService
	fanout        bool
}

// newNextAdapterCache constructs an empty cache with the root action service
// resolved once. The root is shared across all entity types — only the
// ForEntity-narrowed view differs per type.
func newNextAdapterCache(ctx context.Context) (*nextAdapterCache, error) {
	root, err := cli.GetActionService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize action service: %w", err)
	}
	return &nextAdapterCache{
		entries:       make(map[string]*nextAdapters),
		actionSvcRoot: root,
	}, nil
}

// get returns the cached adapter triple for entityType, constructing it on
// first lookup. Errors propagate from buildTransitioner (the only fallible
// builder); ForEntity and buildPlaceholderGenerator never error.
func (c *nextAdapterCache) get(ctx context.Context, entityType string) (*nextAdapters, error) {
	if a, ok := c.entries[entityType]; ok {
		return a, nil
	}
	transitioner, err := nextBuildTransitioner(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}
	a := &nextAdapters{
		transitioner: transitioner,
		generator:    nextBuildPlaceholderGenerator(ctx, entityType),
		actionSvc:    narrowActionServiceForEntity(c.actionSvcRoot, entityType),
	}
	c.entries[entityType] = a
	return a, nil
}

// nextBuildTransitioner and nextBuildPlaceholderGenerator are indirection
// hooks for testing — production code points them at the run.go builders.
// Tests can swap in counting wrappers to assert the cache eliminates
// redundant construction. Not for general use; this package owns both ends.
var (
	nextBuildTransitioner         = buildTransitioner
	nextBuildPlaceholderGenerator = buildPlaceholderGenerator
	nextNewAdapterCache           = newNextAdapterCache
	// nextGetSequentialDispatch is an indirection hook for testing —
	// production code points it at the real cli.GetConfig() accessor.
	// Returns true when sequential_dispatch is configured, forcing today's
	// legacy single-track cascade instead of fan-out.
	nextGetSequentialDispatch = func() bool {
		cfg, err := cli.GetConfig()
		if err != nil {
			return false
		}
		return cfg.GetSequentialDispatch()
	}
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
	Status     string `json:"status"`           // current status of the entity
	Action     string `json:"action"`           // dispatch action (see above)
	AgentType  string `json:"agent_type"`       // agent type to spawn ("" when action != spawn_agent)
	Provider   string `json:"provider"`         // AI provider (e.g., "anthropic", "openai")
	Model      string `json:"model"`            // model override
	Effort     string `json:"effort,omitempty"` // reasoning-effort override (low, medium, high, xhigh)
	Prompt     string `json:"prompt"`           // fully-rendered, skill-inlined prompt

	// ResolvedVia records the parent entity keys the engine traversed when
	// `shark next` was called on a parent whose status mapped to action
	// "cascade". The harness never receives "cascade" on the wire — the
	// engine recurses internally and returns the eventual dispatch step.
	// resolved_via is the audit trail so the harness can understand what
	// got skipped through. Empty when no cascade happened.
	ResolvedVia []string `json:"resolved_via,omitempty"`

	// UnresolvedPlaceholders lists the `<token>` names still present in
	// Prompt after rendering (e.g. "<task_id>"). A non-empty list means the
	// agent will receive literal placeholder text instead of real data —
	// usually a missing variable in the placeholder map or an authoring bug
	// in the template. Empty when the prompt fully rendered (BUG-3/4: this
	// used to be stderr-only, which machine consumers had to parse out of an
	// unstructured log line; it is now part of the stable JSON contract too.
	// The stderr WARN line is still emitted unconditionally per E32-F07
	// REQ-F-003 — this field doesn't replace it, it gives structured
	// consumers the token identities without scraping stderr).
	UnresolvedPlaceholders []string `json:"unresolved_placeholders,omitempty"`

	Error string `json:"error,omitempty"`

	// selection is an unexported carrier used to return a hierarchy selection
	// through the shared NextResponse type instead of the plain wire shape
	// above: `shark plan` sets it for a one-level selection, and keyed
	// `shark next` sets it when cascade stops at a fork. Unexported, so JSON
	// marshaling of NextResponse itself is unaffected.
	selection *HierarchyPlanSelectionResponse
}

var nextCmd = newNextCommand()

func newNextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "next <entity-key>",
		Short: "Get the next entity dispatch step as JSON",
		Long: `Return the next dispatch step for the given entity as JSON, then exit.

An entity key is required. Bare 'shark next' (no key) is invalid — use
'shark plan' to select an epic, hierarchy tier, or standalone-collection root
to work next.

Unlike 'shark run', which executes the full dispatch loop in-process,
'shark next <entity-key>' returns a single step and lets the harness drive the loop.
This is the canonical entry point for harness-side dispatch in Shark 2.0.
While resolving the dispatch, the engine may auto-advance cascade-complete
parents or agentless advance_status placeholders.

Keyed dispatch JSON shape:
  {
    "entity_key":   "<key>",
    "entity_type":  "task" | "feature" | "epic" | "bug" | "change" | "tech_debt",
    "status":       "<current status>",
    "action":       "spawn_agent" | "pause" | "archive",
    "agent_type":   "<agent type if action=spawn_agent>",
    "provider":     "<provider>",
    "model":        "<model override>",
    "prompt":       "<fully-rendered, skill-inlined prompt>",
    "unresolved_placeholders": ["<token>", ...]  // omitted when empty
  }

Examples:
  shark next E07-F01-001              # JSON dispatch step for a task
  shark next E07-F01                  # Feature
  shark next E07                      # Epic

Errors:
  - No entity key       → exit 1, see 'shark plan'
  - Unknown entity key  → exit 1
  - Entity in a state with no orchestrator action → action="pause" (not error)
	  - Internal failure rendering the prompt → exit 2 with stderr context`,
		Args: requireNextEntityKey,
		RunE: runNext,
	}
}

// requireNextEntityKey rejects bare `shark next` before any portfolio,
// planning, or dispatch service is constructed. An entity key is mandatory —
// work selection lives in `shark plan`.
func requireNextEntityKey(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("shark next requires an entity key (e.g. \"shark next E07-F01-001\"); use \"shark plan\" to select what to work on next")
	}
	return cobra.ExactArgs(1)(cmd, args)
}

func init() {
	nextCmd.Flags().Bool("sequential", false, "Force the legacy single-track cascade instead of fan-out")
	cli.RootCmd.AddCommand(nextCmd)
}

func runNext(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Start a span for this dispatch step so harness-side operators can
	// trace prompt rendering latency and catch unresolved placeholder leaks
	// per-dispatch. When OTel is disabled the tracer is a noop and all span
	// calls compile away to sub-microsecond stubs.
	tracer := cli.GetTracer("shark.cli")
	ctx, span := tracer.Start(ctx, "shark.next")
	defer span.End()

	// Step 1: Parse and detect entity type. requireNextEntityKey already
	// rejected a bare invocation, so args[0] is guaranteed to be present.

	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("invalid entity key %q: %w", args[0], err)
	}

	// Build a per-invocation adapter cache so cascade recursion doesn't
	// rebuild the same transitioner / placeholder generator / action service
	// view on every hop (TD-020). Cache is scoped to this top-level call;
	// it's reset for each subsequent `shark next` invocation.
	adapters, err := nextNewAdapterCache(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	adapters.fanout = !resolveSequentialDispatch(cmd)

	resp, err := resolveNext(ctx, adapters, entityType, normalizedKey, 0)
	if err != nil {
		// Unrendered agent-body tokens surface here via the per-step
		// RenderAndLintAgentBody pass inside attachAgentBody. Action-prompt
		// content is no longer post-render scanned because action prompts
		// legitimately use `<...>` as instructional prose (e.g.
		// `<enhancement-feature>`, `<findings>`); only the agent body
		// region — substituted by RenderAgentBody, which is silent on miss —
		// needs the loudness guarantee.
		var tokErr *templates.UnrenderedTokenError
		if errors.As(err, &tokErr) {
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.String("entity_key", normalizedKey),
				attribute.String("entity_type", entityType),
				attribute.String("exit_status", "error"),
			)
			fmt.Fprintf(os.Stderr, "[shark next] %s (entity %s)\n", tokErr.Error(), normalizedKey)
			os.Exit(3)
		}
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.String("entity_key", normalizedKey),
			attribute.String("entity_type", entityType),
			attribute.String("exit_status", "error"),
		)
		return err
	}

	// Fan-out stopped at a fork: emit the candidate tier instead of a single
	// dispatch step. A fork carries no prompt — it is a selection surface, so
	// the placeholder/agent-body annotation below does not apply.
	if resp.selection != nil {
		span.SetAttributes(
			attribute.String("entity_key", normalizedKey),
			attribute.String("entity_type", entityType),
			attribute.String("action", resp.selection.Action),
			attribute.String("selection_reason", resp.selection.SelectionReason),
			attribute.Int("candidates", len(resp.selection.Entities)),
			attribute.String("exit_status", "ok"),
		)
		return outputHierarchyPlanSelectionJSON(*resp.selection)
	}

	// Record per-dispatch span attributes so traces capture the full
	// dispatch decision for this entity step.
	resp = annotateUnresolvedPlaceholders(resp)
	span.SetAttributes(
		attribute.String("entity_key", resp.EntityKey),
		attribute.String("entity_type", resp.EntityType),
		attribute.String("status", resp.Status),
		attribute.String("action", resp.Action),
		attribute.String("agent_type", resp.AgentType),
		attribute.String("model", resp.Model),
		attribute.Int("prompt_bytes", len(resp.Prompt)),
		attribute.Int("unresolved_placeholders", len(resp.UnresolvedPlaceholders)),
		attribute.String("exit_status", "ok"),
	)
	warnUnresolvedPlaceholdersToStderr(resp)

	return outputNextJSON(resp)
}

// resolveNext is the recursive core of `shark next`. It produces a single
// dispatch step for (entityType, key) at the given recursion depth, with
// cascade resolution handled inline: if the entity's status maps to action
// "cascade", resolveNext picks the first dispatchable child and calls
// itself on the child's key, prepending the parent key to resolved_via on
// the way back up. The returned NextResponse is always wire-shaped — no
// "cascade" verb ever leaks out of this function.
func resolveNext(ctx context.Context, cache *nextAdapterCache, entityType, normalizedKey string, depth int) (NextResponse, error) {
	if depth > maxCascadeDepth {
		return NextResponse{
			EntityKey:  normalizedKey,
			EntityType: entityType,
			Action:     "error",
			Error:      fmt.Sprintf("cascade depth limit (%d) exceeded — likely a misconfigured workflow", maxCascadeDepth),
		}, nil
	}

	// Step 2: Resolve the per-entity-type adapter triple (transitioner,
	// placeholder generator, narrowed action service) from the per-invocation
	// cache. On a cold cache the first call for each entity type does the
	// construction work; subsequent recursion hops on the same type are O(1)
	// map lookups (TD-020). Using the same triple as `run` keeps `next` and
	// `run` semantically identical at the per-step level — the only difference
	// is who owns the loop.
	a, err := cache.get(ctx, entityType)
	if err != nil {
		return NextResponse{}, err
	}
	transitioner := a.transitioner
	placeholderGen := a.generator
	actionSvc := a.actionSvc

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
	if nextInfo.IsTerminal || isArchivedStatus(entityType, currentStatus) {
		if isArchivedStatus(entityType, currentStatus) {
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
	//
	// Graceful degradation (B022): when the entity's current status is not
	// defined in the workflow YAML (e.g. a legacy status like "in_approval"
	// or "ready_for_approval" that was removed from the workflow config), we
	// treat it as a terminal pause rather than crashing. This keeps the
	// harness from exiting non-zero on databases that contain any legacy
	// status values — the unknown status is surfaced as a warning on stderr
	// and the harness receives action="pause" so it can surface the situation
	// to the user rather than failing opaquely.
	populated, err := actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
	if err != nil {
		if isStatusNotFoundError(err) {
			// Unknown/legacy status — degrade to pause so the harness can
			// report it without a non-zero exit.
			fmt.Fprintf(os.Stderr, "[shark next] warning: status %q is not defined in the workflow configuration for entity type %q — treating as pause (B022)\n", currentStatus, entityType)
			resp.Action = "pause"
			resp.Error = fmt.Sprintf("status %q is not defined in the workflow configuration; this may be a legacy status that has been removed", currentStatus)
			return resp, nil
		}
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
	if actionRequiresInstruction(internalAction) && strings.TrimSpace(populated.Instruction) == "" {
		return NextResponse{}, fmt.Errorf("workflow action for %s status %q rendered an empty instruction; check the configured prompt path/template", normalizedKey, currentStatus)
	}
	if internalAction == "cascade" {
		if cache.fanout {
			return tryCascadeFanout(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
		}
		return tryCascade(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
	}

	// Step 8: Verb normalization + action application. The YAML's internal
	// verb vocabulary (spawn_agent, check_or_resume, advance_status, pause,
	// archive) is richer than the harness wire vocabulary {spawn_agent,
	// pause, archive}. applyWireAction maps onto the wire set and handles
	// the special-case branches (advance_and_recurse, error) inline so the
	// caller only deals with the simple wire-shaped result.
	wireResp, handled, err := applyWireAction(ctx, cache, entityType, normalizedKey, depth, internalAction, populated, nextInfo, transitioner, resp)
	if err != nil {
		return NextResponse{}, err
	}
	if handled {
		return wireResp, nil
	}
	resp = wireResp

	// Step 9: Auto-inline the agent body so the harness receives the agent
	// persona / config alongside the action prompt. attachAgentBody runs
	// the agent-body region through RenderAndLintAgentBody, which fails
	// loudly if any `<token>` is unmapped — the lint that used to live as
	// a post-render guard on the whole prompt, now scoped to just the
	// agent body region.
	attached, err := assembleDispatchPrompt(resp.Prompt, resp.AgentType, vars)
	if err != nil {
		return NextResponse{}, err
	}
	resp.Prompt = attached

	return resp, nil
}

func actionRequiresInstruction(internalAction string) bool {
	return internalAction == action.ActionSpawnAgent || internalAction == action.ActionCheckOrResume
}

// tryCascade owns the "cascade" verb's children-loop and resolved_via
// threading. It iterates dispatchable children of (entityType, key),
// recursing into resolveNext on each until one returns a non-pause/archive
// action. The first parent on the chain is prepended to ResolvedVia so the
// harness can audit which entities were traversed.
//
// When all children are non-dispatchable, the parent normally pauses. The one
// exception is when the service can prove there were children and every one of
// them is terminal; in that case the parent is auto-advanced one configured
// step and resolveNext recurses on the same entity so feature/epic workflows
// move into code_review/completed instead of stalling at 100% child completion.
//
// resp is the partially-filled NextResponse for the parent entity; the
// function mutates and returns it on the all-paused path.
func tryCascade(
	ctx context.Context,
	cache *nextAdapterCache,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	childrenState, err := nextDescribeDispatchableChildren(ctx, entityType, normalizedKey)
	if err != nil {
		return NextResponse{}, fmt.Errorf("cascade lookup failed for %s: %w", normalizedKey, err)
	}
	// E35-F03: hand out only unclaimed entities. A child held by a live lease
	// (claim within TTL) is skipped like any other non-dispatchable sibling;
	// unclaimed and expired-lease children pass through. Fail-soft: if the
	// claim lookup errors, the child is treated as claimable (no skip), so the
	// claim layer can never wedge dispatch.
	claimSvc := cli.GetClaimService()
	for _, child := range childrenState.Children {
		claimable, cerr := claimSvc.IsClaimable(ctx, string(child.EntityType), child.Key)
		if cerr != nil {
			// Fail-soft: treat as claimable so the claim layer can never wedge
			// dispatch, but surface the lookup failure so it isn't silent.
			fmt.Fprintf(os.Stderr, "[shark next] claim lookup failed for %s %s; treating as claimable: %v\n",
				child.EntityType, child.Key, cerr)
		} else if !claimable {
			continue
		}
		childResp, err := resolveNext(ctx, cache, string(child.EntityType), child.Key, depth+1)
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
	if childrenState.TotalChildren > 0 && childrenState.NonTerminalChildren == 0 {
		return autoAdvanceCascadeParent(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
	}
	// All children either non-dispatchable or absent — pause the parent.
	resp.Action = "pause"
	return resp, nil
}

// tryCascadeFanout is the fan-out counterpart to tryCascade: it cascades while
// there is exactly one thing to do, and stops at a *fork* — a tier with two or
// more tied dispatchable children — returning that candidate tier instead of
// silently picking children[0]. Surfacing the set is what lets the rider fan
// out; tryCascade structurally hides it.
//
// Child selection reuses plan's engine (PlanHierarchyService.DescribeChildren +
// selectPlanChildTier) rather than CascadeService, so claim-filtering,
// dependency-filtering, and tie-tiering are computed exactly once, in one
// place, and dependency terminality is config-driven at every tier. `shark next
// --sequential` (or sequential_dispatch in config) still routes to tryCascade.
//
// The loop preserves tryCascade's sibling fall-through: a selected child that
// resolves to pause/archive is dropped and the tier is recomputed over the
// remaining siblings, because "the next available work" excludes work that is
// paused, claimed, or blocked. Re-tiering (rather than a linear scan to the
// next single child) means that when the order-1 child pauses, the order-2
// children come back as a fork rather than an arbitrary pick — the same
// principle applied one tier down. Termination is trivial: the fork branch
// returns immediately, so each iteration consumes exactly one child.
func tryCascadeFanout(
	ctx context.Context,
	cache *nextAdapterCache,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	childrenState, err := planDescribeDispatchableChildren(ctx, entityType, normalizedKey)
	if err != nil {
		return NextResponse{}, fmt.Errorf("cascade lookup failed for %s: %w", normalizedKey, err)
	}

	remaining := childrenState.Children
	for len(remaining) > 0 {
		selected, reason := selectPlanChildTier(remaining)
		if len(selected) == 0 {
			break
		}

		// Resolve every tied candidate before deciding what to return.
		// DescribeChildren filters on terminal / claimed / dependency state but
		// has no view of the workflow *action*, so a child it hands back can
		// still resolve to pause (e.g. parked at a human gate). Emitting such a
		// tier as parallel_candidates would strand the run: the rider
		// dispatches each candidate, gets pause for each, and stops — never
		// reaching a dispatchable sibling behind them. That is the fall-through
		// tryCascade has always had, and it has to apply to a tie as much as to
		// a lone child.
		dispatchable := make([]services.PlanHierarchyChild, 0, len(selected))
		var firstDispatch NextResponse
		for _, child := range selected {
			childResp, err := resolveNext(ctx, cache, string(child.EntityType), child.Key, depth+1)
			if err != nil {
				return NextResponse{}, err
			}
			if childResp.Action == "error" {
				// Propagate child errors up untouched, as tryCascade does.
				return prependFanoutParent(childResp, normalizedKey), nil
			}
			// A nested selection means the child has dispatchable descendants
			// of its own, so it counts as live work.
			if childResp.selection == nil && childResp.Action != "spawn_agent" {
				continue
			}
			if len(dispatchable) == 0 {
				firstDispatch = childResp
			}
			dispatchable = append(dispatchable, child)
		}

		switch {
		case len(dispatchable) == 0:
			// The whole tier is parked. Drop it and re-tier over the siblings
			// behind it. selected is always the leading run of remaining.
			remaining = remaining[len(selected):]
			continue

		case len(dispatchable) == 1:
			return prependFanoutParent(firstDispatch, normalizedKey), nil
		}

		// A cap below 2 means the operator has asked for no fan-out. Honor it
		// by dispatching the first surviving candidate: letting
		// buildHierarchyPlanSelection truncate the tier to one would emit a
		// singleton `select_<type>` envelope, which carries no prompt and is
		// outside the keyed-next wire vocabulary, stalling the harness.
		maxParallelItems := planGetMaxParallelItems()
		if maxParallelItems < 2 {
			return prependFanoutParent(firstDispatch, normalizedKey), nil
		}

		// Fork: hand the surviving tied candidates back and let the rider
		// decide which subset is integration-safe to run concurrently.
		selection := buildHierarchyPlanSelection(
			normalizedKey,
			entityType,
			dispatchable,
			reason,
			maxParallelItems,
		)
		if err := attachForkCandidateEdges(ctx, &selection); err != nil {
			return NextResponse{}, err
		}
		selection.ResolvedVia = []string{normalizedKey}
		resp.selection = &selection
		return resp, nil
	}

	if childrenState.TotalChildren > 0 && childrenState.NonTerminalChildren == 0 {
		return autoAdvanceCascadeParent(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
	}
	// All children either non-dispatchable or absent — pause the parent.
	resp.Action = "pause"
	return resp, nil
}

// resolveSequentialDispatch reports whether `shark next` should use the legacy
// single-track cascade. Precedence: an explicitly passed --sequential flag
// wins, else the sequential_dispatch config value, else the default (fan-out).
//
// This is a named function rather than three lines inline in runNext so a test
// can exercise the real resolution instead of restating it.
func resolveSequentialDispatch(cmd *cobra.Command) bool {
	if cmd != nil && cmd.Flags().Changed("sequential") {
		sequential, err := cmd.Flags().GetBool("sequential")
		if err == nil {
			return sequential
		}
	}
	return nextGetSequentialDispatch()
}

// prependFanoutParent records parentKey as the entity traversed to reach this
// response, on whichever trail the response carries: a nested fork's
// resolved_via, or a dispatch's ResolvedVia.
func prependFanoutParent(resp NextResponse, parentKey string) NextResponse {
	if resp.selection != nil {
		resp.selection.ResolvedVia = append(
			[]string{parentKey}, resp.selection.ResolvedVia...)
		return resp
	}
	resp.ResolvedVia = append([]string{parentKey}, resp.ResolvedVia...)
	return resp
}

// fanoutDescribeCandidateEdges is a test seam for the fork path's edge load.
// It lives here rather than beside plan.go's hooks because the keyed fork is
// the only caller — `shark plan` selections stay deliberately edge-less.
var fanoutDescribeCandidateEdges = describePlanCandidateEdges

// attachForkCandidateEdges loads each fork candidate's dependency, blocker,
// and link edges and attaches them to the selection.
//
// A load failure is fatal rather than fail-soft, unlike the claim lookup in
// tryCascade. The asymmetry is deliberate: a missing claim errs toward
// offering work that turns out to be taken, which the next call corrects,
// whereas missing edges are indistinguishable on the wire from "this
// candidate has no dependencies" and would invite the rider to launch
// genuinely coupled work in parallel. Silence is the dangerous answer here,
// so the fork fails loudly instead.
func attachForkCandidateEdges(ctx context.Context, selection *HierarchyPlanSelectionResponse) error {
	if selection == nil {
		return nil
	}
	// Cover both envelope shapes. Entities is the fork case; Entity is the
	// singleton buildHierarchyPlanSelection emits when the tier holds one
	// candidate. Skipping the singleton would ship it edge-less, which is
	// exactly the silently-dependency-free state this function exists to
	// prevent.
	candidates := selection.Entities
	if selection.Entity != nil {
		candidates = append(append([]HierarchyPlanCandidate{}, candidates...), *selection.Entity)
	}
	if len(candidates) == 0 {
		return nil
	}
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		keys = append(keys, candidate.EntityKey)
	}
	edges, err := fanoutDescribeCandidateEdges(ctx, candidates[0].EntityType, keys)
	if err != nil {
		return fmt.Errorf(
			"failed to load dependency edges for the %s fork under %s: %w",
			candidates[0].EntityType, selection.RootKey, err,
		)
	}
	applyCandidateEdges(selection, edges)
	return nil
}

// autoAdvanceCascadeParent handles tryCascade's all-children-terminal case:
// the parent is advanced one happy-path step and resolveNext recurses on the
// same entity, so feature/epic workflows move into code_review/completed
// instead of stalling at 100% child completion.
//
// The target is the pass-first default transition (AvailableTransitions[0]) —
// cascade statuses normally expose several transitions (pass/fail/blocked
// targets), so demanding exactly one would never fire against the shipped
// workflows. A status with no forward transition pauses the parent with a
// descriptive error.
func autoAdvanceCascadeParent(
	ctx context.Context,
	cache *nextAdapterCache,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	if nextInfo == nil || len(nextInfo.AvailableTransitions) == 0 {
		resp.Action = "pause"
		resp.Error = fmt.Sprintf(
			"all child work is terminal, but %s %s at status %q has no forward transition to auto-advance to",
			entityType, normalizedKey, resp.Status,
		)
		return resp, nil
	}
	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	opts := services.TransitionOptions{
		Reason: "cascade completion: all child work terminal",
		Agent:  "shark-cascade",
	}
	if _, err := transitioner.TransitionStatus(ctx, normalizedKey, targetStatus, opts); err != nil {
		return NextResponse{}, fmt.Errorf(
			"cascade completion advance from %s to %s failed for %s: %w",
			resp.Status, targetStatus, normalizedKey, err,
		)
	}
	return resolveNext(ctx, cache, entityType, normalizedKey, depth+1)
}

// applyWireAction maps the YAML internal verb onto a wire-vocabulary action
// and handles the two non-trivial branches inline:
//
//   - "advance_and_recurse": the engine transitions the entity forward to
//     transitionalTarget and recurses on the same key, returning whatever the
//     next status dispatches to. Used for auto-advance placeholder statuses
//     (advance_status with no agent_type).
//   - "error": the verb is unknown — surface as pause with a clear Error
//     field rather than silently dropping the entity.
//
// For the simple wire actions (spawn_agent, pause, archive), the function
// fills resp with the populated action's fields and returns handled=false
// so the caller can run any remaining post-processing (e.g., agent body
// inlining). When handled=true, the returned NextResponse is final and
// should be returned to the caller untouched.
func applyWireAction(
	ctx context.Context,
	cache *nextAdapterCache,
	entityType, normalizedKey string,
	depth int,
	internalAction string,
	populated *action.PopulatedAction,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
	resp NextResponse,
) (NextResponse, bool, error) {
	wireAction, transitionalTarget := normalizeWireAction(internalAction, populated.AgentType, nextInfo)
	switch wireAction {
	case "advance_and_recurse":
		// Internal verb advance_status with no agent_type: this status is an
		// auto-transition placeholder. The engine performs the advance and
		// recurses on the same key, returning whatever the next status
		// dispatches to.
		if transitionalTarget == "" {
			// No safe forward transition — leave the entity for human review.
			resp.Action = "pause"
			return resp, true, nil
		}
		if _, err := transitioner.TransitionStatus(ctx, normalizedKey, transitionalTarget, services.TransitionOptions{}); err != nil {
			return NextResponse{}, true, fmt.Errorf("auto-advance from %s to %s failed for %s: %w", resp.Status, transitionalTarget, normalizedKey, err)
		}
		recursed, err := resolveNext(ctx, cache, entityType, normalizedKey, depth+1)
		if err != nil {
			return NextResponse{}, true, err
		}
		// We didn't move to a different entity, only a different status, so
		// resolved_via isn't appropriate (it audits cross-entity hops). The
		// status field on `recursed` already reflects the new state.
		return recursed, true, nil

	case "error":
		// Unknown verb — surface to the harness as pause with a clear error
		// field so the harness shows the failure to the user instead of
		// silently dropping the entity.
		resp.Action = "pause"
		resp.Error = fmt.Sprintf("unknown internal action verb %q for status %q", internalAction, resp.Status)
		return resp, true, nil
	}

	// Simple wire action — fill the response and hand back to the caller
	// for any remaining post-processing (e.g., agent body inlining).
	resp.Action = wireAction
	resp.AgentType = populated.AgentType
	resp.Provider = populated.Provider
	resp.Model = populated.Model
	resp.Effort = populated.Effort
	resp.Prompt = populated.Instruction
	return resp, false, nil
}

// attachAgentBody inlines the agent persona / config above the rendered
// instruction prompt, separated by a horizontal rule. Per the 2026-05-10
// rendering-model decision, the response from `shark next` should be
// self-contained so the harness can spawn the agent without resolving agent
// files from its own filesystem.
//
// Graceful degradation: when agentType is empty, the data root is unknown
// (non-bundle prompt mode), or the agent file doesn't exist, the
// original prompt is returned unchanged — the harness can still spawn the
// agent by type if it has a local copy.
//
// Placeholder rendering: agent files use kebab-case tokens like `<task-id>`
// that the instruction-template engine (Go templates, snake_case) doesn't
// touch — RenderAndLintAgentBody closes that gap before concatenation and
// fails loudly on any surviving `<token>` (the lint scoped to the agent
// body region so action-prompt prose using `<...>` is not falsely flagged).
func attachAgentBody(prompt, agentType string, vars map[string]string) (string, error) {
	if agentType == "" {
		return prompt, nil
	}
	root := templates.GetOrchestratorEngine().IncludeRoot()
	body, ok := LoadAgentBodyForInline(root, agentType)
	if !ok {
		return prompt, nil
	}
	rendered, err := templates.RenderAndLintAgentBody(body, agentType, vars)
	if err != nil {
		return "", err
	}
	return rendered + "\n\n---\n\n" + prompt, nil
}

const workerOwnershipPreamble = `PARENT LOOP OWNERSHIP CONTRACT:
- You are a spawned worker inside a Shark parent-run loop.
- Do NOT run Shark workflow-state commands against the entity this prompt dispatched you for.
- Never run against the dispatched entity: shark claim, shark heartbeat, shark release, shark status advance, shark status set, shark task next-status, shark task set-status, shark feature next-status, or shark epic next-status.
- Exception: if the workflow prompt below explicitly makes you an orchestration loop over OTHER entities (e.g. a sprint step iterating "shark sprint next" and dispatching each child), driving those child entities is the requested work — the prohibition above still applies to the dispatched entity itself.
- Operate in single-worker mode by default. Do NOT spawn or delegate to additional host-native subagents, agent teams, or external AI CLIs unless the workflow prompt explicitly tells you to run a multi-agent skill or recipe.
- If the bundled agent persona describes broader coordination behavior, treat that as background context only. This contract and the concrete workflow prompt override it for the current dispatched step.
- Complete the requested work, write the requested artifacts, then stop and clearly report the recommended outcome and any follow-up guidance for the parent loop.
- The parent loop owns the dispatched entity's lease and workflow transitions.`

// assembleDispatchPrompt is the final Shark-owned prompt assembly step shared
// by `shark next` and `shark run`: take the already-rendered workflow prompt,
// inline the Shark specialist persona, and return the exact prompt the host
// execution primitive should receive.
func assembleDispatchPrompt(prompt, agentType string, vars map[string]string) (string, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	templates.AugmentPlaceholderAliases(vars)
	attached, err := attachAgentBody(prompt, agentType, vars)
	if err != nil {
		return "", err
	}
	return workerOwnershipPreamble + "\n\n---\n\n" + attached, nil
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
	if agentType == "" {
		return "", false
	}
	// NewIncludeResolverWithEmbed falls back to the embedded canonical tree
	// when root is empty (zero-config mode) or a file is absent on disk.
	resolver := templates.NewIncludeResolverWithEmbed(root)
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

// annotateUnresolvedPlaceholders scans resp.Prompt for surviving `<token>`
// placeholders and records their identities on resp.UnresolvedPlaceholders.
// A non-empty result means the agent will receive literal placeholder text
// instead of the real data value — usually a missing variable in the
// placeholder map or an authoring bug in the template. This used to be
// stderr-only (BUG-3/4); the JSON field is now a structured contract
// alongside the still-unconditional stderr WARN (E32-F07 REQ-F-003).
func annotateUnresolvedPlaceholders(resp NextResponse) NextResponse {
	resp.UnresolvedPlaceholders = templates.UnrenderedTokens(resp.Prompt)
	return resp
}

// warnUnresolvedPlaceholdersToStderr emits the "[shark-stats] WARN" line
// unconditionally when resp.UnresolvedPlaceholders is non-empty, per E32-F07
// REQ-F-003 ("so trial defects are impossible to miss" for operators/harnesses
// watching stderr across many invocations without parsing every JSON response
// body). This is a separate stream from the stdout JSON contract — it does
// not affect --json consumers unless they explicitly merge 2>&1.
func warnUnresolvedPlaceholdersToStderr(resp NextResponse) {
	if len(resp.UnresolvedPlaceholders) > 0 {
		fmt.Fprintf(os.Stderr, "[shark-stats] WARN: %s has %d unresolved placeholders\n", resp.EntityKey, len(resp.UnresolvedPlaceholders))
	}
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

// isArchivedStatus reports whether a status name indicates an archived /
// terminal entity for the given entity type. The canonical terminal set is
// resolved against the entity's own workflow.Service (so custom workflows
// that rename "completed" still route correctly — see B028), and we also
// keep a loose `_archived` suffix match for the cross-vocabulary cases
// where some workflows use names like "in_qa_archived".
func isArchivedStatus(entityType, status string) bool {
	s := strings.ToLower(status)
	if strings.HasSuffix(s, "_archived") {
		return true
	}
	wf := cli.GetWorkflowService()
	if entityType != "" {
		// B034: narrow against the normalized type or change-cards silently
		// fall back to the task workflow's terminal-status set.
		wf = wf.ForLevel(normalizeEntityTypeForWorkflow(entityType))
	}
	return wf.IsTerminalStatus(status)
}

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

// isStatusNotFoundError reports whether err (or any error in its chain) is an
// *action.StatusNotFoundError, indicating the entity's current status is not
// defined in the workflow YAML. Used by resolveNext to apply graceful
// degradation (pause) instead of propagating an error.
//
// See B022: "shark next exits 1 on legacy task statuses instead of degrading".
func isStatusNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var snfe *action.StatusNotFoundError
	return errors.As(err, &snfe)
}

// pickAutoAdvanceTarget returns the natural forward transition for an
// auto-advance status. Workflow authors who use `advance_status` without
// an agent_type implicitly declare the status is a no-op placeholder with
// exactly one productive forward path.
//
// For route-based workflows that path is declared explicitly: the step's
// `pass` outcome. Selecting by outcome key (rather than scanning the ordered
// transition slice for the first "forward-looking" status) means a terminal
// pass target — e.g. a placeholder whose pass ends the workflow — is honored
// instead of being skipped in favor of the fail target. A pass self-loop
// yields "" (pause): the step declares no forward motion.
//
// When the pass outcome cannot be resolved (legacy workflow, or a status not
// in the step graph), we fall back to the transition scan: filter out the
// obviously non-productive transitions (a self-loop back to the current
// status, a terminal/abandonment state, a parking step, or any step in the
// blocked phase) and take the first remaining option. Classification is
// derived from the workflow step metadata via the workflow.Service, not from
// hardcoded status names (B028) — so custom workflows that rename
// "completed"/"blocked"/etc. still route correctly.
//
// Returns "" when there is no safe forward transition — the caller treats
// that as "pause" so a misconfigured workflow is surfaced to the user
// rather than spinning.
func pickAutoAdvanceTarget(nextInfo *services.NextStatusInfo) string {
	if nextInfo == nil {
		return ""
	}
	wf := cli.GetWorkflowService()
	if nextInfo.EntityType != "" {
		wf = wf.ForLevel(string(nextInfo.EntityType))
	}
	if wf.IsRouteBased() {
		if target, err := wf.Release(nextInfo.CurrentStatus, configworkflow.OutcomePass); err == nil {
			if strings.EqualFold(target, nextInfo.CurrentStatus) {
				return ""
			}
			return target
		}
	}
	for _, t := range nextInfo.AvailableTransitions {
		// A transition back to the current status (e.g. a fail self-loop) is
		// never a forward move.
		if strings.EqualFold(t.TargetStatus, nextInfo.CurrentStatus) {
			continue
		}
		if wf.IsTerminalStatus(t.TargetStatus) ||
			wf.IsParkingStatus(t.TargetStatus) ||
			wf.IsBlockedStatus(t.TargetStatus) {
			continue
		}
		return t.TargetStatus
	}
	return ""
}
