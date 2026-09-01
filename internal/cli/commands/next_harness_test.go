package commands

// Covers T-E34-F01-004 (docs/plan/E34-prompt-and-skill-improvements/E34-F01-harness-aware-prompt-rendering/tasks/T-E34-F01-004.md):
// TC-003, TC-004, TC-005, TC-006, TC-007, TC-016, TC-018 from the feature
// test-plan.md. All tests drive `runNext`/`resolveNext` — the seam
// TestResolveNext_ReturnsSelfContainedPrompt and the TC-002 provenance suite
// (next_provenance_test.go) already use — through the real
// templates.OrchestratorRenderer (never mocked) with `ClaimReader.Get` as the
// only mocked seam, per test-plan.md's Caller-Path Contract for these TCs.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// harnessBranchA/B are distinctive markers used as the template branch
// output, deliberately not "A"/"B" — the dispatch prompt is prefixed with the
// PARENT LOOP OWNERSHIP CONTRACT preamble (attachAgentBody/assembleDispatchPrompt),
// whose prose legitimately contains bare "A"/"B" letters (e.g. "AI CLIs"), so
// a single-letter marker would produce false negatives on the "must not
// contain the other branch" assertions.
const (
	harnessBranchA = "HARNESS_BRANCH_MARKER_A"
	harnessBranchB = "HARNESS_BRANCH_MARKER_B"
)

var harnessIfTemplate = `{{if isClaude .harness}}` + harnessBranchA + `{{else}}` + harnessBranchB + `{{end}}`

// harnessMockClaimReader is the one mocked seam TC-003/004/005/006/007/016/
// 018 are permitted to use (test-plan.md's Caller-Path Contract for these
// cases forbids mocking HarnessResolver.Resolve or OrchestratorRenderer.Render
// directly).
type harnessMockClaimReader struct {
	claim *models.EntityClaim
	err   error
}

func (m *harnessMockClaimReader) Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claim, nil
}

// harnessDispatchRenderer builds a real templates.OrchestratorRenderer over a
// single fixture template under a temp "task/" directory, returning the
// renderer and the template name to pass to Render.
func harnessDispatchRenderer(t *testing.T, templateBody string) (*templates.OrchestratorRenderer, string) {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))
	const templateName = "task/dispatch.md"
	require.NoError(t, os.WriteFile(filepath.Join(root, templateName), []byte(templateBody), 0644))
	renderer, err := templates.NewOrchestratorRenderer(root)
	require.NoError(t, err)
	return renderer, templateName
}

// harnessTestCache builds a nextAdapterCache for entity type "task" backed by
// a real renderer (via a MockActionService seam that calls Render — the same
// belt-and-braces pattern TestResolveNext_ReturnsSelfContainedPrompt and
// provenanceCache use) and a HarnessResolver constructed over claims. This is
// the "existing next_test.go seam" the task's Design Reference names.
func harnessTestCache(t *testing.T, claims services.ClaimReader, templateBody string) *nextAdapterCache {
	t.Helper()
	renderer, templateName := harnessDispatchRenderer(t, templateBody)

	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*action.PopulatedAction, error) {
			rendered, err := renderer.Render(templateName, vars)
			if err != nil {
				return nil, err
			}
			return &action.PopulatedAction{Action: "spawn_agent", Instruction: rendered}, nil
		},
	}

	return &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"task": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{
					CurrentStatus: "in_progress",
					IsTerminal:    false,
				}},
				generator: fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc: actionSvc,
			},
		},
		actionSvcRoot:   actionSvc,
		harnessResolver: services.NewHarnessResolver(claims),
	}
}

