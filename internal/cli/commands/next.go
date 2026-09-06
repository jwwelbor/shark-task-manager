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
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/integration"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// nextCaptureEpicIntegrationBase is resolveCascade's indirection hook for
// wiring the epic `active` step's cascade action to E34-F08's per-epic
// integration-run capture (REQ-F-004, spec.md "Integration with existing
// code"): the cascade action calls this once per invocation so the epic's
// IntegrationRun.BaseCommit is captured on the first feature dispatch under
// the epic. integration.CaptureBase is itself idempotent (its own doc
// comment: "every later call for the same epicKey ... is a no-op that
// returns the identical, already-persisted record"), so calling it on every
// cascade invocation for the epic — not just a synthetic "first" one — is
// both correct and simpler than tracking first-dispatch state here.
//
// Production points this at integration.CaptureBase. Tests that exercise
// unrelated cascade behavior (fan-out, question-blocking, etc.) override it
// to a no-op so they never touch a real git repository or the real
// `.shark/` directory. TC-011 (next_cascade_traversal_test.go), which
// exists specifically to prove production code calls CaptureBase, must
// leave this at its production default — overriding it there would defeat
// the point of that test.
var nextCaptureEpicIntegrationBase = integration.CaptureBase

// nextIntegrationCaptureFailureRecorder resolves the integration.NoteRecorder
// nextRecordCaptureFailureNote uses to durably surface a CaptureBase
// failure (UAT rework round 2, Finding 1). Production points this at a
// real *services.NoteService via cli.GetNoteService — the same accessor
// service_accessors.go's GetFeatureService uses to wire
// SetIntegrationNoteRecorder. CLI command tests must never touch a real
// database (.claude/rules/testing/cli-tests.md), so tests override this
// with a fake recorder instead of exercising cli.GetNoteService.
var nextIntegrationCaptureFailureRecorder = func(ctx context.Context) (integration.NoteRecorder, error) {
	return cli.GetNoteService(ctx)
}

// integrationCaptureFailureCreatedBy identifies the automated actor
// recorded on the durable capture-failure note nextRecordCaptureFailureNote
// writes, mirroring feature_service.go's integrationCaptureCreatedBy
// literal (unexported there, so not importable — kept as the identical
// string rather than a second, divergent constant).
const integrationCaptureFailureCreatedBy = "shark-integration-capture"

// nextIntegrationCaptureFailureNoteExists reports whether epicKey already
// carries an open `review-finding` note for CaptureBase's own failure stage
// ("capture_base"). nextCaptureEpicIntegrationBase is invoked on every
// cascade attempt for the epic (see its own doc comment), so a persistently
// failing epic — e.g. one with no git repository — would otherwise
// accumulate a new note on every single `shark next` poll a harness makes;
// this dedupes to exactly one open note per epic per failure stage.
//
// The dedupe is scoped to `disposition=="open"` by design, not merely
// because that is the value this call site writes: if an operator (or a
// future closure workflow) ever marks that note's disposition closed while
// the underlying condition still persists, the next poll writes a fresh
// note rather than staying silent — an unresolved capture failure should
// keep re-surfacing once its prior note is no longer treated as open,
// rather than being permanently deduped away by a note that no longer
// reflects the epic's real state.
func nextIntegrationCaptureFailureNoteExists(ctx context.Context, recorder integration.NoteRecorder, epicKey string) (bool, error) {
	notes, err := recorder.ListNotes(ctx, models.EntityTypeEpic, epicKey, []string{"review-finding"})
	if err != nil {
		return false, err
	}
	for _, note := range notes {
		if note.Metadata == nil {
			continue
		}
		var meta map[string]string
		if err := json.Unmarshal([]byte(*note.Metadata), &meta); err != nil {
			continue
		}
		if meta["gate"] == "integration_capture" && meta["stage"] == "capture_base" && meta["disposition"] == "open" {
			return true, nil
		}
	}
	return false, nil
}

