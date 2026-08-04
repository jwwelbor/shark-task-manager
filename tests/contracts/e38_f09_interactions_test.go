// Package contracts exercises the published execution seams consumed by
// E38-F09. This file originates the E38-F09 contract test file: task
// T-E38-F09-006 creates it and every later E38-F09 task appends its own
// TestTC0XX_* function(s) here rather than creating a sibling file — see
// T-E38-F09-006's task spec.
//
// Read convention for skill/prompt repository content, matching
// e38_f07_interactions_test.go's readF07RepositoryFile/readF07EmbeddedFile
// split — appending tasks should reuse readF09RepositoryFile/
// readF09EmbeddedFile below rather than inventing a third pattern:
//   - skills/shark-attack/**  has an embedded mirror under
//     internal/sharkdata/default_data/, and dispatch renders from the
//     embedded copy — read it with readF09EmbeddedFile (sharkdata.ReadEmbedded).
//   - skills/shark-rider/**   has no embedded mirror — read it with
//     readF09RepositoryFile (a plain repo-root file read).
package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	commands "github.com/jwwelbor/shark-task-manager/internal/cli/commands"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"gopkg.in/yaml.v3"
)

func readF09RepositoryFile(t *testing.T, relPath string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(content)
}

func readF09EmbeddedFile(t *testing.T, relPath string) string {
	t.Helper()
	content, err := sharkdata.ReadEmbedded(relPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", relPath, err)
	}
	return string(content)
}

// TestF09ReadConventionHelpersResolveTheirDeclaredTrees is a smoke test for
// the two read-convention helpers declared in this file's header comment.
// Appending E38-F09 tasks call readF09RepositoryFile/readF09EmbeddedFile
// directly; this test keeps both helpers exercised (and therefore not
// dead code) from the moment this file is originated, and pins that each
// helper actually resolves the tree it is declared to own.
func TestF09ReadConventionHelpersResolveTheirDeclaredTrees(t *testing.T) {
	rider := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	if rider == "" {
		t.Fatal("readF09RepositoryFile must resolve a non-empty skills/shark-rider/** file")
	}

	attack := readF09EmbeddedFile(t, "skills/shark-attack/SKILL.md")
	if attack == "" {
		t.Fatal("readF09EmbeddedFile must resolve a non-empty skills/shark-attack/** embedded file")
	}
}

// f09ProjectRoot resolves the repository root from this test file's own
// location, independent of the process's working directory.
func f09ProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// buildSharkF09 builds the real shark binary once for a test. TC-002-09
// deliberately drives the compiled Cobra binary rather than resolveNext's
// mocked seam (see next_provenance_test.go) — the thing under test is the
// OS-level --prompt-out file write inside runNext's body, which only the
// real binary + real filesystem can prove.
func buildSharkF09(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-o", binary, "./cmd/shark")
	build.Dir = f09ProjectRoot(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build shark test binary: %v\n%s", err, output)
	}
	return binary
}

// f09AdversarialPromptFixture is TC-002's adversarial fixture, embedded
// directly as a workflow step's literal (non-.md/.tmpl) `prompt:` instruction
// so the dispatched prompt's instruction body is byte-known: embedded
// newlines (\n and \r\n — the \r\n pair is what TC-002-09d's CRLF-survival
// assertion depends on), a fenced code block, single/double quotes, Unicode
// (accented letter, combining mark, RTL mark, emoji), and shell
// metacharacters.
func f09AdversarialPromptFixture() string {
	return "adversarial line one\n" +
		"adversarial line two\r\n" +
		"fenced ```code fence``` block\n" +
		`single 'quote' and double "quote" pair` + "\n" +
		"unicode: café combining é rtl-mark‏ emoji\U0001F600\n" +
		"shell metacharacters: `cmd` $VAR ; | && > redirect"
}

// TestTC002_09PromptOutRealWritePathByteExactness is TC-002-09 (test-plan.md
// #tc-002, added per codex red-team BLOCKER): a black-box, command-level test
// driving the real `runNext`/flag-parsing/file-write chain via the compiled
// binary against a temp SQLite DB, proving:
//
//	(a) the --prompt-out file contains exactly resp.Prompt bytes, no trailing
//	    newline added by the write itself;
//	(b) sha256(file) == resp.PromptSHA256 from the same invocation's --json
//	    output;
//	(c) len(file) == resp.PromptBytes;
//	(d) the embedded CRLF fixture variant survives the write unmangled (a
//	    text-mode writer normalizing \r\n -> \n would fail this and (b)/(c));
//	(e) an unwritable --prompt-out target (a directory) fails loudly —
//	    non-zero exit, not a silent partial write.
func TestTC002_09PromptOutRealWritePathByteExactness(t *testing.T) {
	projectDir := t.TempDir()
	dbPath := filepath.Join(projectDir, "tc002-09.db")

	// A custom task-only workflow directory: epic/feature entity types fall
	// back to the embedded defaults (LoadMultiLevelWorkflowFromYAMLDir treats
	// each entity file as independently optional), so only task.yaml is
	// written. The "development" step's prompt is the literal adversarial
	// fixture text (no .md/.tmpl suffix => OrchestratorAction.PopulateTemplate
	// takes the plain-string path, not the template engine — see
	// internal/config/action/orchestrator.go PopulateTemplate).
	workflowDir := filepath.Join(projectDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("TC-002-09 create workflow dir: %v", err)
	}
	taskWorkflow := map[string]any{
		"version": "1.0",
		"start":   "development",
		"steps": map[string]any{
			"development": map[string]any{
				"phase":  "development",
				"action": "spawn_agent",
				"agent":  "developer",
				"prompt": f09AdversarialPromptFixture(),
				"outcomes": map[string]any{
					"pass":    "completed",
					"fail":    "development",
					"blocked": "blocked",
				},
			},
			"blocked": map[string]any{
				"phase":   "blocked",
				"action":  "pause",
				"parking": true,
			},
			"completed": map[string]any{
				"phase":    "done",
				"action":   "archive",
				"terminal": true,
			},
		},
	}
	taskYAML, err := yaml.Marshal(taskWorkflow)
	if err != nil {
		t.Fatalf("TC-002-09 marshal task workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "task.yaml"), taskYAML, 0o644); err != nil {
		t.Fatalf("TC-002-09 write task.yaml: %v", err)
	}
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{"workflow_config":%q}`, workflowDir)), 0o644); err != nil {
		t.Fatalf("TC-002-09 write config: %v", err)
	}

	// Seed a task directly at the "development" step via the production
	// repositories (bypassing entity-creation CLI/file scaffolding, which
	// this test does not need) — the same convention
	// seedRegistrationBaselineTC012 uses in e39_interactions_test.go.
	ctx := context.Background()
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-002-09 InitDB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "TC-002-09 provenance epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-002-09 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "TC-002-09 provenance feature"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-002-09 seed feature: %v", err)
	}
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "TC-002-09 provenance task"}, FeatureID: feature.ID, Status: "development", Priority: 5}
	if err := repository.NewTaskRepository(repoDB).Create(ctx, task); err != nil {
		t.Fatalf("TC-002-09 seed task: %v", err)
	}

	binary := buildSharkF09(t)
	runShark := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
		cmd.Dir = projectDir
		output, runErr := cmd.CombinedOutput()
		return string(output), runErr
	}

	promptOutPath := filepath.Join(t.TempDir(), "prompt.txt")
	out, runErr := runShark("next", "T-E01-F01-001", "--json", "--prompt-out", promptOutPath)
	if runErr != nil {
		t.Fatalf("TC-002-09 shark next --prompt-out failed: %v\n%s", runErr, out)
	}

	var resp commands.NextResponse
	if jsonErr := json.Unmarshal([]byte(out), &resp); jsonErr != nil {
		t.Fatalf("TC-002-09 decode next response: %v\n%s", jsonErr, out)
	}
	if resp.Action != "spawn_agent" || resp.PromptSHA256 == "" || resp.PromptBytes == 0 {
		t.Fatalf("TC-002-09 next response = %#v, want a populated spawn_agent dispatch with provenance fields", resp)
	}

	fileBytes, err := os.ReadFile(promptOutPath)
	if err != nil {
		t.Fatalf("TC-002-09 read --prompt-out file: %v", err)
	}

	// (a) the file contains exactly resp.Prompt bytes.
	if string(fileBytes) != resp.Prompt {
		t.Fatalf("TC-002-09 --prompt-out bytes != resp.Prompt (file len=%d, resp.Prompt len=%d)", len(fileBytes), len(resp.Prompt))
	}

	// (b) sha256(file) == resp.PromptSHA256 from the same invocation.
	sum := sha256.Sum256(fileBytes)
	if got := hex.EncodeToString(sum[:]); got != resp.PromptSHA256 {
		t.Fatalf("TC-002-09 sha256(file) = %s, want resp.PromptSHA256 = %s", got, resp.PromptSHA256)
	}

	// (c) len(file) == resp.PromptBytes.
	if len(fileBytes) != resp.PromptBytes {
		t.Fatalf("TC-002-09 len(file) = %d, want resp.PromptBytes = %d", len(fileBytes), resp.PromptBytes)
	}

	// (d) the embedded CRLF fixture variant survives unmangled — a text-mode
	// writer normalizing \r\n -> \n would already have failed (a)/(b)/(c)
	// above; this assertion names the specific defect class those would
	// otherwise leave undiagnosed.
	if !bytes.Contains(fileBytes, []byte("\r\n")) {
		t.Fatalf("TC-002-09 --prompt-out file lost the embedded CRLF sequence (normalized by a text-mode writer?)")
	}

	// (e) an unwritable target (an existing directory) must fail loudly, not
	// silently skip the write while still reporting success.
	unwritableTarget := t.TempDir()
	failOut, failErr := runShark("next", "T-E01-F01-001", "--json", "--prompt-out", unwritableTarget)
	if failErr == nil {
		t.Fatalf("TC-002-09 shark next --prompt-out <directory> succeeded, want a loud failure\n%s", failOut)
	}
}

// ---------------------------------------------------------------------------
// TC-003-01..07 (test-plan.md #tc-003) — T-E38-F09-007.
//
// TC-003-01..03 are content-only per the test plan's Caller-Path Contract:
// worker-control-schema.yaml is a schema/prose contract, not a running
// envelope processor (F09 introduces no Go type for it), so the entrypoint is
// templates.NewIncludeResolver resolving the embedded skill tree — the same
// seam TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly (e38_f04_interactions_
// test.go:99) uses.
//
// TC-003-04..07 drive the real roster-YAML-parsing validator
// (sharkdata.Validate, embed.go:1169's allowedMember map) with real YAML text
// — never a hand-built struct literal — so a missing allowedMember entry is
// caught the same way the existing model_tier coverage in
// internal/sharkdata/shark_attack_test.go already catches one.
// ---------------------------------------------------------------------------

// f09ControlEnvelopeExample mirrors one example_* block in
// worker-control-schema.yaml.
type f09ControlEnvelopeExample struct {
	Kind               string   `yaml:"kind"`
	RecommendedOutcome string   `yaml:"recommended_outcome"`
	Evidence           []string `yaml:"evidence"`
}

// f09WorkerControlSchemaDoc mirrors worker-control-schema.yaml's top-level
// shape: the bare `kind` vocabulary declaration plus one example per variant.
type f09WorkerControlSchemaDoc struct {
	Kind                   string                    `yaml:"kind"`
	ExampleFinal           f09ControlEnvelopeExample `yaml:"example_final"`
	ExampleQuestion        map[string]interface{}    `yaml:"example_question"`
	ExampleNeedsCouncil    f09ControlEnvelopeExample `yaml:"example_needs_council"`
	ExampleBlockedExternal f09ControlEnvelopeExample `yaml:"example_blocked_external"`
	ExampleFailed          f09ControlEnvelopeExample `yaml:"example_failed"`
}

// f09ResolveWorkerControlSchema resolves worker-control-schema.yaml through
// the real embedded tree via templates.NewIncludeResolver (TC-003-01..03's
// declared entrypoint), returning both the raw resolved text and its parsed
// shape.
func f09ResolveWorkerControlSchema(t *testing.T) (string, f09WorkerControlSchemaDoc) {
	t.Helper()
	projectRoot := t.TempDir()
	if _, err := sharkdata.Init(projectRoot); err != nil {
		t.Fatalf("TC-003 init embedded bundle: %v", err)
	}
	dataRoot := filepath.Join(projectRoot, sharkdata.SharkDataDirName)
	resolver := templates.NewIncludeResolver(dataRoot)

	raw, err := resolver.Resolve("{{include: skills/shark-attack/context/worker-control-schema.yaml}}")
	if err != nil {
		t.Fatalf("TC-003 resolve worker-control-schema.yaml: %v", err)
	}

	var doc f09WorkerControlSchemaDoc
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("TC-003 parse worker-control-schema.yaml: %v\n%s", err, raw)
	}
	return raw, doc
}

// TestTC003_01ControlEnvelopeRecommendedOutcomeRoundTripsVerbatim is
// TC-003-01: worker-control-schema.yaml round-trips a `kind: final,
// recommended_outcome: deep_verify` fixture unchanged through to the parent's
// `shark status advance --outcome deep_verify` call — the outcome string
// passed to status advance must equal the fixture's recommended_outcome
// verbatim, byte-for-byte, with no normalization/mapping table involved.
func TestTC003_01ControlEnvelopeRecommendedOutcomeRoundTripsVerbatim(t *testing.T) {
	raw, doc := f09ResolveWorkerControlSchema(t)

	if doc.ExampleFinal.Kind != "final" {
		t.Fatalf("TC-003-01 example_final.kind = %q, want %q", doc.ExampleFinal.Kind, "final")
	}
	if doc.ExampleFinal.RecommendedOutcome != "deep_verify" {
		t.Fatalf("TC-003-01 example_final.recommended_outcome = %q, want %q", doc.ExampleFinal.RecommendedOutcome, "deep_verify")
	}

	// Independently extract the literal value via regex against the raw
	// resolved text, so this assertion does not merely restate the YAML
	// unmarshaler's own output — an implementation that transformed the value
	// during YAML decoding (e.g. via a custom UnmarshalYAML) would still be
	// caught because the two extraction paths must agree.
	match := regexp.MustCompile(`recommended_outcome:\s*(\S+)`).FindStringSubmatch(raw)
	if match == nil {
		t.Fatalf("TC-003-01 could not find a recommended_outcome literal in the resolved schema text:\n%s", raw)
	}
	rawOutcome := match[1]
	if rawOutcome != doc.ExampleFinal.RecommendedOutcome {
		t.Fatalf("TC-003-01 raw recommended_outcome %q != YAML-parsed value %q — a transformation exists between the two extraction paths", rawOutcome, doc.ExampleFinal.RecommendedOutcome)
	}

	// Simulate the documented parent action:
	// `shark status advance <entity-key> --outcome <recommended_outcome>`.
	// The constructed flag value must equal the fixture value byte-for-byte.
	outcomeArgs := []string{"status", "advance", "E01-F01-001", "--outcome", doc.ExampleFinal.RecommendedOutcome}
	if got := outcomeArgs[len(outcomeArgs)-1]; got != rawOutcome {
		t.Fatalf("TC-003-01 constructed --outcome argument %q != raw fixture value %q", got, rawOutcome)
	}
}

// TestTC003_02ControlEnvelopeRecommendedOutcomeOpaqueToKindVocabulary is
// TC-003-02: a recommended_outcome value absent from any control-vocabulary
// enum (e.g. `simple`, `standard`) still passes through — no enum-membership
// check exists in the envelope-to-advance path.
func TestTC003_02ControlEnvelopeRecommendedOutcomeOpaqueToKindVocabulary(t *testing.T) {
	_, doc := f09ResolveWorkerControlSchema(t)

	kindVocabulary := strings.Split(doc.Kind, "|")
	if len(kindVocabulary) != 5 {
		t.Fatalf("TC-003-02 kind vocabulary = %v, want exactly 5 members (final|question|needs_council|blocked_external|failed)", kindVocabulary)
	}
	kindSet := make(map[string]struct{}, len(kindVocabulary))
	for _, k := range kindVocabulary {
		kindSet[k] = struct{}{}
	}
	if _, collides := kindSet[doc.ExampleFinal.RecommendedOutcome]; collides {
		t.Fatalf("TC-003-02 recommended_outcome %q unexpectedly collides with a kind vocabulary member %v", doc.ExampleFinal.RecommendedOutcome, kindVocabulary)
	}

	// Several outcome values named directly in REQ-F-003 ("pass", "simple",
	// "standard") plus an arbitrary unrecognized key must all be absent from
	// the kind vocabulary too — proving the pass-through has no special case
	// for any particular outcome string, only a single five-member enum
	// (`kind`) exists anywhere in this schema.
	for _, outcome := range []string{"pass", "simple", "standard", "an-unrecognized-outcome-key"} {
		if _, collides := kindSet[outcome]; collides {
			t.Fatalf("TC-003-02 fixture outcome %q unexpectedly collides with the kind vocabulary %v", outcome, kindVocabulary)
		}
	}
}

// TestTC003_03QuestionEnvelopeCarriesNoQuestionID is TC-003-03: the
// `question` envelope variant schema carries no `question_id` field (D-005)
// — the parent, never the worker, mints the E39 Q### identity.
func TestTC003_03QuestionEnvelopeCarriesNoQuestionID(t *testing.T) {
	_, doc := f09ResolveWorkerControlSchema(t)

	if doc.ExampleQuestion == nil {
		t.Fatal("TC-003-03 worker-control-schema.yaml missing example_question")
	}
	if kind, _ := doc.ExampleQuestion["kind"].(string); kind != "question" {
		t.Fatalf("TC-003-03 example_question.kind = %v, want %q", doc.ExampleQuestion["kind"], "question")
	}
	if _, exists := doc.ExampleQuestion["question_id"]; exists {
		t.Fatalf("TC-003-03 example_question must not declare question_id — D-005 requires the parent, not the worker, to mint the E39 Q### identity")
	}
	for _, required := range []string{"entity_key", "category", "question", "why_blocking", "evidence"} {
		if _, exists := doc.ExampleQuestion[required]; !exists {
			t.Fatalf("TC-003-03 example_question missing required field %q", required)
		}
	}
}

// f09RosterFixture builds a minimal, otherwise-valid shark-attack roster
// fixture with a single member, whose extra field lines (already correctly
// YAML-indented, e.g. "    capability_profile: deep\n") are injected by the
// caller. This drives real YAML text through the real validator rather than
// hand-constructing a sharkAttackRosterMember struct literal.
func f09RosterFixture(memberExtraLines string) string {
	return "team: shark-attack\n" +
		"chair: developer\n" +
		"memory_root: docs/council\n" +
		"communication:\n" +
		"  inbox_root: docs/council/inbox\n" +
		"  acknowledge_after_read: true\n" +
		"  retain_decisions: true\n" +
		"escalation:\n" +
		"  triggers_file: docs/product/escalation_triggers.md\n" +
		"  route: council-review\n" +
		"members:\n" +
		"  - id: developer\n" +
		"    role: Developer\n" +
		"    responsibilities:\n" +
		"      - Implement scoped work\n" +
		memberExtraLines
}

// f09ValidateRosterFixture materializes the embedded bundle into a temp
// project, overwrites its roster-schema.yaml with the given fixture, and
// returns the resulting sharkdata.Validate report — the same
// Init-then-Validate seam internal/sharkdata/shark_attack_test.go's TC-101/
// TC-102 already use.
func f09ValidateRosterFixture(t *testing.T, memberExtraLines string) *sharkdata.ValidationReport {
	t.Helper()
	root := t.TempDir()
	if _, err := sharkdata.Init(root); err != nil {
		t.Fatalf("TC-003 init embedded bundle: %v", err)
	}
	rosterPath := filepath.Join(root, sharkdata.SharkDataDirName, "skills", "shark-attack", "context", "roster-schema.yaml")
	if err := os.WriteFile(rosterPath, []byte(f09RosterFixture(memberExtraLines)), 0o644); err != nil {
		t.Fatalf("TC-003 write roster fixture: %v", err)
	}
	report, err := sharkdata.Validate(root)
	if err != nil {
		t.Fatalf("TC-003 validate roster fixture: %v", err)
	}
	return report
}

// f09RosterIssuesAtLevel filters a validation report's issues to the given
// level AND to the roster file itself — Validate() walks the whole embedded
// bundle, so unrelated pre-existing issues on other skills/prompts (unrelated
// to this fixture's roster change) must not leak into the roster-specific
// assertions below.
func f09RosterIssuesAtLevel(report *sharkdata.ValidationReport, level string) []sharkdata.ValidationIssue {
	var out []sharkdata.ValidationIssue
	for _, issue := range report.Issues {
		if issue.Level == level && strings.Contains(issue.Path, "roster-schema.yaml") {
			out = append(out, issue)
		}
	}
	return out
}

// TestTC003_04RosterCapabilityProfileValidatesCleanly is TC-003-04: a roster
// fixture with `capability_profile: deep` (no requirements) validates
// cleanly, and its Negative Cases companion: an invalid capability_profile
// value (equivalence class outside {fast,balanced,deep}) fails validation.
func TestTC003_04RosterCapabilityProfileValidatesCleanly(t *testing.T) {
	report := f09ValidateRosterFixture(t, "    capability_profile: deep\n")
	if report.HasErrors() {
		t.Fatalf("TC-003-04 capability_profile: deep must validate cleanly: %+v", report.Issues)
	}

	t.Run("invalid capability_profile value is rejected", func(t *testing.T) {
		report := f09ValidateRosterFixture(t, "    capability_profile: turbo\n")
		if !report.HasErrors() {
			t.Fatal("TC-003-04 capability_profile: turbo (outside fast|balanced|deep) must fail validation")
		}
		found := false
		for _, issue := range report.Issues {
			if issue.Level == sharkdata.IssueLevelError && strings.Contains(issue.Message, "capability_profile") {
				found = true
			}
		}
		if !found {
			t.Fatalf("TC-003-04 expected a capability_profile validation error, got: %+v", report.Issues)
		}
	})

	t.Run("whitespace-only capability_profile is rejected", func(t *testing.T) {
		report := f09ValidateRosterFixture(t, "    capability_profile: '   '\n")
		if !report.HasErrors() {
			t.Fatal("TC-003-04 whitespace-only capability_profile must fail validation")
		}
	})

	t.Run("optional requirements mapping is accepted", func(t *testing.T) {
		report := f09ValidateRosterFixture(t, "    requirements:\n      tools: [git, go]\n      context: bounded\n      messaging: follow_up\n      isolation: directed\n")
		if report.HasErrors() {
			t.Fatalf("TC-003-04 valid requirements mapping must validate cleanly: %+v", report.Issues)
		}
	})
}

// TestTC003_05RosterLegacyModelTierValidatesWithDeprecationWarning is
// TC-003-05: a roster fixture with legacy `model_tier: opus` validates and
// produces a deprecation warning — asserted as an emitted warning-level
// issue, not merely the absence of an error.
func TestTC003_05RosterLegacyModelTierValidatesWithDeprecationWarning(t *testing.T) {
	report := f09ValidateRosterFixture(t, "    model_tier: opus\n")
	if report.HasErrors() {
		t.Fatalf("TC-003-05 model_tier: opus must validate without errors: %+v", report.Issues)
	}
	warnings := f09RosterIssuesAtLevel(report, sharkdata.IssueLevelWarning)
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "model_tier") && strings.Contains(strings.ToLower(w.Message), "deprecat") {
			found = true
		}
	}
	if !found {
		t.Fatalf("TC-003-05 expected a model_tier deprecation warning to be emitted, got issues: %+v", report.Issues)
	}
}

// TestTC003_06RosterBothFieldsValidatesModelTierUnmapped is TC-003-06: a
// roster fixture with BOTH capability_profile and model_tier set validates;
// model_tier still produces no provider mapping — asserted here as an
// externally observable property of the validator's output: the only issue
// concerning model_tier is its deprecation warning, and it never names or
// selects a provider/model.
func TestTC003_06RosterBothFieldsValidatesModelTierUnmapped(t *testing.T) {
	report := f09ValidateRosterFixture(t, "    capability_profile: deep\n    model_tier: opus\n")
	if report.HasErrors() {
		t.Fatalf("TC-003-06 capability_profile + model_tier together must validate without errors: %+v", report.Issues)
	}

	warnings := f09RosterIssuesAtLevel(report, sharkdata.IssueLevelWarning)
	if len(warnings) != 1 {
		t.Fatalf("TC-003-06 expected exactly one warning (model_tier deprecation only — capability_profile must add none), got: %+v", warnings)
	}
	warning := warnings[0]
	if !strings.Contains(warning.Message, "model_tier") {
		t.Fatalf("TC-003-06 the single warning must concern model_tier, got: %q", warning.Message)
	}
	for _, providerName := range []string{"anthropic", "openai", "claude-", "gpt-", "codex"} {
		if strings.Contains(strings.ToLower(warning.Message), providerName) {
			t.Fatalf("TC-003-06 model_tier deprecation warning names a provider/model (%q) — model_tier must produce no provider mapping: %q", providerName, warning.Message)
		}
	}
}

// TestTC003_07RosterNeitherFieldValidatesSilently is TC-003-07: a roster
// fixture with neither capability_profile nor model_tier validates silently
// — no warning, no error.
func TestTC003_07RosterNeitherFieldValidatesSilently(t *testing.T) {
	report := f09ValidateRosterFixture(t, "")
	if report.HasErrors() {
		t.Fatalf("TC-003-07 a roster omitting both fields must validate without errors: %+v", report.Issues)
	}
	if warnings := f09RosterIssuesAtLevel(report, sharkdata.IssueLevelWarning); len(warnings) != 0 {
		t.Fatalf("TC-003-07 a roster omitting both fields must validate silently (no warnings), got: %+v", warnings)
	}
}

// ---------------------------------------------------------------------------
// TC-005-01..13 (test-plan.md #tc-005) — T-E38-F09-008.
//
// operating-model.md is content-only (D-002, no runtime change): its scenario
// table and 9-row decision table are asserted via readF09EmbeddedFile +
// sharkdata.ReadEmbedded, the same pattern e38_f04/e38_f07_interactions_
// test.go use for skill-prose contracts.
//
// TC-005-05 is the one sub-case with a live runtime decision (AC-002): it
// adds a real-DB CLI seam — seeding two tied, unclaimed, evidence-free
// sibling features under a cascading epic and driving the compiled shark
// binary — to prove no parallel dispatch occurs absent recorded evidence.
// TC-004's own real-DB CLI seam (T-E38-F09-011/012) does not exist yet at
// this task's point in the dependency graph (T-008 depends only on T-006),
// so this sub-case builds its own minimal fixture using the same
// db.InitDB-seed + compiled-binary pattern e39_interactions_test.go's
// TestTC308_PublicRunCascadeFallsThroughBlockedChild already establishes,
// rather than referencing test code that has not landed yet.
// ---------------------------------------------------------------------------

// f09ResolveOperatingModel resolves operating-model.md through the real
// embedded tree via sharkdata.ReadEmbedded (readF09EmbeddedFile), the
// declared read convention for skills/shark-attack/** content in this file's
// header comment.
func f09ResolveOperatingModel(t *testing.T) string {
	t.Helper()
	return readF09EmbeddedFile(t, "skills/shark-attack/context/operating-model.md")
}

// f09ParseMarkdownTable extracts a GitHub-flavored markdown table's data rows
// (header and separator rows excluded) from content, locating the table by a
// substring unique to its header row. Cell values are trimmed. Parsing the
// table directly — rather than hand-transcribing its rows into Go literals —
// means an edit that breaks or reorders a row is caught by these tests
// instead of silently drifting from what operating-model.md actually says.
func f09ParseMarkdownTable(t *testing.T, content, headerContains string) [][]string {
	t.Helper()
	lines := strings.Split(content, "\n")
	var rows [][]string
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inTable {
			if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, headerContains) {
				inTable = true
			}
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			break
		}
		// Skip the "|---|---|..." separator row.
		if strings.Trim(strings.ReplaceAll(trimmed, "|", ""), "- ") == "" {
			continue
		}
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		t.Fatalf("no markdown table found in operating-model.md containing header %q", headerContains)
	}
	return rows
}

