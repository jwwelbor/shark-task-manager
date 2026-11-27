package templates

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestProductCriticalPathGuardPartial_RendersStandalone verifies that the
// checked-in product-critical-path guard partial (E34-F10) defines exactly
// one named template (spec.md AC-1, test-plan.md TC-001) and renders cleanly
// through the production template engine (text/template.Parse + Funcs +
// ExecuteTemplate, the same engine NewOrchestratorRenderer uses to compile
// every prompt) with no data supplied — this repository's actual state has
// none of the four REQ-F-032 source files, so the guard's static reporting
// text must name each one as an unresolved prerequisite (D-F10-03), name the
// five REQ-F-034 disqualified evidence classes verbatim, and contain no bare
// workflow status name or `shark` CLI verb (REQ-NF-002/003).
func TestProductCriticalPathGuardPartial_RendersStandalone(t *testing.T) {
	partialPath := canonicalPromptPath("_partials", "_product_critical_path_guard.md")

	raw, err := os.ReadFile(partialPath)
	require.NoError(t, err, "product-critical-path guard partial must exist at %s", partialPath)
	source := string(raw)

	// AC-T1: exactly one {{define}} block, naming the guard template.
	assert.Equal(t, 1, strings.Count(source, "{{define"),
		"guard partial must contain exactly one {{define}} block")
	assert.Contains(t, source, `{{define "_product_critical_path_guard"}}`,
		"guard partial must define the _product_critical_path_guard template")

	tmpl, err := template.New("guard").Funcs(orchestratorFuncs()).Parse(source)
	require.NoError(t, err, "guard partial must parse as a valid Go template")

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "_product_critical_path_guard", nil)
	require.NoError(t, err, "guard partial must render cleanly through the production template engine")
	out := buf.String()

	// AC-T2 / D-F10-03: all four REQ-F-032 source files are absent from this
	// repository today, and the guard must report each one, by its exact
	// path, as an unresolved prerequisite.
	for _, sourceFile := range []string{
		"docs/product/D01-vision-statement.md",
		"docs/product/D02-success-criteria.md",
		"docs/plan/product-delivery-roadmap.md",
		"docs/plan/product-critical-path.md",
	} {
		assert.Contains(t, out, "unresolved prerequisite: "+sourceFile+" missing",
			"guard must report %s as an unresolved prerequisite when absent", sourceFile)
	}

	// AC-T3 / REQ-F-034: the five disqualified evidence classes, reusing
	// E34-F02's evidence-authenticity vocabulary verbatim (research-report.md
	// finding 5).
	for _, term := range productCriticalPathGuardEvidenceClassTerms {
		assert.Contains(t, out, term, "guard must name disqualified evidence class %q", term)
	}

	// AC-T3 / REQ-NF-002/003: no bare workflow status name (from any entity
	// workflow YAML's steps: keys) or shark CLI verb anywhere in the guard.
	wordBoundary := regexp.MustCompile(`\bshark\b`)
	assert.False(t, wordBoundary.MatchString(out), "guard must not name the `shark` CLI")
	for _, status := range workflowStepKeysUnion(t, canonicalWorkflowDir) {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(status) + `\b`)
		assert.False(t, re.MatchString(out), "guard must not contain bare workflow status name %q", status)
	}
}

// productCriticalPathGuardInvocation is the exact literal invocation string
// spec.md REQ-F-033 requires each of the twelve consuming prompts to contain
// (test-plan.md TC-002).
const productCriticalPathGuardInvocation = `{{template "_product_critical_path_guard" .}}`

// productCriticalPathGuardConsumingPrompts is the exact twelve-prompt set
// spec.md's Architecture table and T-E34-F10-002's task Scope name.
var productCriticalPathGuardConsumingPrompts = []string{
	"sprint/planning.md",
	"sprint/active.md",
	"sprint/closing.md",
	"epic/assessment.md",
	"epic/decomposition.md",
	"epic/active.md",
	"feature/specification.md",
	"feature/test_planning.md",
	"feature/task_generation.md",
	"feature/task_review.md",
	"feature/approval.md",
	"task/development.md",
}