// nextRecordCaptureFailureNote durably records a CaptureBase failure as an
// epic-level `review-finding` note (UAT rework round 2, Finding 1),
// mirroring feature_service.go's recordIntegrationFailureNote convention
// (gate=="integration_capture") so `shark epic notes`/`shark search`
// surface both this feature's pre-dispatch and post-transition
// integration-capture failures the same way. Deduped via
// nextIntegrationCaptureFailureNoteExists.
//
// Best-effort with respect to the note write itself — a failure to resolve
// the recorder, list existing notes, or write the note is logged to
// stderr, never raised. This is safe because the note is a visibility aid,
// not the blocking mechanism: callers (resolveCascade for `shark next`,
// cascadeIntegrationGuard for `shark run` — see
// ensureEpicIntegrationBaseCaptured, run_cascade_integration_guard.go) block
// dispatch before ever calling this function's caller regardless of whether
// this note write succeeds.
//
// commandLabel identifies which dispatch surface hit the failure (e.g.
// "next" or "run") for the stderr warning prefix only; the durable note
// content and metadata are surface-agnostic; note text and metadata are
// shared and never mention which command hit the failure, so `shark epic
// notes`/`shark search` present the finding identically regardless of
// dispatch surface.
func nextRecordCaptureFailureNote(ctx context.Context, commandLabel, epicKey string, cause error) {
	recorder, err := nextIntegrationCaptureFailureRecorder(ctx)
	if err != nil || recorder == nil {
		fmt.Fprintf(os.Stderr,
			"[shark %s] warning: could not resolve a note recorder to record the integration-capture failure for epic %s: %v\n",
			commandLabel, epicKey, err)
		return
	}
	exists, err := nextIntegrationCaptureFailureNoteExists(ctx, recorder, epicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"[shark %s] warning: could not check for an existing integration-capture failure note for epic %s: %v\n",
			commandLabel, epicKey, err)
		return
	}
	if exists {
		return
	}
	content := fmt.Sprintf(
		"integration base capture failed for epic %s: %v — the epic cascade is blocked (no feature will be dispatched under this epic) until this is resolved.",
		epicKey, cause,
	)
	metadata, err := json.Marshal(map[string]string{
		"gate":             "integration_capture",
		"severity":         "critical",
		"closure_category": "reference",
		"disposition":      "open",
		"epic_key":         epicKey,
		"stage":            "capture_base",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"[shark %s] warning: could not encode the integration-capture failure note metadata for epic %s: %v\n",
			commandLabel, epicKey, err)
		return
	}
	if _, err := recorder.AddNoteWithMetadata(
		ctx, models.EntityTypeEpic, epicKey, "review-finding", content, integrationCaptureFailureCreatedBy, string(metadata),
	); err != nil {
		fmt.Fprintf(os.Stderr,
			"[shark %s] warning: could not record the integration-capture failure note for epic %s: %v\n",
			commandLabel, epicKey, err)
	}
}

// ensureEpicIntegrationBaseCaptured is the ONE shared pre-dispatch
// integration-evidence guard for every epic cascade dispatch entrypoint
// (UAT round-3 rejection Finding 1; docs/plan/tech-debt/TD-208.md).
// `shark next`'s resolveCascade and `shark run`'s cascadeIntegrationGuard
// (run_cascade_integration_guard.go) both call this exact function rather
// than each maintaining their own capture-or-block implementation — the
// UAT's defect-class statement ("every supported cascade dispatch
// entrypoint must initialize and enforce the shared integration evidence
// precondition") is satisfied by construction: there is exactly one call
// site that can fail to block, not two independently-maintained copies that
// could drift again the way `shark run` drifted from `shark next` in round 2.
//
// nextCaptureEpicIntegrationBase is itself idempotent (see its doc comment),
// so calling it on every cascade invocation for the epic is both correct and
// simpler than tracking first-dispatch state here. On failure — of ANY
// kind, including an ordinary I/O or git error, not only a corrupt existing
// run record — this returns a non-nil error and the caller must block: no
// feature may be dispatched under this epic. A corrupt run record additionally
// gets a stderr warning (a genuinely exceptional, non-recurring state); every
// failure kind gets a durable, deduped epic-level `review-finding` note via
// nextRecordCaptureFailureNote.
func ensureEpicIntegrationBaseCaptured(ctx context.Context, commandLabel, epicKey string) error {
	if _, err := nextCaptureEpicIntegrationBase(epicKey); err != nil {
		var corruptErr *integration.CorruptRunError
		if errors.As(err, &corruptErr) {
			fmt.Fprintf(os.Stderr,
				"[shark %s] warning: epic %s has a corrupt integration-run record, not recreating it: %v\n",
				commandLabel, epicKey, err,
			)
		}
		nextRecordCaptureFailureNote(ctx, commandLabel, epicKey, err)
		return err
	}
	return nil
}

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