// f09OperatingModelExamplesTable parses the "Illustrative examples" table
// (columns: Work, Coordination, Topology, Why).
func f09OperatingModelExamplesTable(t *testing.T) [][]string {
	t.Helper()
	return f09ParseMarkdownTable(t, f09ResolveOperatingModel(t), "| Work |")
}

// f09OperatingModelDecisionTable parses the 9-row two-axis decision table
// (columns: #, Requested coordination, Requested topology, Ownership
// evidence recorded?, Isolation evidence recorded?, Resolved topology).
func f09OperatingModelDecisionTable(t *testing.T) [][]string {
	t.Helper()
	rows := f09ParseMarkdownTable(t, f09ResolveOperatingModel(t), "| # |")
	if len(rows) != 9 {
		t.Fatalf("TC-005 decision table has %d rows, want the full 9-row table", len(rows))
	}
	return rows
}

// TestTC005_01DirectSequentialScenarioExists is TC-005-01: a `Direct` +
// `Sequential` scenario row exists in the illustrative examples table
// (AC-001).
func TestTC005_01DirectSequentialScenarioExists(t *testing.T) {
	rows := f09OperatingModelExamplesTable(t)
	for _, row := range rows {
		if len(row) >= 3 && row[1] == "Direct" && row[2] == "Sequential" {
			return
		}
	}
	t.Fatalf("TC-005-01 no Direct + Sequential row found in operating-model.md's examples table: %+v", rows)
}

// TestTC005_02BatchParallelResearchThenSequentialWritesScenarioExists is
// TC-005-02: a `Batch` + parallel-research-then-sequential-writes scenario
// row exists (AC-001).
func TestTC005_02BatchParallelResearchThenSequentialWritesScenarioExists(t *testing.T) {
	rows := f09OperatingModelExamplesTable(t)
	for _, row := range rows {
		if len(row) >= 3 && row[1] == "Batch" &&
			strings.Contains(row[2], "Parallel research") &&
			strings.Contains(row[2], "Sequential") {
			return
		}
	}
	t.Fatalf("TC-005-02 no Batch + parallel-research-then-sequential-writes row found: %+v", rows)
}

// TestTC005_03CouncilMixedScenarioExists is TC-005-03: a `Council` +
// mixed-topology scenario row exists (AC-001).
func TestTC005_03CouncilMixedScenarioExists(t *testing.T) {
	rows := f09OperatingModelExamplesTable(t)
	for _, row := range rows {
		if len(row) >= 3 && row[1] == "Council" && strings.Contains(strings.ToLower(row[2]), "mixed") {
			return
		}
	}
	t.Fatalf("TC-005-03 no Council + mixed-topology row found: %+v", rows)
}

// TestTC005_04CoordinationLevelDoesNotDetermineTopology is TC-005-04:
// changing only the coordination-level column of a fixture row does not
// change the topology column value.
//
// The primary assertion is a direct, mechanical reading of that sentence:
// for every pair of decision-table rows whose requested topology, ownership
// evidence, and isolation evidence are identical, the resolved topology
// must also be identical — i.e. the requested-coordination column is the
// *only* thing allowed to differ between such a pair, and it must have no
// effect. Rows 1/7 (Direct vs. Council, both requesting Sequential with n/a
// evidence) and rows 2/8 (Batch vs. Council, both requesting Parallel with
// ownership with yes/no evidence) are the table's two such pairs.
//
// A secondary, coarser check proves the same independence the other way:
// the same coordination level (Direct) resolves to more than one topology
// across the table (Sequential in row 1, Parallel with ownership in row 9).
func TestTC005_04CoordinationLevelDoesNotDetermineTopology(t *testing.T) {
	rows := f09OperatingModelDecisionTable(t)

	type fixtureKey struct{ topology, ownership, isolation string }
	resolvedByFixture := map[fixtureKey]map[string]string{} // fixture -> coordination -> resolved topology
	topologiesByCoordination := map[string]map[string]bool{}
	for _, row := range rows {
		if len(row) < 6 {
			t.Fatalf("TC-005-04 decision table row has %d cells, want 6: %+v", len(row), row)
		}
		coordination, topology, ownership, isolation, resolved := row[1], row[2], row[3], row[4], row[5]
		// Bucket by the leading resolved-topology word set so a parenthetical
		// rationale (e.g. "(degraded — ...)" or a per-row independence note)
		// doesn't fragment rows that resolve to the same actual topology.
		resolvedBucket := strings.TrimSpace(strings.SplitN(resolved, "(", 2)[0])

		key := fixtureKey{topology, ownership, isolation}
		if resolvedByFixture[key] == nil {
			resolvedByFixture[key] = map[string]string{}
		}
		resolvedByFixture[key][coordination] = resolvedBucket

		if topologiesByCoordination[coordination] == nil {
			topologiesByCoordination[coordination] = map[string]bool{}
		}
		topologiesByCoordination[coordination][resolvedBucket] = true
	}

	pairsChecked := 0
	for key, byCoordination := range resolvedByFixture {
		if len(byCoordination) < 2 {
			continue // only one coordination level requests this exact fixture
		}
		var want string
		for coordination, resolved := range byCoordination {
			if want == "" {
				want = resolved
			} else if resolved != want {
				t.Fatalf("TC-005-04 fixture %+v resolves to %q under one coordination level but %q under %q — coordination level must not affect the resolved topology when the rest of the row is unchanged", key, want, resolved, coordination)
			}
			pairsChecked++
		}
	}
	if pairsChecked == 0 {
		t.Fatal("TC-005-04 found no pair of decision-table rows sharing the same requested topology/evidence — the table cannot demonstrate coordination-level independence without at least one such pair")
	}

	if got := len(topologiesByCoordination["Direct"]); got < 2 {
		t.Fatalf("TC-005-04 coordination level %q resolves to %d distinct topologies %+v, want >= 2 (coordination must not determine topology)", "Direct", got, topologiesByCoordination["Direct"])
	}
}

// f09DecodeNextJSON decodes `shark next`'s JSON output generically via
// map[string]interface{} rather than a strict struct, so an unexpected key's
// mere presence — e.g. a stray "prompt" on a fork response, which strict
// struct decoding would silently drop — is itself detectable by this file's
// TC-005-05 runtime sub-case.
func f09DecodeNextJSON(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("decode shark next JSON: %v\n%s", err, raw)
	}
	return decoded
}

// f09Tc005RuntimeForkFixture seeds a real temp SQLite database (via
// db.InitDB, matching e39_interactions_test.go's TestTC308 cascade-fixture
// pattern) with an epic at a cascading status and two sibling features tied
// for the same dispatch tier — both non-terminal, unclaimed, dependency-free,
// with no execution_order set — so `shark next` on the epic must return a
// `parallel_candidates` fork rather than a single dispatch. A minimal custom
// workflow (mirroring TestTC308's own trimmed epic/feature YAML) supplies the
// cascade/spawn_agent actions; no evidence of ownership or isolation is
// recorded anywhere in this fixture, which is the scenario under test.
func f09Tc005RuntimeForkFixture(t *testing.T) (dbPath, projectDir string, sqlDB *sql.DB) {
	t.Helper()
	ctx := context.Background()
	projectDir = t.TempDir()
	dbPath = filepath.Join(projectDir, "tc005-05.db")

	workflowDir := filepath.Join(projectDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("TC-005-05 create workflow dir: %v", err)
	}
	epicYAML := `version: "1.0"
start: active
steps:
  active:
    phase: execution
    action: cascade
    outcomes:
      pass: completed
  completed:
    phase: done
    action: archive
    terminal: true
`
	featureYAML := `version: "1.0"
start: ready
steps:
  ready:
    phase: development
    action: spawn_agent
    agent: developer
    prompt: "TC-005-05 fixture dispatch prompt: no ownership or isolation evidence is recorded for this fork."
    outcomes:
      pass: completed
      fail: ready
      blocked: blocked
  blocked:
    phase: blocked
    action: pause
    parking: true
  completed:
    phase: done
    action: archive
    terminal: true
`
	if err := os.WriteFile(filepath.Join(workflowDir, "epic.yaml"), []byte(epicYAML), 0o644); err != nil {
		t.Fatalf("TC-005-05 write epic workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "feature.yaml"), []byte(featureYAML), 0o644); err != nil {
		t.Fatalf("TC-005-05 write feature workflow: %v", err)
	}
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{"workflow_config":%q}`, workflowDir)), 0o644); err != nil {
		t.Fatalf("TC-005-05 write config: %v", err)
	}

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-005-05 InitDB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "TC-005-05 fork root"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-005-05 seed epic: %v", err)
	}
	featureRepo := repository.NewFeatureRepository(repoDB)
	for _, key := range []string{"E01-F01", "E01-F02"} {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Key: key, Title: "TC-005-05 tied candidate " + key}, EpicID: epic.ID, Status: models.FeatureStatus("ready")}
		if err := featureRepo.Create(ctx, feature); err != nil {
			t.Fatalf("TC-005-05 seed feature %s: %v", key, err)
		}
	}
	return dbPath, projectDir, sqlDB
}

// TestTC005_05NoOwnershipEvidenceDegradesToSequential is TC-005-05 (AC-002):
// a fixture row requesting a parallel topology with no recorded
// ownership/isolation evidence resolves to `Sequential`. The content
// subtest asserts the documented rule (row 3 of the decision table); the
// runtime subtest reuses a real-DB CLI seam (the fixture above) to show that
// `shark next` itself never dispatches more than one entity from a fork —
// the concrete Shark-side signal the task's brownfield context calls for.
//
// Boundary this runtime subtest does NOT cover: it asserts the *floor*, not
// the AC-002 condition itself. It would pass identically whether or not
// ownership/isolation evidence were recorded, because Shark's fork mechanism
// carries no evidence concept at all (D-002: operating-model.md is
// content-only, no runtime change) — the fixture's "no evidence recorded"
// state is simply this fixture's absence of any evidence-recording
// mechanism, not a Shark-enforced check of one. What the runtime subtest
// proves is that *nothing* about calling `shark next` on a fork — regardless
// of what a chair has or hasn't recorded — can produce concurrent dispatch;
// the degrade-on-missing-evidence *decision* is enforced chair-side by
// operating-model.md's prose, which the content subtest above (and
// TestTC005_07to12's row-3/row-5 pair) cover.
func TestTC005_05NoOwnershipEvidenceDegradesToSequential(t *testing.T) {
	t.Run("content: decision table row 3 documents the degrade", func(t *testing.T) {
		rows := f09OperatingModelDecisionTable(t)
		row := rows[2] // row 3: Batch | Parallel with ownership | no | no | Sequential (degraded)
		if row[1] != "Batch" || row[2] != "Parallel with ownership" || row[3] != "no" || row[4] != "no" {
			t.Fatalf("TC-005-05 decision table row 3 = %+v, want the no-ownership/no-isolation degrade row", row)
		}
		if !strings.HasPrefix(row[5], "Sequential") || !strings.Contains(strings.ToLower(row[5]), "degrad") {
			t.Fatalf("TC-005-05 decision table row 3 resolved topology = %q, want a Sequential degrade", row[5])
		}
	})

	t.Run("runtime: shark next never dispatches more than one candidate from a fork", func(t *testing.T) {
		dbPath, projectDir, sqlDB := f09Tc005RuntimeForkFixture(t)
		binary := buildSharkF09(t)
		runShark := func(args ...string) (string, error) {
			t.Helper()
			cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
			cmd.Dir = projectDir
			out, runErr := cmd.CombinedOutput()
			return string(out), runErr
		}
		countEntityClaims := func() int {
			t.Helper()
			var count int
			if err := sqlDB.QueryRow("SELECT COUNT(*) FROM entity_claims").Scan(&count); err != nil {
				t.Fatalf("TC-005-05 count entity_claims: %v", err)
			}
			return count
		}

		if before := countEntityClaims(); before != 0 {
			t.Fatalf("TC-005-05 entity_claims has %d rows before the fork call, want 0", before)
		}

		forkOut, err := runShark("next", "E01", "--json")
		if err != nil {
			t.Fatalf("TC-005-05 shark next E01 (fork) failed: %v\n%s", err, forkOut)
		}
		// The fork call itself leaves no lease behind — the plan's own
		// framing of this seam ("assert against the real absence of a
		// recorded claim"). This is not the load-bearing assertion for
		// "no parallel dispatch": `shark next` never claims on any call,
		// single or forked, so an empty entity_claims table alone would not
		// distinguish this fork from an ordinary single dispatch. The
		// prompt/agent_type absence checks below carry that weight.
		if after := countEntityClaims(); after != 0 {
			t.Fatalf("TC-005-05 entity_claims has %d rows after the fork call, want 0 (the fork call must not itself claim any candidate)", after)
		}
		fork := f09DecodeNextJSON(t, forkOut)
		if fork["action"] != "parallel_candidates" {
			t.Fatalf("TC-005-05 fork action = %v, want parallel_candidates\n%s", fork["action"], forkOut)
		}
		entities, ok := fork["entities"].([]interface{})
		if !ok || len(entities) != 2 {
			t.Fatalf("TC-005-05 fork entities = %v, want exactly 2 tied candidates\n%s", fork["entities"], forkOut)
		}
		// The fork envelope is read-only: no worker prompt or agent type is
		// present anywhere on it, so a caller cannot spawn work directly from
		// this response — it can only enumerate candidates.
		if _, hasPrompt := fork["prompt"]; hasPrompt {
			t.Fatalf("TC-005-05 fork response unexpectedly carries a prompt field: %s", forkOut)
		}
		if _, hasAgentType := fork["agent_type"]; hasAgentType {
			t.Fatalf("TC-005-05 fork response unexpectedly carries an agent_type field: %s", forkOut)
		}
		for _, raw := range entities {
			candidate, ok := raw.(map[string]interface{})
			if !ok {
				t.Fatalf("TC-005-05 fork candidate is not an object: %v", raw)
			}
			if _, hasPrompt := candidate["prompt"]; hasPrompt {
				t.Fatalf("TC-005-05 fork candidate unexpectedly carries a prompt field: %v", candidate)
			}
		}

		// Dispatching one candidate individually returns exactly one live
		// dispatch step, and its sibling is untouched by that call — proving
		// the two candidates are never dispatched together. Absent recorded
		// evidence, the fork degrades to the same one-at-a-time behavior
		// Sequential topology would produce.
		firstOut, err := runShark("next", "E01-F01", "--json")
		if err != nil {
			t.Fatalf("TC-005-05 shark next E01-F01 failed: %v\n%s", err, firstOut)
		}
		first := f09DecodeNextJSON(t, firstOut)
		if first["action"] != "spawn_agent" || first["prompt"] == "" || first["prompt"] == nil {
			t.Fatalf("TC-005-05 individual dispatch of E01-F01 = %v, want a populated spawn_agent dispatch\n%s", first, firstOut)
		}

		secondOut, err := runShark("next", "E01-F02", "--json")
		if err != nil {
			t.Fatalf("TC-005-05 shark next E01-F02 failed: %v\n%s", err, secondOut)
		}
		second := f09DecodeNextJSON(t, secondOut)
		if second["action"] != "spawn_agent" || second["prompt"] == "" || second["prompt"] == nil {
			t.Fatalf("TC-005-05 individual dispatch of E01-F02 = %v, want its own independent spawn_agent dispatch, unaffected by dispatching E01-F01 first\n%s", second, secondOut)
		}
	})
}

// TestTC005_06IsolationAloneDoesNotParallelizeDependentWork is TC-005-06
// (D-007's explicit non-goal): isolation evidence present does not by itself
// make logically dependent (producer/consumer contract-ordered) work run in
// parallel — content assertion against operating-model.md's degradation
// rule section.
func TestTC005_06IsolationAloneDoesNotParallelizeDependentWork(t *testing.T) {
	content := f09ResolveOperatingModel(t)
	if !strings.Contains(content, "Isolation does not make logically dependent work parallel") {
		t.Fatalf("TC-005-06 operating-model.md must state the D-007 non-goal explicitly (isolation does not parallelize dependent work); content:\n%s", content)
	}
	if !strings.Contains(content, "producer/consumer contract order still applies") {
		t.Fatalf("TC-005-06 operating-model.md must state that producer/consumer contract order still applies regardless of topology; content:\n%s", content)
	}
}

// TestTC005_07to12DecisionTableRemainingRowsResolveExpectedTopology is
// TC-005-07..12: each remaining decision-table row resolves to its
// documented topology, including both independent degrade-to-Sequential
// paths (row 3 — TC-005-05, asserted above — is the missing-ownership path;
// row 5 — TC-005-09 — is the missing-isolation path; a fix for one path
// could silently leave the other broken, so both are asserted).
func TestTC005_07to12DecisionTableRemainingRowsResolveExpectedTopology(t *testing.T) {
	rows := f09OperatingModelDecisionTable(t)

	cases := []struct {
		name         string
		rowIndex     int // 0-based index into the parsed decision table
		coordination string
		topology     string
		ownership    string
		isolation    string
		wantPrefix   string
		wantDegrade  bool
	}{
		{"TC-005-07", 1, "Batch", "Parallel with ownership", "yes", "no", "Parallel with ownership", false},
		{"TC-005-08", 3, "Batch", "Parallel with isolation", "no", "yes", "Parallel with isolation", false},
		{"TC-005-09", 4, "Batch", "Parallel with isolation", "no", "no", "Sequential", true},
		{"TC-005-10", 5, "Batch", "Parallel with isolation", "yes", "yes", "Parallel with isolation", false},
		// Row 7 (Council + Sequential). test-plan.md's decision table (#tc-005)
		// labels this row "TC-005-03" too, reusing the sub-case ID already
		// covered by TestTC005_03CouncilMixedScenarioExists above (which
		// asserts the *examples* table's Council+mixed row, not this one).
		// Named "council-sequential" here rather than duplicating "TC-005-03"
		// as a Go subtest name, since the plan's ID reuse cannot be mirrored
		// as two identically-named subtests.
		{"council-sequential", 6, "Council", "Sequential", "n/a", "n/a", "Sequential", false},
		{"TC-005-11", 7, "Council", "Parallel with ownership", "yes", "no", "Parallel with ownership", false},
		{"TC-005-12", 8, "Direct", "Parallel with ownership", "yes", "no", "Parallel with ownership", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := rows[tc.rowIndex]
			if row[1] != tc.coordination || row[2] != tc.topology || row[3] != tc.ownership || row[4] != tc.isolation {
				t.Fatalf("%s decision table row %d = %+v, want coordination=%q topology=%q ownership=%q isolation=%q", tc.name, tc.rowIndex+1, row, tc.coordination, tc.topology, tc.ownership, tc.isolation)
			}
			if !strings.HasPrefix(row[5], tc.wantPrefix) {
				t.Fatalf("%s resolved topology = %q, want it to start with %q", tc.name, row[5], tc.wantPrefix)
			}
			if tc.wantDegrade && !strings.Contains(strings.ToLower(row[5]), "degrad") {
				t.Fatalf("%s resolved topology = %q, want an explicit degrade rationale", tc.name, row[5])
			}
		})
	}
}

// TestTC005_13DirectPairsWithParallelTopologyProvingBothDirectionIndependence
// is TC-005-13: row 9 (`Direct` + a parallel topology request) additionally
// proves coordination level and topology vary independently in *both*
// directions — not just that `Batch`/`Council` can pair with a non-parallel
// topology (rows 1/7 already show that), but that `Direct` itself can pair
// with a parallel topology.
func TestTC005_13DirectPairsWithParallelTopologyProvingBothDirectionIndependence(t *testing.T) {
	rows := f09OperatingModelDecisionTable(t)
	row9 := rows[8]
	if row9[1] != "Direct" || row9[2] != "Parallel with ownership" {
		t.Fatalf("TC-005-13 decision table row 9 = %+v, want a Direct + Parallel with ownership request", row9)
	}
	if !strings.HasPrefix(row9[5], "Parallel with ownership") {
		t.Fatalf("TC-005-13 row 9 resolved topology = %q, want Parallel with ownership (coordination level Direct must not force a Sequential degrade)", row9[5])
	}

	// Cross-check against the examples table's Direct + Sequential row
	// (TC-005-01): the *same* coordination level, Direct, appears with two
	// different resolved topologies across the two tables — the clearest
	// available proof that coordination level does not determine topology.
	examples := f09OperatingModelExamplesTable(t)
	directSequentialFound := false
	for _, row := range examples {
		if len(row) >= 3 && row[1] == "Direct" && row[2] == "Sequential" {
			directSequentialFound = true
		}
	}
	if !directSequentialFound {
		t.Fatalf("TC-005-13 expected a Direct + Sequential examples row to contrast against the decision table's Direct + Parallel with ownership row 9")
	}

	content := f09ResolveOperatingModel(t)
	if !strings.Contains(content, "independent in **both**") && !strings.Contains(content, "independent in both") {
		t.Fatalf("TC-005-13 operating-model.md must explicitly state the axes are independent in both directions")
	}
}

// TestTC005_17Row9IndependenceIsClassificationOnlyNotOperationalEquivalence
// is a regression guard added during T-E38-F09-008's rework
// (code-review-2026-08-03T0801-E38-F09.md finding #1 / blocker): row 9's
// "independent in both directions" proof is a classification-time property
// only. `direct.md` (TC-005-15) never consults the resolved topology and
// always dispatches exactly one worker — a single bounded entity has no wave
// to shape. operating-model.md must say so explicitly, and must never again
// claim that a `Direct`-classified request executes an actual parallel
// dispatch "exactly like" `Batch`/`Council` — that specific phrase is what
// the code review's blocker cited: it contradicted `coordinate.md`'s routing
// (topology never changes which procedure file is selected) and `direct.md`'s
// own "performs no topology-selection step of its own" statement.
func TestTC005_17Row9IndependenceIsClassificationOnlyNotOperationalEquivalence(t *testing.T) {
	content := f09ResolveOperatingModel(t)

	const forbidden = "exactly like Batch"
	if strings.Contains(content, forbidden) {
		t.Fatalf("TC-005-17 operating-model.md must not claim Direct executes a parallel dispatch operationally equivalent to Batch/Council (found %q) — direct.md always dispatches exactly one worker regardless of resolved topology", forbidden)
	}

	if !strings.Contains(content, "`direct.md`") || !strings.Contains(content, "classification-time property") {
		t.Fatalf("TC-005-17 operating-model.md must explicitly connect row 9's independence claim to direct.md's actual (classification-only, no-execution-effect) behavior")
	}

	rows := f09OperatingModelDecisionTable(t)
	row9 := rows[8]
	if !strings.Contains(row9[5], "classification only") {
		t.Fatalf("TC-005-17 row 9's resolved-topology cell = %q, want it to state the independence it demonstrates is classification-time only, not an operational parallel dispatch", row9[5])
	}
}

// ---------------------------------------------------------------------------
// TC-005-14..16 (test-plan.md #tc-005, "Executable-procedure sub-cases") —
// T-E38-F09-023. TC-005-01..13 above assert the two-axis rules exist in
// operating-model.md; these three assert the four executable procedures
// (coordinate.md, direct.md, batch.md, execute-wave.md) apply those rules
// rather than restating them.
// ---------------------------------------------------------------------------

// TestTC005_14CoordinateRoutesDeterministicallyByCoordinationLevel is
// TC-005-14: coordinate.md is the two-axis entry point and routes
// deterministically to direct.md (Direct), batch.md (Batch), and
// council.md (Council) per the selected coordination level — content
// assertion against coordinate.md's routing table/links.
func TestTC005_14CoordinateRoutesDeterministicallyByCoordinationLevel(t *testing.T) {
	content := readF09EmbeddedFile(t, "skills/shark-attack/workflows/coordinate.md")
	rows := f09ParseMarkdownTable(t, content, "| Coordination level |")

	want := map[string]string{
		"Direct":  "`direct.md`",
		"Batch":   "`batch.md`",
		"Council": "`council.md`",
	}
	got := map[string]string{}
	for _, row := range rows {
		if len(row) < 2 {
			t.Fatalf("TC-005-14 coordinate.md routing table row has %d cells, want 2: %+v", len(row), row)
		}
		level := strings.Trim(row[0], "`")
		got[level] = row[1]
	}
	for level, wantProcedure := range want {
		gotProcedure, ok := got[level]
		if !ok {
			t.Fatalf("TC-005-14 coordinate.md routing table has no row for coordination level %q; parsed rows: %+v", level, rows)
		}
		if gotProcedure != wantProcedure {
			t.Fatalf("TC-005-14 coordinate.md routes %q to %q, want %q", level, gotProcedure, wantProcedure)
		}
	}
	if len(got) != 3 {
		t.Fatalf("TC-005-14 coordinate.md routing table has %d distinct coordination levels, want exactly 3 (Direct/Batch/Council); parsed rows: %+v", len(got), rows)
	}
}

// TestTC005_15DirectSingleWorkerBatchWaveShapesLinkNotRestate is TC-005-15:
// direct.md documents single-worker dispatch with no topology-selection step
// (Direct implies one worker); batch.md/execute-wave.md document the
// parallel-with-ownership and parallel-with-isolation wave shapes and link
// to, rather than restate, operating-model.md's degradation rule.
func TestTC005_15DirectSingleWorkerBatchWaveShapesLinkNotRestate(t *testing.T) {
	direct := readF09EmbeddedFile(t, "skills/shark-attack/workflows/direct.md")
	if !strings.Contains(direct, "one worker") {
		t.Fatalf("TC-005-15 direct.md must document single-worker dispatch; content:\n%s", direct)
	}
	if !strings.Contains(direct, "no topology-selection step") && !strings.Contains(direct, "No topology axis is consulted") {
		t.Fatalf("TC-005-15 direct.md must state it performs no topology-selection step; content:\n%s", direct)
	}
	for _, forbidden := range []string{"Parallel with ownership", "Parallel with isolation"} {
		if strings.Contains(direct, forbidden) {
			t.Fatalf("TC-005-15 direct.md must not perform topology selection; found forbidden phrase %q", forbidden)
		}
	}

	// The canonical degradation-rule marker operating-model.md states
	// verbatim (D-007/AC-002). batch.md/execute-wave.md must apply the
	// rule's consequence in their own words, never paste this marker.
	const degradeRuleMarker = "MUST degrade to `Sequential`"
	operatingModel := f09ResolveOperatingModel(t)
	if !strings.Contains(operatingModel, degradeRuleMarker) {
		t.Fatalf("TC-005-15 fixture assumption broken: operating-model.md no longer states the canonical degradation-rule marker %q", degradeRuleMarker)
	}

	for _, relPath := range []string{
		"skills/shark-attack/workflows/batch.md",
		"skills/shark-attack/workflows/execute-wave.md",
	} {
		content := readF09EmbeddedFile(t, relPath)
		for _, wantWaveShape := range []string{"Parallel with ownership", "Parallel with isolation"} {
			if !strings.Contains(content, wantWaveShape) {
				t.Fatalf("TC-005-15 %s must document the %q wave shape; content:\n%s", relPath, wantWaveShape, content)
			}
		}
		if !strings.Contains(content, "context/operating-model.md") {
			t.Fatalf("TC-005-15 %s must link to context/operating-model.md rather than restate its degradation rule; content:\n%s", relPath, content)
		}
		if strings.Contains(content, degradeRuleMarker) {
			t.Fatalf("TC-005-15 %s must not restate operating-model.md's verbatim degradation-rule text (found %q); link to it instead", relPath, degradeRuleMarker)
		}
	}
}