// productCriticalPathGuardRestatementSignals are fragments unique to the
// guard partial's own reporting-field prose (REQ-F-033/034). None of the
// twelve consuming prompts pre-dated these phrases (grep-confirmed empty
// before this task, per research-report.md); their presence in a consuming
// prompt outside the single template invocation would mean the prompt
// paraphrased the guard's fields inline instead of referencing it — the
// twelve-copy drift risk research-report.md finding 3 warns against.
var productCriticalPathGuardRestatementSignals = []string{
	"unresolved prerequisite:",
	"side quest",
	"side-quest",
	"executable advancement evidence",
	"hand-authored test actor",
	"component-level test suite",
}

// TestTwelvePromptsInvokeProductCriticalPathGuardExactlyOnce is test-plan.md
// TC-002 (spec.md AC-2): each of the twelve prompts named in REQ-F-033
// contains the guard's literal invocation string exactly once, and none of
// them restates the guard's own reporting-field language inline.
func TestTwelvePromptsInvokeProductCriticalPathGuardExactlyOnce(t *testing.T) {
	for _, prompt := range productCriticalPathGuardConsumingPrompts {
		t.Run(prompt, func(t *testing.T) {
			body, err := sharkdata.ReadEmbedded("prompts/" + prompt)
			require.NoError(t, err, "embedded prompt %s must exist", prompt)
			content := string(body)

			count := strings.Count(content, productCriticalPathGuardInvocation)
			assert.Equal(t, 1, count,
				"%s must invoke the guard template exactly once (found %d)", prompt, count)

			for _, signal := range productCriticalPathGuardRestatementSignals {
				assert.NotContains(t, content, signal,
					"%s must not restate the guard's reporting fields inline (found %q)", prompt, signal)
			}
		})
	}
}

// productCriticalPathGuardBaselineSHA256 records the SHA-256 of each of the
// twelve wired prompts' content as of their latest intentional edits; the task
// baseline was updated by CC-044 after its development-prompt change and by
// B081 after the active-sprint selection-prompt change, and for the
// dispatch-prompt defect corrections (durable sources, quality-gate sources,
// precedence and unclear-requirement text, and the task baseline by E31-F01
// after its quality-block change). Originally
// the values recorded content exactly as it existed before T-E34-F10-002
// added the guard invocation (captured from the pre-edit checked-in files).
// TestTwelvePromptsAddGuardAsPureAddition reconstructs each file's
// pre-invocation content by removing the single guard paragraph and
// re-hashes it — any other change to the file (a reordered section, an
// altered pre-existing line) changes the hash and fails this test
// (test-plan.md TC-003, spec.md AC-3/REQ-NF-001).
var productCriticalPathGuardBaselineSHA256 = map[string]string{
	"sprint/planning.md":         "f27d56ae7e922347d611389a6d5797fa39402b4887e41bb1dc815a10799d4248",
	"sprint/active.md":           "a8150439c3ee91ffdf636b69587f88da96dbd7be415c919a42eee7f759c10853",
	"sprint/closing.md":          "94f8057747fae8ffa85b95fab0cf86146c8f8298e285322737c5457f4f9e3a59",
	"epic/assessment.md":         "1b3affa6488e62d866923e6ec8a4e56e29c3ed0e31be349039e934596111f825",
	"epic/decomposition.md":      "65156f2adb679d23dac78d90456627ab6de94c3533362182345c55c057d2869f",
	"epic/active.md":             "d821d8ffef1ed1f3422133397e318c9bb92e7be702ec2e695ce0430f04c9f23a",
	"feature/specification.md":   "94f0e0c9dff600be5f0a05a00ad061d02254db0e651d91fdbf41f64087bfedce",
	"feature/test_planning.md":   "2cf2ba57af7d65bb392f3af6ceb2e887134486851cd430860182a863256615c7",
	"feature/task_generation.md": "3100315ace5f1aaaa817d5ceaf945fc22a8543cfba261ff314cb2a28e0d6d577",
	"feature/task_review.md":     "0360562975bb8a1dbb908e3be0a49dace76b4434b3deba973c8101d7ac009c16",
	"feature/approval.md":        "b555814fcb69df48ce3d1b777c1ffafe37bf3eaaa770b9f8157f11b8e873f41a",
	"task/development.md":        "d428ce207569f914cf3bdea819fa91faf7d7f6b5ac8b100fcf05df3ac98a4b82",
}