// questionBlockChecker is the narrow read-only I-03 seam used at keyed-next
// dispatch boundaries. The service owns qualification; commands only decide
// whether a qualifying compact handoff means dispatch must pause.
type questionBlockChecker interface {
	Check(ctx context.Context, candidateType models.EntityType, candidateKey string) (*services.QuestionBlock, error)
}

// nextAdapterCache is the per-invocation cache hoisted into runNext and passed
// through resolveNext recursion. Lookup is keyed by entity type; entries are
// populated lazily on first use and reused for the remainder of the call.
// Reset between top-level `shark next` calls (no cross-invocation caching).
type nextAdapterCache struct {
	entries         map[string]*nextAdapters
	actionSvcRoot   action.ActionService
	surfaceForks    bool
	questionBlocker questionBlockChecker

	// harnessResolver resolves harness identity per spec.md REQ-F-002's
	// flag > claim > env > zero precedence (T-E34-F01-003/004). nil is a
	// valid, supported state — resolveEntity treats it as "no resolution
	// available" and merges the zero identity's three empty keys, never an
	// absent key (D-F01-07).
	harnessResolver *services.HarnessResolver

	// harnessOverride carries the explicit --harness/--harness-version/
	// --harness-model flag values read once in runNext (spec.md §3.3); it is
	// the top precedence tier for every entity resolveEntity visits during
	// this invocation, including cascade recursion.
	harnessOverride services.HarnessIdentity
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
		entries:         make(map[string]*nextAdapters),
		actionSvcRoot:   root,
		questionBlocker: cli.GetQuestionBlocker(),
		harnessResolver: cli.GetHarnessResolver(),
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
	// Returns true when sequential_dispatch is configured, collapsing a
	// surviving keyed-next fork to its first eligible candidate.
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

	// Harness, HarnessVersion, and HarnessModel carry the resolved harness
	// identity (spec.md §3.2 AC-T1) used to branch prompt rendering via the
	// `isHarness`/`isClaude`/`isCodex` template helpers. All three are
	// additive and `omitempty` so a run with no resolvable harness metadata
	// (no claim, no flag, no env var — spec.md AC-04) emits a JSON object
	// byte-identical to today's (REQ-NF-001).
	Harness        string `json:"harness,omitempty"`
	HarnessVersion string `json:"harness_version,omitempty"`
	HarnessModel   string `json:"harness_model,omitempty"`

	// ResultContract is the REQ-F-006 resolved worker-result contract for
	// the current step: "legacy" or "gate_result_v1". Always populated
	// (never omitted) so both the core runner and Rider consume the exact
	// same resolved value instead of deriving it independently.
	ResultContract string `json:"result_contract"`

	// OutcomeRoles maps each of the current step's configured outcome keys
	// to its REQ-F-006 semantic role. Empty/omitted for a "legacy" step.
	OutcomeRoles map[string]gateresult.OutcomeRole `json:"outcome_roles,omitempty"`

	Prompt string `json:"prompt"` // fully-rendered, skill-inlined prompt

	// PromptSHA256 is the hex-encoded SHA-256 digest of the exact Prompt bytes
	// (REQ-F-011). It is computed once, immediately after assembleDispatchPrompt,
	// over the fully assembled prompt — including the ownership preamble and
	// agent body (D-004) — so it always matches what --prompt-out writes and
	// what the harness actually spawns the worker with. Empty and omitted from
	// the wire when the response carries no prompt (pause/archive/fork).
	PromptSHA256 string `json:"prompt_sha256,omitempty"`

	// PromptBytes is len(Prompt) in bytes, computed alongside PromptSHA256 from
	// the same materialized string. runNext's OTel prompt_bytes span attribute
	// reuses this value rather than recomputing len(Prompt) a second time
	// (REQ-NF-007: a single SHA-256 pass, computed once per response).
	PromptBytes int `json:"prompt_bytes,omitempty"`

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

	// QuestionBlock is the optional, compact I-03 handoff for a directly
	// linked open blocking Question. It is omitted from ordinary next output.
	QuestionBlock *services.QuestionBlock `json:"question_block,omitempty"`

	// CurrentResponder is the derived Question responder identity needed by a
	// host to claim and record a routed Question response. It is omitted for
	// other entity types and Questions without a pending responder.
	CurrentResponder string `json:"current_responder,omitempty"`

	// selection is an unexported carrier used to return a hierarchy selection
	// through the shared NextResponse type instead of the plain wire shape
	// above: `shark plan` sets it for a one-level selection, and keyed
	// `shark next` sets it when cascade stops at a fork. Unexported, so JSON
	// marshaling of NextResponse itself is unaffected.
	selection *HierarchyPlanSelectionResponse
}