// runHarnessNextCommand swaps nextNewAdapterCache to return cache, executes a
// fresh newNextCommand() with args against it, and returns the captured
// stdout and any execution error. This is the real runNext/cobra execution
// chain test-plan.md's Caller-Path Contract table names for TC-003 through
// TC-007/016/018 — flags (TC-006) only work through this path since
// resolveNext alone has no flag-parsing seam.
func runHarnessNextCommand(t *testing.T, cache *nextAdapterCache, args []string) (stdout string, execErr error) {
	t.Helper()
	origFactory := nextNewAdapterCache
	t.Cleanup(func() { nextNewAdapterCache = origFactory })
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) { return cache, nil }

	cmd := newNextCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)

	stdout = capturingOutput(func() {
		execErr = cmd.ExecuteContext(context.Background())
	})
	return stdout, execErr
}

// captureSlogOutput temporarily installs a buffer-backed slog default logger
// for the duration of fn, then restores the previous default. The stdlib
// default logger caches its output destination at construction time, so
// swapping the os.Stderr *variable* around a call (as captureStderrOutput
// does for the plain fmt.Fprintf/log.Default() warnings elsewhere in this
// package) does not reliably capture slog.Warn output — installing a
// dedicated handler is the deterministic way to observe it.
func captureSlogOutput(fn func()) string {
	origLogger := slog.Default()
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(origLogger)
	fn()
	return buf.String()
}

// unsetHarnessEnv unsets all three SHARK_HARNESS* vars for the duration of a
// test and restores whatever was previously set, per TC-005's precondition:
// "use os.Unsetenv ... not t.Setenv(k, \"\"), the two are observably different
// states to a resolver implemented via os.LookupEnv."
func unsetHarnessEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"SHARK_HARNESS", "SHARK_HARNESS_VERSION", "SHARK_HARNESS_MODEL"} {
		prev, had := os.LookupEnv(k)
		require.NoError(t, os.Unsetenv(k))
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, prev)
			}
		})
	}
}

// TestTC003_HarnessClaimClaudeRendersClaudeBranch covers TC-003: a claim
// carrying harness=claude renders the Claude branch of an isClaude-branching
// template, and the wire response reports harness="claude".
func TestTC003_HarnessClaimClaudeRendersClaudeBranch(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "claude", LastHeartbeat: time.Now().UTC()}}
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))

	assert.Contains(t, resp.Prompt, harnessBranchA)
	assert.NotContains(t, resp.Prompt, harnessBranchB)
	assert.Equal(t, "claude", resp.Harness)
}

// TestTC003_IsHarnessGeneralFormAlsoBranchesCorrectly is TC-003's documented
// edge case: the same assertion using the general isHarness helper directly
// (not just the isClaude convenience wrapper), proving the general helper
// also resolves against real vars.
func TestTC003_IsHarnessGeneralFormAlsoBranchesCorrectly(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "claude", LastHeartbeat: time.Now().UTC()}}
	template := `{{if isHarness "claude" .harness}}` + harnessBranchA + `{{else}}` + harnessBranchB + `{{end}}`
	cache := harnessTestCache(t, claims, template)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.Contains(t, resp.Prompt, harnessBranchA)
	assert.NotContains(t, resp.Prompt, harnessBranchB)
}

// TestTC004_HarnessClaimCodexRendersGenericBranch covers TC-004: a claim
// carrying harness=codex renders the generic branch, completing the two-row
// decision table with TC-003. isClaude must return false for "codex" — this
// guards against a helper that defaults to true.
func TestTC004_HarnessClaimCodexRendersGenericBranch(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "codex", LastHeartbeat: time.Now().UTC()}}
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))

	assert.Contains(t, resp.Prompt, harnessBranchB)
	assert.NotContains(t, resp.Prompt, harnessBranchA)
	assert.Equal(t, "codex", resp.Harness)
}

// TestTC005_NoClaimNoEnvRendersGenericBranchAndOmitsHarness covers TC-005:
// the all-sources-empty boundary of REQ-F-002's precedence chain. Exit 0,
// generic branch renders, and "harness" is absent from the JSON response
// (AC-04's literal wording, asserted via an untyped map so omitempty is
// actually exercised rather than assumed).
func TestTC005_NoClaimNoEnvRendersGenericBranchAndOmitsHarness(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: nil}
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.Contains(t, resp.Prompt, harnessBranchB)
	assert.NotContains(t, resp.Prompt, harnessBranchA)
	assert.NotContains(t, resp.Prompt, "<no value>")

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &asMap))
	_, hasHarness := asMap["harness"]
	assert.False(t, hasHarness, "AC-04: harness must be entirely absent from the JSON response, not present-and-empty")
}