// TestTC005_16BatchAndExecuteWaveAssertProducerConsumerOrderSurvivesIsolation
// is TC-005-16 (D-007 non-parallelization assertion): batch.md and
// execute-wave.md positively state that isolation evidence alone never
// authorizes running logically dependent (producer/consumer
// contract-ordered) work in parallel — grep both files for an explicit
// ordering-preserved statement; absence is a FAIL.
func TestTC005_16BatchAndExecuteWaveAssertProducerConsumerOrderSurvivesIsolation(t *testing.T) {
	cases := []struct {
		relPath string
		markers []string
	}{
		{
			relPath: "skills/shark-attack/workflows/batch.md",
			markers: []string{"reorder dependent writes", "never in the same wave as its producer"},
		},
		{
			relPath: "skills/shark-attack/workflows/execute-wave.md",
			markers: []string{"Producer/consumer order survives isolation", "not eligible for the same wave as its producer"},
		},
	}
	for _, tc := range cases {
		content := readF09EmbeddedFile(t, tc.relPath)
		for _, marker := range tc.markers {
			if !strings.Contains(content, marker) {
				t.Fatalf("TC-005-16 %s must positively state that isolation evidence never authorizes running logically dependent (producer/consumer contract-ordered) work in parallel; missing marker %q; content:\n%s", tc.relPath, marker, content)
			}
		}
	}
}

// f09PullByRoleRetiredVocabulary is the "sanctioned claim route → retire"
// subset of the 16 phrases T-E38-F09-009 adjudicated across
// internal/sharkdata/shark_attack_pull_test.go's TestTC107_TC003 (11
// pinned phrases) and TestTC110 (5 pinned phrases): the direct-claim
// mechanic itself, plus the fallback recommendations that only applied to
// the retired worker-owned direct-claim decision tree. The remaining 10 of
// the 16 are "authority description → keep": they describe the role/roster
// authority model or the read-only selector, both of which the sanctioned
// Rider re-entry path still relies on, so they remain valid, unconfined
// guidance in pull-by-role.md and needed no change to their pinned
// assertions in e38_f04_interactions_test.go, e38_f07_interactions_test.go,
// or shark_attack_pull_test.go's own TestTC107_TC003 "keep" entries.
func f09PullByRoleRetiredVocabulary() []string {
	return []string{
		"ClaimService.Claim",
		"missing product gates",
		"bootstrap or escalation",
		"explicit sequential fallback",
		"do not guess product decisions",
		"ordinary `/shark-rider run` routing",
	}
}

// f09PullByRoleHistoricalMarker is the heading that begins pull-by-role.md's
// clearly marked historical/compatibility-only section. Everything at or
// after this heading is compatibility-reference content; everything before
// it is live, sanctioned guidance.
const f09PullByRoleHistoricalMarker = "Historical reference: worker-owned child mode (compatibility only)"

// f09WalkEmbeddedSharkAttackMarkdown returns the sharkdata.ReadEmbedded-style
// relative paths of every .md file under the embedded skills/shark-attack/
// tree, for TC-007-02's corpus-wide scan.
func f09WalkEmbeddedSharkAttackMarkdown(t *testing.T) []string {
	t.Helper()
	efs, prefix := sharkdata.EmbeddedFS()
	root := prefix + "/skills/shark-attack"
	var paths []string
	err := fs.WalkDir(efs, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(p, prefix+"/"))
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded skills/shark-attack: %v", err)
	}
	return paths
}

// TestTC007_01PullByRoleFramesItselfAsCompatibilityReference is TC-007-01:
// pull-by-role.md (post-restructure) frames itself as a compatibility
// reference, not a sanctioned normal-path claim route. It reuses and extends
// the existing F07-owned assertions (don't delete them — see
// e38_f07_interactions_test.go's TestTC005) and additionally asserts the
// file's introductory framing explicitly labels itself historical/
// compatibility-only.
func TestTC007_01PullByRoleFramesItselfAsCompatibilityReference(t *testing.T) {
	content := readF09EmbeddedFile(t, "skills/shark-attack/workflows/pull-by-role.md")
	normalized := strings.Join(strings.Fields(content), " ")

	for _, want := range []string{
		"worker-owned child mode",
		"not `/shark-rider run`",
		"Do not hand this child session to `/shark-rider run`.",
	} {
		if !strings.Contains(normalized, want) {
			t.Errorf("pull-by-role.md must retain existing F07-pinned phrase %q", want)
		}
	}

	for _, want := range []string{
		"historical / compatibility reference",
		"no longer describes a sanctioned normal-path claim",
		"Sanctioned path: Rider re-entry",
		f09PullByRoleHistoricalMarker,
	} {
		if !strings.Contains(normalized, want) {
			t.Errorf("pull-by-role.md introductory/section framing omits %q", want)
		}
	}
}

// TestTC007_02NoOtherCorpusFileSanctionsPullByRoleAsNormalPath is TC-007-02:
// no other file in the rendered skills/shark-attack/ corpus references
// pull-by-role.md as the normal claim path. Every workflow file that cites
// pull-by-role.md by name is checked for a still-sanctioning cross-reference
// (any of the retired direct-claim/fallback phrases).
func TestTC007_02NoOtherCorpusFileSanctionsPullByRoleAsNormalPath(t *testing.T) {
	forbidden := f09PullByRoleRetiredVocabulary()
	for _, rel := range f09WalkEmbeddedSharkAttackMarkdown(t) {
		if rel == "skills/shark-attack/workflows/pull-by-role.md" {
			continue
		}
		content := readF09EmbeddedFile(t, rel)
		normalized := strings.Join(strings.Fields(content), " ")
		if !strings.Contains(normalized, "pull-by-role.md") {
			continue // does not cross-reference pull-by-role.md at all
		}
		lower := strings.ToLower(normalized)
		for _, phrase := range forbidden {
			if strings.Contains(lower, strings.ToLower(phrase)) {
				t.Errorf("%s cites pull-by-role.md and still sanctions retired phrase %q as a normal path", rel, phrase)
			}
		}
	}
}

// TestTC007_03PullByRoleRetiredVocabularyConfinedToHistoricalSection is
// TC-007-03 (added per codex red-team CONCERN): none of the retired
// forbidden-vocabulary phrases, nor any close paraphrase, survives anywhere
// in pull-by-role.md outside the clearly marked historical/compatibility
// section. The list is derived directly from f09PullByRoleRetiredVocabulary
// (this task's own retire-list), not authored independently, so this gate
// and the pinning tests it is derived from cannot silently diverge.
func TestTC007_03PullByRoleRetiredVocabularyConfinedToHistoricalSection(t *testing.T) {
	content := readF09EmbeddedFile(t, "skills/shark-attack/workflows/pull-by-role.md")
	normalized := strings.Join(strings.Fields(content), " ")

	idx := strings.Index(normalized, f09PullByRoleHistoricalMarker)
	if idx < 0 {
		t.Fatalf("pull-by-role.md must contain a clearly marked historical/compatibility section (%q)", f09PullByRoleHistoricalMarker)
	}
	before, after := normalized[:idx], normalized[idx:]

	for _, phrase := range f09PullByRoleRetiredVocabulary() {
		if strings.Contains(before, phrase) {
			t.Errorf("retired phrase %q survives as live guidance outside the historical/compatibility section", phrase)
		}
		if !strings.Contains(after, phrase) {
			t.Errorf("retired phrase %q must still survive inside the historical/compatibility section (compatibility reference, not deletion)", phrase)
		}
	}
}

// ---------------------------------------------------------------------------
// TC-007-04/05 (UAT round-1 rework, finding UAT-001, HIGH, AC-018/REQ-F-017
// violation): TC-007-02 above only rejects a small literal phrase list in
// files that mention pull-by-role.md by name. It missed the actual defect —
// SKILL.md's router pointed a reader at worker-owned child mode as something
// to "use ... when an existing coordinator explicitly owns a delegated
// child's separate lease lifecycle", and context/worker-ownership.md then
// authorized a worker to "Claim, heartbeat and release" its own child
// lease — neither file mentions pull-by-role.md by name, so TC-007-02 never
// scanned them. TC-007-04 sweeps the COMPLETE rendered skills/shark-attack/**
// corpus (not just pull-by-role.md cross-references) for live worker-owned
// child claim/heartbeat/release authorization, honoring the same clearly
// marked historical/compatibility section pull-by-role.md's own retirement
// already established (D-010's one-boundary-marker pattern, reused rather
// than inventing a second one). TC-007-05 is the companion positive control:
// the retired phrase must survive inside worker-ownership.md's historical
// section as a compatibility reference, not vanish via silent deletion —
// the same reference-not-deletion discipline TC-007-03 already proves for
// pull-by-role.md.
// ---------------------------------------------------------------------------

// f09WorkerChildClaimHistoricalMarker is the heading that begins a clearly
// marked historical/compatibility-only section documenting the retired
// worker-owned-child-mode direct-claim procedure. Both pull-by-role.md and
// context/worker-ownership.md use this exact heading text; content at or
// after it in either file is compatibility-reference material, not live
// guidance. Reuses f09PullByRoleHistoricalMarker's string value rather than
// declaring a second literal, so the two markers cannot silently diverge.
const f09WorkerChildClaimHistoricalMarker = f09PullByRoleHistoricalMarker

// f09LiveChildClaimAuthorizationVocabulary is UAT-001's (round 1) defect-class
// phrase set: literal instructions that authorize a worker to claim,
// heartbeat, or release its own child lease outside Rider re-entry. AC-018/
// REQ-F-017 require Rider re-entry to be the only sanctioned claim path in
// the rendered corpus; neither phrase may survive as *live* guidance
// anywhere in skills/shark-attack/**, even in a file that never cites
// pull-by-role.md by name.
func f09LiveChildClaimAuthorizationVocabulary() []string {
	return []string{
		// context/worker-ownership.md's retired direct-claim authorization.
		"Claim, heartbeat and release only its own child lease",
		// SKILL.md's retired live pointer into worker-owned child mode.
		"use it only when an existing coordinator explicitly owns a delegated child's separate lease lifecycle",
	}
}

// TestTC007_04NoLiveChildClaimAuthorizationAnywhereInRenderedCorpus is
// TC-007-04: sweeps every .md file under the embedded skills/shark-attack/
// tree — not just files referencing pull-by-role.md by name — for live
// authorization of a worker to claim, heartbeat, or release its own child
// lease. A file may still document the retired procedure for compatibility,
// but only inside a clearly marked historical/compatibility section
// (f09WorkerChildClaimHistoricalMarker); content before that marker, or the
// whole file when no such marker exists at all, is live guidance and must
// not contain either forbidden phrase.
func TestTC007_04NoLiveChildClaimAuthorizationAnywhereInRenderedCorpus(t *testing.T) {
	forbidden := f09LiveChildClaimAuthorizationVocabulary()
	for _, rel := range f09WalkEmbeddedSharkAttackMarkdown(t) {
		content := readF09EmbeddedFile(t, rel)
		normalized := strings.Join(strings.Fields(content), " ")

		live := normalized
		if idx := strings.Index(normalized, f09WorkerChildClaimHistoricalMarker); idx >= 0 {
			live = normalized[:idx]
		}

		for _, phrase := range forbidden {
			if strings.Contains(live, phrase) {
				t.Errorf("%s authorizes a live worker-owned child claim/heartbeat/release outside any historical/compatibility section: contains %q", rel, phrase)
			}
		}
	}
}

// TestTC007_05WorkerOwnershipRetiredClaimPhrasePreservedAsHistoricalReference
// is TC-007-05, the positive-control companion to TC-007-04: the retired
// direct-claim phrase must still exist inside worker-ownership.md's own
// historical/compatibility section rather than being silently deleted,
// matching the retirement-not-deletion pattern TC-007-03 already proves for
// pull-by-role.md.
func TestTC007_05WorkerOwnershipRetiredClaimPhrasePreservedAsHistoricalReference(t *testing.T) {
	content := readF09EmbeddedFile(t, "skills/shark-attack/context/worker-ownership.md")
	normalized := strings.Join(strings.Fields(content), " ")

	idx := strings.Index(normalized, f09WorkerChildClaimHistoricalMarker)
	if idx < 0 {
		t.Fatalf("worker-ownership.md must contain a clearly marked historical/compatibility section (%q)", f09WorkerChildClaimHistoricalMarker)
	}
	after := normalized[idx:]
	want := "Claim, heartbeat and release only its own child lease"
	if !strings.Contains(after, want) {
		t.Errorf("worker-ownership.md must retain %q inside its historical/compatibility section", want)
	}
}

// ---------------------------------------------------------------------------
// TC-003-08/08b, TC-003-09/09b (test-plan.md #tc-003) — T-E38-F09-010.
//
// providers/codex.md and providers/claude-code.md are content-only per the
// TC-003-08..09 Caller-Path Contract row: read via readF09EmbeddedFile
// (sharkdata.ReadEmbedded), the same seam TC-003-01..03 already use for
// worker-control-schema.yaml and e38_f07_interactions_test.go's
// readF07EmbeddedFile use for their own trees. The thing actually under
// test is the *check*, not a running Go type: a naive "file exists" check
// would pass a provider reference that claims a supported operation with no
// cited evidence, so these tests grep for an evidence-citation marker
// (`**Evidence:**`) adjacent to every claimed-supported operation, and each
// TC-003-08/09 test also proves that check has teeth via a constructed
// bad-fixture (test-plan.md's Negative Cases paragraph, TC-003-08b's
// inverse) — independent of whatever the shipped file's own content says.
// ---------------------------------------------------------------------------

// f09ProviderOp is one operation bullet parsed out of a provider capability
// reference's "## Supported Operations" / "## Unsupported Operations"
// section. Bullets are single lines of the shape
// `- **<Name>** — <prose> **Evidence:** <citation>`; Evidence is "" when the
// marker itself is absent from the bullet's line.
type f09ProviderOp struct {
	Name     string
	Evidence string
}

var f09ProviderOpBulletPattern = regexp.MustCompile(`(?m)^- \*\*([^*]+)\*\*(.*)$`)
var f09ProviderOpEvidencePattern = regexp.MustCompile(`\*\*Evidence:\*\*\s*(.+)$`)

// f09ParseProviderReferenceSection extracts every operation bullet from the
// named "## <heading>" section of a provider capability reference. A
// section runs from its heading to the next "## " heading (or EOF).
func f09ParseProviderReferenceSection(t *testing.T, content, heading string) []f09ProviderOp {
	t.Helper()
	sectionPattern := regexp.MustCompile(`(?ms)^## ` + regexp.QuoteMeta(heading) + `\s*\n(.*?)(\n## |\z)`)
	match := sectionPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("provider reference missing a %q section", heading)
	}
	body := match[1]

	bulletMatches := f09ProviderOpBulletPattern.FindAllStringSubmatch(body, -1)
	if len(bulletMatches) == 0 {
		t.Fatalf("provider reference %q section declares no operation bullets", heading)
	}

	ops := make([]f09ProviderOp, 0, len(bulletMatches))
	for _, m := range bulletMatches {
		op := f09ProviderOp{Name: strings.TrimSpace(m[1])}
		if evMatch := f09ProviderOpEvidencePattern.FindStringSubmatch(m[2]); evMatch != nil {
			op.Evidence = strings.TrimSpace(evMatch[1])
		}
		ops = append(ops, op)
	}
	return ops
}

// f09MissingEvidenceOps returns the names of every op whose Evidence field
// is empty or is the explicit "none captured" admission — i.e. every op
// with zero captured evidence backing its claim.
func f09MissingEvidenceOps(ops []f09ProviderOp) []string {
	var names []string
	for _, op := range ops {
		if op.Evidence == "" || strings.HasPrefix(strings.ToLower(op.Evidence), "none captured") {
			names = append(names, op.Name)
		}
	}
	return names
}

// f09AssertProviderReferenceSupportedOpsCiteEvidence is TC-003-08/09's
// shared body: every "## Supported Operations" bullet must cite non-empty,
// non-"none captured" evidence, a "## Sequential Fallback" section must
// exist, and — the negative case — stripping one real Evidence marker must
// make the same check fail, proving the check does not just rubber-stamp
// whatever the file happens to contain.
func f09AssertProviderReferenceSupportedOpsCiteEvidence(t *testing.T, tcID, relPath string) {
	t.Helper()
	content := readF09EmbeddedFile(t, relPath)

	fallbackPattern := regexp.MustCompile(`(?ms)^## Sequential Fallback\s*\n(.*?)(\n## |\z)`)
	fallbackMatch := fallbackPattern.FindStringSubmatch(content)
	if fallbackMatch == nil || len(strings.TrimSpace(fallbackMatch[1])) < 40 {
		t.Fatalf("%s %s \"## Sequential Fallback\" section is missing or too thin to describe an actual fallback", tcID, relPath)
	}

	supported := f09ParseProviderReferenceSection(t, content, "Supported Operations")
	if violations := f09MissingEvidenceOps(supported); len(violations) > 0 {
		t.Fatalf("%s %s Supported Operations missing an evidence citation for: %v", tcID, relPath, violations)
	}

	// Negative case (test-plan.md TC-003 Negative Cases paragraph): prove
	// the check itself has teeth by stripping the first bullet's Evidence
	// marker and confirming the same parse-and-check catches it.
	target := supported[0]
	badFixture := strings.Replace(content, "**Evidence:** "+target.Evidence, "", 1)
	if badFixture == content {
		t.Fatalf("%s negative-case fixture construction did not remove %s's Evidence marker — cannot prove the checker has teeth", tcID, target.Name)
	}
	badOps := f09ParseProviderReferenceSection(t, badFixture, "Supported Operations")
	if violations := f09MissingEvidenceOps(badOps); len(violations) == 0 {
		t.Fatalf("%s checker failed to flag %s's Supported Operations bullet once its Evidence marker was stripped — the check does not actually verify evidence citations", tcID, target.Name)
	}
}

// f09AssertProviderReferenceUnsupportedOpCorrectlyMarked is TC-003-08b/09b's
// shared body: the file must declare at least one Unsupported Operations
// entry with zero captured evidence (proving the recorder records what it
// could not verify rather than silently omitting it), and that entry's name
// must not simultaneously appear, supported, in Supported Operations.
func f09AssertProviderReferenceUnsupportedOpCorrectlyMarked(t *testing.T, tcID, relPath string) {
	t.Helper()
	content := readF09EmbeddedFile(t, relPath)

	unsupported := f09ParseProviderReferenceSection(t, content, "Unsupported Operations")
	noEvidence := f09MissingEvidenceOps(unsupported)
	if len(noEvidence) == 0 {
		t.Fatalf("%s %s declares no Unsupported Operations entry with zero captured evidence — an op the recorder could not verify must be listed here, not silently omitted", tcID, relPath)
	}

	supported := f09ParseProviderReferenceSection(t, content, "Supported Operations")
	supportedNames := make(map[string]struct{}, len(supported))
	for _, op := range supported {
		supportedNames[strings.ToLower(op.Name)] = struct{}{}
	}
	for _, name := range noEvidence {
		if _, dup := supportedNames[strings.ToLower(name)]; dup {
			t.Fatalf("%s operation %q has no captured evidence yet also appears in Supported Operations — conflicting classification", tcID, name)
		}
	}
}

// TestTC003_08CodexProviderReferenceDeclaresSupportedOpsWithEvidence is
// TC-003-08: providers/codex.md declares supported ops, unsupported ops, and
// the sequential fallback, and cites an installed-host evidence marker for
// every claimed-supported operation.
func TestTC003_08CodexProviderReferenceDeclaresSupportedOpsWithEvidence(t *testing.T) {
	f09AssertProviderReferenceSupportedOpsCiteEvidence(t, "TC-003-08", "skills/shark-attack/providers/codex.md")
}

// TestTC003_08bCodexProviderReferenceUnsupportedOpCorrectlyMarked is
// TC-003-08b (added per codex red-team CONCERN): providers/codex.md includes
// at least one operation with no captured evidence, and that operation is
// correctly listed under Unsupported Operations rather than silently
// omitted.
func TestTC003_08bCodexProviderReferenceUnsupportedOpCorrectlyMarked(t *testing.T) {
	f09AssertProviderReferenceUnsupportedOpCorrectlyMarked(t, "TC-003-08b", "skills/shark-attack/providers/codex.md")
}

// TestTC003_08cCodexLiveFollowUpFallsBackToReplacement pins the distinction
// between Codex's post-exit session resume and an unavailable live follow-up.
func TestTC003_08cCodexLiveFollowUpFallsBackToReplacement(t *testing.T) {
	content := readF09EmbeddedFile(t, "skills/shark-attack/providers/codex.md")
	unsupported := f09ParseProviderReferenceSection(t, content, "Unsupported Operations")
	for _, op := range unsupported {
		if op.Name == "Live follow-up (same still-running worker)" && strings.Contains(op.Evidence, "none captured") && strings.Contains(content, "bounded-replacement fallback") {
			return
		}
	}
	t.Fatal("TC-003-08c Codex must classify live same-worker follow-up as unsupported and require bounded replacement")
}

// TestTC003_09ClaudeCodeProviderReferenceDeclaresSupportedOpsWithEvidence is
// TC-003-09: same as TC-003-08 for providers/claude-code.md.
func TestTC003_09ClaudeCodeProviderReferenceDeclaresSupportedOpsWithEvidence(t *testing.T) {
	f09AssertProviderReferenceSupportedOpsCiteEvidence(t, "TC-003-09", "skills/shark-attack/providers/claude-code.md")
}

// TestTC003_09bClaudeCodeProviderReferenceUnsupportedOpCorrectlyMarked is
// TC-003-09b: same as TC-003-08b for providers/claude-code.md.
func TestTC003_09bClaudeCodeProviderReferenceUnsupportedOpCorrectlyMarked(t *testing.T) {
	f09AssertProviderReferenceUnsupportedOpCorrectlyMarked(t, "TC-003-09b", "skills/shark-attack/providers/claude-code.md")
}

// f09SkillRouterDegradationRuleMarker is the same canonical degradation-rule
// marker TestTC005_15 already pins on operating-model.md (D-007/AC-002):
// "`Sequential` is the default topology... MUST degrade to `Sequential`" is
// the sequential-default rule test-plan.md's TC-003-10 names as its example
// of a rule that may live in exactly one file. Reusing the established
// constant, rather than inventing a second marker string, keeps both tests
// pinned to the same sentence if it ever moves.
const f09SkillRouterDegradationRuleMarker = "MUST degrade to `Sequential`"

// f09ResolveSkillRouter resolves SKILL.md through the real embedded tree,
// the declared read convention for skills/shark-attack/** content.
func f09ResolveSkillRouter(t *testing.T) string {
	t.Helper()
	return readF09EmbeddedFile(t, "skills/shark-attack/SKILL.md")
}

// TestTC003_10SkillRouterLinksToOperatingModelAuthorityAndModeWorkflows is
// TC-003-10 (test-plan.md #tc-003, AC-017): T-E38-F09-024's restructured
// SKILL.md must link to context/operating-model.md, context/authority.md,
// and each of workflows/{coordinate,direct,batch,council,route-question,
// execute-wave,resume}.md — the full set of dedicated per-mode files
// REQ-F-015 requires the router to reduce to links for, rather than restated
// rule text. The router's own degradation-rule-adjacent prose must not
// restate operating-model.md's canonical degradation-rule marker verbatim;
// TC-003-11 (T-E38-F09-025's scope) does the full corpus-wide near-duplicate
// scan — this sub-case only proves the one rule test-plan.md names by
// example is not doubly stated.
func TestTC003_10SkillRouterLinksToOperatingModelAuthorityAndModeWorkflows(t *testing.T) {
	skill := f09ResolveSkillRouter(t)

	for _, wantLink := range []string{
		"context/operating-model.md",
		"context/authority.md",
		"workflows/coordinate.md",
		"workflows/direct.md",
		"workflows/batch.md",
		"workflows/council.md",
		"workflows/route-question.md",
		"workflows/execute-wave.md",
		"workflows/resume.md",
	} {
		if !strings.Contains(skill, wantLink) {
			t.Errorf("TC-003-10 SKILL.md must link %q so a fresh agent can reach it; content:\n%s", wantLink, skill)
		}
	}

	operatingModel := f09ResolveOperatingModel(t)
	if !strings.Contains(operatingModel, f09SkillRouterDegradationRuleMarker) {
		t.Fatalf("TC-003-10 fixture assumption broken: operating-model.md no longer states the canonical degradation-rule marker %q", f09SkillRouterDegradationRuleMarker)
	}
	if strings.Contains(skill, f09SkillRouterDegradationRuleMarker) {
		t.Fatalf("TC-003-10 SKILL.md must not restate operating-model.md's verbatim degradation-rule marker %q; link to context/operating-model.md instead", f09SkillRouterDegradationRuleMarker)
	}
}

// ---------------------------------------------------------------------------
// TC-003-11 (test-plan.md #tc-003, AC-017's no-duplication half; T-E38-F09-025's
// scope) — the corpus-wide rule-inventory near-duplication scan. TC-003-10
// above spot-checks the one degradation-rule sentence test-plan.md names by
// example; this section mechanically proves the general claim across every
// rule in tests/contracts/testdata/e38_f09_rule_inventory.yaml and every file
// in the restructured skills/shark-attack/** corpus, per the codex red-team
// BLOCKER that a single spot-checked sentence cannot establish "no rule is
// duplicated" tree-wide.

// f09NearDupThreshold is TC-003-11's ">80% token overlap" near-duplicate
// cutoff (test-plan.md TC-003-11, task spec's Brownfield Context: "a cheap
// near-duplicate heuristic (>80% token overlap), not full NLP").
const f09NearDupThreshold = 0.8

// f09RuleInventoryEntry is one row of the YAML fixture: a rule's canonical
// source, its verbatim canonical text, and the files allowed to reference it
// via a link/pointer (informational — TC-003-10/12 already prove SKILL.md's
// specific links resolve; this fixture does not re-check link existence).
// canonical_file is empty for a rule enforced entirely outside
// skills/shark-attack/** (e.g. the parity-gate row) — such a rule must
// appear in zero corpus files rather than exactly one.
type f09RuleInventoryEntry struct {
	RuleID                 string   `yaml:"rule_id"`
	CanonicalFile          string   `yaml:"canonical_file"`
	CanonicalText          string   `yaml:"canonical_text"`
	AllowedCrossReferences []string `yaml:"allowed_cross_references"`
}

type f09RuleInventoryFixture struct {
	Rules []f09RuleInventoryEntry `yaml:"rules"`
}

// f09LoadRuleInventory parses the maintained rule-inventory fixture. It is a
// test-only artifact under tests/contracts/testdata/ — outside
// skills/shark-attack/** and therefore outside REQ-F-016's parity gate —
// read with readF09RepositoryFile per this file's declared read convention.
func f09LoadRuleInventory(t *testing.T) []f09RuleInventoryEntry {
	t.Helper()
	raw := readF09RepositoryFile(t, "tests/contracts/testdata/e38_f09_rule_inventory.yaml")
	var fixture f09RuleInventoryFixture
	if err := yaml.Unmarshal([]byte(raw), &fixture); err != nil {
		t.Fatalf("parse rule inventory fixture: %v", err)
	}
	if len(fixture.Rules) == 0 {
		t.Fatal("rule inventory fixture must declare at least one rule")
	}
	for _, rule := range fixture.Rules {
		if strings.TrimSpace(rule.RuleID) == "" {
			t.Fatalf("rule inventory fixture has an entry with an empty rule_id")
		}
		if strings.TrimSpace(rule.CanonicalText) == "" {
			t.Fatalf("rule %s: canonical_text must not be empty", rule.RuleID)
		}
	}
	return fixture.Rules
}