var nextCmd = newNextCommand()

func newNextCommand() *cobra.Command {
	cmd := &cobra.Command{
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
    "entity_type":  "task" | "feature" | "epic" | "bug" | "change" | "tech_debt" | "question",
    "status":       "<current status>",
    "action":       "spawn_agent" | "pause" | "archive",
    "agent_type":   "<agent type if action=spawn_agent>",
    "provider":     "<provider>",
    "model":        "<model override>",
    "prompt":       "<fully-rendered, skill-inlined prompt>",
    "prompt_sha256": "<hex sha256 of the exact prompt bytes>",
    "prompt_bytes":  <byte length of the exact prompt>,
    "unresolved_placeholders": ["<token>", ...]  // omitted when empty
  }

Cascade fork JSON shape (the default when tied candidates survive):
  {
    "mode":               "hierarchy_selection",
    "action":             "parallel_candidates",
    "root_key":           "<requested parent key>",
    "root_type":          "epic" | "feature",
    "selection_reason":   "<why this tier was selected>",
    "resolved_via":       ["<traversed parent key>", ...],
    "parallel_execution": "available",
    "entities":           [{"entity_key": "<child key>", "entity_type": "<type>"}, ...]
  }
This read-only selection does not claim candidates and does not include a worker
prompt. Choose one or more candidates, then call 'shark next <child-key>' for
each chosen key. Pass --sequential to collapse a surviving fork to its first
eligible candidate.

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
	// Flags are registered on the *cobra.Command instance here, inside the
	// constructor, rather than in init() against the package-level nextCmd
	// singleton — so every newNextCommand() construction, including the
	// fresh instances CLI tests build to swap in mocked adapters, carries a
	// complete, self-contained flag set (spec.md §3.3).
	cmd.Flags().Bool(
		"sequential",
		false,
		"Collapse a keyed-next fork to its first eligible candidate",
	)
	cmd.Flags().String(
		"prompt-out",
		"",
		"Write the exact UTF-8 prompt bytes to <path> (no trailing newline); fails loudly if the target is unwritable",
	)
	cmd.Flags().String(
		"harness",
		"",
		"Override the resolved harness type (e.g. claude, codex); wins over the active claim and SHARK_HARNESS",
	)
	cmd.Flags().String(
		"harness-version",
		"",
		"Override the resolved harness version; wins over the active claim and SHARK_HARNESS_VERSION",
	)
	cmd.Flags().String(
		"harness-model",
		"",
		"Override the resolved harness model; wins over the active claim and SHARK_HARNESS_MODEL",
	)
	return cmd
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
	// Flags are registered inside newNextCommand() itself (see there) so
	// every constructed instance — including fresh test instances — is
	// self-contained; init() only wires the package singleton into the root
	// command.
	cli.RootCmd.AddCommand(nextCmd)
}

// harnessOverrideFromFlags reads the --harness/--harness-version/
// --harness-model flags into a HarnessIdentity override (spec.md §3.3). Flag
// values are read verbatim (no trimming/normalization here) — HarnessResolver
// treats non-empty as "set" per-field precedence (D-F01-04).
func harnessOverrideFromFlags(cmd *cobra.Command) (services.HarnessIdentity, error) {
	harnessType, err := cmd.Flags().GetString("harness")
	if err != nil {
		return services.HarnessIdentity{}, fmt.Errorf("read --harness flag: %w", err)
	}
	harnessVersion, err := cmd.Flags().GetString("harness-version")
	if err != nil {
		return services.HarnessIdentity{}, fmt.Errorf("read --harness-version flag: %w", err)
	}
	harnessModel, err := cmd.Flags().GetString("harness-model")
	if err != nil {
		return services.HarnessIdentity{}, fmt.Errorf("read --harness-model flag: %w", err)
	}
	return services.HarnessIdentity{Type: harnessType, Version: harnessVersion, Model: harnessModel}, nil
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

	adapters.surfaceForks = !resolveSequentialDispatch(cmd)

	harnessOverride, err := harnessOverrideFromFlags(cmd)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	adapters.harnessOverride = harnessOverride

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

	// --prompt-out: write the exact prompt bytes the wire response carries, no
	// trailing newline. Written before the JSON response is emitted so an
	// unwritable target fails loudly (non-zero exit, no success output)
	// instead of silently skipping the write (REQ-F-011/AC-013).
	promptOutPath, err := cmd.Flags().GetString("prompt-out")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("read --prompt-out flag: %w", err)
	}
	if promptOutPath != "" {
		if writeErr := os.WriteFile(promptOutPath, []byte(resp.Prompt), 0o644); writeErr != nil {
			span.SetStatus(codes.Error, writeErr.Error())
			return fmt.Errorf("failed to write --prompt-out file %q: %w", promptOutPath, writeErr)
		}
	}

	span.SetAttributes(
		attribute.String("entity_key", resp.EntityKey),
		attribute.String("entity_type", resp.EntityType),
		attribute.String("status", resp.Status),
		attribute.String("action", resp.Action),
		attribute.String("agent_type", resp.AgentType),
		attribute.String("model", resp.Model),
		// harness (not harness_version/harness_model) is added as a span
		// attribute per spec.md §5: the harness type is a small, bounded
		// vocabulary ("claude", "codex", ...), while version/model are
		// free-text and would risk unbounded attribute cardinality.
		attribute.String("harness", resp.Harness),
		attribute.Int("prompt_bytes", resp.PromptBytes),
		attribute.Int("unresolved_placeholders", len(resp.UnresolvedPlaceholders)),
		attribute.String("exit_status", "ok"),
	)
	warnUnresolvedPlaceholdersToStderr(resp)

	return outputNextJSON(resp)
}