// TestTC006_FlagBeatsClaimBeatsEnv covers TC-006: --harness=claude wins over
// both a codex claim and a SHARK_HARNESS env var (agreeing or disagreeing
// with the flag) — flag is the top precedence tier for the type field.
func TestTC006_FlagBeatsClaimBeatsEnv(t *testing.T) {
	tests := []struct {
		name    string
		envType string
	}{
		{"env agrees with flag", "claude"},
		{"env disagrees with flag", "codex"}, // TC-006 edge case
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetHarnessEnv(t)
			t.Setenv("SHARK_HARNESS", tt.envType)
			claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "codex", LastHeartbeat: time.Now().UTC()}}
			cache := harnessTestCache(t, claims, harnessIfTemplate)

			stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001", "--harness=claude"})
			require.NoError(t, err)

			var resp NextResponse
			require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
			assert.Equal(t, "claude", resp.Harness,
				"flag must win over both the codex claim and the env var")
			assert.NotEqual(t, "codex", resp.Harness, "claim must not outrank the flag")
		})
	}
}

// TestTC007_PerFieldPrecedenceClaimTypeEnvVersion covers TC-007, the
// discriminating test between per-field (D-F01-04, correct) and per-source
// (rejected) resolution: a claim supplies only the type (codex, no version);
// SHARK_HARNESS_VERSION=9.9 supplies only the version. Both fields must
// resolve independently, not "claim wins the whole record."
func TestTC007_PerFieldPrecedenceClaimTypeEnvVersion(t *testing.T) {
	unsetHarnessEnv(t)
	t.Setenv("SHARK_HARNESS_VERSION", "9.9")
	claims := &harnessMockClaimReader{claim: &models.EntityClaim{Harness: "codex", LastHeartbeat: time.Now().UTC()}} // no version on the claim
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.Equal(t, "codex", resp.Harness, "type must come from the claim, the only non-empty type source")
	assert.Equal(t, "9.9", resp.HarnessVersion,
		"version must come from env — a per-source resolver would leave this empty")
}

// TestTC016_BareHarnessFormNeverLeaksNoValue covers TC-016: the bare
// `{{.harness}}` form (not wrapped in {{if}}) with no claim and no env must
// never leak the literal "<no value>" sentinel and must not error — the
// second of D-F01-07's two named render-failure classes (the first,
// typed-helper-absent-key, is TC-005/TC-019's job).
func TestTC016_BareHarnessFormNeverLeaksNoValue(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{claim: nil}
	cache := harnessTestCache(t, claims, "before[{{.harness}}]after")

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.NotContains(t, resp.Prompt, "<no value>")
	assert.True(t, strings.HasSuffix(resp.Prompt, "before[]after"),
		"the substitution position must be empty, contiguous with the surrounding literal text; got %q", resp.Prompt)
}

// TestTC018_ClaimReadFailureDegradesToZeroIdentity covers TC-018 (D-F01-05):
// a ClaimReader.Get error must degrade to the zero identity and log a
// warning, never surface as a fatal render error. Exit 0, generic branch,
// harness absent from the JSON response (same shape as TC-005's zero-identity
// case), and the warning is observable via the resolver's slog.Warn call.
func TestTC018_ClaimReadFailureDegradesToZeroIdentity(t *testing.T) {
	unsetHarnessEnv(t)
	claims := &harnessMockClaimReader{err: harnessTestError("simulated claim store outage")}
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	var stdout string
	var err error
	logOutput := captureSlogOutput(func() {
		stdout, err = runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	})
	require.NoError(t, err, "a claim-read error must not surface as a fatal dispatch failure")

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.Contains(t, resp.Prompt, harnessBranchB)
	assert.NotContains(t, resp.Prompt, harnessBranchA)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &asMap))
	_, hasHarness := asMap["harness"]
	assert.False(t, hasHarness, "a claim-read error must degrade to the same absent-harness shape as no claim at all")

	assert.True(t,
		strings.Contains(logOutput, "harness resolution") || strings.Contains(logOutput, "failed to read claim"),
		"a warning mirroring the next.go:585-601 degrade-and-warn posture must be logged on claim-read failure; log=%q", logOutput,
	)
}