// f09WalkEmbeddedSharkAttackAllFiles returns the sharkdata.ReadEmbedded-style
// relative paths of every file (not just .md) under the embedded
// skills/shark-attack/ tree. TC-003-11 parses "every file under
// skills/shark-attack/**", which includes context/roster-schema.yaml and
// context/worker-control-schema.yaml — f09WalkEmbeddedSharkAttackMarkdown
// (TC-007-02's helper) is intentionally .md-only and is not reused here.
func f09WalkEmbeddedSharkAttackAllFiles(t *testing.T) []string {
	t.Helper()
	efs, prefix := sharkdata.EmbeddedFS()
	root := prefix + "/skills/shark-attack"
	var paths []string
	err := fs.WalkDir(efs, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(p, prefix+"/"))
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded skills/shark-attack: %v", err)
	}
	return paths
}

// f09EmbeddedSharkAttackCorpus reads every file the walk above finds, keyed
// by its sharkdata.ReadEmbedded relPath, so each rule needs only one corpus
// build rather than re-reading the tree per rule.
func f09EmbeddedSharkAttackCorpus(t *testing.T) map[string]string {
	t.Helper()
	corpus := make(map[string]string)
	for _, rel := range f09WalkEmbeddedSharkAttackAllFiles(t) {
		corpus[rel] = readF09EmbeddedFile(t, rel)
	}
	return corpus
}

// f09NormalizeWhitespace collapses all whitespace runs to single spaces so a
// verbatim-substring check is insensitive to line-wrap differences between
// the fixture's YAML-folded canonical_text and the source file's own line
// breaks.
func f09NormalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// f09TokenPattern extracts word tokens for the near-duplicate heuristic:
// lowercase alphanumeric runs, dropping Markdown punctuation (backticks,
// asterisks, pipes, em dashes) rather than treating it as meaningful
// content.
var f09TokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// f09Tokenize returns s's word tokens in order (duplicates kept) — the
// "cheap... not full NLP" heuristic the task spec's Brownfield Context calls
// for, not a stemmed/stopword-filtered NLP pipeline.
func f09Tokenize(s string) []string {
	return f09TokenPattern.FindAllString(strings.ToLower(s), -1)
}

func f09TokenSet(tokens []string) map[string]struct{} {
	set := make(map[string]struct{}, len(tokens))
	for _, tok := range tokens {
		set[tok] = struct{}{}
	}
	return set
}

// f09Jaccard is the intersection-over-union overlap ratio between two token
// sets. Jaccard (rather than a canonical-relative containment ratio) is
// deliberate: a containment ratio inflates toward 1.0 as the *candidate*
// paragraph grows longer and happens to include the canonical rule's shared
// domain vocabulary, which these tightly cross-referenced files have in
// common by design — that inflation would produce false positives on
// deliberate link-not-restate pointers. Jaccard penalizes a candidate window
// that adds unrelated tokens, not just one that omits canonical ones.
func f09Jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for tok := range a {
		if _, ok := b[tok]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// f09MaxWindowJaccard slides a window the length of canonicalTokens across
// candidateTokens (step 1) and returns the maximum Jaccard overlap observed.
// A fixed-size sliding window — rather than comparing whole blank-line
// paragraphs — is what catches a restated rule regardless of where the
// candidate file's own paragraph or sentence boundaries fall.
func f09MaxWindowJaccard(canonicalTokens, candidateTokens []string) float64 {
	canonicalSet := f09TokenSet(canonicalTokens)
	windowSize := len(canonicalTokens)
	if windowSize == 0 {
		return 0
	}
	if len(candidateTokens) <= windowSize {
		return f09Jaccard(canonicalSet, f09TokenSet(candidateTokens))
	}
	max := 0.0
	for i := 0; i+windowSize <= len(candidateTokens); i++ {
		ratio := f09Jaccard(canonicalSet, f09TokenSet(candidateTokens[i:i+windowSize]))
		if ratio > max {
			max = ratio
		}
	}
	return max
}

// f09AssertRuleStatedInExactlyOneFile is TC-003-11 sub-case (a): the rule's
// canonical text appears verbatim (whitespace-normalized) in exactly one
// corpus file. For a rule whose fixture row declares an empty
// canonical_file (enforced entirely outside skills/shark-attack/**, e.g.
// parity-gate), the assertion inverts: the text must appear in zero files.
func f09AssertRuleStatedInExactlyOneFile(t *testing.T, rule f09RuleInventoryEntry, corpus map[string]string) {
	t.Helper()
	normalizedCanonical := f09NormalizeWhitespace(rule.CanonicalText)

	var occurrences []string
	for rel, content := range corpus {
		if strings.Contains(f09NormalizeWhitespace(content), normalizedCanonical) {
			occurrences = append(occurrences, rel)
		}
	}
	sort.Strings(occurrences)

	if rule.CanonicalFile == "" {
		if len(occurrences) != 0 {
			t.Errorf("rule %s is declared enforced outside skills/shark-attack/** (empty canonical_file), but its text appears verbatim in %v — it must appear in zero skill-tree files", rule.RuleID, occurrences)
		}
		return
	}

	if len(occurrences) == 0 {
		t.Fatalf("rule %s: canonical_text not found verbatim in declared canonical_file %s — fixture assumption broken (canonical_file's prose moved; update the fixture's canonical_text to match)", rule.RuleID, rule.CanonicalFile)
	}
	if len(occurrences) > 1 {
		t.Errorf("rule %s canonical text appears verbatim in %d files (%v); it must appear in exactly one — its declared canonical source %s", rule.RuleID, len(occurrences), occurrences, rule.CanonicalFile)
	}
	if occurrences[0] != rule.CanonicalFile {
		t.Errorf("rule %s canonical text appears verbatim in %s, not its declared canonical_file %s", rule.RuleID, occurrences[0], rule.CanonicalFile)
	}
}

// f09AssertRuleNotNearDuplicatedElsewhere is TC-003-11 sub-case (b): no
// corpus file other than the rule's own declared canonical_file restates it
// as a near-duplicate paraphrase (>f09NearDupThreshold token-window Jaccard
// overlap). Restating within canonical_file itself is not cross-file
// duplication and is out of AC-017's "no rule stated in two files" scope.
func f09AssertRuleNotNearDuplicatedElsewhere(t *testing.T, rule f09RuleInventoryEntry, corpus map[string]string) {
	t.Helper()
	canonicalTokens := f09Tokenize(rule.CanonicalText)
	if len(canonicalTokens) == 0 {
		t.Fatalf("rule %s: canonical_text tokenizes to zero words", rule.RuleID)
	}

	rels := make([]string, 0, len(corpus))
	for rel := range corpus {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	for _, rel := range rels {
		if rel == rule.CanonicalFile {
			continue
		}
		ratio := f09MaxWindowJaccard(canonicalTokens, f09Tokenize(corpus[rel]))
		t.Logf("TC-003-11 rule %s vs %s: max token-window Jaccard overlap = %.3f", rule.RuleID, rel, ratio)
		if ratio > f09NearDupThreshold {
			t.Errorf("rule %s is near-duplicated (%.0f%% token overlap, threshold %.0f%%) in %s — that file must link to %s instead of restating the rule", rule.RuleID, ratio*100, f09NearDupThreshold*100, rel, rule.CanonicalFile)
		}
	}
}

// f09AssertDetectorFiresOnRuleOwnCanonicalFile is the identity-case control
// for a rule with a real canonical_file: the near-duplicate detector, run
// against the rule's *own* canonical file content (not a hand-written Go
// string literal, unlike TestTC003_11NearDuplicateDetectorFlagsParaphraseNot
// UnrelatedText's control), must itself score above f09NearDupThreshold.
// Without this, a YAML-folding mishap that mangled canonical_text into
// something that still happens to satisfy the verbatim substring check
// (e.g. truncated at a folding boundary) could silently make every
// cross-file comparison for that rule meaningless, while the hand-written
// paraphrase control above kept passing because it never touches the real
// fixture text against the real corpus.
func f09AssertDetectorFiresOnRuleOwnCanonicalFile(t *testing.T, rule f09RuleInventoryEntry, corpus map[string]string) {
	t.Helper()
	if rule.CanonicalFile == "" {
		return // no real corpus file to self-match against (e.g. parity-gate)
	}
	content, ok := corpus[rule.CanonicalFile]
	if !ok {
		t.Fatalf("rule %s: declared canonical_file %s is not in the scanned corpus", rule.RuleID, rule.CanonicalFile)
	}
	ratio := f09MaxWindowJaccard(f09Tokenize(rule.CanonicalText), f09Tokenize(content))
	if ratio <= f09NearDupThreshold {
		t.Fatalf("rule %s: detector measured only %.3f overlap between canonical_text and its own declared canonical_file %s (want > %.2f) — the fixture's canonical_text has likely drifted from the real file content, making the cross-file scan for this rule meaningless", rule.RuleID, ratio, rule.CanonicalFile, f09NearDupThreshold)
	}
}

// TestTC003_11RuleInventoryNoDuplicationAcrossRestructuredTree is TC-003-11:
// for every rule the maintained inventory fixture names, its canonical text
// appears verbatim in exactly one corpus file (or zero, for a rule enforced
// outside the skill tree) and no other file near-duplicates it as a
// paraphrase. One subtest per rule_id so a single rule's regression does not
// mask another's pass/fail in the summary.
func TestTC003_11RuleInventoryNoDuplicationAcrossRestructuredTree(t *testing.T) {
	rules := f09LoadRuleInventory(t)
	corpus := f09EmbeddedSharkAttackCorpus(t)

	for _, rule := range rules {
		rule := rule
		t.Run(rule.RuleID, func(t *testing.T) {
			f09AssertRuleStatedInExactlyOneFile(t, rule, corpus)
			f09AssertDetectorFiresOnRuleOwnCanonicalFile(t, rule, corpus)
			f09AssertRuleNotNearDuplicatedElsewhere(t, rule, corpus)
		})
	}
}

// TestTC003_11SkillRouterCarriesZeroRuleStatementsFromInventory is TC-003-11
// sub-case (c) (this file's own Notes for Agent: "SKILL.md must contain zero
// rule statements from the inventory — router prose and links only"),
// asserted explicitly rather than left as an implicit corollary of the
// corpus-wide scan above, so a SKILL.md regression is named by file rather
// than surfacing only as "some file duplicated this rule."
func TestTC003_11SkillRouterCarriesZeroRuleStatementsFromInventory(t *testing.T) {
	rules := f09LoadRuleInventory(t)
	skill := f09ResolveSkillRouter(t)
	normalizedSkill := f09NormalizeWhitespace(skill)
	skillTokens := f09Tokenize(skill)

	for _, rule := range rules {
		normalizedCanonical := f09NormalizeWhitespace(rule.CanonicalText)
		if strings.Contains(normalizedSkill, normalizedCanonical) {
			t.Errorf("SKILL.md restates rule %s verbatim; SKILL.md must carry router prose and links only, per REQ-F-015", rule.RuleID)
		}
		ratio := f09MaxWindowJaccard(f09Tokenize(rule.CanonicalText), skillTokens)
		t.Logf("TC-003-11(c) rule %s vs SKILL.md: max token-window Jaccard overlap = %.3f", rule.RuleID, ratio)
		if ratio > f09NearDupThreshold {
			t.Errorf("SKILL.md near-duplicates rule %s (%.0f%% token overlap); SKILL.md must carry router prose and links only, not a restated rule statement", rule.RuleID, ratio*100)
		}
	}
}

// TestTC003_11NearDuplicateDetectorFlagsParaphraseNotUnrelatedText is
// TC-003-11's required positive/negative control (test-plan.md's
// counter-factual column: "A router that duplicated operating-model rules
// inline ... would still pass a naive single-sentence keyword-presence
// check; the inventory-driven near-duplicate scan ... is what a one-sentence
// spot check cannot catch"). Without this control, a broken tokenizer or a
// threshold set past 1.0 would make the corpus-wide scan above permanently
// green regardless of the tree's actual content. It is a pure in-memory
// fixture — no skill content changes, no new files.
func TestTC003_11NearDuplicateDetectorFlagsParaphraseNotUnrelatedText(t *testing.T) {
	canonical := "`Sequential` is the default topology. A parallel topology " +
		"(`Parallel with ownership` or `Parallel with isolation`) requires " +
		"captured, recorded evidence matching that specific topology. " +
		"Whenever the required evidence cannot be produced, the request " +
		"**MUST degrade to `Sequential`** — regardless of the requested " +
		"coordination level."
	// Same rule, reworded: light synonym/order changes, same underlying
	// vocabulary and claim — the paraphrase class TC-003-11 exists to catch.
	paraphrase := "Sequential is the default execution topology. A parallel " +
		"topology, whether parallel with ownership or parallel with " +
		"isolation, requires captured and recorded evidence matching that " +
		"specific topology, and whenever the required evidence cannot be " +
		"produced the request must degrade back to sequential regardless of " +
		"the requested coordination level."
	// No shared rule content at all — a different rule's subject matter.
	unrelated := "The roster schema declares an optional capability_profile " +
		"field with allowed values fast, balanced, or deep, and retains " +
		"model_tier as a deprecated preference that never selects work, " +
		"overrides workflow metadata, or maps to a provider or model."

	canonicalTokens := f09Tokenize(canonical)

	paraphraseRatio := f09MaxWindowJaccard(canonicalTokens, f09Tokenize(paraphrase))
	if paraphraseRatio <= f09NearDupThreshold {
		t.Errorf("near-duplicate detector failed to flag a reworded paraphrase of the same rule: measured overlap %.3f, want > %.2f (detector cannot be trusted to catch a real restated rule)", paraphraseRatio, f09NearDupThreshold)
	}

	unrelatedRatio := f09MaxWindowJaccard(canonicalTokens, f09Tokenize(unrelated))
	if unrelatedRatio > f09NearDupThreshold {
		t.Errorf("near-duplicate detector false-flagged unrelated text sharing no rule content: measured overlap %.3f, want <= %.2f (detector would false-positive on legitimate distinct prose)", unrelatedRatio, f09NearDupThreshold)
	}
}

// f09SkillRouterScenarioDestination is TC-003-12's expected mechanical
// link-chain target for one coordination-level scenario, expressed exactly
// as the cell values appear in SKILL.md's own coordination-routing table.
type f09SkillRouterScenarioDestination struct {
	level       string
	destination string
}

// TestTC003_12FreshAgentReachabilityScenarioLinkChainsFromSkillRouter is
// TC-003-12 (test-plan.md #tc-003, AC-017): three fresh-agent-reachability
// scenario fixtures (Direct, Batch, Council) — starting from only SKILL.md's
// content, a deterministic link-chain must lead to the correct
// workflows/*.md file for that scenario class. This is mechanical
// link-following, not a semantic judgment call: it parses SKILL.md's own
// coordination-routing table (the same GitHub-flavored-markdown-table
// convention f09ParseMarkdownTable already uses for operating-model.md) and
// asserts each scenario row's Destination cell names the one correct
// workflow file — a router that listed Batch's row but pointed it at
// direct.md, or omitted Council's row entirely, fails here even though a
// human skimming the prose might not notice.
func TestTC003_12FreshAgentReachabilityScenarioLinkChainsFromSkillRouter(t *testing.T) {
	skill := f09ResolveSkillRouter(t)
	rows := f09ParseMarkdownTable(t, skill, "| Coordination level |")

	want := map[string]f09SkillRouterScenarioDestination{
		"`Direct`":  {level: "`Direct`", destination: "`workflows/direct.md`"},
		"`Batch`":   {level: "`Batch`", destination: "`workflows/batch.md`"},
		"`Council`": {level: "`Council`", destination: "`workflows/council.md`"},
	}
	found := map[string]bool{}
	for _, row := range rows {
		if len(row) < 3 {
			t.Fatalf("TC-003-12 SKILL.md coordination-routing table row has %d cells, want at least 3 (level, scenario, destination): %v", len(row), row)
		}
		level, destination := row[0], row[2]
		wantRow, known := want[level]
		if !known {
			continue
		}
		found[level] = true
		if destination != wantRow.destination {
			t.Errorf("TC-003-12 SKILL.md routes scenario class %s to %q, want %q (deterministic link-chain reachability)", level, destination, wantRow.destination)
		}
	}
	for level, wantRow := range want {
		if !found[level] {
			t.Errorf("TC-003-12 SKILL.md coordination-routing table is missing a %s row entirely, so a fresh agent starting from only SKILL.md cannot reach %s", level, wantRow.destination)
		}
	}
}

// ---------------------------------------------------------------------------
// TC-004-01..14 (test-plan.md #tc-004) — X-06 consumer activation, the full
// mint -> configure -> gate (question_blocks link) -> route -> respond ->
// resolve -> unblock lifecycle, plus the competing-claim collapse-to-pause
// case (TC-004-08), the sqlite_master no-new-table snapshot (TC-004-13), and
// the bespoke-type import guard (TC-004-14, its own function below — it
// needs no DB and no lifecycle state). TC-004-01..08 is T-E38-F09-011's
// scope; TC-004-09..14 (respond/resolve transcription, idempotent replay,
// session-mismatch rejection, the table snapshot, and the import guard) is
// T-E38-F09-012's scope, extending the same ordered narrative function
// rather than restating its setup in a sibling.
//
// This drives the real Shark CLI end-to-end against a temp SQLite DB via the
// compiled binary (buildSharkF09, defined above for TC-002-09) because this
// sub-index issues many sequential invocations against one shared, ordered
// Question lifecycle; the compiled-binary + real-DB pattern is the same
// tests/contracts convention TestTC004_X06ProducerPublicQuestionHandoffIsRead
// Only (e39_interactions_test.go:898) already uses for the producer side.
// ---------------------------------------------------------------------------

// f09ForbiddenQuestionBlockFields is the leak-check list TC-004-03 applies to
// a compact question_block handoff, drawn from e39_interactions_test.go:926's
// assertCompact forbidden-field list and scoped to fields a question_block
// specifically must never carry (context_data, full response/resolution
// material, and the raw relationship row).
var f09ForbiddenQuestionBlockFields = []string{
	"context_data", "responses", "evidence_pointer", "resolution_pointer",
	"resolution_kind", "relationship_id", "credential",
}

// tc004TableNames returns the sorted, comma-joined set of non-sqlite_ table
// names in sqlDB. TC-004-13 compares this before and after the full TC-004
// lifecycle: REQ-NF-001 forbids a second question/handoff/resolution store,
// and a bespoke store would require at least one new table, so an identical
// before/after set is the structural proof no such table was created. This
// intentionally captures only table identity, not row contents — the
// lifecycle legitimately writes rows to existing tables (questions, the
// question_blocks relationship, claims, task_history), and asserting "zero
// rows changed anywhere" would fail on that expected, in-scope activity.
func tc004TableNames(t *testing.T, sqlDB *sql.DB) string {
	t.Helper()
	rows, err := sqlDB.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatalf("TC-004-13 list database tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("TC-004-13 scan table: %v", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("TC-004-13 iterate tables: %v", err)
	}
	return strings.Join(tables, ",")
}

// TestTC004_01to13X06ConsumerActivationFullLifecycleMintThroughResolve
// drives the full mint -> configure -> gate -> route -> respond -> resolve ->
// unblock narrative in the order test-plan.md's TC-004-01..13 states it,
// since each step's assertion depends on the durable state the previous step
// left behind (the same reason TestTC002_I02SerialQuestionKeyedDispatch and
// TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly in
// e39_interactions_test.go are each one ordered function rather than
// independent per-assertion tests).
//
// TC-004-11 (mismatched-session respond rejected) runs immediately after
// TC-004-08 rather than after TC-004-09/10, out of its stated numeric order:
// it must observe Q001 still `answering` with bob's claim live so it
// exercises RecordResponse's claim/session-match branch
// (question_workflow_service.go:116). TC-004-09 is bob's real response and,
// since bob is the last configured responder, it also completes the
// lifecycle — Q001 moves to `ready_for_resolution` — so running TC-004-11
// after TC-004-09 would hit the earlier "must be open or answering" branch
// instead and prove nothing about session matching.
func TestTC004_01to13X06ConsumerActivationFullLifecycleMintThroughResolve(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tc004-x06-activation.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-004 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)

	// TC-004-13's baseline: captured before any TC-004-01..12 mutation runs.
	tablesBeforeLifecycle := tc004TableNames(t, sqlDB)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "TC-004 X-06 activation epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-004 seed epic: %v", err)
	}
	blocked := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "TC-004 dispatched entity"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, blocked); err != nil {
		t.Fatalf("TC-004 seed blocked feature: %v", err)
	}
	// TC-004-04's independent second target: never touched by Q001, so it
	// stays available to prove a draft-status Question's edge is inert rather
	// than accidentally inheriting Q001's own qualified block.
	secondTarget := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F02", Title: "TC-004 reversed-order target"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, secondTarget); err != nil {
		t.Fatalf("TC-004 seed second target feature: %v", err)
	}

	binary := buildSharkF09(t)
	projectRoot := f09ProjectRoot(t)
	runShark := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
		cmd.Dir = projectRoot
		out, runErr := cmd.CombinedOutput()
		return string(out), runErr
	}
	runOK := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr != nil {
			t.Fatalf("TC-004 %s: shark %s failed: %v\n%s", label, strings.Join(args, " "), runErr, out)
		}
		return out
	}
	runFail := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr == nil {
			t.Fatalf("TC-004 %s: shark %s succeeded, want rejection\n%s", label, strings.Join(args, " "), out)
		}
		return out
	}

	// TC-004-01: mint Q001 in draft.
	createOut := runOK("TC-004-01 create", "--json", "question", "create", "X-06 activation question", "--summary", "Choose the consultation outcome", "--requester", "release-owner", "--blocking")
	var created struct {
		Key      string `json:"key"`
		Status   string `json:"status"`
		Blocking bool   `json:"blocking"`
	}
	if err := json.Unmarshal([]byte(createOut), &created); err != nil {
		t.Fatalf("TC-004-01 decode create: %v\n%s", err, createOut)
	}
	if created.Key != "Q001" || created.Status != "draft" || !created.Blocking {
		t.Fatalf("TC-004-01 created Question = %#v, want Q001/draft/blocking=true", created)
	}

	// TC-004-02: configure-workflow moves the Question from draft to open —
	// the step that makes the gate below reachable at all.
	configureOut := runOK("TC-004-02 configure", "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice", "--responder", "bob")
	var configured struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(configureOut), &configured); err != nil || configured.Status != "open" {
		t.Fatalf("TC-004-02 configure-workflow = %s, decode error = %v, want status open", configureOut, err)
	}

	// TC-004-03: link created AFTER configure-workflow qualifies the gate
	// immediately — the dispatched entity's keyed next pauses with a compact
	// question_block matching {question_key, summary, resolution_owner,
	// current_responder} exactly, with no forbidden leakage.
	runOK("TC-004-03 link", "--json", "link", "Q001", "E01-F01", "--type", "question_blocks")

	assertQuestionBlock := func(label, out string, want services.QuestionBlock) {
		t.Helper()
		var envelope struct {
			Action        string                  `json:"action"`
			QuestionBlock *services.QuestionBlock `json:"question_block"`
		}
		if err := json.Unmarshal([]byte(out), &envelope); err != nil {
			t.Fatalf("TC-004 %s decode: %v\n%s", label, err, out)
		}
		if envelope.Action != "pause" || envelope.QuestionBlock == nil || *envelope.QuestionBlock != want {
			t.Fatalf("TC-004 %s = action=%s block=%#v, want pause/%#v", label, envelope.Action, envelope.QuestionBlock, want)
		}
		encoded, marshalErr := json.Marshal(envelope.QuestionBlock)
		if marshalErr != nil {
			t.Fatalf("TC-004 %s re-marshal question_block: %v", label, marshalErr)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil || len(fields) != 4 {
			t.Fatalf("TC-004 %s compact question_block fields = %v (decode error = %v), want exactly 4", label, fields, err)
		}
		for _, forbidden := range f09ForbiddenQuestionBlockFields {
			if _, present := fields[forbidden]; present {
				t.Fatalf("TC-004 %s question_block leaks forbidden field %q: %s", label, forbidden, out)
			}
		}
	}

	blockedNextOut := runOK("TC-004-03 next blocked entity", "--json", "next", "E01-F01")
	assertQuestionBlock("TC-004-03 next E01-F01", blockedNextOut, services.QuestionBlock{
		QuestionKey: "Q001", Summary: "Choose the consultation outcome",
		ResolutionOwner: "release-owner", CurrentResponder: "alice",
	})

	// TC-004-04 (the load-bearing ordering case): a SECOND Question, still
	// draft and never configured, linked to a fresh target. D-006:
	// QualifyQuestionBlock only recognizes open/answering, so this edge is
	// silently inert — the fixture, not prose, proves configure-before-link.
	runOK("TC-004-04 create Q002", "--json", "question", "create", "TC-004-04 reversed order question", "--summary", "Never configured", "--requester", "release-owner", "--blocking")
	runOK("TC-004-04 link draft Q002", "--json", "link", "Q002", "E01-F02", "--type", "question_blocks")
	reversedNextOut := runOK("TC-004-04 next E01-F02", "--json", "next", "E01-F02")
	var reversedNext commands.NextResponse
	if err := json.Unmarshal([]byte(reversedNextOut), &reversedNext); err != nil {
		t.Fatalf("TC-004-04 decode next E01-F02: %v\n%s", err, reversedNextOut)
	}
	if reversedNext.QuestionBlock != nil {
		t.Fatalf("TC-004-04 next E01-F02 = %#v, want no question_block — a still-draft Question's question_blocks edge must stay inert", reversedNext)
	}

	// TC-004-05: while Q001 is open, a status-advance attempt on the blocked
	// entity is rejected — reuses guardQuestionBlockedStatusAdvance
	// (status_group.go:545), relied upon rather than reimplemented here.
	advanceRejectedOut := runFail("TC-004-05 advance while open", "--json", "status", "advance", "E01-F01")
	if !strings.Contains(advanceRejectedOut, `"code": "QUESTION_BLOCKED"`) {
		t.Fatalf("TC-004-05 advance-while-open output = %s, want QUESTION_BLOCKED", advanceRejectedOut)
	}

	// TC-004-05b (added per codex red-team CONCERN): the same rejection holds
	// while Q001 is answering, not just open. Alice responds (bob remains
	// pending), which moves Q001 from open to answering; status advance on
	// the blocked entity is rejected again. QualifyQuestionBlock checks both
	// states (question_blocker.go:86); this closes the gap between the
	// checked states and the tested ones.
	runOK("TC-004-05b claim alice", "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session")
	respondOut := runOK("TC-004-05b respond alice", "--json", "question", "respond", "Q001", "--session", "alice-session", "--responder", "alice", "--summary", "alice's bounded response", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	var afterRespond struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(respondOut), &afterRespond); err != nil || afterRespond.Status != "answering" {
		t.Fatalf("TC-004-05b question respond = %s, decode error = %v, want status answering", respondOut, err)
	}
	answeringAdvanceOut := runFail("TC-004-05b advance while answering", "--json", "status", "advance", "E01-F01")
	if !strings.Contains(answeringAdvanceOut, `"code": "QUESTION_BLOCKED"`) {
		t.Fatalf("TC-004-05b advance-while-answering output = %s, want QUESTION_BLOCKED", answeringAdvanceOut)
	}
	runOK("TC-004-05b release alice", "--json", "release", "Q001", "--session", "alice-session", "--outcome", "pass")

	// TC-004-06: status set --force succeeds despite the still-open Question
	// — the documented human escape hatch (AC-006, D-006) that intentionally
	// does not check the question gate.
	setOut := runOK("TC-004-06 status set --force", "--json", "status", "set", "E01-F01", "assessment", "--force", "--reason", "TC-004-06 human escape hatch despite open Question")
	var setResult struct {
		Changed  bool `json:"changed"`
		IsForced bool `json:"is_forced"`
	}
	if err := json.Unmarshal([]byte(setOut), &setResult); err != nil || !setResult.Changed || !setResult.IsForced {
		t.Fatalf("TC-004-06 status set --force = %s, decode error = %v, want a forced changed transition despite the open Question", setOut, err)
	}

	// TC-004-07: next Q001 dispatches a single scoped prompt naming only the
	// current pending responder (bob — alice already completed), never the
	// other configured responder and never the blocked entity's key.
	questionNextOut := runOK("TC-004-07 next Q001", "--json", "next", "Q001")
	var questionNext commands.NextResponse
	if err := json.Unmarshal([]byte(questionNextOut), &questionNext); err != nil {
		t.Fatalf("TC-004-07 decode next Q001: %v\n%s", err, questionNextOut)
	}
	if questionNext.EntityKey != "Q001" || questionNext.Action != "spawn_agent" {
		t.Fatalf("TC-004-07 next Q001 = %#v, want Q001/spawn_agent", questionNext)
	}
	if !strings.Contains(questionNext.Prompt, "currently routed responder: bob") {
		t.Fatalf("TC-004-07 next Q001 prompt = %q, want it to name bob as the current pending responder", questionNext.Prompt)
	}
	if strings.Contains(questionNext.Prompt, "alice") {
		t.Fatalf("TC-004-07 next Q001 prompt = %q, must not expose the already-completed responder alice", questionNext.Prompt)
	}
	if strings.Contains(questionNextOut, "E01-F01") {
		t.Fatalf("TC-004-07 next Q001 output = %s, must not reference the blocked entity's key", questionNextOut)
	}

	// TC-004-08: a second, competing next Q001 while a live claim exists on
	// Q001 collapses to pause instead of naming a competing responder
	// (AC-007).
	runOK("TC-004-08 claim bob", "--json", "claim", "Q001", "--by", "bob", "--session", "bob-session")
	competingOut := runOK("TC-004-08 competing next Q001", "--json", "next", "Q001")
	var competing commands.NextResponse
	if err := json.Unmarshal([]byte(competingOut), &competing); err != nil {
		t.Fatalf("TC-004-08 decode competing next Q001: %v\n%s", err, competingOut)
	}
	if competing.Action != "pause" {
		t.Fatalf("TC-004-08 competing next Q001 = %#v, want pause while bob's claim is live", competing)
	}

	// TC-004-11 (executed here — see the function doc comment for why this
	// runs before TC-004-09/10): a question respond call whose --session
	// does not match bob's live claim (session bob-session) is rejected
	// while Q001 is still answering — AC-008's negative case.
	mismatchOut := runFail("TC-004-11 respond wrong session", "--json", "question", "respond", "Q001", "--session", "not-bobs-session", "--responder", "bob", "--summary", "bob's response under the wrong session", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/test-plan.md")
	if !strings.Contains(mismatchOut, "active claim does not match responder session") {
		t.Fatalf("TC-004-11 mismatched-session respond output = %s, want rejection naming the claim/session mismatch", mismatchOut)
	}

	// TC-004-09: the parent (not the worker) transcribes bob's answer under
	// the parent-held claim (session bob-session) into question respond
	// (REQ-F-005). Bob is the last pending responder, so this also
	// completes the lifecycle: Q001 transitions to ready_for_resolution.
	bobRespondOut := runOK("TC-004-09 respond bob", "--json", "question", "respond", "Q001", "--session", "bob-session", "--responder", "bob", "--summary", "bob's bounded response", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	var afterBobRespond struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(bobRespondOut), &afterBobRespond); err != nil || afterBobRespond.Status != "ready_for_resolution" {
		t.Fatalf("TC-004-09 question respond = %s, decode error = %v, want status ready_for_resolution", bobRespondOut, err)
	}

	// TC-004-10: replaying the identical respond call is idempotent — no
	// error and no duplicate response row (responseReplayMatches,
	// question_workflow_service.go:156). question full (authorized here as
	// the configured resolution owner) exposes the raw Responses slice so
	// the response count, not just the status, proves no duplicate write.
	readFullResponseCount := func(label string) int {
		t.Helper()
		out := runOK(label, "--json", "question", "full", "Q001", "--actor", "release-owner")
		var full struct {
			Responses []struct {
				Responder string `json:"responder"`
			} `json:"responses"`
		}
		if err := json.Unmarshal([]byte(out), &full); err != nil {
			t.Fatalf("%s decode: %v\n%s", label, err, out)
		}
		return len(full.Responses)
	}
	responsesBeforeReplay := readFullResponseCount("TC-004-10 full before replay")
	replayOut := runOK("TC-004-10 replay respond bob", "--json", "question", "respond", "Q001", "--session", "bob-session", "--responder", "bob", "--summary", "bob's bounded response", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	var afterReplay struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(replayOut), &afterReplay); err != nil || afterReplay.Status != "ready_for_resolution" {
		t.Fatalf("TC-004-10 replay question respond = %s, decode error = %v, want unchanged status ready_for_resolution", replayOut, err)
	}
	responsesAfterReplay := readFullResponseCount("TC-004-10 full after replay")
	if responsesAfterReplay != responsesBeforeReplay {
		t.Fatalf("TC-004-10 replay wrote a duplicate response: before=%d after=%d", responsesBeforeReplay, responsesAfterReplay)
	}
	runOK("TC-004-10 release bob", "--json", "release", "Q001", "--session", "bob-session", "--outcome", "pass")

	// TC-004-12: resolve closes Q001 (REQ-F-006); the question_blocks
	// predicate no longer qualifies and the previously-blocked entity's
	// status advance now succeeds (AC-009).
	resolveOut := runOK("TC-004-12 resolve", "--json", "question", "resolve", "Q001", "--owner", "release-owner", "--resolution-kind", "no_lasting_consequence")
	var resolved struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(resolveOut), &resolved); err != nil || resolved.Status != "resolved" {
		t.Fatalf("TC-004-12 resolve = %s, decode error = %v, want status resolved", resolveOut, err)
	}
	runOK("TC-004-12 advance after resolve", "--json", "status", "advance", "E01-F01")

	// TC-004-13 (added per codex red-team BLOCKER — "zero bespoke records"
	// was previously asserted only by not writing one in the test's own
	// code, which proves nothing about the production schema): the
	// sqlite_master table set captured before TC-004-01 ran must be
	// byte-identical to the table set captured now, after the full mint ->
	// configure -> link -> route -> respond -> resolve -> unblock lifecycle.
	if after := tc004TableNames(t, sqlDB); after != tablesBeforeLifecycle {
		t.Fatalf("TC-004-13 sqlite_master table set changed during the lifecycle:\nbefore: %s\nafter:  %s", tablesBeforeLifecycle, after)
	}
}