type entityResolutionMode uint8

const (
	nextResolutionMode entityResolutionMode = iota
	planResolutionMode
)

type entityResolutionStrategy struct {
	mode         entityResolutionMode
	commandLabel string
	nextCache    *nextAdapterCache
	planCache    *planAdapterCache
}

func nextResolutionStrategy(cache *nextAdapterCache) entityResolutionStrategy {
	return entityResolutionStrategy{mode: nextResolutionMode, commandLabel: "next", nextCache: cache}
}

func planResolutionStrategy(cache *planAdapterCache) entityResolutionStrategy {
	return entityResolutionStrategy{
		mode: planResolutionMode, commandLabel: "plan", planCache: cache,
	}
}

func (s entityResolutionStrategy) resolveCascade(
	ctx context.Context,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	// REQ-F-004 cascade wiring (T-E34-F08-008): the epic `active` step's
	// cascade action captures the epic's integration-run base commit here —
	// this is the confirmed real handler for `action: cascade` (see this
	// function's callers in resolveEntity). Scoped to keyed `shark next`
	// dispatch only: `shark plan` (planResolutionMode) is a read-only
	// advisory surface and must never side-effect the filesystem, and
	// scoped to entityType == epic so a nested feature-level cascade (e.g.
	// a feature's own `active` step cascading into tasks) never re-fires
	// it for the wrong entity.
	//
	// UAT rework round 2, Finding 1 (uat-20260905-142000-E34-F08.md): a
	// CaptureBase failure — of ANY kind, including an ordinary I/O or git
	// error (e.g. the project root is not a git repository, so `git
	// rev-parse HEAD` fails) and not only a *integration.CorruptRunError —
	// must stop this cascade from ever dispatching a feature under this
	// epic. The previous fix here treated an ordinary error as a
	// best-effort side channel (mirroring services' indexEntityIfConfigured
	// / search_indexer.go) and let dispatch proceed regardless; that
	// silently produced exactly the state REQ-F-004 forbids — feature work
	// dispatched with no integration base ever captured, so the
	// terminal-transition hook in feature_service.go later finds no
	// IntegrationRun and records nothing, either. run.go states the
	// governing principle for this whole feature's fail-closed posture:
	// "the parent cannot acknowledge active entry or dispatch a feature
	// until head and note reconcile" (architecture.md "Epic integration
	// candidate identity") — that applies just as much to the base capture
	// itself as it does to RegisterRun's own reconciliation. There is no
	// carve-out for a git-less project: spec.md's REQ-F-004 has no such
	// exception, and a project that can never resolve a base commit can
	// never produce integration_review's required evidence anyway, so
	// blocking the cascade until an operator resolves the underlying
	// condition (initializes git, fixes permissions, or otherwise makes
	// CaptureBase succeed) is the correct fail-closed behavior rather than
	// a silent, permanently-unauditable gap.
	//
	// This still costs a stderr warning line per corrupt-run failure (kept
	// from the prior fix, since a corrupt run file is a genuinely
	// exceptional, non-recurring state) and a durable, deduped epic-level
	// `review-finding` note via nextRecordCaptureFailureNote for every
	// failure kind (nextIntegrationCaptureFailureNoteExists keeps a
	// persistently-failing epic — e.g. one with no git repository, which
	// CaptureBase is invoked against on every cascade attempt per this
	// function's own idempotent-call convention — from accumulating a new
	// note on every `shark next` poll).
	//
	// This capture-or-block call is shared verbatim with `shark run`'s
	// cascade path (cascadeIntegrationGuard in
	// run_cascade_integration_guard.go, wired via
	// runner.RunControllerDeps.IntegrationGuard) — see
	// ensureEpicIntegrationBaseCaptured's doc comment (UAT round-3 rejection
	// Finding 1; docs/plan/tech-debt/TD-208.md).
	if s.mode == nextResolutionMode && entityType == string(models.EntityTypeEpic) {
		if err := ensureEpicIntegrationBaseCaptured(ctx, s.commandLabel, normalizedKey); err != nil {
			resp.Action = "pause"
			resp.Error = fmt.Sprintf(
				"integration base capture failed for epic %s: %v — the epic cascade is blocked (no feature will be dispatched under this epic) until the underlying condition is resolved; see `shark epic notes %s` for the durable finding",
				normalizedKey, err, normalizedKey,
			)
			return resp, nil
		}
	}

	switch s.mode {
	case nextResolutionMode:
		if s.nextCache == nil {
			return NextResponse{}, fmt.Errorf("next resolution strategy requires a next adapter cache")
		}
		return tryCascadeCandidates(
			ctx,
			s.nextCache,
			s,
			s.nextCache.surfaceForks,
			entityType,
			normalizedKey,
			depth,
			resp,
			nextInfo,
			transitioner,
		)
	case planResolutionMode:
		if s.planCache == nil {
			return NextResponse{}, fmt.Errorf("plan resolution strategy requires a plan cache")
		}
		return tryPlanHierarchy(
			ctx, s.planCache, s, entityType, normalizedKey, depth, resp, nextInfo, transitioner,
		)
	default:
		return NextResponse{}, fmt.Errorf("unknown entity resolution strategy %d", s.mode)
	}
}

