package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file covers test-plan.md TC-002-01..08 (../../../docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/test-plan.md#tc-002):
// prompt-hash/byte-length provenance for keyed `shark next` dispatch
// (REQ-F-011, REQ-NF-003/005/007; AC-013, AC-024 provenance slice).
//
// TC-002-01..08 drive resolveNext through the same mocked nextAdapterCache
// seam TestResolveNext_ReturnsSelfContainedPrompt already uses (next_test.go
// TestResolveNext_ReturnsSelfContainedPrompt) — no real DB, matching the
// golden-rule CLI-test convention (mocked adapters). TC-002-09, the real
// `--prompt-out` write path driven through the actual `runNext`/cobra
// execution chain against a temp SQLite DB, lives in
// tests/contracts/e38_f09_interactions_test.go per the test plan's
// real-process/real-DB convention — flag introspection and in-memory
// resolveNext assertions cannot prove OS-level file-write correctness.

// f09AdversarialPrompt is the shared TC-002 adversarial fixture: embedded
// newlines (\n and \r\n), a fenced code block, single/double quotes, Unicode
// (accented letter, combining mark, RTL mark, emoji), and shell
// metacharacters.
func f09AdversarialPrompt() string {
	return "adversarial line one\n" +
		"adversarial line two\r\n" +
		"fenced ```code fence``` block\n" +
		`single 'quote' and double "quote" pair` + "\n" +
		"unicode: café combining é rtl-mark‏ emoji\U0001F600\n" +
		"shell metacharacters: `cmd` $VAR ; | && > redirect"
}

// f09PromptFixtureVariants isolates each adversarial dimension the test plan
// enumerates, plus the combined fixture, so TC-002-01..03 exercise each
// individually as well as together.
func f09PromptFixtureVariants() map[string]string {
	return map[string]string{
		"newlines":            "line one\nline two\r\nline three",
		"fenced_code":         "before\n```go\nfunc x() {}\n```\nafter",
		"quotes":              `single 'quote' and double "quote" pair`,
		"unicode":             "café combining é rtl-mark‏ emoji\U0001F600",
		"shell_metachars":     "`cmd` $VAR ; | && > redirect",
		"trailing_newline":    "line with a trailing newline\n",
		"no_trailing_newline": "line without a trailing newline",
		"combined":            f09AdversarialPrompt(),
	}
}

// provenanceCache builds a mocked nextAdapterCache for entity type "task"
// whose action service returns instruction verbatim as a spawn_agent
// dispatch with no agent persona (AgentType left empty so
// attachAgentBody is a no-op) — isolating the hash/byte-length assertions
// from persona-inlining content while still exercising the full
// resolveEntity -> assembleDispatchPrompt path production dispatch uses.
func provenanceCache(instruction string) *nextAdapterCache {
	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{
				Action:      "spawn_agent",
				Instruction: instruction,
			}, nil
		},
	}
	return &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"task": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{
					CurrentStatus: "development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}},
				generator: fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc: actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
	}
}

// TestTC002_01_02_03_PromptHashAndByteLengthAcrossAdversarialVariants covers
// TC-002-01 (PromptSHA256 == hex(sha256(resp.Prompt))), TC-002-02
// (PromptBytes == len(resp.Prompt) in bytes), and TC-002-03 (hashing is
// stable across two independent calls) for every fixture variant.
func TestTC002_01_02_03_PromptHashAndByteLengthAcrossAdversarialVariants(t *testing.T) {
	for name, instruction := range f09PromptFixtureVariants() {
		t.Run(name, func(t *testing.T) {
			resp, err := resolveNext(context.Background(), provenanceCache(instruction), "task", "T-E01-F01-001", 0)
			require.NoError(t, err)

			// TC-002-01
			wantSum := sha256.Sum256([]byte(resp.Prompt))
			assert.Equal(t, hex.EncodeToString(wantSum[:]), resp.PromptSHA256,
				"PromptSHA256 must equal an independently computed hex(sha256(resp.Prompt))")

			// TC-002-02
			assert.Equal(t, len([]byte(resp.Prompt)), resp.PromptBytes,
				"PromptBytes must equal len([]byte(resp.Prompt))")

			// TC-002-03: a second, independent call over the same fixture
			// produces identical hash/byte-length values.
			resp2, err := resolveNext(context.Background(), provenanceCache(instruction), "task", "T-E01-F01-001", 0)
			require.NoError(t, err)
			assert.Equal(t, resp.PromptSHA256, resp2.PromptSHA256, "hashing must be deterministic across calls")
			assert.Equal(t, resp.PromptBytes, resp2.PromptBytes, "byte length must be deterministic across calls")
		})
	}
}

// TestTC002_04_PromptOutFlagRegistration proves --prompt-out registration
// only — write behavior is TC-002-09's job (real runNext execution).
func TestTC002_04_PromptOutFlagRegistration(t *testing.T) {
	flag := nextCmd.Flags().Lookup("prompt-out")
	require.NotNil(t, flag, "shark next must register a --prompt-out flag")
	assert.Equal(t, "string", flag.Value.Type())
	assert.Equal(t, "", flag.DefValue)
}