// tc004ForbiddenBespokeQuestionStoreIdentifiers are the retired
// abandoned-branch type names (research-report Findings #5) for the bespoke
// QuestionControl-style store F09 rejected in favor of extending E39's
// public Question API (D-001, REQ-NF-001). TC-004-14 fails if any of these
// ever reappears anywhere under internal/.
var tc004ForbiddenBespokeQuestionStoreIdentifiers = []string{
	"council_artifact", "question_control", "parent_control", "roster_profile",
}

// TestTC004_14NoBespokeQuestionControlStoreReintroducedUnderInternal is
// TC-004-14 (test-plan.md #tc-004, added per codex red-team BLOCKER,
// REQ-NF-001 static guard): a source-level guard over every .go file under
// internal/ asserting none of the retired abandoned-branch identifiers
// appear anywhere in the tree — catching an accidental reintroduction of the
// rejected bespoke design at compile-dependency granularity, not just by
// code review. This test's own tests/contracts package sits outside
// internal/ and is not walked, which is exactly what lets it name the
// forbidden identifiers as literal pattern strings without self-matching.
func TestTC004_14NoBespokeQuestionControlStoreReintroducedUnderInternal(t *testing.T) {
	root := filepath.Join(f09ProjectRoot(t), "internal")
	projectRoot := f09ProjectRoot(t)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lower := strings.ToLower(string(content))
		for _, forbidden := range tc004ForbiddenBespokeQuestionStoreIdentifiers {
			if strings.Contains(lower, forbidden) {
				rel, relErr := filepath.Rel(projectRoot, path)
				if relErr != nil {
					rel = path
				}
				t.Errorf("TC-004-14 %s contains forbidden retired identifier %q — a bespoke QuestionControl-style store must not be reintroduced under internal/ (REQ-NF-001)", rel, forbidden)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("TC-004-14 walk internal/: %v", walkErr)
	}
}

// ---------------------------------------------------------------------------
// TC-006-01..04 (test-plan.md #tc-006, I-04 council routing threshold) —
// T-E38-F09-013. council.md defines the material-question threshold and
// carries forward the two procedures T-024 retires (communicate.md's inbox/
// acknowledgement rule, escalate.md's material-question routing procedure);
// route-question.md points at the threshold rather than restating it.
// ---------------------------------------------------------------------------

// f09CouncilVocabularyFromSchema extracts the category vocabulary
// (`product|requirements|architecture|quality|process`) directly from
// context/worker-control-schema.yaml's `example_question.category` line, so
// TC-006-04's agreement check is against the live schema value, not a second
// hand-transcribed list that could silently drift from it.
func f09CouncilVocabularyFromSchema(t *testing.T) []string {
	t.Helper()
	schema := readF09EmbeddedFile(t, "skills/shark-attack/context/worker-control-schema.yaml")
	re := regexp.MustCompile(`(?m)^\s*category:\s*(\S+)\s+# required`)
	m := re.FindStringSubmatch(schema)
	if m == nil {
		t.Fatalf("TC-006-04 fixture assumption broken: worker-control-schema.yaml no longer declares a %q category line", "category: <vocab>  # required")
	}
	return strings.Split(m[1], "|")
}

// f09ResolveCouncil resolves council.md through the embedded tree.
func f09ResolveCouncil(t *testing.T) string {
	t.Helper()
	return readF09EmbeddedFile(t, "skills/shark-attack/workflows/council.md")
}

// f09ResolveRouteQuestion resolves route-question.md through the embedded
// tree.
func f09ResolveRouteQuestion(t *testing.T) string {
	t.Helper()
	return readF09EmbeddedFile(t, "skills/shark-attack/workflows/route-question.md")
}

// f09CouncilCategoryTable parses council.md's "Category | Default path |
// Off-default when" table (columns: category, default path, off-default
// condition).
func f09CouncilCategoryTable(t *testing.T) [][]string {
	t.Helper()
	return f09ParseMarkdownTable(t, f09ResolveCouncil(t), "| Category |")
}

// f09CouncilMaterialThresholdMarker is the canonical, single-source-of-truth
// sentence opener for council.md's material-threshold definition. TC-006-03
// asserts it appears exactly once in council.md and zero times in
// route-question.md (which must point at the rule, never restate it).
const f09CouncilMaterialThresholdMarker = "A question is material when"

// TestTC006_01RoutineQuestionRoutesToE39LoopWithNoCouncilArtifact is
// TC-006-01: a fixture question classified "routine" (scope-bounded,
// single-role, no architecture/quality/product impact) is documented as
// routing to the E39 `Q###` path (route-question.md) and creating no
// `docs/council/` artifact. Anchored to the category table's own routine row
// (not a floating string) so a "route everything to council" implementation
// — which would leave no routine-default row — fails this test, per the
// test plan's explicit counter-factual.
func TestTC006_01RoutineQuestionRoutesToE39LoopWithNoCouncilArtifact(t *testing.T) {
	rows := f09CouncilCategoryTable(t)

	foundRoutine := false
	for _, row := range rows {
		if len(row) < 2 {
			t.Fatalf("TC-006-01 council.md category table row has %d cells, want >= 2: %+v", len(row), row)
		}
		category := strings.Trim(row[0], "`")
		path := row[1]
		if category == "requirements" || category == "process" {
			if !strings.Contains(path, "route-question.md") {
				t.Fatalf("TC-006-01 category %q must default to the E39 Q### loop (route-question.md); got path %q", category, path)
			}
			foundRoutine = true
		}
	}
	if !foundRoutine {
		t.Fatal("TC-006-01 council.md's category table has no routine-default (route-question.md) row; a route-everything-to-council implementation would fail this check")
	}

	council := f09ResolveCouncil(t)
	normalized := strings.Join(strings.Fields(council), " ")
	for _, want := range []string{
		"no `docs/council/` artifact",
		"route-question.md",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("TC-006-01 council.md's routine-path prose must state %q; content:\n%s", want, council)
		}
	}
}

// TestTC006_02MaterialQuestionRoutesThroughCouncilOnlyNoEscalateReference is
// TC-006-02: a fixture question classified "material" (crosses a named
// threshold: scope, architecture, or quality-gate impact) is documented as
// routing through council.md only — escalate.md (retired by T-024) must not
// appear in this assertion or in council.md's prose — and creating exactly
// one immutable decision/handoff record, not a `Q###`.
func TestTC006_02MaterialQuestionRoutesThroughCouncilOnlyNoEscalateReference(t *testing.T) {
	rows := f09CouncilCategoryTable(t)

	foundMaterial := false
	for _, row := range rows {
		if len(row) < 2 {
			t.Fatalf("TC-006-02 council.md category table row has %d cells, want >= 2: %+v", len(row), row)
		}
		category := strings.Trim(row[0], "`")
		path := row[1]
		if category == "product" || category == "architecture" || category == "quality" {
			if !strings.Contains(path, "Council") || strings.Contains(path, "route-question.md") {
				t.Fatalf("TC-006-02 category %q must default to Council (this file); got path %q", category, path)
			}
			foundMaterial = true
		}
	}
	if !foundMaterial {
		t.Fatal("TC-006-02 council.md's category table has no Council-default row")
	}

	council := f09ResolveCouncil(t)
	if strings.Contains(council, "escalate.md") {
		t.Fatal("TC-006-02 council.md must not reference escalate.md (retired by T-024)")
	}

	normalized := strings.Join(strings.Fields(council), " ")
	for _, want := range []string{
		"routes through this file only",
		"Exactly one artifact file exists per material question",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("TC-006-02 council.md must state exclusive routing and single-record language; missing %q; content:\n%s", want, council)
		}
	}
}

// TestTC006_03ThresholdStatedOnceInCouncilRouteQuestionPointsNotRestates is
// TC-006-03: the threshold rule itself (what makes a question "material") is
// stated once, in one file, and referenced — not duplicated — from the
// other (route-question.md points at council.md's definition rather than
// restating it).
func TestTC006_03ThresholdStatedOnceInCouncilRouteQuestionPointsNotRestates(t *testing.T) {
	council := f09ResolveCouncil(t)
	if count := strings.Count(council, f09CouncilMaterialThresholdMarker); count != 1 {
		t.Fatalf("TC-006-03 council.md must state the material-threshold marker %q exactly once, found %d", f09CouncilMaterialThresholdMarker, count)
	}

	routeQuestion := f09ResolveRouteQuestion(t)
	if !strings.Contains(routeQuestion, "workflows/council.md") {
		t.Fatal("TC-006-03 route-question.md must point at workflows/council.md for the material-question path")
	}
	if strings.Contains(routeQuestion, f09CouncilMaterialThresholdMarker) {
		t.Fatalf("TC-006-03 route-question.md must not restate council.md's threshold marker %q", f09CouncilMaterialThresholdMarker)
	}
	if strings.Contains(routeQuestion, "| Category |") {
		t.Fatal("TC-006-03 route-question.md must not restate council.md's category vocabulary table")
	}
}

// TestTC006_04CategoryVocabularyAgreesNoOrphanNoAmbiguity is TC-006-04:
// route-question.md and council.md agree on which vocabulary/category
// values (product|requirements|architecture|quality|process per REQ-F-003)
// map to which path — no category is orphaned (routable to neither) or
// ambiguous (routable to both without a tie-break rule). The vocabulary is
// read from the live schema (f09CouncilVocabularyFromSchema), not a second
// hand-authored list, so this test cannot silently diverge from
// worker-control-schema.yaml.
func TestTC006_04CategoryVocabularyAgreesNoOrphanNoAmbiguity(t *testing.T) {
	wantCategories := f09CouncilVocabularyFromSchema(t)
	if len(wantCategories) == 0 {
		t.Fatal("TC-006-04 fixture assumption broken: no category vocabulary extracted from the schema")
	}

	rows := f09CouncilCategoryTable(t)
	gotPaths := map[string]string{}
	for _, row := range rows {
		if len(row) < 2 {
			t.Fatalf("TC-006-04 council.md category table row has %d cells, want >= 2: %+v", len(row), row)
		}
		category := strings.Trim(row[0], "`")
		if _, dup := gotPaths[category]; dup {
			t.Fatalf("TC-006-04 category %q appears more than once in council.md's table", category)
		}
		gotPaths[category] = row[1]
	}

	for _, category := range wantCategories {
		path, ok := gotPaths[category]
		if !ok {
			t.Fatalf("TC-006-04 category %q from the schema vocabulary is orphaned: no row in council.md's category table", category)
		}
		isRoutine := strings.Contains(path, "route-question.md")
		isCouncil := strings.Contains(path, "Council")
		if isRoutine == isCouncil {
			// Either neither path matched (isRoutine=false, isCouncil=false)
			// or both matched (ambiguous, routable to both) — both are
			// failures the AC explicitly names.
			t.Fatalf("TC-006-04 category %q must resolve to exactly one path (route-question.md xor Council), got path %q (routine=%v, council=%v)", category, path, isRoutine, isCouncil)
		}
	}
	if len(gotPaths) != len(wantCategories) {
		t.Fatalf("TC-006-04 council.md's category table has %d rows, want exactly %d (one per schema category); got categories: %+v", len(gotPaths), len(wantCategories), gotPaths)
	}

	// route-question.md must not define a second, potentially conflicting
	// category table — council.md is the single source of truth.
	routeQuestion := f09ResolveRouteQuestion(t)
	if strings.Contains(routeQuestion, "| Category |") {
		t.Fatal("TC-006-04 route-question.md must not define its own category vocabulary table; council.md is the single source of truth")
	}
}

// ---------------------------------------------------------------------------
// TC-010-05,06,08..13,15,16 (test-plan.md #tc-010) — T-E38-F09-014. These
// cover the resume lifecycle's core procedure: same-worker follow-up vs
// bounded replacement (AC-021), capability-discovery-before-selection
// ordering (AC-022), and interrupt-driven cancellation ordering (AC-023).
// TC-010-01..04, 07, 14, 17..20 (the silent-responder escalation ladder and
// the handoff no-rendered-prompt sweep) belong to T-E38-F09-016 alone — see
// that task's spec for why they are not duplicated here. F09 ships no
// dispatcher Go code (D-002), so every assertion below is content/fixture
// level against the installed skill tree via readF09EmbeddedFile, matching
// this file's existing TC-003/TC-005 conventions.
// ---------------------------------------------------------------------------

// f09ResolveResume resolves resume.md through the real embedded tree via
// sharkdata.ReadEmbedded (readF09EmbeddedFile), the declared read
// convention for skills/shark-attack/** content in this file's header
// comment.
func f09ResolveResume(t *testing.T) string {
	t.Helper()
	return readF09EmbeddedFile(t, "skills/shark-attack/workflows/resume.md")
}

// f09ResumeCapabilityVectorTable parses resume.md's 5-row capability-vector
// decision table (columns: #, Isolation detected?, Follow-up detected?,
// Interrupt detected?, Expected resolved behavior, Sub-case). Parsing the
// table directly, rather than hand-transcribing its rows into Go literals,
// means an edit that breaks or reorders a row is caught here instead of
// silently drifting from what resume.md actually says.
func f09ResumeCapabilityVectorTable(t *testing.T) [][]string {
	t.Helper()
	// "| Isolation detected? |" (rather than "| # |", which also matches the
	// responder-outcome ladder table T-E38-F09-016 added later in the file)
	// is the substring that uniquely anchors this table.
	rows := f09ParseMarkdownTable(t, f09ResolveResume(t), "| Isolation detected? |")
	if len(rows) != 5 {
		t.Fatalf("TC-010 resume.md capability-vector table has %d rows, want the full 5-row table", len(rows))
	}
	return rows
}

// f09ParseBehaviorSegments splits one "Expected resolved behavior" cell
// (e.g. "Topology: Sequential; Follow-up: same-worker (zero new workers);
// Interrupt: cancel-then-replace") into a Topology/Follow-up/Interrupt
// segment map, so each capability's resolved behavior is individually
// assertable rather than checked with a single brittle Contains.
func f09ParseBehaviorSegments(t *testing.T, cell string) map[string]string {
	t.Helper()
	segments := map[string]string{}
	for _, part := range strings.Split(cell, "; ") {
		name, value, ok := strings.Cut(part, ": ")
		if !ok {
			t.Fatalf("TC-010 behavior cell segment %q is not in \"Name: value\" form; full cell: %q", part, cell)
		}
		segments[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	for _, want := range []string{"Topology", "Follow-up", "Interrupt"} {
		if _, ok := segments[want]; !ok {
			t.Fatalf("TC-010 behavior cell %q is missing the %q segment", cell, want)
		}
	}
	return segments
}

// f09ResumeForbiddenProviderCommandLiterals is the set of concrete provider
// CLI invocation tokens resume.md must never contain. resume.md names
// capabilities abstractly and points at providers/{codex,claude-code}.md,
// which own the actual commands (REQ-F-012) — a resume.md that hard-codes
// one of these strings would be inventing/duplicating a provider command
// rather than deferring to the provider reference.
func f09ResumeForbiddenProviderCommandLiterals() []string {
	return []string{
		"codex exec", "codex resume", "codex fork",
		"claude -p", "claude --resume", "claude --bg", "claude -w", "claude --worktree",
		"git worktree",
	}
}

// TestTC010_05FollowUpSupportedDeliversToSameWorkerZeroNewWorkers is
// TC-010-05: the fixture "host declares resume supported" branch of
// resume.md's resume-path procedure delivers the answer to the same worker
// identity and creates zero new workers (AC-021).
func TestTC010_05FollowUpSupportedDeliversToSameWorkerZeroNewWorkers(t *testing.T) {
	content := f09ResolveResume(t)
	branchPattern := regexp.MustCompile(`(?s)\*\*Follow-up supported\.\*\*(.*?)\n\d+\.\s+\*\*Follow-up unsupported`)
	match := branchPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-05 resume.md must document a \"Follow-up supported\" branch immediately followed by a \"Follow-up unsupported\" branch; content:\n%s", content)
	}
	branch := match[1]
	for _, want := range []string{"same", "worker", "zero new workers"} {
		if !strings.Contains(branch, want) {
			t.Fatalf("TC-010-05 resume.md's Follow-up-supported branch must state %q; branch text:\n%s", want, branch)
		}
	}
	if strings.Contains(branch, "replacement") {
		t.Fatalf("TC-010-05 resume.md's Follow-up-supported branch must not mention starting a replacement worker; branch text:\n%s", branch)
	}
}

// TestTC010_06FollowUpUnsupportedCreatesExactlyOneReplacementWorker is
// TC-010-06: the fixture "host declares resume unsupported" branch creates
// exactly one replacement worker from a bounded immutable handoff (AC-021).
func TestTC010_06FollowUpUnsupportedCreatesExactlyOneReplacementWorker(t *testing.T) {
	content := f09ResolveResume(t)
	branchPattern := regexp.MustCompile(`(?s)\*\*Follow-up unsupported\.\*\*(.*?)\n### Handoff content`)
	match := branchPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-06 resume.md must document a \"Follow-up unsupported\" branch followed by a \"Handoff content\" section; content:\n%s", content)
	}
	// Normalize whitespace before substring checks: markdown line-wraps a
	// phrase across lines, and a literal Contains would otherwise break on
	// the embedded newline.
	branch := strings.Join(strings.Fields(match[1]), " ")
	for _, want := range []string{"bounded immutable handoff", "exactly one replacement worker"} {
		if !strings.Contains(branch, want) {
			t.Fatalf("TC-010-06 resume.md's Follow-up-unsupported branch must state %q; branch text:\n%s", want, branch)
		}
	}
	for _, forbidden := range []string{"more than one replacement", "second time"} {
		if !strings.Contains(branch, forbidden) {
			t.Fatalf("TC-010-06 resume.md's Follow-up-unsupported branch must positively forbid %q; branch text:\n%s", forbidden, branch)
		}
	}
}

// TestTC010_08IsolationUndetectedResolvesSequentialWithoutDetectionCommand
// is TC-010-08: with isolation undetected (and follow-up/interrupt both
// detected), the capability-vector table row resolves to Sequential, and
// resume.md never describes an isolation-detection command attempt beyond
// the capability-discovery step itself (REQ-F-012 ordering) — asserted by
// the absence of any concrete provider isolation-command literal anywhere
// in the file.
func TestTC010_08IsolationUndetectedResolvesSequentialWithoutDetectionCommand(t *testing.T) {
	rows := f09ResumeCapabilityVectorTable(t)
	row := rows[1] // row 2: isolation=no, follow-up=yes, interrupt=yes
	if row[1] != "no" || row[2] != "yes" || row[3] != "yes" {
		t.Fatalf("TC-010-08 capability-vector row 2 = %+v, want isolation=no/follow-up=yes/interrupt=yes", row)
	}
	segments := f09ParseBehaviorSegments(t, row[4])
	if !strings.HasPrefix(segments["Topology"], "Sequential") || !strings.Contains(segments["Topology"], "isolation undetected") {
		t.Fatalf("TC-010-08 row 2 Topology segment = %q, want a Sequential resolution citing isolation undetected", segments["Topology"])
	}
	if row[5] != "TC-010-08" {
		t.Fatalf("TC-010-08 row 2 Sub-case column = %q, want TC-010-08", row[5])
	}

	content := f09ResolveResume(t)
	for _, forbidden := range f09ResumeForbiddenProviderCommandLiterals() {
		if strings.Contains(content, forbidden) {
			t.Fatalf("TC-010-08 resume.md must not hard-code a provider command literal (found %q) — isolation status comes from providers/{codex,claude-code}.md's captured evidence, never an invented detection command", forbidden)
		}
	}
}

// TestTC010_09FollowUpUndetectedForcesReplacementPath is TC-010-09: with
// follow-up undetected (isolation and interrupt both detected), the
// question loop forces the bounded-replacement path, not same-worker.
func TestTC010_09FollowUpUndetectedForcesReplacementPath(t *testing.T) {
	rows := f09ResumeCapabilityVectorTable(t)
	row := rows[2] // row 3: isolation=yes, follow-up=no, interrupt=yes
	if row[1] != "yes" || row[2] != "no" || row[3] != "yes" {
		t.Fatalf("TC-010-09 capability-vector row 3 = %+v, want isolation=yes/follow-up=no/interrupt=yes", row)
	}
	segments := f09ParseBehaviorSegments(t, row[4])
	if !strings.Contains(segments["Follow-up"], "bounded replacement") || !strings.Contains(segments["Follow-up"], "exactly one replacement worker") {
		t.Fatalf("TC-010-09 row 3 Follow-up segment = %q, want the bounded-replacement resolution", segments["Follow-up"])
	}
	if strings.Contains(segments["Follow-up"], "same-worker") {
		t.Fatalf("TC-010-09 row 3 Follow-up segment = %q, must not resolve to same-worker when follow-up is undetected", segments["Follow-up"])
	}
	if row[5] != "TC-010-09" {
		t.Fatalf("TC-010-09 row 3 Sub-case column = %q, want TC-010-09", row[5])
	}
}

