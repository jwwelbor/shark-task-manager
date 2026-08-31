package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateGolden, when set on the test command line, rewrites the rendered-prompt
// golden corpus under testdata/rendered-prompts/ instead of asserting against
// it. Phase 4 of the E32-F02 plan: we can't diff against pre-migration
// .tmpl output anymore, so the substitute is a forward-looking snapshot that
// gates future prompt-corpus drift on an intentional `-update` step.
//
//	go test ./internal/cli/commands/ -run TestRenderedPromptsGolden -update
var updateGolden = flag.Bool("update", false,
	"rewrite the rendered-prompt golden corpus instead of comparing")

// goldenVars are the deterministic placeholder values used to render every
// shipped prompt for the golden corpus. The keys cover every `{{.<var>}}` the
// shipped prompts reference today — discovered via:
//
//	grep -hroE '{{\.[a-zA-Z_][a-zA-Z0-9_]*}}' shark-data/prompts/ | sort -u
//
// Values are synthetic so a diff reads obviously as "test data" rather than as
// real-looking content; if you change them, regenerate the corpus with -update
// and commit both the source change and the regenerated .golden files.
func goldenVars() map[string]string {
	return map[string]string{
		"id":                  "E07-F01",
		"key":                 "Q001",
		"title":               "Sample feature for golden test",
		"summary":             "Sample Question summary",
		"current_responder":   "alice",
		"file_path":           "docs/plan/E07/E07-F01/E07-F01.md",
		"epic_id":             "E07",
		"task_id":             "E07-F01-001",
		"category":            "feature",
		"severity":            "medium",
		"primary_doc":         "docs/plan/E07/E07-F01/E07-F01.md",
		"doc_friendly_name":   "Feature spec",
		"related_docs":        "docs/plan/E07/E07-F01/spec.md",
		"related_tasks":       "E07-F01-001, E07-F01-002",
		"review_base":         "docs/review/E07/E07-F01/",
		"is_resume":           "false",
		"advance_summary":     "Advancing entity to next status",
		"blocked_reason":      "External dependency unavailable",
		"fail_reason_summary": "Validation failed",
	}
}

// promptSHA256 hex-encodes the SHA-256 digest of s using the exact same
// algorithm runNext uses to compute NextResponse.PromptSHA256
// (sha256.Sum256 + hex.EncodeToString — see next.go's post-assembly hashing
// step), so this test's digest comparison is a faithful stand-in for AC-07's
// "prompt_sha256 is identical" wording rather than a parallel, divergent
// hashing scheme.
func promptSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestRenderedPromptsGolden is the F02 AC #3 regression guard. It renders
// every shipped prompt under shark-data/prompts/<entity>/*.md against
// goldenVars() and either compares to or rewrites a corresponding file under
// testdata/rendered-prompts/<entity>/<status>.golden.
//
// Coverage is the corpus itself: every (entity_type, status) pair shipped in
// shark-data/prompts/ produces one .golden file. Partials under _partials/
// are skipped because they have no standalone rendering — they're composed
// into the entity prompts via {{template ...}}.
// The _shared/ directory IS covered: it holds dispatchable shared prompts
// (e.g. code_review.md, qa.md) that are referenced by multiple entity
// workflows and render independently.
func TestRenderedPromptsGolden(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes + partials resolved")

	entities, err := os.ReadDir(promptsDir)
	require.NoError(t, err)

	// varsWithHarness mirrors what resolveEntity actually merges into the
	// placeholder map in production (next.go) before every render: the zero
	// HarnessIdentity's three vars keys, unconditionally, per D-F01-07.
	// varsWithoutHarness is the pre-feature shape (no harness keys at all) —
	// the "before this feature" state TC-008 compares against.
	varsWithoutHarness := goldenVars()
	varsWithHarness := goldenVars()
	for k, v := range (services.HarnessIdentity{}).Vars() {
		varsWithHarness[k] = v
	}
	goldenRoot := filepath.Join("testdata", "rendered-prompts")

	for _, ent := range entities {
		// Skip non-directories and the _partials/ directory (partials have no
		// standalone rendering — they're composed into entity prompts via
		// {{template ...}}). The _shared/ directory is NOT skipped: it contains
		// dispatchable shared prompts that render independently.
		if !ent.IsDir() || ent.Name() == "_partials" {
			continue
		}
		entity := ent.Name()

		t.Run(entity, func(t *testing.T) {
			files, err := os.ReadDir(filepath.Join(promptsDir, entity))
			require.NoError(t, err)

			var names []string
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
					names = append(names, f.Name())
				}
			}
			sort.Strings(names)

			for _, name := range names {
				status := strings.TrimSuffix(name, ".md")
				t.Run(status, func(t *testing.T) {
					// Match the renderer's lookup key: subdirectory-prefixed
					// relative path (e.g., "feature/ready_for_assessment.md").
					tmplName := filepath.ToSlash(filepath.Join(entity, name))
					rendered, err := renderer.Render(tmplName, varsWithHarness)
					require.NoErrorf(t, err, "render %s", tmplName)

					goldenPath := filepath.Join(goldenRoot, entity, status+".golden")
					if *updateGolden {
						require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
						require.NoError(t, os.WriteFile(goldenPath, []byte(rendered), 0644))
						return
					}

					want, err := os.ReadFile(goldenPath)
					if err != nil {
						t.Fatalf("missing golden file %s — run with -update to generate: %v",
							goldenPath, err)
					}
					if string(want) != rendered {
						t.Errorf("rendered prompt %s differs from %s; if the change is intentional, regenerate with -update",
							tmplName, goldenPath)
					}

					// T-E34-F01-006/TC-008 (AC-07, REQ-NF-001): a genuine
					// before/after comparison, independent of the committed
					// .golden file (which a future -update would overwrite,
					// destroying it as a "before this feature" anchor).
					// renderedWithoutHarness reproduces the pre-feature vars
					// shape (no harness keys at all); rendered (above) is the
					// post-feature shape with this feature's harness-vars merge
					// applied. PromptSHA256 is computed the identical way
					// NextResponse.PromptSHA256 is. None of the shipped prompts
					// in this corpus branch on .harness (spec.md §2.4 item 5),
					// so the two digests must match — this is exactly the "vars
					// construction order" / "map iteration behavior" drift
					// TC-008's Negative Cases call out, caught across the full
					// corpus rather than a hand-picked fixture.
					renderedWithoutHarness, err := renderer.Render(tmplName, varsWithoutHarness)
					require.NoErrorf(t, err, "render %s without harness keys", tmplName)
					assert.Equalf(t, promptSHA256(renderedWithoutHarness), promptSHA256(rendered),
						"prompt_sha256 must be identical before/after this feature's harness-vars merge for %s (REQ-NF-001/AC-07)", tmplName)
				})
			}
		})
	}
}