// resolveNext is the top-level and child-recursion entry point for `shark
// next`. The shared resolver keeps all non-cascade behavior identical to
// keyed `shark plan`; the explicit strategy retains next's sequential/fan-out
// cascade policy.
func resolveNext(ctx context.Context, cache *nextAdapterCache, entityType, normalizedKey string, depth int) (NextResponse, error) {
	return resolveEntity(
		ctx, cache, nextResolutionStrategy(cache), entityType, normalizedKey, depth,
	)
}

// resolveEntity is the shared per-entity status/action/prompt pipeline for
// keyed next and plan. Only a populated cascade action is delegated to the
// command-specific strategy; same-entity recursion retains that strategy.
func resolveEntity(
	ctx context.Context,
	cache *nextAdapterCache,
	strategy entityResolutionStrategy,
	entityType, normalizedKey string,
	depth int,
) (NextResponse, error) {
	if depth > maxCascadeDepth {
		return NextResponse{
			EntityKey:      normalizedKey,
			EntityType:     entityType,
			Action:         "error",
			Error:          fmt.Sprintf("cascade depth limit (%d) exceeded — likely a misconfigured workflow", maxCascadeDepth),
			ResultContract: configworkflow.ResultContractLegacy,
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
		EntityKey:      normalizedKey,
		EntityType:     entityType,
		Status:         currentStatus,
		ResultContract: resolvedResultContract(nextInfo),
		OutcomeRoles:   nextInfo.OutcomeRoles,
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
	// A live lease is an operational dispatch boundary, not a terminal
	// workflow state. Pause a competing keyed-next caller, while preserving
	// the non-terminal NextStatusInfo used by the owning parent to advance and
	// release its worker stage.
	if nextInfo.IsClaimed {
		resp.Action = "pause"
		return resp, nil
	}

	// Keyed next alone owns the F03 dispatch gate. Planning remains advisory;
	// it deliberately reuses this resolver without turning ordinary planning
	// reads into a blocking surface. The check occurs after identity/status are
	// known and before placeholders, action/prompt work, or cascade traversal.
	if strategy.mode == nextResolutionMode && cache.questionBlocker != nil {
		block, err := cache.questionBlocker.Check(ctx, models.EntityType(entityType), normalizedKey)
		if err != nil {
			return NextResponse{}, fmt.Errorf("check Question block for %s: %w", normalizedKey, err)
		}
		if block != nil {
			resp.Action = action.ActionPause
			resp.QuestionBlock = block
			return resp, nil
		}
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

	// Resolve harness identity (spec.md §3.2/§3.4 AC-T2) and merge it into
	// vars before the template renders, so `{{if isClaude .harness}}`
	// branches see real values. All three vars keys are always present —
	// HarnessIdentity.Vars() never omits a key, even when unresolved
	// (D-F01-07) — and resp.Harness/Version/Model mirror the resolution onto
	// the wire response (`omitempty` keeps an unresolved run byte-identical
	// to today's JSON, REQ-NF-001).
	if cache.harnessResolver != nil {
		identity, hErr := cache.harnessResolver.Resolve(ctx, entityType, normalizedKey, cache.harnessOverride)
		if hErr != nil {
			// HarnessResolver.Resolve is documented to always return a nil
			// error (claim-read failures degrade internally per D-F01-05);
			// this branch exists only to fail loudly if that contract is
			// ever violated, rather than silently dropping the error.
			return NextResponse{}, fmt.Errorf("failed to resolve harness identity for %s: %w", normalizedKey, hErr)
		}
		for k, v := range identity.Vars() {
			vars[k] = v
		}
		resp.Harness = identity.Type
		resp.HarnessVersion = identity.Version
		resp.HarnessModel = identity.Model
	} else {
		zero := services.HarnessIdentity{}
		for k, v := range zero.Vars() {
			vars[k] = v
		}
	}

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
			fmt.Fprintf(
				os.Stderr,
				"[shark %s] warning: status %q is not defined in the workflow configuration for entity type %q — treating as pause (B022)\n",
				strategy.commandLabel,
				currentStatus,
				entityType,
			)
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
		return strategy.resolveCascade(
			ctx, entityType, normalizedKey, depth, resp, nextInfo, transitioner,
		)
	}

	// Step 8: Verb normalization + action application. The YAML's internal
	// verb vocabulary (spawn_agent, check_or_resume, advance_status, pause,
	// archive) is richer than the harness wire vocabulary {spawn_agent,
	// pause, archive}. applyWireAction maps onto the wire set and handles
	// the special-case branches (advance_and_recurse, error) inline so the
	// caller only deals with the simple wire-shaped result.
	wireResp, handled, err := applyWireAction(
		ctx,
		cache,
		strategy,
		entityType,
		normalizedKey,
		depth,
		internalAction,
		populated,
		nextInfo,
		transitioner,
		resp,
	)
	if err != nil {
		return NextResponse{}, err
	}
	if handled {
		return wireResp, nil
	}
	resp = wireResp
	if entityType == string(models.EntityTypeQuestion) {
		resp.CurrentResponder = vars["current_responder"]
	}
	// Question's F01 workflow is an explicit read-only pause/archive fixture.
	// It is not a worker dispatch, so its exact response envelope intentionally
	// omits both an instruction and the parent-loop ownership preamble.
	if entityType == "question" && (resp.Action == action.ActionPause || resp.Action == action.ActionArchive) {
		resp.AgentType = ""
		resp.Provider = ""
		resp.Model = ""
		resp.Effort = ""
		resp.Prompt = ""
		return resp, nil
	}

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

	// Hash the fully assembled prompt — post-assembly, including the
	// ownership preamble and agent body (D-004) — so PromptSHA256/PromptBytes
	// always describe exactly what the harness receives and what --prompt-out
	// writes. Computed once here; runNext's OTel span attribute and the
	// --prompt-out write both reuse resp.Prompt/resp.PromptBytes rather than
	// re-hashing or re-measuring (REQ-NF-007).
	sum := sha256.Sum256([]byte(attached))
	resp.PromptSHA256 = hex.EncodeToString(sum[:])
	resp.PromptBytes = len(attached)

	return resp, nil
}

// resolvedResultContract returns nextInfo's REQ-F-006 result_contract,
// defaulting to "legacy" when nil or unset — e.g. a NextStatusInfo built by
// a caller that predates the field (Question's handoff literals in
// question_service.go) or a legacy (non-route-based) workflow. This keeps
// the "always populated, legacy by default" wire contract even though not
// every NextStatusInfo constructor sets ResultContract explicitly.
func resolvedResultContract(nextInfo *services.NextStatusInfo) string {
	if nextInfo == nil || nextInfo.ResultContract == "" {
		return configworkflow.ResultContractLegacy
	}
	return nextInfo.ResultContract
}

func actionRequiresInstruction(internalAction string) bool {
	return internalAction == action.ActionSpawnAgent || internalAction == action.ActionCheckOrResume
}

// tryCascadeCandidates owns keyed-next child enumeration and traversal for
// both emission modes. Every mode consumes the same set-oriented hierarchy
// snapshot and leading-tier policy. surfaceForks controls only whether the
// first live candidate returns immediately or a surviving tie is emitted.
//
// A selected child that
// resolves to pause/archive is dropped and the tier is recomputed over the
// remaining siblings, because "the next available work" excludes work that is
// parked. Sequential mode short-circuits at the first live candidate; fork
// mode resolves the leading tier far enough to exclude parked candidates.
func tryCascadeCandidates(
	ctx context.Context,
	cache *nextAdapterCache,
	strategy entityResolutionStrategy,
	surfaceForks bool,
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
	// Preserve the first compact Question handoff encountered while every
	// candidate is parked. If a later sibling is dispatchable it is intentionally
	// discarded with the parked child; if none is, the parent pause must retain
	// the actionable reason that made the cascade unavailable.
	var allParkedQuestionBlock *services.QuestionBlock
	nonProgressingCandidates := 0
	questionBlockedCandidates := 0
	for len(remaining) > 0 {
		selected, reason := selectPlanChildTier(remaining)
		if len(selected) == 0 {
			break
		}

		// Resolve candidates in tier order until the active emission mode has
		// enough information to return.
		// DescribeChildren filters on terminal / claimed / dependency state but
		// has no view of the workflow *action*, so a child it hands back can
		// still resolve to pause (e.g. parked at a human gate). Emitting such a
		// tier as parallel_candidates would strand the run: the rider
		// dispatches each candidate, gets pause for each, and stops — never
		// reaching a dispatchable sibling behind them. That is the fall-through
		// keyed next has always had, and it has to apply to a tie as much as to
		// a lone child.
		dispatchable := make([]services.PlanHierarchyChild, 0, len(selected))
		var firstDispatch NextResponse
		for _, child := range selected {
			childResp, err := resolveEntity(
				ctx, cache, strategy, string(child.EntityType), child.Key, depth+1,
			)
			if err != nil {
				return NextResponse{}, err
			}
			if childResp.Action == "error" {
				// Propagate child errors up untouched.
				return prependFanoutParent(childResp, normalizedKey), nil
			}
			if childResp.QuestionBlock != nil && allParkedQuestionBlock == nil {
				allParkedQuestionBlock = childResp.QuestionBlock
			}
			// A nested selection means the child has dispatchable descendants
			// of its own, so it counts as live work.
			if childResp.selection == nil && childResp.Action != "spawn_agent" {
				nonProgressingCandidates++
				if childResp.QuestionBlock != nil {
					questionBlockedCandidates++
				}
				continue
			}
			if !surfaceForks {
				return prependFanoutParent(childResp, normalizedKey), nil
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
		return autoAdvanceCascadeParent(
			ctx, cache, strategy, entityType, normalizedKey, depth, resp, nextInfo, transitioner,
		)
	}
	// All children either non-dispatchable or absent — pause the parent.
	resp.Action = "pause"
	if questionBlockedCandidates != nonProgressingCandidates {
		allParkedQuestionBlock = nil
	}
	resp.QuestionBlock = allParkedQuestionBlock
	return resp, nil
}

// resolveSequentialDispatch reports whether `shark next` should collapse a
// surviving fork to its first eligible candidate. Precedence: an explicitly
// passed --sequential flag wins, else the sequential_dispatch config value,
// else the default (surface forks).
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
// the historical per-child cascade. The asymmetry is deliberate: a missing claim errs toward
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

// autoAdvanceCascadeParent handles keyed-next's all-children-terminal case:
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
	strategy entityResolutionStrategy,
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
	return resolveEntity(ctx, cache, strategy, entityType, normalizedKey, depth+1)
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
	strategy entityResolutionStrategy,
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
		recursed, err := resolveEntity(
			ctx, cache, strategy, entityType, normalizedKey, depth+1,
		)
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
	return scanForwardTransition(wf, nextInfo)
}

// scanForwardTransition preserves the legacy auto-advance fallback for
// workflows without a usable route-based pass outcome.
func scanForwardTransition(wf *workflow.Service, nextInfo *services.NextStatusInfo) string {
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