// TestTC010_10InterruptUndetectedForcesDeadlineOnlyExpiry is TC-010-10:
// with interrupt undetected (isolation and follow-up both detected), the
// silent-responder handling forces deadline-only expiry, with no cancel
// attempt described.
func TestTC010_10InterruptUndetectedForcesDeadlineOnlyExpiry(t *testing.T) {
	rows := f09ResumeCapabilityVectorTable(t)
	row := rows[3] // row 4: isolation=yes, follow-up=yes, interrupt=no
	if row[1] != "yes" || row[2] != "yes" || row[3] != "no" {
		t.Fatalf("TC-010-10 capability-vector row 4 = %+v, want isolation=yes/follow-up=yes/interrupt=no", row)
	}
	segments := f09ParseBehaviorSegments(t, row[4])
	if !strings.Contains(segments["Interrupt"], "deadline-only") || !strings.Contains(segments["Interrupt"], "no cancel attempt") {
		t.Fatalf("TC-010-10 row 4 Interrupt segment = %q, want the deadline-only, no-cancel resolution", segments["Interrupt"])
	}
	if strings.Contains(segments["Interrupt"], "cancel-then-replace") {
		t.Fatalf("TC-010-10 row 4 Interrupt segment = %q, must not resolve to cancel-then-replace when interrupt is undetected", segments["Interrupt"])
	}
	if row[5] != "TC-010-10" {
		t.Fatalf("TC-010-10 row 4 Sub-case column = %q, want TC-010-10", row[5])
	}

	content := f09ResolveResume(t)
	unsupportedPattern := regexp.MustCompile(`(?s)\*\*Interrupt unsupported\.\*\*(.*?)(\n## |\z)`)
	match := unsupportedPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-10 resume.md must document an \"Interrupt unsupported\" branch; content:\n%s", content)
	}
	if !strings.Contains(match[1], "deadline") {
		t.Fatalf("TC-010-10 resume.md's Interrupt-unsupported branch must describe the deadline-only fallback; branch text:\n%s", match[1])
	}
	if strings.Contains(match[1], "Cancel the stale consultation") {
		t.Fatalf("TC-010-10 resume.md's Interrupt-unsupported branch must not describe a cancel step; branch text:\n%s", match[1])
	}
}

// TestTC010_11CapabilityDiscoveryPrecedesTopologyCoordinationSelection is
// TC-010-11: capability discovery is documented as the first step, strictly
// before any topology/coordination-selection content — asserted by section
// ordering in the rendered workflow file, not by mere Contains (a heading
// name that happened to contain the word "topology" would otherwise let
// this test pass without actually proving the ordering).
func TestTC010_11CapabilityDiscoveryPrecedesTopologyCoordinationSelection(t *testing.T) {
	content := f09ResolveResume(t)

	capIdx := strings.Index(content, "\n## Capability discovery")
	if capIdx == -1 {
		t.Fatalf("TC-010-11 resume.md must have a \"## Capability discovery\" heading; content:\n%s", content)
	}

	// "Topology:" is resume.md's own load-bearing marker for a resolved
	// topology outcome (the capability-vector table's segment name) — the
	// first place resume.md names a topology-resolution outcome at all.
	topoIdx := strings.Index(content, "Topology:")
	if topoIdx == -1 {
		t.Fatalf("TC-010-11 resume.md must resolve a Topology outcome somewhere (the capability-vector table); content:\n%s", content)
	}
	if topoIdx < capIdx {
		t.Fatalf("TC-010-11 resume.md names a Topology resolution at byte offset %d, before its own \"## Capability discovery\" heading at offset %d — capability discovery must come first", topoIdx, capIdx)
	}

	before := content[:capIdx]
	if strings.Contains(before, "Topology:") {
		t.Fatalf("TC-010-11 resume.md must not resolve any Topology outcome before the \"## Capability discovery\" heading; text before that heading:\n%s", before)
	}

	// context/operating-model.md — the file that actually owns topology and
	// coordination-level selection — must only be referenced after
	// capability discovery is introduced.
	modelIdx := strings.Index(content, "context/operating-model.md")
	if modelIdx == -1 {
		t.Fatalf("TC-010-11 resume.md must reference context/operating-model.md; content:\n%s", content)
	}
	if modelIdx < capIdx {
		t.Fatalf("TC-010-11 resume.md references context/operating-model.md (topology/coordination selection) at offset %d, before its own \"## Capability discovery\" heading at offset %d", modelIdx, capIdx)
	}
}

// TestTC010_12InterruptSupportedCancelsBeforeRoutingReplacement is
// TC-010-12: with interrupt supported, resume.md documents cancelling the
// stale consultation before routing the replacement responder (an ordering
// assertion by numbered-step index, not prose position), and states that
// cancelling changes no Shark state.
func TestTC010_12InterruptSupportedCancelsBeforeRoutingReplacement(t *testing.T) {
	content := f09ResolveResume(t)

	cancelIdx := strings.Index(content, "Cancel the stale consultation before routing the replacement")
	if cancelIdx == -1 {
		t.Fatalf("TC-010-12 resume.md must document cancelling the stale consultation before routing the replacement responder; content:\n%s", content)
	}
	routeIdx := strings.Index(content, "Route exactly one replacement responder")
	if routeIdx == -1 {
		t.Fatalf("TC-010-12 resume.md must document routing exactly one replacement responder; content:\n%s", content)
	}
	if routeIdx < cancelIdx {
		t.Fatalf("TC-010-12 resume.md routes the replacement responder (offset %d) before cancelling the stale consultation (offset %d) — cancel must come first", routeIdx, cancelIdx)
	}

	supportedPattern := regexp.MustCompile(`(?s)\*\*Interrupt supported\.\*\*(.*?)\n\d+\.\s+\*\*Interrupt unsupported`)
	match := supportedPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-12 resume.md must document an \"Interrupt supported\" branch immediately followed by an \"Interrupt unsupported\" branch; content:\n%s", content)
	}
	branch := match[1]
	if !strings.Contains(branch, "changes no Shark state") {
		t.Fatalf("TC-010-12 resume.md's Interrupt-supported branch must state that cancelling changes no Shark state; branch text:\n%s", branch)
	}
	noStatePattern := regexp.MustCompile(`no (claim|status|history)`)
	if !noStatePattern.MatchString(branch) {
		t.Fatalf("TC-010-12 resume.md's Interrupt-supported branch must name at least one of claim/status/history as unaffected by the cancel; branch text:\n%s", branch)
	}
}

// TestTC010_13InterruptUnsupportedFallbackInvokesNoUnverifiedProviderCommand
// is TC-010-13: where interrupt is unsupported, the documented fallback
// runs and no unverified provider command is invoked — cross-checked
// against both provider references' own declared-unsupported-ops list
// (reusing f09ParseProviderReferenceSection/f09MissingEvidenceOps from
// TC-003-08/09 rather than hand-rolling a second parser), proving the
// fallback never silently invokes an op either provider reference itself
// marked unsupported.
func TestTC010_13InterruptUnsupportedFallbackInvokesNoUnverifiedProviderCommand(t *testing.T) {
	for _, relPath := range []string{
		"skills/shark-attack/providers/codex.md",
		"skills/shark-attack/providers/claude-code.md",
	} {
		content := readF09EmbeddedFile(t, relPath)
		unsupported := f09ParseProviderReferenceSection(t, content, "Unsupported Operations")
		noEvidence := f09MissingEvidenceOps(unsupported)
		found := false
		for _, name := range noEvidence {
			if strings.Contains(strings.ToLower(name), "interrupt") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("TC-010-13 fixture assumption broken: %s no longer lists Interrupt under Unsupported Operations with no captured evidence", relPath)
		}
	}

	resume := f09ResolveResume(t)
	unsupportedPattern := regexp.MustCompile(`(?s)\*\*Interrupt unsupported\.\*\*(.*?)(\n## |\z)`)
	match := unsupportedPattern.FindStringSubmatch(resume)
	if match == nil {
		t.Fatalf("TC-010-13 resume.md must document an \"Interrupt unsupported\" fallback branch; content:\n%s", resume)
	}
	branch := match[1]
	if !strings.Contains(branch, "deadline") {
		t.Fatalf("TC-010-13 resume.md's Interrupt-unsupported branch must describe the deadline-only fallback; branch text:\n%s", branch)
	}
	for _, forbidden := range f09ResumeForbiddenProviderCommandLiterals() {
		if strings.Contains(branch, forbidden) {
			t.Fatalf("TC-010-13 resume.md's Interrupt-unsupported branch must not invoke a concrete provider command literal (found %q)", forbidden)
		}
	}
}

// TestTC010_15CapabilityVectorBaselineAllSupported is TC-010-15: with all
// three capabilities detected, the capability-vector table's row 1
// resolves isolation-eligible topology, same-worker follow-up, and
// interrupt-then-replace — the baseline every other row is compared
// against.
func TestTC010_15CapabilityVectorBaselineAllSupported(t *testing.T) {
	rows := f09ResumeCapabilityVectorTable(t)
	row := rows[0] // row 1: all yes
	if row[1] != "yes" || row[2] != "yes" || row[3] != "yes" {
		t.Fatalf("TC-010-15 capability-vector row 1 = %+v, want isolation=yes/follow-up=yes/interrupt=yes", row)
	}
	segments := f09ParseBehaviorSegments(t, row[4])
	if !strings.Contains(segments["Topology"], "Parallel-with-isolation eligible") {
		t.Fatalf("TC-010-15 row 1 Topology segment = %q, want Parallel-with-isolation eligible", segments["Topology"])
	}
	if !strings.Contains(segments["Follow-up"], "same-worker") {
		t.Fatalf("TC-010-15 row 1 Follow-up segment = %q, want same-worker", segments["Follow-up"])
	}
	if !strings.Contains(segments["Interrupt"], "cancel-then-replace") {
		t.Fatalf("TC-010-15 row 1 Interrupt segment = %q, want cancel-then-replace", segments["Interrupt"])
	}
	if row[5] != "TC-010-15" {
		t.Fatalf("TC-010-15 row 1 Sub-case column = %q, want TC-010-15", row[5])
	}
}

// TestTC010_16CapabilityVectorAllUndetectedProvesIndependentFallbacks is
// TC-010-16 (codex red-team BLOCKER fix): with all three capabilities
// undetected, all three fallbacks apply simultaneously and none masks
// another — proven by showing row 5's Topology segment matches row 2's
// (isolation-only-undetected) Topology segment, row 5's Follow-up segment
// matches row 3's (follow-up-only-undetected), and row 5's Interrupt
// segment matches row 4's (interrupt-only-undetected). An implementation
// that only checks one flag and assumes the others follow from it would
// fail at least one of these three cross-row equalities.
func TestTC010_16CapabilityVectorAllUndetectedProvesIndependentFallbacks(t *testing.T) {
	rows := f09ResumeCapabilityVectorTable(t)
	row1 := rows[0]
	row2 := rows[1]
	row3 := rows[2]
	row4 := rows[3]
	row5 := rows[4]

	if row5[1] != "no" || row5[2] != "no" || row5[3] != "no" {
		t.Fatalf("TC-010-16 capability-vector row 5 = %+v, want isolation=no/follow-up=no/interrupt=no", row5)
	}
	if row5[5] != "TC-010-16" {
		t.Fatalf("TC-010-16 row 5 Sub-case column = %q, want TC-010-16", row5[5])
	}

	seg1 := f09ParseBehaviorSegments(t, row1[4])
	seg2 := f09ParseBehaviorSegments(t, row2[4])
	seg3 := f09ParseBehaviorSegments(t, row3[4])
	seg4 := f09ParseBehaviorSegments(t, row4[4])
	seg5 := f09ParseBehaviorSegments(t, row5[4])

	if seg5["Topology"] != seg2["Topology"] {
		t.Fatalf("TC-010-16 row 5 Topology segment %q must match row 2's (isolation-only-undetected) Topology segment %q — the isolation fallback must apply the same way whether it is the only undetected capability or one of three", seg5["Topology"], seg2["Topology"])
	}
	if seg5["Follow-up"] != seg3["Follow-up"] {
		t.Fatalf("TC-010-16 row 5 Follow-up segment %q must match row 3's (follow-up-only-undetected) Follow-up segment %q", seg5["Follow-up"], seg3["Follow-up"])
	}
	if seg5["Interrupt"] != seg4["Interrupt"] {
		t.Fatalf("TC-010-16 row 5 Interrupt segment %q must match row 4's (interrupt-only-undetected) Interrupt segment %q", seg5["Interrupt"], seg4["Interrupt"])
	}

	// And row 5 must differ from the all-supported baseline in all three
	// segments at once — proving no single fallback masks another.
	if seg5["Topology"] == seg1["Topology"] || seg5["Follow-up"] == seg1["Follow-up"] || seg5["Interrupt"] == seg1["Interrupt"] {
		t.Fatalf("TC-010-16 row 5 must differ from row 1's all-supported baseline in every segment; row1=%+v row5=%+v", seg1, seg5)
	}
}

// ---------------------------------------------------------------------------
// TC-010-01..04, 07, 14, 17..20 (test-plan.md #tc-010) — T-E38-F09-016. These
// complete the resume lifecycle's silent-responder escalation ladder (ping
// once -> interrupt-or-wait -> hard deadline stop, AC-012's resume-half) and
// the handoff provenance guard (AC-024's resume-half): the bounded handoff
// never carries rendered-prompt content, and no handoff/decision/note
// template anywhere under skills/shark-attack/** declares a prompt-
// persistence placeholder or field. F09 ships no dispatcher Go code (D-002),
// so every assertion below is content/fixture level against the installed
// skill tree via readF09EmbeddedFile/f09WalkEmbeddedSharkAttackMarkdown,
// matching this file's existing TC-003/TC-005/TC-010 conventions.
// ---------------------------------------------------------------------------

// f09ResumeResponderOutcomeLadderTable parses resume.md's 5-row
// responder-outcome ladder table (columns: #, Responder behavior, Expected
// result, Sub-case). "| Responder behavior |" (rather than "| # |", which
// also matches the capability-vector table earlier in the file) is the
// substring that uniquely anchors this table.
func f09ResumeResponderOutcomeLadderTable(t *testing.T) [][]string {
	t.Helper()
	rows := f09ParseMarkdownTable(t, f09ResolveResume(t), "| Responder behavior |")
	if len(rows) != 5 {
		t.Fatalf("TC-010 resume.md responder-outcome ladder table has %d rows, want the full 5-row table", len(rows))
	}
	return rows
}

// TestTC010_01SilentResponderPingedExactlyOnceBeforeEscalating is
// TC-010-01: a silent responder is pinged exactly once (a documented step),
// and that ping precedes the interrupt-or-wait choice.
func TestTC010_01SilentResponderPingedExactlyOnceBeforeEscalating(t *testing.T) {
	content := f09ResolveResume(t)
	normalized := strings.Join(strings.Fields(content), " ")

	pingIdx := strings.Index(normalized, "the parent pings once")
	if pingIdx == -1 {
		t.Fatalf("TC-010-01 resume.md must document that the parent pings the silent responder once; content:\n%s", normalized)
	}
	for _, want := range []string{
		"sends at most one ping per responder",
		"never mints a second",
		"never pinged at all",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("TC-010-01 resume.md must state %q; content:\n%s", want, normalized)
		}
	}

	chooseIdx := strings.Index(normalized, "Whether the parent then cancels the stale consultation before routing a replacement responder, or simply waits to the deadline, depends on the interrupt capability discovered above")
	if chooseIdx == -1 {
		t.Fatalf("TC-010-01 resume.md must document the post-ping interrupt-or-wait choice; content:\n%s", normalized)
	}
	if chooseIdx < pingIdx {
		t.Fatalf("TC-010-01 resume.md describes the interrupt-or-wait choice (offset %d) before describing the ping (offset %d) — the ping must come first", chooseIdx, pingIdx)
	}
}

// TestTC010_02NoResponseAfterPingInterruptsOrWaitsPerCapability is
// TC-010-02: once the ping goes unanswered, the parent interrupts and
// replaces where interrupt is supported, else waits to the deadline only —
// and both named branches exist.
func TestTC010_02NoResponseAfterPingInterruptsOrWaitsPerCapability(t *testing.T) {
	content := f09ResolveResume(t)
	for _, want := range []string{"Interrupt supported", "Interrupt unsupported", "deadline-only fallback"} {
		if !strings.Contains(content, want) {
			t.Fatalf("TC-010-02 resume.md must document %q; content:\n%s", want, content)
		}
	}
}

// TestTC010_03ExactlyOneReplacementResponderRouted is TC-010-03: at most one
// replacement responder is ever routed for a stale consultation — asserted
// both by the single documented occurrence of the routing statement and by
// the explicit absence of any "more than one" phrasing.
func TestTC010_03ExactlyOneReplacementResponderRouted(t *testing.T) {
	content := f09ResolveResume(t)
	const want = "Route exactly one replacement responder"
	if count := strings.Count(content, want); count != 1 {
		t.Fatalf("TC-010-03 resume.md states %q %d times, want exactly 1 occurrence — more than one would suggest a second, competing replacement path", want, count)
	}
	lower := strings.ToLower(content)
	for _, forbidden := range []string{"route two replacement", "a second replacement responder", "multiple replacement responders"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("TC-010-03 resume.md must not describe routing more than one replacement responder (found %q)", forbidden)
		}
	}
}

// TestTC010_04DeadlineStopRunsOrderedNonOptionalSequence is TC-010-04: once
// the consultation deadline passes with no answer recorded, the parent
// stops write workers, records a bounded unresolved handoff, records a
// blocker, and releases the lease — all four, in that order, as one
// non-optional sequence (not "some of the above").
func TestTC010_04DeadlineStopRunsOrderedNonOptionalSequence(t *testing.T) {
	content := f09ResolveResume(t)
	sectionPattern := regexp.MustCompile(`(?s)### Deadline stop\n(.*?)\n### `)
	match := sectionPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-04 resume.md must have a \"### Deadline stop\" section; content:\n%s", content)
	}
	section := match[1]

	steps := []string{
		"Stop the dispatched entity's write worker(s).",
		"Record a bounded unresolved handoff",
		"Record a blocker against the dispatched entity.",
		"Release the dispatched entity's lease.",
	}
	prevIdx := -1
	for _, step := range steps {
		idx := strings.Index(section, step)
		if idx == -1 {
			t.Fatalf("TC-010-04 resume.md's \"### Deadline stop\" section must document step %q; section:\n%s", step, section)
		}
		if idx <= prevIdx {
			t.Fatalf("TC-010-04 step %q appears at offset %d, not after the previous step's offset %d — the four deadline-stop actions must be documented as one ordered sequence", step, idx, prevIdx)
		}
		prevIdx = idx
	}
	if !strings.Contains(section, "non-optional") {
		t.Fatalf("TC-010-04 resume.md's \"### Deadline stop\" section must state the sequence is non-optional; section:\n%s", section)
	}
}

// TestTC010_07HandoffCarriesRequiredFieldsAndExcludesRenderedPrompt is
// TC-010-07: the resume-path handoff carries entity key, question, answer,
// and evidence pointers, and explicitly excludes rendered-prompt content —
// grepping the documented handoff schema for a prompt/transcript field and
// asserting absence, with a negative-case fixture proving the check has
// teeth.
func TestTC010_07HandoffCarriesRequiredFieldsAndExcludesRenderedPrompt(t *testing.T) {
	content := f09ResolveResume(t)
	sectionPattern := regexp.MustCompile(`(?s)### Handoff content\n(.*?)\n## `)
	match := sectionPattern.FindStringSubmatch(content)
	if match == nil {
		t.Fatalf("TC-010-07 resume.md must have a \"### Handoff content\" section; content:\n%s", content)
	}
	section := strings.Join(strings.Fields(match[1]), " ")

	for _, want := range []string{
		"the dispatched entity's key",
		"the consultation question",
		"the recorded answer",
		"evidence pointers",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("TC-010-07 resume.md's handoff schema must carry %q; section:\n%s", want, section)
		}
	}

	const exclusionPhrase = "excludes rendered prompt content"
	if !strings.Contains(section, exclusionPhrase) {
		t.Fatalf("TC-010-07 resume.md's handoff schema must state %q; section:\n%s", exclusionPhrase, section)
	}
	for _, forbidden := range []string{"prompt:", "rendered_prompt", "transcript:", "{{prompt}}"} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("TC-010-07 resume.md's handoff schema must not declare a prompt/transcript field (found %q); section:\n%s", forbidden, section)
		}
	}

	// Negative case: prove the exclusion check has teeth by stripping the
	// exclusion phrase and confirming the same check would then fail.
	badFixture := strings.Replace(content, exclusionPhrase, "omits nothing of note", 1)
	if badFixture == content {
		t.Fatalf("TC-010-07 negative-case fixture construction did not remove the exclusion phrase — cannot prove the check has teeth")
	}
	badMatch := sectionPattern.FindStringSubmatch(badFixture)
	if badMatch == nil {
		t.Fatalf("TC-010-07 negative-case fixture broke the \"### Handoff content\" section boundary")
	}
	badSection := strings.Join(strings.Fields(badMatch[1]), " ")
	if strings.Contains(badSection, exclusionPhrase) {
		t.Fatalf("TC-010-07 negative-case fixture still contains the exclusion phrase — construction failed")
	}
}

// TestTC010_17AnswersBeforeAnyPingFastPath is TC-010-17 (responder-outcome
// ladder row 1): a responder that answers before any ping is sent has its
// answer recorded normally, with no ping documented for this path.
func TestTC010_17AnswersBeforeAnyPingFastPath(t *testing.T) {
	rows := f09ResumeResponderOutcomeLadderTable(t)
	row := rows[0]
	if row[3] != "TC-010-17" {
		t.Fatalf("TC-010-17 ladder row 1 Sub-case column = %q, want TC-010-17", row[3])
	}
	if !strings.Contains(row[1], "Answers before any ping") {
		t.Fatalf("TC-010-17 ladder row 1 Responder-behavior column = %q, want the fast-path description", row[1])
	}
	for _, want := range []string{"No ping sent", "recorded normally"} {
		if !strings.Contains(row[2], want) {
			t.Fatalf("TC-010-17 ladder row 1 Expected-result column = %q, want it to state %q", row[2], want)
		}
	}
}

// TestTC010_18SilentThenAnswersAfterPingNoReplacementRouted is TC-010-18
// (responder-outcome ladder row 2): a responder that answers after the one
// ping (but before any replacement is routed) has its answer recorded from
// the original responder, and no replacement is routed.
func TestTC010_18SilentThenAnswersAfterPingNoReplacementRouted(t *testing.T) {
	rows := f09ResumeResponderOutcomeLadderTable(t)
	row := rows[1]
	if row[3] != "TC-010-18" {
		t.Fatalf("TC-010-18 ladder row 2 Sub-case column = %q, want TC-010-18", row[3])
	}
	for _, want := range []string{"No replacement routed", "original responder"} {
		if !strings.Contains(row[2], want) {
			t.Fatalf("TC-010-18 ladder row 2 Expected-result column = %q, want it to state %q", row[2], want)
		}
	}
}

// TestTC010_19SilentThroughPingReplacementAnswersOriginalClosedOut is
// TC-010-19 (responder-outcome ladder row 3): once a replacement responder
// is routed and answers, the replacement's answer is recorded and the
// original responder's pending state is explicitly closed out, not left
// dangling.
func TestTC010_19SilentThroughPingReplacementAnswersOriginalClosedOut(t *testing.T) {
	rows := f09ResumeResponderOutcomeLadderTable(t)
	row := rows[2]
	if row[3] != "TC-010-19" {
		t.Fatalf("TC-010-19 ladder row 3 Sub-case column = %q, want TC-010-19", row[3])
	}
	for _, want := range []string{"Replacement's answer is recorded", "closed out", "not left dangling"} {
		if !strings.Contains(row[2], want) {
			t.Fatalf("TC-010-19 ladder row 3 Expected-result column = %q, want it to state %q", row[2], want)
		}
	}
}

// TestTC010_20AnswersAtDeadlineBoundaryStatesExplicitRule is TC-010-20
// (responder-outcome ladder row 5, a BVA edge case): the procedure must
// state an explicit, unambiguous rule for an answer landing exactly at the
// deadline instant — either "counts" or "too late" — rather than leaving
// the boundary silently unresolved. Asserting exactly one of the two rule
// markers is present (not zero, not both) is what would catch a doc edit
// that quietly drops the boundary rule.
func TestTC010_20AnswersAtDeadlineBoundaryStatesExplicitRule(t *testing.T) {
	rows := f09ResumeResponderOutcomeLadderTable(t)
	row := rows[4]
	if row[3] != "TC-010-20" {
		t.Fatalf("TC-010-20 ladder row 5 Sub-case column = %q, want TC-010-20", row[3])
	}
	if !strings.Contains(row[1], "exactly the deadline boundary") {
		t.Fatalf("TC-010-20 ladder row 5 Responder-behavior column = %q, want the deadline-boundary BVA description", row[1])
	}
	counts := strings.Contains(row[2], "Counts as answered")
	tooLate := strings.Contains(strings.ToLower(row[2]), "ruled too late") || strings.Contains(strings.ToLower(row[2]), "treated as too late")
	if counts == tooLate {
		t.Fatalf("TC-010-20 ladder row 5 Expected-result column must state exactly one explicit rule (\"counts\" or \"too late\"), not leave the deadline boundary ambiguous; column: %q", row[2])
	}
}

// f09ProhibitedPromptPersistenceMarkers is the set of concrete
// prompt-persistence placeholders TC-010-14 forbids anywhere under
// skills/shark-attack/**.
func f09ProhibitedPromptPersistenceMarkers() []string {
	return []string{
		"{{prompt}}", "{{ prompt }}",
		"{{rendered_prompt}}", "{{ rendered_prompt }}",
	}
}

// f09PromptFieldNamePattern matches a YAML/markdown field declaration named
// "prompt:" or "rendered_prompt:" at the start of a line — the field-name
// half of TC-010-14's "placeholder or field name suggesting prompt
// persistence" check.
var f09PromptFieldNamePattern = regexp.MustCompile(`(?im)^\s*(rendered_)?prompt\s*:`)

// TestTC010_14NoTemplateCarriesAPromptPlaceholderOrField is TC-010-14
// (AC-024's resume-half): across every documented handoff/decision/note
// template under skills/shark-attack/**, none contains a
// `{{prompt}}`/`{{rendered_prompt}}`-style placeholder or a `prompt:`/
// `rendered_prompt:` field name. A negative-case fixture (an injected
// marker) proves the sweep would actually catch a real leak, not just
// rubber-stamp a corpus that happens to be clean today.
func TestTC010_14NoTemplateCarriesAPromptPlaceholderOrField(t *testing.T) {
	var violations []string
	for _, rel := range f09WalkEmbeddedSharkAttackMarkdown(t) {
		content := readF09EmbeddedFile(t, rel)
		for _, marker := range f09ProhibitedPromptPersistenceMarkers() {
			if strings.Contains(content, marker) {
				violations = append(violations, rel+": contains placeholder "+marker)
			}
		}
		if loc := f09PromptFieldNamePattern.FindString(content); loc != "" {
			violations = append(violations, rel+": contains field name "+strings.TrimSpace(loc))
		}
	}
	if len(violations) > 0 {
		t.Fatalf("TC-010-14 no handoff/decision/note template may declare a rendered-prompt placeholder or field; violations:\n%s", strings.Join(violations, "\n"))
	}

	// Negative case: inject a placeholder and a field name into a real
	// template's content and confirm the same checks would catch both.
	resume := f09ResolveResume(t)
	poisoned := strings.Replace(resume, "### Handoff content", "### Handoff content\n\nrendered_prompt: {{prompt}}\n", 1)
	if poisoned == resume {
		t.Fatalf("TC-010-14 negative-case fixture construction failed to inject a marker into resume.md — cannot prove the sweep has teeth")
	}
	poisonedHasPlaceholder := false
	for _, marker := range f09ProhibitedPromptPersistenceMarkers() {
		if strings.Contains(poisoned, marker) {
			poisonedHasPlaceholder = true
			break
		}
	}
	if !poisonedHasPlaceholder {
		t.Fatalf("TC-010-14 sweep would not have caught an injected {{prompt}} placeholder — the placeholder check does not have teeth")
	}
	if f09PromptFieldNamePattern.FindString(poisoned) == "" {
		t.Fatalf("TC-010-14 sweep would not have caught an injected rendered_prompt: field — the field-name check does not have teeth")
	}
}