// TestTC002_05_FieldPromptDiffersFromPromptOutByExactlyOneTrailingNewline
// reuses the existing, documented cli.OutputField "--field" behavior
// (Fprintln always appends exactly one trailing newline for a string field)
// to prove AC-013's stated invariant: `--field prompt` differs from the
// `--prompt-out` file (== resp.Prompt bytes exactly, see TC-002-09) by
// exactly one trailing newline.
func TestTC002_05_FieldPromptDiffersFromPromptOutByExactlyOneTrailingNewline(t *testing.T) {
	resp, err := resolveNext(context.Background(), provenanceCache(f09AdversarialPrompt()), "task", "T-E01-F01-001", 0)
	require.NoError(t, err)

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	os.Stdout = w

	fieldErr := cli.OutputField(resp, "prompt")

	require.NoError(t, w.Close())
	os.Stdout = oldStdout
	require.NoError(t, fieldErr)

	out, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	assert.Equal(t, resp.Prompt+"\n", string(out),
		"--field prompt must equal resp.Prompt plus exactly one trailing newline")
	assert.Equal(t, resp.Prompt, strings.TrimSuffix(string(out), "\n"),
		"stripping the single trailing newline must recover exactly the --prompt-out bytes")
}

// TestTC002_06_PromptHashCoversFullyAssembledPromptIncludingPreamble proves
// D-004: PromptSHA256 hashes the fully assembled prompt (ownership preamble
// + agent body + instruction), not the pre-assembly instruction alone. A
// regression to the rejected pre-assembly-hash alternative would make
// PromptSHA256 equal the pre-assembly hash below, which this test catches.
func TestTC002_06_PromptHashCoversFullyAssembledPromptIncludingPreamble(t *testing.T) {
	instruction := f09AdversarialPrompt()
	resp, err := resolveNext(context.Background(), provenanceCache(instruction), "task", "T-E01-F01-001", 0)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(resp.Prompt, "PARENT LOOP OWNERSHIP CONTRACT:"),
		"resp.Prompt must lead with the assembled ownership preamble")

	preAssemblySum := sha256.Sum256([]byte(instruction))
	preAssemblyHex := hex.EncodeToString(preAssemblySum[:])
	assert.NotEqual(t, preAssemblyHex, resp.PromptSHA256,
		"PromptSHA256 must not equal the hash of the pre-assembly instruction alone (D-004's rejected alternative)")

	fullSum := sha256.Sum256([]byte(resp.Prompt))
	assert.Equal(t, hex.EncodeToString(fullSum[:]), resp.PromptSHA256,
		"PromptSHA256 must equal the hash of the fully assembled prompt")
}

// TestTC002_07_ProvenanceFieldsCarryNoPromptText covers AC-024's
// provenance-only slice for this task's scope: PromptSHA256 and PromptBytes
// are pure identifiers (a hash and a byte count) and must never themselves
// carry the rendered prompt text. This task adds no separate persisted
// provenance record (council handoff, decision, note) — REQ-F-008/009 land
// in later E38-F09 tasks — so this covers the provenance fields T-006
// actually introduces; the full leak-sink enumeration is TC-SEC-01.
func TestTC002_07_ProvenanceFieldsCarryNoPromptText(t *testing.T) {
	instruction := f09AdversarialPrompt()
	resp, err := resolveNext(context.Background(), provenanceCache(instruction), "task", "T-E01-F01-001", 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.PromptSHA256)

	provenanceOnly := fmt.Sprintf(
		`{"entity_key":%q,"entity_type":%q,"prompt_sha256":%q,"prompt_bytes":%d}`,
		resp.EntityKey, resp.EntityType, resp.PromptSHA256, resp.PromptBytes,
	)
	assert.NotContains(t, provenanceOnly, instruction,
		"the provenance-only field slice must not contain the literal fixture prompt text")
	assert.NotContains(t, provenanceOnly, "adversarial",
		"the provenance-only field slice must not contain any fragment of the fixture prompt text")
}

// TestTC002_08_OTelPromptBytesReusesResponseValueNotRecomputed is the
// "reasoned code-review check" REQ-NF-007 explicitly permits in place of a
// benchmark: it proves runNext's OTel prompt_bytes span attribute reuses the
// once-computed resp.PromptBytes rather than a second len(resp.Prompt) pass.
func TestTC002_08_OTelPromptBytesReusesResponseValueNotRecomputed(t *testing.T) {
	src, err := os.ReadFile("next.go")
	require.NoError(t, err)
	body := string(src)

	assert.Contains(t, body, `attribute.Int("prompt_bytes", resp.PromptBytes)`,
		"runNext's OTel prompt_bytes span attribute must reuse resp.PromptBytes (computed once in resolveEntity)")
	assert.NotContains(t, body, `attribute.Int("prompt_bytes", len(resp.Prompt))`,
		"a second len(resp.Prompt) computation would defeat REQ-NF-007's single-pass guarantee")
}