// stripProductCriticalPathGuardParagraph removes the guard invocation's
// standalone paragraph block from content and returns what remains. content
// is expected to end in exactly one trailing newline (true of every shipped
// prompt), so that single newline is set aside before splitting on the
// blank-line paragraph separator ("\n\n") and restored afterward — this
// makes the reconstruction independent of where in the file the guard
// paragraph was inserted, so long as it was inserted as its own
// blank-line-delimited block.
func stripProductCriticalPathGuardParagraph(t *testing.T, content string) string {
	t.Helper()
	require.True(t, strings.HasSuffix(content, "\n"), "prompt file must end with a trailing newline")
	trimmed := strings.TrimSuffix(content, "\n")

	blocks := strings.Split(trimmed, "\n\n")
	var kept []string
	removed := 0
	for _, b := range blocks {
		if b == productCriticalPathGuardInvocation {
			removed++
			continue
		}
		kept = append(kept, b)
	}
	require.Equal(t, 1, removed,
		"guard invocation must appear as exactly one standalone paragraph block")

	return strings.Join(kept, "\n\n") + "\n"
}

// TestTwelvePromptsAddGuardAsPureAddition is test-plan.md TC-003 (spec.md
// AC-3/REQ-NF-001): removing the guard's standalone paragraph from each of
// the twelve wired prompts' current content reproduces that file's exact
// pre-edit bytes, proving the guard was wired in as a pure addition with
// every pre-existing line byte-identical.
func TestTwelvePromptsAddGuardAsPureAddition(t *testing.T) {
	for _, prompt := range productCriticalPathGuardConsumingPrompts {
		t.Run(prompt, func(t *testing.T) {
			body, err := sharkdata.ReadEmbedded("prompts/" + prompt)
			require.NoError(t, err, "embedded prompt %s must exist", prompt)
			content := string(body)

			stripped := stripProductCriticalPathGuardParagraph(t, content)

			wantHash, ok := productCriticalPathGuardBaselineSHA256[prompt]
			require.True(t, ok, "missing pre-edit baseline hash for %s", prompt)

			sum := sha256.Sum256([]byte(stripped))
			gotHash := hex.EncodeToString(sum[:])
			assert.Equal(t, wantHash, gotHash,
				"%s: removing the guard paragraph must reproduce the file's exact pre-edit content; a hash mismatch means some other line changed", prompt)
		})
	}
}

// TestDevelopmentPromptGuardSitsInCompletionReportingSection is test-plan.md
// TC-003's structural-position check for task/development.md (spec.md AC-T3):
// the guard invocation must land inside the completion-reporting section —
// specifically after the closing "When done: stop and summarize" line — not
// prepended to the file.
func TestDevelopmentPromptGuardSitsInCompletionReportingSection(t *testing.T) {
	body, err := sharkdata.ReadEmbedded("prompts/task/development.md")
	require.NoError(t, err, "embedded prompts/task/development.md must exist")
	content := string(body)

	closingIdx := strings.Index(content, "When done: stop and summarize")
	require.NotEqual(t, -1, closingIdx, "development.md must contain the closing completion-reporting line")

	guardIdx := strings.Index(content, productCriticalPathGuardInvocation)
	require.NotEqual(t, -1, guardIdx, "development.md must contain the guard invocation")

	assert.Greater(t, guardIdx, closingIdx,
		"guard invocation must come after the closing completion-reporting line, not be prepended to the file")
}