// ---------------------------------------------------------------------------
// TC-001-01..03 (test-plan.md #tc-001, REQ-F-007, AC-010/AC-011) —
// T-E38-F09-015. Parent-owned claim/heartbeat invariants during a live
// consultation: a heartbeat renewed on the dispatched entity's own lease is
// silent (AC-010), and simulated loss of the Question's routing lease
// mid-consultation blocks the in-flight answer and the dispatched entity's
// status advance until a fresh keyed dispatch and a fresh claim recover it
// (AC-011). This drives the real Shark CLI end-to-end against a temp SQLite
// DB via the compiled binary, reusing TC-004's real-DB CLI seam per this
// task's Brownfield Context (buildSharkF09, defined above for TC-002-09).
// ---------------------------------------------------------------------------

// tc001FeatureSnapshot captures the dispatched feature's own status plus
// every entity_history row recorded against it, concatenated into one
// deterministic string. TC-001-01 compares this before and after a
// heartbeat call: Renew (claim_repository.go:101) only ever updates
// entity_claims, so an identical snapshot is the byte-level proof AC-010
// requires, not just "the heartbeat command didn't error."
func tc001FeatureSnapshot(t *testing.T, sqlDB *sql.DB, featureID int64) string {
	t.Helper()
	var status string
	if err := sqlDB.QueryRow(`SELECT status FROM features WHERE id = ?`, featureID).Scan(&status); err != nil {
		t.Fatalf("TC-001 snapshot feature status: %v", err)
	}
	rows, err := sqlDB.Query(`
		SELECT id, COALESCE(from_status, ''), to_status, COALESCE(changed_by, ''), COALESCE(notes, ''), forced, COALESCE(rejection_reason, ''), changed_at
		FROM entity_history WHERE entity_type = 'feature' AND entity_id = ? ORDER BY id`, featureID)
	if err != nil {
		t.Fatalf("TC-001 snapshot feature history: %v", err)
	}
	defer rows.Close()
	var b strings.Builder
	b.WriteString("status=")
	b.WriteString(status)
	for rows.Next() {
		var id int64
		var from, to, by, notes, reason, at string
		var forced int
		if err := rows.Scan(&id, &from, &to, &by, &notes, &forced, &reason, &at); err != nil {
			t.Fatalf("TC-001 scan feature history row: %v", err)
		}
		fmt.Fprintf(&b, "|%d:%s>%s:by=%s:notes=%s:forced=%d:reason=%s:at=%s", id, from, to, by, notes, forced, reason, at)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("TC-001 iterate feature history: %v", err)
	}
	return b.String()
}

// TestTC001_01to03ParentOwnedClaimHeartbeatInvariantsDuringConsultation
// drives TC-001-01, TC-001-02, and TC-001-03 as one ordered narrative — each
// sub-case depends on the durable state the previous one left behind, the
// same reason TestTC004_01to13X06ConsumerActivationFullLifecycleMintThroughResolve
// is one function rather than three independent ones.
func TestTC001_01to03ParentOwnedClaimHeartbeatInvariantsDuringConsultation(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tc001-parent-owned-loop.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-001 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "TC-001 parent-owned loop epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-001 seed epic: %v", err)
	}
	dispatched := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "TC-001 dispatched entity"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, dispatched); err != nil {
		t.Fatalf("TC-001 seed dispatched feature: %v", err)
	}

	binary := buildSharkF09(t)
	projectRoot := f09ProjectRoot(t)
	runShark := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
		cmd.Dir = projectRoot
		out, runErr := cmd.CombinedOutput()
		return string(out), runErr
	}
	runOK := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr != nil {
			t.Fatalf("TC-001 %s: shark %s failed: %v\n%s", label, strings.Join(args, " "), runErr, out)
		}
		return out
	}
	runFail := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr == nil {
			t.Fatalf("TC-001 %s: shark %s succeeded, want rejection\n%s", label, strings.Join(args, " "), out)
		}
		return out
	}

	// Precondition: "A dispatched entity is claimed by the parent" — the
	// parent claims E01-F01 for the normal dispatch loop before the worker
	// ever returns a kind:question control envelope.
	claimEntityOut := runOK("precondition claim dispatched entity", "--json", "claim", "E01-F01", "--by", "parent", "--session", "parent-session")
	var claimedEntity struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(claimEntityOut), &claimedEntity); err != nil || claimedEntity.SessionID != "parent-session" {
		t.Fatalf("TC-001 precondition claim = %s, decode error = %v, want session_id parent-session", claimEntityOut, err)
	}

	// Precondition: "a question_blocks Question is open" — mint, configure,
	// and link Q001 exactly as route-question.md's Mint/Configure/Gate
	// sections document.
	createOut := runOK("precondition create Q001", "--json", "question", "create", "TC-001 consultation question", "--summary", "Choose the parent-owned loop outcome", "--requester", "release-owner", "--blocking")
	var created struct {
		Key    string `json:"key"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(createOut), &created); err != nil || created.Key != "Q001" || created.Status != "draft" {
		t.Fatalf("TC-001 precondition create = %s, decode error = %v, want Q001/draft", createOut, err)
	}
	configureOut := runOK("precondition configure Q001", "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice")
	var configured struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(configureOut), &configured); err != nil || configured.Status != "open" {
		t.Fatalf("TC-001 precondition configure = %s, decode error = %v, want status open", configureOut, err)
	}
	runOK("precondition link question_blocks", "--json", "link", "Q001", "E01-F01", "--type", "question_blocks")

	// Confirm the gate actually qualifies before relying on it below: the
	// blocked entity's own keyed next pauses with a compact question_block.
	gatedNextOut := runOK("precondition next E01-F01 gated", "--json", "next", "E01-F01")
	var gated struct {
		Action        string                  `json:"action"`
		QuestionBlock *services.QuestionBlock `json:"question_block"`
	}
	if err := json.Unmarshal([]byte(gatedNextOut), &gated); err != nil || gated.Action != "pause" || gated.QuestionBlock == nil || gated.QuestionBlock.QuestionKey != "Q001" {
		t.Fatalf("TC-001 precondition gated next = %s, decode error = %v, want pause/Q001 question_block", gatedNextOut, err)
	}

	// TC-001-01: a heartbeat renewed mid-consultation on the dispatched
	// entity's own lease is silent — the entity's status and its
	// entity_history rows are byte-unchanged (AC-010).
	snapshotBeforeHeartbeat := tc001FeatureSnapshot(t, sqlDB, dispatched.ID)
	runOK("TC-001-01 heartbeat dispatched entity", "--json", "heartbeat", "E01-F01", "--session", "parent-session", "--progress", "0.4", "--note", "TC-001-01 mid-consultation heartbeat")
	snapshotAfterHeartbeat := tc001FeatureSnapshot(t, sqlDB, dispatched.ID)
	if snapshotBeforeHeartbeat != snapshotAfterHeartbeat {
		t.Fatalf("TC-001-01 heartbeat mutated the dispatched entity's status/history:\nbefore: %s\nafter:  %s", snapshotBeforeHeartbeat, snapshotAfterHeartbeat)
	}

	// Route the consultation to alice — route-question.md's Route section:
	// next Q### names the pending responder, then the parent claims Q###
	// on alice's behalf under a session it tracks (separate from the
	// dispatched entity's own parent-session lease above).
	questionNextOut := runOK("route next Q001", "--json", "next", "Q001")
	var questionNext commands.NextResponse
	if err := json.Unmarshal([]byte(questionNextOut), &questionNext); err != nil || questionNext.EntityKey != "Q001" || questionNext.Action != "spawn_agent" {
		t.Fatalf("TC-001 route next Q001 = %s, decode error = %v, want Q001/spawn_agent", questionNextOut, err)
	}
	if !strings.Contains(questionNext.Prompt, "currently routed responder: alice") {
		t.Fatalf("TC-001 route next Q001 prompt = %q, want it to name alice as the current pending responder", questionNext.Prompt)
	}
	runOK("route claim Q001 as alice", "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session-1")

	// TC-001-02: simulated lease loss. A concurrent --force claim replaces
	// Q001's live session out from under the parent — exactly what an
	// expired lease reclaimed and reissued to someone else would look like
	// from the parent's point of view. The next attempt to record alice's
	// answer under the now-stale alice-session-1 is rejected, reusing
	// RecordResponse's claim.SessionID == input.SessionID check
	// (question_workflow_service.go:116) — the same check TC-004-11 already
	// exercises with a merely-wrong string, but here the claim really was
	// live and really was lost, not just guessed wrong.
	runOK("TC-001-02 simulate lease loss", "--json", "claim", "Q001", "--by", "reclaimer", "--session", "reclaimer-session", "--force")
	staleRespondOut := runFail("TC-001-02 stale-session respond after lease loss", "--json", "question", "respond", "Q001", "--session", "alice-session-1", "--responder", "alice", "--summary", "alice's answer under a lease that was just lost", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	if !strings.Contains(staleRespondOut, "active claim does not match responder session") {
		t.Fatalf("TC-001-02 stale-session respond output = %s, want rejection naming the claim/session mismatch", staleRespondOut)
	}

	// TC-001-03: after lease loss, no council answer was delivered — Q001
	// carries zero responses — and the dispatched entity's status advance is
	// still rejected, because the Question the gate depends on never
	// resolved.
	readResponseCount := func(label string) int {
		t.Helper()
		out := runOK(label, "--json", "question", "full", "Q001", "--actor", "release-owner")
		var full struct {
			Responses []struct {
				Responder string `json:"responder"`
			} `json:"responses"`
		}
		if err := json.Unmarshal([]byte(out), &full); err != nil {
			t.Fatalf("%s decode: %v\n%s", label, err, out)
		}
		return len(full.Responses)
	}
	if got := readResponseCount("TC-001-03 full after lease loss"); got != 0 {
		t.Fatalf("TC-001-03 Q001 has %d responses after lease loss, want 0 — no council answer must be delivered under lost authority", got)
	}
	advanceRejectedOut := runFail("TC-001-03 advance after lease loss", "--json", "status", "advance", "E01-F01")
	if !strings.Contains(advanceRejectedOut, `"code": "QUESTION_BLOCKED"`) {
		t.Fatalf("TC-001-03 advance-after-lease-loss output = %s, want QUESTION_BLOCKED", advanceRejectedOut)
	}

	// The dispatched entity's own lease survived the whole ordeal untouched
	// — it is a separate claim from Q001's, exactly as route-question.md's
	// Route section documents, so it is still parent-session, not reclaimer
	// or alice's.
	var dispatchedSessionAfterLeaseLoss string
	if err := sqlDB.QueryRow(`SELECT session_id FROM entity_claims WHERE entity_type = 'feature' AND entity_key = 'E01-F01'`).Scan(&dispatchedSessionAfterLeaseLoss); err != nil {
		t.Fatalf("TC-001-03 read dispatched entity claim: %v", err)
	}
	if dispatchedSessionAfterLeaseLoss != "parent-session" {
		t.Fatalf("TC-001-03 dispatched entity claim session = %q after Q001's lease was lost, want unchanged parent-session — the two leases are separate concerns", dispatchedSessionAfterLeaseLoss)
	}

	// Recovery: merely releasing the stolen claim does not restore
	// authority — a stale session still cannot respond once no claim
	// matches it at all (REQ-F-010: "the handoff supplies context but never
	// restores authority").
	runOK("TC-001-03 release stolen claim", "--json", "release", "Q001", "--force")
	staleAfterReleaseOut := runFail("TC-001-03 stale session still rejected after release", "--json", "question", "respond", "Q001", "--session", "alice-session-1", "--responder", "alice", "--summary", "retrying with the old session after release", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	if !strings.Contains(staleAfterReleaseOut, "active claim does not match responder session") {
		t.Fatalf("TC-001-03 stale-session-after-release output = %s, want rejection — release alone does not restore authority", staleAfterReleaseOut)
	}

	// Recovery proper: a fresh keyed dispatch, then a fresh claim, is what
	// restores authority (AC-011, REQ-F-010).
	freshNextOut := runOK("TC-001-03 fresh keyed next Q001", "--json", "next", "Q001")
	var freshNext commands.NextResponse
	if err := json.Unmarshal([]byte(freshNextOut), &freshNext); err != nil || freshNext.EntityKey != "Q001" || freshNext.Action != "spawn_agent" {
		t.Fatalf("TC-001-03 fresh next Q001 = %s, decode error = %v, want a fresh Q001/spawn_agent dispatch", freshNextOut, err)
	}
	if !strings.Contains(freshNext.Prompt, "currently routed responder: alice") {
		t.Fatalf("TC-001-03 fresh next Q001 prompt = %q, want it to still name alice as the pending responder", freshNext.Prompt)
	}
	runOK("TC-001-03 fresh claim Q001 as alice", "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session-2")
	recoveredRespondOut := runOK("TC-001-03 recovered respond", "--json", "question", "respond", "Q001", "--session", "alice-session-2", "--responder", "alice", "--summary", "alice's answer delivered after recovery", "--evidence-pointer", "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md")
	var afterRecoveredRespond struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(recoveredRespondOut), &afterRecoveredRespond); err != nil || afterRecoveredRespond.Status != "ready_for_resolution" {
		t.Fatalf("TC-001-03 recovered respond = %s, decode error = %v, want status ready_for_resolution", recoveredRespondOut, err)
	}
	if got := readResponseCount("TC-001-03 full after recovery"); got != 1 {
		t.Fatalf("TC-001-03 Q001 has %d responses after recovery, want exactly 1", got)
	}
	runOK("TC-001-03 release recovered claim", "--json", "release", "Q001", "--session", "alice-session-2", "--outcome", "pass")
	runOK("TC-001-03 resolve Q001", "--json", "question", "resolve", "Q001", "--owner", "release-owner", "--resolution-kind", "no_lasting_consequence")
	runOK("TC-001-03 advance after recovery", "--json", "status", "advance", "E01-F01")
}

// ---------------------------------------------------------------------------
// TC-003-13 (test-plan.md #tc-003, "added during task-decomposition rework —
// closes T-017's I-10 orphan pointer gap") — T-E38-F09-017.
//
// skills/shark-rider/context/host-adapter-contract.md has no embedded mirror
// (skills/shark-rider/** is never rendered from the embedded bundle — see
// this file's header comment), so it is read via readF09RepositoryFile, the
// declared real-filesystem convention for that tree, matching
// readF07RepositoryFile in e38_f07_interactions_test.go. The cross-check
// reads internal/cli/commands/next.go's actual JSON struct tags the same
// way, so a rename on either side (e.g. host-adapter-contract.md drifting to
// "prompt_hash" while next.go keeps "prompt_sha256") fails this test instead
// of silently shipping a mismatched I-10 pointer for F10/F11.
// ---------------------------------------------------------------------------

func TestTC003_13HostAdapterContractFieldsMatchNextGoWireShapeVerbatim(t *testing.T) {
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	nextGoSource := readF09RepositoryFile(t, "internal/cli/commands/next.go")

	// next.go is the wire-shape source of truth TC-002 already proves at
	// runtime (TestTC002_09PromptOutRealWritePathByteExactness). Pin its own
	// JSON tags here so a future rename of either field is caught even if
	// nobody remembers to update this test's literals.
	for _, wireTag := range []string{
		`json:"prompt_sha256,omitempty"`,
		`json:"prompt_bytes,omitempty"`,
	} {
		if !strings.Contains(nextGoSource, wireTag) {
			t.Fatalf("TC-003-13 internal/cli/commands/next.go no longer declares %s — update this test's cross-check target, not just host-adapter-contract.md", wireTag)
		}
	}

	// The provenance field names in the adapter contract must match next.go's
	// wire fields verbatim — no rename, no duplicate shape.
	for _, field := range []string{"prompt_sha256", "prompt_bytes"} {
		if !strings.Contains(contract, field) {
			t.Errorf("TC-003-13 host-adapter-contract.md omits wire field %q (must match next.go's NextResponse verbatim)", field)
		}
	}

	// The request shape is exactly shark next --json's wire fields — the
	// adapter receives them unchanged.
	for _, field := range []string{
		"entity_key", "entity_type", "status", "action", "agent_type",
		"provider", "model", "prompt",
	} {
		if !strings.Contains(contract, field) {
			t.Errorf("TC-003-13 host-adapter-contract.md omits provider-neutral request field %q", field)
		}
	}

	// The result shape is the worker's control envelope — declared once in
	// worker-control-schema.yaml; this file must point at it, not redeclare
	// a second kind vocabulary.
	if !strings.Contains(contract, "worker-control-schema.yaml") {
		t.Error("TC-003-13 host-adapter-contract.md must point at context/worker-control-schema.yaml for the result envelope shape rather than redeclaring it")
	}
	for _, kind := range []string{"final", "question", "needs_council", "blocked_external", "failed"} {
		if !strings.Contains(contract, kind) {
			t.Errorf("TC-003-13 host-adapter-contract.md omits control-envelope kind %q", kind)
		}
	}
}

// ---------------------------------------------------------------------------
// Rider-path half of TC-001/TC-003/I-07 (test-plan.md Integration Scenarios
// "Keyed dispatch → prompt provenance → Rider dispatch") — T-E38-F09-017.
//
// run.md's exact-transport section is extended, not restructured (D-002: no
// runner change) — these assertions pin only the new content T-017 adds
// around the untouched e38_f07_interactions_test.go pins (:42,57,107,133).
// ---------------------------------------------------------------------------

func TestTC017_RunMdExtendsProvenanceQuestionRoutingHeartbeatAndResumeAdapter(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")

	// Prompt-hash provenance (REQ-F-011, I-10) is surfaced on the wire
	// fields the loop already parses.
	for _, want := range []string{
		"response.prompt_sha256",
		"response.prompt_bytes",
		"--prompt-out",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("TC-017 run.md omits prompt-hash provenance content %q", want)
		}
	}

	// The question control-envelope path (REQ-F-004, I-07) routes around
	// F07's loop, never inside a second one — it must link to
	// route-question.md and council.md, not restate their procedures.
	for _, want := range []string{
		"worker-control-schema.yaml",
		"kind: question",
		"kind: needs_council",
		"route-question.md",
		"council.md",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("TC-017 run.md omits question-routing content %q", want)
		}
	}

	// Heartbeat-during-consultation (AC-010) keeps the dispatched entity's
	// own lease alive for the whole consultation window.
	if !strings.Contains(run, "Keep heartbeating the dispatched entity") && !strings.Contains(run, "heartbeating the dispatched entity's own lease") {
		t.Error("TC-017 run.md must instruct the parent to keep heartbeating the dispatched entity's lease during a consultation")
	}

	// Same-worker/replacement resume adapter contract (REQ-F-008, I-10)
	// points at the new host-adapter-contract.md and reuses resume.md.
	for _, want := range []string{
		"host-adapter-contract.md",
		"resume.md",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("TC-017 run.md omits resume-adapter content %q", want)
		}
	}

	// This task's own additions must not introduce a second `shark status
	// set` occurrence beyond the one pre-existing kickback-application use
	// (run.md's "Apply task kickbacks" step) — T-026 owns the full-corpus
	// zero-occurrence grep sweep across skills/shark-attack/workflows/ and
	// this file together; that pre-existing occurrence is out of this
	// task's scope to remove.
	if got := strings.Count(run, "shark status set"); got > 1 {
		t.Errorf("TC-017 run.md contains %d `shark status set` occurrences, want at most the 1 pre-existing kickback-application use — this task's additions must not introduce a new one", got)
	}
}

// ---------------------------------------------------------------------------
// TC-003-03b, TC-009-01..03 (test-plan.md #tc-003, #tc-009; AC-006; REQ-F-018)
// — T-E38-F09-026. Two cross-cutting static prose/import guards spanning both
// producers this task's dependencies create: T-024's restructured
// skills/shark-attack/ tree and T-017's skills/shark-rider/verbs/run.md.
// Per the task spec's declared TDD-structure exception, this is a bounded
// feature-level cross-component prose-guard suite (content-only scan + a
// static import/symbol-absence check, no new runtime code), checkable only
// once both producers exist — it cannot merge into either task's own scope.
// ---------------------------------------------------------------------------

// f09RiderStatusSetMarker is the literal invocation TC-003-03b's grep
// targets (AC-006: "shark status set ... appears on no Rider path").
const f09RiderStatusSetMarker = "shark status set"

// f09KnownRiderKickbackStatusSetExceptionCount is TC-003-03b's honestly
// documented gap in AC-006's literal "appears on no Rider path" claim:
// run.md's "Apply task kickbacks" step (test-plan.md TC-004-06's own fixture
// target) applies a --force status set to a KICKED-BACK SIBLING task —
// never the entity currently dispatched — as the deliberately documented
// human escape hatch (D-006, AC-006's own second sentence). T-E38-F09-017's
// TestTC017_RunMdExtendsProvenanceQuestionRoutingHeartbeatAndResumeAdapter
// already pins this single occurrence as "at most 1" for run.md alone and
// explicitly delegates "the full-corpus zero-occurrence grep sweep" to this
// task. Rather than either silently asserting a "zero" claim production does
// not meet, or leaving the gap unrecorded, this test treats the count as a
// bounded, named exception — mirroring how REQ-F-018 requires F09 to record
// the I-08/I-09 degraded-upstream gaps as evidence instead of assuming
// compliance; this task's own scope note says it "mirrors that degraded
// state." PARENT NOTE: AC-006's prose ("appears on no Rider path") does not
// literally hold given this pre-existing, independently-documented
// exception; flagging for the owner rather than silently treating it as
// compliant or leaving it to regress unnoticed.
const f09KnownRiderKickbackStatusSetExceptionCount = 1

// TestTC003_03bShardStatusSetAbsentFromRiderExecutablePaths is TC-003-03b
// (test-plan.md #tc-003, AC-006, added per codex red-team CONCERN): greps
// every file under the embedded skills/shark-attack/workflows/ tree and the
// repository skills/shark-rider/verbs/run.md — together "any Rider path"
// per the task spec's Notes for Agent — for a `shark status set`
// invocation, proving AC-006's claim across the whole executable-workflow
// surface rather than the one file TC-006's council-routing prose happens
// to cover.
func TestTC003_03bShardStatusSetAbsentFromRiderExecutablePaths(t *testing.T) {
	// skills/shark-attack/workflows/** — T-024/T-025's restructured content
	// carries zero legacy exceptions to document, so the bar here is a hard
	// zero.
	corpus := f09EmbeddedSharkAttackCorpus(t)
	scanned := 0
	for rel, content := range corpus {
		if !strings.HasPrefix(rel, "skills/shark-attack/workflows/") {
			continue
		}
		scanned++
		if got := strings.Count(content, f09RiderStatusSetMarker); got > 0 {
			t.Errorf("TC-003-03b: %s contains %d %q occurrence(s), want 0 — AC-006 requires `shark status set` never appear on a Rider-executable path", rel, got, f09RiderStatusSetMarker)
		}
	}
	if scanned == 0 {
		t.Fatal("TC-003-03b fixture assumption broken: embedded skills/shark-attack/workflows/ tree resolved zero files")
	}

	// skills/shark-rider/verbs/run.md carries exactly the one documented
	// kickback-application exception above; a second occurrence — anywhere,
	// including a new dispatched-entity-advancing use AC-006 exists to
	// forbid — fails this test.
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	if !strings.Contains(run, "Apply task kickbacks") {
		t.Fatal("TC-003-03b fixture assumption broken: run.md's kickback-application step (the documented exception's context) no longer exists — re-verify the sole `shark status set` occurrence's context before trusting the exception count below")
	}
	if got := strings.Count(run, f09RiderStatusSetMarker); got != f09KnownRiderKickbackStatusSetExceptionCount {
		t.Errorf("TC-003-03b: run.md contains %d %q occurrence(s), want exactly %d (the documented kickback-application escape hatch, AC-006/D-006) — a changed count means either a regression (new occurrence) or the documented exception was removed; either way the exception count and AC-006's prose need reconciling by the parent/owner", got, f09RiderStatusSetMarker, f09KnownRiderKickbackStatusSetExceptionCount)
	}
}

// f09ShardPlanReadOnlyProbeMarker is the literal substring TC-009-01
// forbids: an instruction to call `shark plan` from within the
// skills/shark-attack/ corpus. Per spec.md's "Degraded upstream
// dependencies" table, `plan.go:631` still calls `autoAdvanceCascadeParent`
// on current trunk (pinned by `plan_dispatch_test.go:254`), so any such
// instruction would silently mutate state under F08's unrepaired gap.
const f09ShardPlanReadOnlyProbeMarker = "shark plan"

// TestTC009_01NoShardAttackFileInstructsReadOnlyShardPlanProbe is TC-009-01
// (test-plan.md #tc-009, REQ-F-018, I-08): no file under the embedded
// skills/shark-attack/** tree instructs a chair/parent to call `shark plan
// <key>` as a read-only inspection step — F09's own selection is chair-side
// and uses keyed `shark next` instead (spec.md's degraded-upstream table).
func TestTC009_01NoShardAttackFileInstructsReadOnlyShardPlanProbe(t *testing.T) {
	corpus := f09EmbeddedSharkAttackCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("TC-009-01 fixture assumption broken: embedded skills/shark-attack/ corpus resolved zero files")
	}
	for rel, content := range corpus {
		if strings.Contains(content, f09ShardPlanReadOnlyProbeMarker) {
			t.Errorf("TC-009-01: %s contains %q — F08's `shark plan` still mutates on trunk (plan.go:631 autoAdvanceCascadeParent); no skill file may instruct a read-only inspection call", rel, f09ShardPlanReadOnlyProbeMarker)
		}
	}
}

// f09CouncilArtifactImportPath is the nonexistent package TC-009-02
// forbids importing (per spec.md's "Degraded upstream dependencies" table:
// F05's council_artifact model/repository/service/CLI tooling is absent
// from main).
const f09CouncilArtifactImportPath = "internal/models/council_artifact"

// f09AdminCouncilInvocationMarker is the CLI invocation TC-009-02/03
// forbid in skill prose: `shark admin council ...` is not a registered
// subcommand on trunk (F05's tooling is absent), so any prose instructing
// it would describe a nonexistent command even though the prose itself
// compiles fine.
const f09AdminCouncilInvocationMarker = "admin council"

// f09GoSourceScanRoots are the repository directories that could contain a
// "new F09 Go file" per TC-009-02's wording. Deliberately scoped rather than
// walking the full repository root: the root also contains .worktrees/ (a
// sibling checkout of the unmerged F05 branch, which legitimately imports
// council_artifact on its own branch and must not be scanned as if it were
// this branch's tree), .git/, bin/, and dev-artifacts/.
var f09GoSourceScanRoots = []string{"internal", "cmd", "tests"}

// TestTC009_02NoGoFileImportsNonexistentCouncilArtifactPackage is TC-009-02
// (test-plan.md #tc-009, REQ-F-018, I-09): a compile-time-proof-by-
// construction static guard. internal/models/council_artifact does not
// exist on `main` (F05's tooling — model, repository, service, `admin
// council` CLI — is entirely absent per spec.md's degraded-upstream table),
// so an import of it would fail `go build`; this test proves that absence
// concretely (directory-existence check) and additionally greps every .go
// file's contents for the literal qualified import path so a future
// partial vendoring attempt is also caught. It also greps skill prose
// (which compiles fine but can still describe a nonexistent command) for an
// `admin council` CLI invocation.
func TestTC009_02NoGoFileImportsNonexistentCouncilArtifactPackage(t *testing.T) {
	root := f09ProjectRoot(t)

	// The package must not exist on trunk — this is what makes an import
	// impossible "by construction," per the test-plan's own phrasing.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f09CouncilArtifactImportPath))); !os.IsNotExist(err) {
		t.Fatalf("TC-009-02 fixture assumption broken: %s exists on trunk (stat err=%v) — F05's tooling may have landed; re-scope this guard rather than deleting it, per REQ-F-018's honesty requirement", f09CouncilArtifactImportPath, err)
	}

	// Explicit import-path grep across every .go file under this branch's
	// own source roots, independent of the directory-absence check above.
	importPath := "github.com/jwwelbor/shark-task-manager/" + f09CouncilArtifactImportPath
	for _, sub := range f09GoSourceScanRoots {
		scanRoot := filepath.Join(root, sub)
		walkErr := filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(content), importPath) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				t.Errorf("TC-009-02: %s imports %q, which does not exist on `main` (F05's council-artifact tooling is unreachable — spec.md 'Degraded upstream dependencies')", rel, importPath)
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("TC-009-02 walk %s for .go files: %v", sub, walkErr)
		}
	}

	// Documentation-only reference check: skill prose compiles fine but can
	// still describe a nonexistent `admin council` command.
	corpus := f09EmbeddedSharkAttackCorpus(t)
	for rel, content := range corpus {
		if strings.Contains(content, f09AdminCouncilInvocationMarker) {
			t.Errorf("TC-009-02: %s references %q — no such subcommand is registered on trunk (F05's `admin council` CLI tooling is absent)", rel, f09AdminCouncilInvocationMarker)
		}
	}
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	if strings.Contains(run, f09AdminCouncilInvocationMarker) {
		t.Errorf("TC-009-02: skills/shark-rider/verbs/run.md references %q — no such subcommand is registered on trunk", f09AdminCouncilInvocationMarker)
	}
}