// harnessTestError is a tiny local error type so this file doesn't need to
// import "errors" solely for one fixture value.
type harnessTestError string

func (e harnessTestError) Error() string { return string(e) }

// TestTC017_HarnessTypeAddedAsSpanAttributeVersionModelAreNot covers TC-017
// (spec.md §5 / test-plan.md's observability-design row): a claimed entity's
// resolved harness *type* must appear as an attribute on runNext's existing
// "shark.next" OTel span, while harness_version/harness_model must NOT be
// added — bounded cardinality per §5. Drives the real runNext span emission
// via an in-memory exporter (the tracetest.NewInMemoryExporter pattern
// already used by TestRunPlanBareEmitsEpicOnlyParallelCandidatesWithBoundedTelemetry
// in plan_parallel_test.go); no mock stands in for the tracer/span itself.
func TestTC017_HarnessTypeAddedAsSpanAttributeVersionModelAreNot(t *testing.T) {
	unsetHarnessEnv(t)
	originalTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
		otel.SetTracerProvider(originalTracerProvider)
	}()

	claims := &harnessMockClaimReader{claim: &models.EntityClaim{
		Harness:        "claude",
		HarnessVersion: "2.1.0",
		HarnessModel:   "opus",
		LastHeartbeat:  time.Now().UTC(),
	}}
	cache := harnessTestCache(t, claims, harnessIfTemplate)

	stdout, err := runHarnessNextCommand(t, cache, []string{"E01-F01-001"})
	require.NoError(t, err)

	var resp NextResponse
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	require.Equal(t, "claude", resp.Harness, "precondition: harness must resolve to claude")

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "shark.next", spans[0].Name)

	var gotHarness string
	var hasHarness, hasHarnessVersion, hasHarnessModel bool
	for _, attr := range spans[0].Attributes {
		switch string(attr.Key) {
		case "harness":
			hasHarness = true
			gotHarness = attr.Value.AsString()
		case "harness_version":
			hasHarnessVersion = true
		case "harness_model":
			hasHarnessModel = true
		}
	}
	assert.True(t, hasHarness, "span must carry a harness attribute")
	assert.Equal(t, "claude", gotHarness, "span's harness attribute must equal the resolved type")
	assert.False(t, hasHarnessVersion, "harness_version must NOT be added as a span attribute (bounded cardinality, spec.md §5)")
	assert.False(t, hasHarnessModel, "harness_model must NOT be added as a span attribute (bounded cardinality, spec.md §5)")
}

// TestNewNextAdapterCache_WiresHarnessResolver pins the production wiring
// line every TC-003..018 test above bypasses by construction (they all
// replace nextNewAdapterCache with a stub cache carrying a hand-built
// harnessResolver, so no test above exercises newNextAdapterCache itself).
// Without this, deleting `harnessResolver: cli.GetHarnessResolver()` from
// newNextAdapterCache would leave production `shark next` silently
// resolving no harness identity ever, while every test above still passes.
// Source-scan, following TestTC002_08_OTelPromptBytesReusesResponseValueNotRecomputed's
// established pattern for the same "no real DB in a CLI test" constraint.
func TestNewNextAdapterCache_WiresHarnessResolver(t *testing.T) {
	src, err := os.ReadFile("next.go")
	require.NoError(t, err)
	body := string(src)

	assert.Contains(t, body, `harnessResolver: cli.GetHarnessResolver()`,
		"newNextAdapterCache must construct the production nextAdapterCache with a real HarnessResolver, or shark next silently resolves no harness identity")
}