// ============================================================================
// E34-F10 T-003 — cross-prompt guard-block invariant checks (TC-004/TC-005)
// ============================================================================
//
// TestProductCriticalPathGuardPartial_RendersStandalone (T-001) already
// proves the guard partial's own static text carries the REQ-F-034 evidence
// terms and no bare status/CLI-verb token, in isolation. These tests are the
// feature-level integration check test-plan.md TC-004/TC-005 call for: render
// each of the twelve wired prompts through the same production renderer
// shipped prompts render through (NewOrchestratorRenderer over
// canonicalPromptsDir, the same pipeline TestRenderedPromptsGolden exercises
// in internal/cli/commands), extract only the guard's own block of text from
// each full render, and assert the invariant against that extracted block —
// proving the wiring itself (not just the standalone partial) preserves both
// invariants.

// canonicalWorkflowDir is the on-disk canonical workflow YAML directory,
// relative to this package, mirroring canonicalPromptsDir's convention in
// orchestrator_partials_test.go.
const canonicalWorkflowDir = "../../internal/sharkdata/default_data/workflow"

// productCriticalPathGuardBlockStartMarker and …EndMarker bound the guard's
// rendered text within a full prompt render. TC-004/TC-005 scope their checks
// to the guard block only, not the whole rendered prompt, because the
// surrounding prompt content legitimately contains status names and `shark`
// commands outside the guard (test-plan.md TC-005 edge case).
const (
	productCriticalPathGuardBlockStartMarker = "PRODUCT CRITICAL-PATH GUARD"
	productCriticalPathGuardBlockEndMarker   = "makes the choice for you."
)

// extractProductCriticalPathGuardBlock renders templateName through renderer
// (the production orchestrator renderer, the same pipeline shipped prompts
// render through) and returns only the guard's own block of text, delimited
// by its known start/end markers.
func extractProductCriticalPathGuardBlock(t *testing.T, renderer *OrchestratorRenderer, templateName string) string {
	t.Helper()

	rendered, err := renderer.Render(templateName, map[string]string{})
	require.NoError(t, err, "render %s through the production renderer", templateName)
	require.Equal(t, 1, strings.Count(rendered, productCriticalPathGuardBlockStartMarker),
		"%s: rendered output must contain exactly one guard block start marker", templateName)
	require.Equal(t, 1, strings.Count(rendered, productCriticalPathGuardBlockEndMarker),
		"%s: rendered output must contain exactly one guard block end marker", templateName)

	startIdx := strings.Index(rendered, productCriticalPathGuardBlockStartMarker)
	require.NotEqual(t, -1, startIdx,
		"%s: rendered output must contain the guard block start marker %q", templateName, productCriticalPathGuardBlockStartMarker)

	endMarkerIdx := strings.Index(rendered[startIdx:], productCriticalPathGuardBlockEndMarker)
	require.NotEqual(t, -1, endMarkerIdx,
		"%s: rendered output must contain the guard block end marker %q after the start marker", templateName, productCriticalPathGuardBlockEndMarker)

	endIdx := startIdx + endMarkerIdx + len(productCriticalPathGuardBlockEndMarker)
	return rendered[startIdx:endIdx]
}

// workflowStepsFile is the minimal shape needed to read a workflow YAML
// file's `steps:` key set — every other field is irrelevant here.
type workflowStepsFile struct {
	Steps map[string]interface{} `yaml:"steps"`
}

// workflowStepKeysUnion enumerates the union of every step key (bare status
// name) across every workflow YAML file under dir — e.g.
// internal/sharkdata/default_data/workflow/*.yaml (task, feature, epic, bug,
// change, question, sprint, tech-debt). Enumerated programmatically per
// T-E34-F10-003's Notes for Agent, so this negative-token list tracks the
// shipped workflows without hand-maintenance as steps change.
func workflowStepKeysUnion(t *testing.T, dir string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	require.NoError(t, err, "glob workflow YAML files under %s", dir)
	require.NotEmpty(t, files, "expected at least one workflow YAML file under %s", dir)

	seen := map[string]bool{}
	var keys []string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		require.NoError(t, err, "read workflow file %s", f)

		var wf workflowStepsFile
		require.NoError(t, yaml.Unmarshal(raw, &wf), "parse workflow file %s", f)

		for step := range wf.Steps {
			if !seen[step] {
				seen[step] = true
				keys = append(keys, step)
			}
		}
	}
	return keys
}