// TestTC009_03CouncilArtifactsLandUnderDocsCouncilAsPlainFilesNotCLISubcommand
// is TC-009-03 (test-plan.md #tc-009, REQ-F-018, I-09): F09's own durable
// decision/handoff artifacts land under docs/council/ as plain files per
// message-schema.md's existing conventions (spec.md's degraded-upstream
// table: "F09's durable decisions/handoffs ... are files under docs/council/
// written per the existing message-schema.md conventions"), not through a
// `shark admin council` subcommand — which does not exist on trunk.
// Positive half: the files that actually author decisions/handoffs
// (council.md, resume.md, message-schema.md) declare the docs/council/
// layout. Negative half: none of them instructs the parent to call the
// nonexistent subcommand — the same marker TC-009-02 checks corpus-wide;
// this sub-case anchors the assertion to the specific authoring files the
// test-plan names, rather than the whole tree.
func TestTC009_03CouncilArtifactsLandUnderDocsCouncilAsPlainFilesNotCLISubcommand(t *testing.T) {
	authoringFiles := []string{
		"skills/shark-attack/workflows/council.md",
		"skills/shark-attack/workflows/resume.md",
		"skills/shark-attack/context/message-schema.md",
	}
	for _, rel := range authoringFiles {
		content := readF09EmbeddedFile(t, rel)
		if !strings.Contains(content, "docs/council/") {
			t.Errorf("TC-009-03: %s does not declare the docs/council/ plain-file layout for its durable decision/handoff artifacts", rel)
		}
		if strings.Contains(content, f09AdminCouncilInvocationMarker) {
			t.Errorf("TC-009-03: %s references %q — F09's durable artifacts must be plain files under docs/council/, never a `shark admin council` subcommand (absent on trunk)", rel, f09AdminCouncilInvocationMarker)
		}
	}
}

// ---------------------------------------------------------------------------
// TC-SEC-01-01..03 (test-plan.md #tc-sec-01, AC-024, REQ-NF-003, REQ-NF-004)
// — T-E38-F09-021. Prompt/credential leak-sink enumeration: every durable/
// transport sink F09 writes, checked against every ValidateQuestionBoundedText
// denylist class. Lands as sub-cases of the existing I-10 TC-002/TC-010
// pointers rather than a new top-level TC number (task spec's Scope).
//
// This is a bounded feature-level cross-component security suite (a
// declared TDD-structure exception, per the workflow's "Exception" clause):
// it enumerates three sinks T-006 (--prompt-out), T-011/T-012 (question
// mint/respond/resolve), and T-016 (docs/council/ handoffs) each created, so
// it can only run once all three have landed. It builds on
// f09ProhibitedPromptPersistenceMarkers/f09PromptFieldNamePattern (T-016,
// above) rather than duplicating them.
// ---------------------------------------------------------------------------

// tcSec01DenylistClasses is the exact denylist marker set
// ValidateQuestionBoundedText enforces (internal/models/question.go:190) —
// the denylist-class enumeration dimension of TC-SEC-01-01/02's Caller-Path
// Contract. Deliberately duplicated here rather than imported from the
// production package: the contract calls for asserting against the same
// literal strings production checks, not a derived list, so an uncoordinated
// production edit to the denylist shows up here as a mismatch instead of
// silently tracking the change.
func tcSec01DenylistClasses() []string {
	return []string{
		"api_key", "password=", "authorization:", "bearer ",
		"system prompt", "user prompt", "assistant:",
	}
}

// tcSec01QuestionField is one of the "2 Question free-text fields" the task
// spec names — RecordQuestionResponseInput's Summary/EvidencePointer
// (question_workflow_service.go:26), the field pair `question respond`
// exposes and question.go:101/104 validate as "responses.summary" /
// "responses.evidence_pointer" with their production bounds
// (questionResponseMaxBytes=1000, questionPointerMaxBytes=2048).
//
// Interpretation note (PARENT NOTE material — the task spec's literal field
// list is looser than what production actually exposes): the spec names
// "summary/evidence_pointer on ConfigureWorkflowInput/RecordQuestionResponse
// Input/ResolveQuestionInput". ConfigureWorkflowInput carries no free-text
// field of its own (only ResolutionOwner, identity-bounded at 256 bytes),
// and ResolveQuestionInput's Pointer field validates as "resolution_pointer"
// (question.go:161), not "evidence_pointer". RecordQuestionResponseInput is
// the one input type that actually carries both named fields together, so
// this table uses it as the concrete field set; all three input types are
// still exercised end-to-end by TC-SEC-01-03's lifecycle below.
//
// minBytes/maxBytes are deliberately re-declared as literals here (1000,
// 2048) rather than importing the unexported questionResponseMaxBytes/
// questionPointerMaxBytes constants — same "assert against the same
// literal contract production checks, not a derived value" rationale as
// tcSec01DenylistClasses' duplicated marker list above: a production bound
// change without a matching test update shows up as a mismatch here.
type tcSec01QuestionField struct {
	name           string
	validatorField string
	minBytes       int
	maxBytes       int
}

func tcSec01QuestionFields() []tcSec01QuestionField {
	return []tcSec01QuestionField{
		{name: "summary", validatorField: "responses.summary", minBytes: 1, maxBytes: 1000},
		{name: "evidence_pointer", validatorField: "responses.evidence_pointer", minBytes: 1, maxBytes: 2048},
	}
}

// TestTCSEC01_01DenylistClassesRejectedAcrossQuestionFreeTextFieldsBeforePersistence
// is TC-SEC-01-01: for each denylist class x each of the 2 Question
// free-text fields, a fixture containing the forbidden substring is
// rejected. Two layers, matching the Caller-Path Contract's two entrypoint
// dimensions:
//
//  1. Direct calls to the real models.ValidateQuestionBoundedText (no mock,
//     per "Lowest allowed mock seam: none for (a)") across every class x
//     field combination — 14 cases.
//  2. A real-DB, real-CLI persistence check (the TC-004 seam) proving
//     rejection happens BEFORE persistence: a forbidden-marker `question
//     respond` call fails and leaves the Question's status/context_data
//     byte-identical to before the call — run once per named field so both
//     fields are proven at the sink layer, not only the direct-call layer.
func TestTCSEC01_01DenylistClassesRejectedAcrossQuestionFreeTextFieldsBeforePersistence(t *testing.T) {
	for _, field := range tcSec01QuestionFields() {
		for _, marker := range tcSec01DenylistClasses() {
			t.Run(field.name+"/"+marker, func(t *testing.T) {
				value := "prefix " + marker + " suffix"
				err := models.ValidateQuestionBoundedText(field.validatorField, value, field.minBytes, field.maxBytes)
				if err == nil {
					t.Fatalf("TC-SEC-01-01 ValidateQuestionBoundedText(%q, %q, ...) = nil, want rejection for denylist class %q", field.validatorField, value, marker)
				}
				if !strings.Contains(err.Error(), "forbidden credential, rendered prompt, or transcript material") {
					t.Fatalf("TC-SEC-01-01 rejection error = %q, want the forbidden-material message", err.Error())
				}
			})
		}
	}

	// Persistence-layer proof: real DB + real CLI, reusing the TC-004
	// convention.
	projectRoot := f09ProjectRoot(t)
	dbPath := filepath.Join(t.TempDir(), "tc-sec-01-01-persistence.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-SEC-01-01 InitDB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	binary := buildSharkF09(t)
	runShark := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
		cmd.Dir = projectRoot
		out, runErr := cmd.CombinedOutput()
		return string(out), runErr
	}
	runOK := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr != nil {
			t.Fatalf("TC-SEC-01-01 %s: shark %s failed: %v\n%s", label, strings.Join(args, " "), runErr, out)
		}
		return out
	}

	runOK("create Q001", "--json", "question", "create", "TC-SEC-01-01 persistence question", "--summary", "Bounded creation summary", "--requester", "release-owner", "--blocking")
	runOK("configure Q001", "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice")
	runOK("claim Q001", "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session")

	snapshotQ001 := func(label string) string {
		t.Helper()
		var status string
		var contextData sql.NullString
		if err := sqlDB.QueryRow(`SELECT status, context_data FROM questions WHERE key = ?`, "Q001").Scan(&status, &contextData); err != nil {
			t.Fatalf("TC-SEC-01-01 %s snapshot Q001: %v", label, err)
		}
		return status + "|" + contextData.String
	}

	for _, tc := range []struct {
		name            string
		summary         string
		evidencePointer string
	}{
		{
			name:            "summary carries forbidden marker",
			summary:         "leaked api_key material",
			evidencePointer: "docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/spec.md",
		},
		{
			name:            "evidence_pointer carries forbidden marker",
			summary:         "bounded, non-denylisted summary",
			evidencePointer: "leaked authorization: material",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := snapshotQ001(tc.name)
			out, runErr := runShark("--json", "question", "respond", "Q001", "--session", "alice-session", "--responder", "alice", "--summary", tc.summary, "--evidence-pointer", tc.evidencePointer)
			if runErr == nil {
				t.Fatalf("TC-SEC-01-01 %s: shark question respond succeeded, want rejection\n%s", tc.name, out)
			}
			if !strings.Contains(out, "forbidden credential, rendered prompt, or transcript material") {
				t.Fatalf("TC-SEC-01-01 %s: rejection output = %s, want the forbidden-material message", tc.name, out)
			}
			after := snapshotQ001(tc.name)
			if before != after {
				t.Fatalf("TC-SEC-01-01 %s: Q001 status/context_data mutated by a rejected respond call (before=%q after=%q) — rejection must happen before persistence", tc.name, before, after)
			}
		})
	}
}

// TestTCSEC01_02CaseAndWhitespaceDenylistVariantsProveNonNaiveMatch is
// TC-SEC-01-02: case and whitespace variants of each denylist string are
// swept in two labeled tables.
//
// Table 1 (case variants): ValidateQuestionBoundedText lowercases the value
// before matching, so any case permutation of a marker must reject — this
// proves the match isn't a naive case-sensitive exact-match.
//
// Table 2 (known_gap — whitespace-permuted near-misses): the match is
// strings.Contains(strings.ToLower(value), marker) against an exact literal
// marker ("bearer ", "password=", "authorization:"). That normalizes CASE
// but not WHITESPACE/PUNCTUATION, so a marker with an inserted/removed
// space or colon slips through undetected — verified empirically while
// building this test (models.ValidateQuestionBoundedText currently ACCEPTS
// "BEARER:", "password =", and "Authorization :"). REQ-NF-004 commits F09 to
// relying on E39's existing denylist and adding no second validator, and
// this dispatch's Production-Go budget is spent, so this task cannot close
// the gap. Filed as TD-073
// (docs/plan/tech-debt/TD-073.md). This table pins the CURRENT (accepting)
// behavior as a characterization test: it fails loudly the moment the
// denylist is strengthened, forcing this comment and TD-073 to be revisited
// rather than letting the gap go silently unnoticed either way.
func TestTCSEC01_02CaseAndWhitespaceDenylistVariantsProveNonNaiveMatch(t *testing.T) {
	caseVariants := []string{
		"API_KEY", "Api_Key",
		"PASSWORD=", "Password=",
		"AUTHORIZATION:", "Authorization:",
		"BEARER ", "Bearer ",
		"SYSTEM PROMPT", "System Prompt",
		"USER PROMPT", "User Prompt",
		"ASSISTANT:", "Assistant:",
	}
	for _, variant := range caseVariants {
		t.Run("case/"+variant, func(t *testing.T) {
			value := "prefix " + variant + " suffix"
			if err := models.ValidateQuestionBoundedText("responses.summary", value, 1, 1000); err == nil {
				t.Fatalf("TC-SEC-01-02 case variant %q was accepted, want rejection — the denylist match must be case-insensitive", variant)
			}
		})
	}

	knownGapVariants := []string{"BEARER:", "password =", "Authorization :"}
	for _, variant := range knownGapVariants {
		t.Run("known_gap/"+variant, func(t *testing.T) {
			value := "prefix " + variant + " suffix"
			err := models.ValidateQuestionBoundedText("responses.summary", value, 1, 1000)
			if err != nil {
				t.Fatalf("TC-SEC-01-02 whitespace-permuted variant %q is now rejected (err=%v) — the denylist gap this characterization test pins has been closed; update this test and close TD-073", variant, err)
			}
		})
	}
}

// f09TruncateUTF8Trimmed truncates s to at most maxBytes bytes without
// splitting a multi-byte UTF-8 rune, then trims leading/trailing whitespace
// so the result satisfies ValidateQuestionBoundedText's "must be trimmed"
// rule. TC-SEC-01-03 uses it to store a strictly shorter, non-identical
// excerpt of the shared adversarial fixture as a Question's own bounded
// field content — short enough that it can never itself equal or contain
// the full fixture text.
func f09TruncateUTF8Trimmed(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return strings.TrimSpace(s)
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return strings.TrimSpace(s[:cut])
}

// f09CountFiles counts regular files under dir (recursively). A dir that
// does not exist counts as zero files rather than an error — TC-SEC-01-03
// uses this to snapshot docs/council/ before any F09 command has had a
// chance to create it.
func f09CountFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("TC-SEC-01-03 count files under %s: %v", dir, walkErr)
	}
	return count
}

// TestTCSEC01_03FullLifecycleRenderedPromptNeverLeaksOutsideItsOwnDispatchArtifacts
// is TC-SEC-01-03: after a full mint -> configure -> respond -> resolve
// Question lifecycle running alongside a real F09 dispatch of the shared
// TC-002 adversarial fixture (rendered as an entity's instruction, per
// TC-002-09's proven pattern), the fixture's literal, untruncated text must
// survive in ONLY the two artifacts that are its designed destination — the
// dispatching entity's `next --json` `prompt` field and its `--prompt-out`
// file (already proven byte-exact by TC-002-09) — and must be absent from
// every other durable/transport sink F09 writes.
//
// Sink-list deviations from the test-plan's literal wording, each required
// to make the sub-case writable (documented per this task's declared
// TDD-exception latitude):
//
//   - "--prompt-out output" / "CLI stdout": split into the ONE dispatch call
//     that legitimately renders the fixture (excluded from the sweep — its
//     byte-exactness is TC-002-09's job) versus every OTHER command's
//     stdout in this run (included). A literal "must never contain the
//     prompt" reading is unwritable: --prompt-out's entire purpose is to
//     contain the rendered prompt (REQ-NF-005), and `next --json`'s prompt
//     field is the dispatch payload itself (TC-004-07).
//   - "docs/council/ handoffs": no runtime writer exists anywhere under
//     internal/ (grep-verified — only embed.go's roster canonical-path
//     validation constants reference the string; D-002 committed to zero
//     new runtime code). Asserted as a directory file-count snapshot diff
//     rather than a content grep of files that are never written.
//   - OTel span attributes: cannot be captured from a compiled-binary
//     subprocess invocation (no in-memory-exporter CLI seam exists in this
//     repo for `shark next`). Substituted with a source-level check:
//     next.go's only prompt-related attribute is `attribute.Int(
//     "prompt_bytes", ...)` (never the prompt string via attribute.String),
//     and next.go has no AddEvent calls at all.
//   - The Question's own summary/evidence_pointer content is a strictly
//     shorter, non-identical excerpt of the same fixture (per this task's
//     Notes: "reuse T-006's fixture, truncated to field limits") — proving
//     ordinary adversarial-shaped-but-non-denylisted content passes through
//     the pipeline safely (complementing TC-SEC-01-01/02's rejection cases)
//     while remaining too short to ever satisfy a "contains the FULL
//     literal fixture text" check, which is what lets the zero-match sweep
//     apply to the Question's own context_data without contradicting this
//     deliberate write.
func TestTCSEC01_03FullLifecycleRenderedPromptNeverLeaksOutsideItsOwnDispatchArtifacts(t *testing.T) {
	fixture := f09AdversarialPromptFixture()
	projectDir := t.TempDir()
	dbPath := filepath.Join(projectDir, "tc-sec-01-03.db")

	// Custom task-only workflow: the "development" step's literal prompt IS
	// the shared adversarial fixture — TC-002-09's proven pattern.
	workflowDir := filepath.Join(projectDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("TC-SEC-01-03 create workflow dir: %v", err)
	}
	taskWorkflow := map[string]any{
		"version": "1.0",
		"start":   "development",
		"steps": map[string]any{
			"development": map[string]any{
				"phase":  "development",
				"action": "spawn_agent",
				"agent":  "developer",
				"prompt": fixture,
				"outcomes": map[string]any{
					"pass":    "completed",
					"fail":    "development",
					"blocked": "blocked",
				},
			},
			"blocked": map[string]any{
				"phase":   "blocked",
				"action":  "pause",
				"parking": true,
			},
			"completed": map[string]any{
				"phase":    "done",
				"action":   "archive",
				"terminal": true,
			},
		},
	}
	taskYAML, err := yaml.Marshal(taskWorkflow)
	if err != nil {
		t.Fatalf("TC-SEC-01-03 marshal task workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "task.yaml"), taskYAML, 0o644); err != nil {
		t.Fatalf("TC-SEC-01-03 write task.yaml: %v", err)
	}
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{"workflow_config":%q}`, workflowDir)), 0o644); err != nil {
		t.Fatalf("TC-SEC-01-03 write config: %v", err)
	}

	ctx := context.Background()
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-SEC-01-03 InitDB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "TC-SEC-01-03 epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-SEC-01-03 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "TC-SEC-01-03 feature"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-SEC-01-03 seed feature: %v", err)
	}
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "TC-SEC-01-03 fixture-carrying task"}, FeatureID: feature.ID, Status: "development", Priority: 5}
	if err := repository.NewTaskRepository(repoDB).Create(ctx, task); err != nil {
		t.Fatalf("TC-SEC-01-03 seed task: %v", err)
	}

	binary := buildSharkF09(t)
	runShark := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--db", dbPath}, args...)...)
		cmd.Dir = projectDir
		out, runErr := cmd.CombinedOutput()
		return string(out), runErr
	}
	runOK := func(label string, args ...string) string {
		t.Helper()
		out, runErr := runShark(args...)
		if runErr != nil {
			t.Fatalf("TC-SEC-01-03 %s: shark %s failed: %v\n%s", label, strings.Join(args, " "), runErr, out)
		}
		return out
	}

	councilDir := filepath.Join(projectDir, "docs", "council")
	councilFilesBefore := f09CountFiles(t, councilDir)

	// Negative-case check that f09CountFiles has teeth: on a throwaway
	// directory (never councilDir itself, so this cannot pollute the real
	// snapshot below), zero files count as zero and a written file counts
	// as one -- proving the "unchanged file count" assertion further down
	// can actually detect a write, not just default to a vacuous "still
	// zero" for an unrelated reason (e.g. the walk short-circuiting).
	teethDir := filepath.Join(t.TempDir(), "f09-count-files-teeth-check")
	if got := f09CountFiles(t, teethDir); got != 0 {
		t.Fatalf("TC-SEC-01-03 f09CountFiles(%s) = %d on a nonexistent directory, want 0", teethDir, got)
	}
	if err := os.MkdirAll(teethDir, 0o755); err != nil {
		t.Fatalf("TC-SEC-01-03 create teeth-check dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teethDir, "marker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("TC-SEC-01-03 write teeth-check marker: %v", err)
	}
	if got := f09CountFiles(t, teethDir); got != 1 {
		t.Fatalf("TC-SEC-01-03 f09CountFiles(%s) = %d after writing one file, want 1 -- the docs/council/ unchanged-count assertion below would not have teeth", teethDir, got)
	}

	// The run's ONE legitimate rendering of the fixture — excluded from the
	// zero-match sweep below by construction. Doubles as the positive
	// control proving the sweep would have teeth: if the fixture failed to
	// render here, every zero-match assertion below would be vacuous.
	promptOutPath := filepath.Join(t.TempDir(), "tc-sec-01-03-prompt.txt")
	dispatchOut := runOK("dispatch fixture task", "--json", "next", "T-E01-F01-001", "--prompt-out", promptOutPath)
	var dispatchResp commands.NextResponse
	if err := json.Unmarshal([]byte(dispatchOut), &dispatchResp); err != nil {
		t.Fatalf("TC-SEC-01-03 decode fixture dispatch: %v\n%s", err, dispatchOut)
	}
	if !strings.Contains(dispatchResp.Prompt, fixture) {
		t.Fatalf("TC-SEC-01-03 positive control failed: dispatched prompt does not contain the fixture text — the zero-match sweep below would be vacuous")
	}
	promptOutBytes, err := os.ReadFile(promptOutPath)
	if err != nil {
		t.Fatalf("TC-SEC-01-03 read --prompt-out: %v", err)
	}
	if !bytes.Contains(promptOutBytes, []byte(fixture)) {
		t.Fatalf("TC-SEC-01-03 positive control failed: --prompt-out file does not contain the fixture text")
	}

	// Question lifecycle: summary/evidence_pointer hold a strictly shorter,
	// non-identical excerpt of the same fixture.
	summaryExcerpt := f09TruncateUTF8Trimmed(fixture, 60)
	evidenceExcerpt := f09TruncateUTF8Trimmed(fixture, 90)
	if len(summaryExcerpt) >= len(fixture) || len(evidenceExcerpt) >= len(fixture) || strings.Contains(summaryExcerpt, fixture) || strings.Contains(evidenceExcerpt, fixture) {
		t.Fatalf("TC-SEC-01-03 fixture excerpts must be strictly shorter than, and must not contain, the full fixture (summary=%q evidence=%q)", summaryExcerpt, evidenceExcerpt)
	}

	var stdoutCapture []string
	capture := func(label string, args ...string) string {
		t.Helper()
		out := runOK(label, args...)
		stdoutCapture = append(stdoutCapture, out)
		return out
	}

	createOut := capture("create Q001", "--json", "question", "create", "TC-SEC-01-03 question", "--summary", "TC-SEC-01-03 bounded lifecycle question", "--requester", "release-owner", "--blocking")
	var created struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(createOut), &created); err != nil || created.Key != "Q001" {
		t.Fatalf("TC-SEC-01-03 create Q001: decode error = %v, out=%s", err, createOut)
	}
	capture("configure Q001", "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice")
	runOK("claim Q001", "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session")
	capture("respond Q001", "--json", "question", "respond", "Q001", "--session", "alice-session", "--responder", "alice", "--summary", summaryExcerpt, "--evidence-pointer", evidenceExcerpt)
	runOK("release Q001", "--json", "release", "Q001", "--session", "alice-session", "--outcome", "pass")
	capture("resolve Q001", "--json", "question", "resolve", "Q001", "--owner", "release-owner", "--resolution-kind", "no_lasting_consequence")
	capture("full Q001", "--json", "question", "full", "Q001", "--actor", "release-owner")

	// Sink 1: the Q### row's own context_data holds the excerpts (sanity —
	// proves the write actually happened) but never the full fixture.
	// context_data is JSON, so its raw text escapes the fixture's embedded
	// \n/\r\n as literal backslash sequences — decode the field values
	// rather than substring-matching the raw JSON text, which would never
	// match either the escaped excerpt (false sanity failure) or the
	// escaped full fixture (a vacuous negative check).
	var contextData sql.NullString
	if err := sqlDB.QueryRow(`SELECT context_data FROM questions WHERE key = ?`, "Q001").Scan(&contextData); err != nil {
		t.Fatalf("TC-SEC-01-03 read Q001 context_data: %v", err)
	}
	if !contextData.Valid {
		t.Fatalf("TC-SEC-01-03 Q001 context_data is NULL, want the recorded response state")
	}
	var decodedState struct {
		QuestionState struct {
			Responses []struct {
				Summary         string `json:"summary"`
				EvidencePointer string `json:"evidence_pointer"`
			} `json:"responses"`
		} `json:"question_state"`
	}
	if err := json.Unmarshal([]byte(contextData.String), &decodedState); err != nil {
		t.Fatalf("TC-SEC-01-03 decode Q001 context_data: %v\n%s", err, contextData.String)
	}
	if len(decodedState.QuestionState.Responses) != 1 {
		t.Fatalf("TC-SEC-01-03 Q001 context_data has %d responses, want exactly 1: %s", len(decodedState.QuestionState.Responses), contextData.String)
	}
	storedSummary := decodedState.QuestionState.Responses[0].Summary
	storedEvidence := decodedState.QuestionState.Responses[0].EvidencePointer
	if storedSummary != summaryExcerpt || storedEvidence != evidenceExcerpt {
		t.Fatalf("TC-SEC-01-03 sanity check failed: Q001 context_data response = {summary=%q evidence_pointer=%q}, want the stored excerpts {%q, %q}", storedSummary, storedEvidence, summaryExcerpt, evidenceExcerpt)
	}
	if strings.Contains(storedSummary, fixture) || strings.Contains(storedEvidence, fixture) {
		t.Fatalf("TC-SEC-01-03 Q001 context_data contains the full untruncated fixture text — rendered-prompt leak into Question state")
	}

	// Sink 2: every Question-CLI command's own --json stdout in this run.
	// The captured output is JSON, which escapes the fixture's embedded
	// \n/\r\n as literal backslash sequences (the same encoding mismatch
	// Sink 1 above corrects for) -- check both the raw form and the
	// JSON-escaped form (json.Marshal of the fixture, minus the surrounding
	// quotes it adds) so a leak landing inside a JSON string field is not
	// missed the way a raw substring check alone would miss it.
	escapedFixtureJSON, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("TC-SEC-01-03 marshal fixture for escaped-form comparison: %v", err)
	}
	escapedFixture := strings.Trim(string(escapedFixtureJSON), `"`)
	for i, out := range stdoutCapture {
		if strings.Contains(out, fixture) || strings.Contains(out, escapedFixture) {
			t.Fatalf("TC-SEC-01-03 Question CLI stdout call #%d contains the full fixture text: %s", i, out)
		}
	}

	// Sink 3: docs/council/ — no runtime writer exists, so its file count
	// must be unchanged.
	councilFilesAfter := f09CountFiles(t, councilDir)
	if councilFilesAfter != councilFilesBefore {
		t.Fatalf("TC-SEC-01-03 docs/council/ file count changed (%d -> %d) though no F09 code writes there", councilFilesBefore, councilFilesAfter)
	}

	// Sink 4: OTel span attributes/events — source-level substitute (see
	// function doc comment for why a runtime capture isn't available here).
	nextGoSource := readF09RepositoryFile(t, "internal/cli/commands/next.go")
	for _, forbidden := range []string{`attribute.String("prompt"`, `attribute.String("rendered_prompt"`, "AddEvent("} {
		if strings.Contains(nextGoSource, forbidden) {
			t.Fatalf("TC-SEC-01-03 next.go contains %q — a span attribute/event could carry the rendered prompt text", forbidden)
		}
	}
}