// productCriticalPathGuardEvidenceClassTerms are the five REQ-F-034
// disqualified-evidence-class terms, reusing E34-F02's evidence-authenticity
// vocabulary verbatim (research-report.md finding 5) — the same list
// TestProductCriticalPathGuardPartial_RendersStandalone already checks
// against the standalone partial render.
var productCriticalPathGuardEvidenceClassTerms = []string{
	"fixture data",
	"a captured/recorded run",
	"a hand-authored test actor",
	"a contract-only test",
	"a component-level test suite",
}

func newCanonicalOrchestratorRenderer(t *testing.T) *OrchestratorRenderer {
	t.Helper()
	renderer, err := NewOrchestratorRenderer(canonicalPromptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes + partials resolved")
	return renderer
}

// TestSpecificationTestPlanningApprovalRendersNameEvidenceClassesInGuardBlock
// is test-plan.md TC-004 (spec.md AC-4 / task AC-T1): rendering
// feature/specification.md, feature/test_planning.md, and feature/approval.md
// through the full production renderer and extracting each render's guard
// block must find all five REQ-F-034 evidence-class terms verbatim in every
// one of the three guard blocks (15 assertions: 5 terms x 3 files).
func TestSpecificationTestPlanningApprovalRendersNameEvidenceClassesInGuardBlock(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)

	targetPrompts := []string{
		"feature/specification.md",
		"feature/test_planning.md",
		"feature/approval.md",
	}

	for _, prompt := range targetPrompts {
		t.Run(prompt, func(t *testing.T) {
			block := extractProductCriticalPathGuardBlock(t, renderer, prompt)
			for _, term := range productCriticalPathGuardEvidenceClassTerms {
				assert.Contains(t, block, term,
					"%s: guard block must name disqualified evidence class %q verbatim", prompt, term)
			}
		})
	}
}

// productCriticalPathGuardCLIVerbPatterns are the `shark ` + CLI-verb
// substrings test-plan.md TC-005 (spec.md AC-5) names as disqualified from
// appearing inside a wired prompt's guard block.
var productCriticalPathGuardCLIVerbPatterns = []string{
	"shark claim",
	"shark heartbeat",
	"shark release",
	"shark status advance",
	"shark status set",
	"shark next-status",
	"shark set-status",
}

// TestTwelvePromptRendersGuardBlockContainsNoStatusOrCLIVerb is test-plan.md
// TC-005 (spec.md AC-5 / task AC-T2): rendering all twelve wired prompts
// through the full production renderer and extracting each render's guard
// block must find zero matches against the full `steps:` key set unioned
// from every entity workflow YAML file, and zero `shark ` + CLI-verb matches
// — scoped to the guard block only, since the surrounding prompt content
// legitimately contains status names and shark commands outside the guard.
func TestTwelvePromptRendersGuardBlockContainsNoStatusOrCLIVerb(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)

	statusTokens := workflowStepKeysUnion(t, canonicalWorkflowDir)
	require.NotEmpty(t, statusTokens, "expected at least one status token enumerated from workflow YAML files")

	for _, prompt := range productCriticalPathGuardConsumingPrompts {
		t.Run(prompt, func(t *testing.T) {
			block := extractProductCriticalPathGuardBlock(t, renderer, prompt)

			for _, status := range statusTokens {
				re := regexp.MustCompile(`\b` + regexp.QuoteMeta(status) + `\b`)
				assert.False(t, re.MatchString(block),
					"%s: guard block must not contain bare workflow status token %q", prompt, status)
			}
			for _, verbPattern := range productCriticalPathGuardCLIVerbPatterns {
				assert.False(t, strings.Contains(block, verbPattern),
					"%s: guard block must not contain shark CLI verb pattern %q", prompt, verbPattern)
			}
		})
	}
}
