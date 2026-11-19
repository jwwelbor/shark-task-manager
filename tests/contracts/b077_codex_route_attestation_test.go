package contracts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestB077PreflightRunsBeforeClaimAndSpawn(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	preflight := normalizeContractText(e36F04RawSection(t, run,
		"### Route attestation pre-flight (B077)",
		"Claim the concrete entity returned by Shark:"))

	preflightStart := strings.Index(run, "### Route attestation pre-flight (B077)")
	claim := strings.Index(run, "Claim the concrete entity returned by Shark:")
	spawn := strings.Index(run, "Spawn the host worker using a host-safe adapter:")
	assert.Less(t, preflightStart, claim, "route preflight must precede claim")
	assert.Less(t, preflightStart, spawn, "route preflight must precede spawn")
	for _, invariant := range []string{
		"parent/adapter pre-dispatch rejection",
		"no worker control envelope or worker result",
		"no claim",
		"no status advance",
		"blocked_external is reserved for a dispatched worker",
	} {
		assert.Contains(t, strings.ToLower(preflight), strings.ToLower(invariant))
	}
}

func TestB077PreflightAllowsOnlyDocumentedCompatibleRoutes(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	preflight := normalizeContractText(e36F04RawSection(t, run,
		"### Route attestation pre-flight (B077)",
		"Claim the concrete entity returned by Shark:"))

	for _, invariant := range []string{
		"prefer the exact requested provider/model route",
		"for an anthropic request, claude code is preferred when available",
		"its absence alone is not a rejection reason",
		"fallback only through a documented compatible host selector",
		"worker-correlated effective provider/model identity",
		"reject before claim only when no exact route or documented compatible fallback can be selected and observed",
		"do not use the parent's model as an implicit fallback",
		"never silently substitute the requested route",
		"keep response.prompt byte-for-byte unchanged",
	} {
		assert.Contains(t, preflight, invariant)
	}
}

func TestB077DefaultAnthropicBugRouteSelectsCodexCLIModel(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	selection := normalizeContractText(e36F04RawSection(t, run,
		"### B077 default bug-stage Codex fallback mappings",
		"### Claude Code Agent tool adapter"))
	for _, invariant := range []string{
		"host_worker_class=codex-cli",
		"selected_provider=openai",
		"selected_model=gpt-6.2-astra",
		"compatibility mapping",
		"does not claim that any codex model is equivalent to the requested anthropic model",
		"never derived from the parent session, codex configuration default, or environment",
		"table entry does not match",
		"codex selector or worker-identity evidence path is unavailable",
		"reject before claim",
		"codex exec --model",
		"codex login status reports an authenticated session",
		"persistent rollout records can be read",
		"local capability checks and make no model/api call",
		"current host evidence is codex-cli 0.170.0",
	} {
		assert.Contains(t, selection, invariant)
	}

	// TestB077RecordingCodexReceivesSelectedRoute executes the launch and checks the expanded arguments.
}

// C01/C02/C15: run Rider's actual launch command. The recording Codex
// executable starts in a competing provider's configuration.
func TestB077RecordingCodexReceivesSelectedRoute(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	section := e36F04RawSection(t, run, "### Long-running Codex CLI adapter", "### Mid-run consultation")
	_, excerpt, found := strings.Cut(section, "```bash\n")
	require.True(t, found)
	launch, _, found := strings.Cut(excerpt, "\n```")
	require.True(t, found)

	for _, tc := range []struct{ name, request, model, effort string }{
		{"mapped explicit effort", "opus", "gpt-6.2-astra", "low"},
		{"exact explicit effort", "different-exact-model", "different-exact-model", "low"},
		{"mapped omitted effort", "opus", "gpt-6.2-astra", ""},
		{"exact omitted effort", "different-exact-model", "different-exact-model", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			workdir := filepath.Join(dir, "worker checkout")
			require.NoError(t, os.Mkdir(workdir, 0700))
			prompt := []byte("Quoted '$' 🙂\nsecond line  \n")
			require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt"), prompt, 0600))
			const recorder = `#!/bin/bash
set -eu
provider=other
for arg; do
    case "$arg" in model_provider=*) provider=${arg#model_provider=} ;; esac
done
printf '%s\0' "$@" > "$REPO_ROOT/argv"
pwd > "$REPO_ROOT/cwd"
cat > "$REPO_ROOT/submitted-$provider"
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "codex"), []byte(recorder), 0700))
			cmd := exec.Command("bash", "-eu", "-c", launch+"\nwait \"$CODEX_PID\"")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"),
				"REPO_ROOT="+dir, "WORKDIR="+workdir, "SELECTED_PROVIDER=openai",
				"SELECTED_MODEL="+tc.model, "REQUESTED_MODEL="+tc.request, "REQUESTED_EFFORT="+tc.effort,
				"SELECTED_SANDBOX=read-only", "CODEX_BIN="+filepath.Join(dir, "codex"), "SELECTED_PROFILE=",
				"PROMPT_FILE="+filepath.Join(dir, "prompt"), "EVENT_FILE="+filepath.Join(dir, "events"),
				"RESULT_FILE="+filepath.Join(dir, "result"))
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			args, err := os.ReadFile(filepath.Join(dir, "argv"))
			require.NoError(t, err)
			assert.Contains(t, string(args), "--config\x00model_provider=openai\x00")
			assert.Contains(t, string(args), "--model\x00"+tc.model+"\x00")
			assert.Contains(t, string(args), "-C\x00"+workdir+"\x00")
			if tc.effort == "" {
				assert.NotContains(t, string(args), "model_reasoning_effort=")
			} else {
				assert.Contains(t, string(args), "--config\x00model_reasoning_effort="+tc.effort+"\x00")
			}
			cwd, err := os.ReadFile(filepath.Join(dir, "cwd"))
			require.NoError(t, err)
			assert.Equal(t, workdir+"\n", string(cwd))
			assert.NoFileExists(t, filepath.Join(dir, "submitted-other"))
			received, err := os.ReadFile(filepath.Join(dir, "submitted-openai"))
			require.NoError(t, err)
			assert.Equal(t, prompt, received)
		})
	}
}

func TestB077CodexFallbackCoversEveryAnthropicBugWorkflowStage(t *testing.T) {
	workflowText := readF09RepositoryFile(t, "internal/sharkdata/default_data/workflow/bug.yaml")
	var workflow struct {
		Steps map[string]struct {
			Action   string `yaml:"action"`
			Agent    string `yaml:"agent"`
			Provider string `yaml:"provider"`
			Model    string `yaml:"model"`
		} `yaml:"steps"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(workflowText), &workflow))

	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	selection := e36F04RawSection(t, run,
		"### B077 default bug-stage Codex fallback mappings",
		"### Claude Code Agent tool adapter")
	rows := b077MarkdownTableRows(t, selection, "| status |")
	mappings := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		status := strings.ToLower(row["status"])
		require.NotContains(t, mappings, status, "fallback mappings must not duplicate workflow states")
		mappings[status] = row
	}

	spawnableAnthropicStages := 0
	for status, step := range workflow.Steps {
		if step.Action != "spawn_agent" || step.Provider != "anthropic" {
			continue
		}
		spawnableAnthropicStages++
		status, step := status, step
		t.Run(status, func(t *testing.T) {
			row, ok := mappings[status]
			require.True(t, ok, "missing Codex fallback mapping for embedded bug workflow status %q", status)
			assert.Equal(t, "bug", strings.ToLower(row["entity_type"]))
			assert.Equal(t, strings.ToLower(step.Agent), strings.ToLower(row["agent_type"]))
			assert.Equal(t, step.Provider, strings.ToLower(row["requested_provider"]))
			assert.Equal(t, step.Model, strings.ToLower(row["requested_model"]))
			assert.Equal(t, "codex-cli", strings.ToLower(row["host_worker_class"]))
			assert.Equal(t, "openai", strings.ToLower(row["selected_provider"]))
			assert.Equal(t, "gpt-6.2-astra", strings.ToLower(row["selected_model"]))
			assert.Equal(t, "b077-default-bug-anthropic-codex-v1", strings.ToLower(row["mapping_ref"]))
		})
	}
	require.Equal(t, 6, spawnableAnthropicStages, "test must cover all current Anthropic bug workflow stages")
	require.Len(t, mappings, spawnableAnthropicStages, "fallback table must cover only the Anthropic spawn stages")
}

func TestB077PersonaClassAndEffortDoNotSelectTheModel(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	preflight := normalizeContractText(e36F04RawSection(t, run,
		"### Route attestation pre-flight (B077)",
		"Claim the concrete entity returned by Shark:"))

	for _, invariant := range []string{
		"response.agent_type",
		"host worker class",
		"response.effort",
		"are separate values",
		"do not infer the model from response.agent_type, host worker class, or response.effort",
		"documented compatible host selector",
	} {
		assert.Contains(t, preflight, invariant)
	}
}

func TestB077EmptyModelUsesADocumentedObservedDefault(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	claudeAdapter := normalizeContractText(e36F04RawSection(t, run,
		"### Claude Code Agent tool adapter",
		"### Host-neutral handoff and lifecycle"))
	defaultContract := normalizeContractText(e36F04RawSection(t, contract,
		"## Host route attestation (ephemeral adapter metadata)",
		"## Lease lifetime"))

	assert.Contains(t, claudeAdapter, "response.model is empty")
	assert.Contains(t, claudeAdapter, "claude's documented provider default")
	assert.Contains(t, claudeAdapter, "capture the worker-correlated effective provider and concrete model")
	assert.Contains(t, defaultContract, "when requested_model is empty, keep it empty")
	assert.Contains(t, defaultContract, "effective_model records that observed value")
	assert.Contains(t, defaultContract, "if the provider default or its identity cannot be evidenced, reject before claim")
}

func TestB077ResponseModelIsProviderNeutralAndClaudeSelectorIsScoped(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	modelDescription := normalizeContractText(b077MarkdownTableRow(t, run, "response.model"))
	assert.Contains(t, modelDescription, "requested model")
	assert.Contains(t, modelDescription, "regardless of provider")
	assert.NotContains(t, modelDescription, "anthropic")

	claudeAdapter := normalizeContractText(e36F04RawSection(t, run,
		"### Claude Code Agent tool adapter",
		"### Host-neutral handoff and lifecycle"))
	assert.Contains(t, claudeAdapter, "response.provider")
	assert.Contains(t, claudeAdapter, "anthropic")
	assert.Contains(t, claudeAdapter, "agent tool's model parameter")
}

func TestB077RouteMatrixDefinesEveryPhaseAndFieldInvariant(t *testing.T) {
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	rows := b077RouteMatrixRows(t, contract)
	require.Len(t, rows, 5, "route-state matrix must define all five valid phase/status rows")

	actual := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		key := strings.ToLower(row["phase"]) + "/" + strings.ToLower(row["route_status"])
		require.NotContains(t, actual, key, "route-state rows must be unique")
		actual[key] = row
	}

	base := []string{"route_status", "requested_provider", "requested_model", "host_worker_class", "evidence_ref"}
	selected := []string{"selected_provider", "selected_model"}
	worker := []string{"worker_id"}
	effective := []string{"effective_provider", "effective_model"}
	for _, testCase := range b077ExpectedRouteStates(base, selected, worker, effective) {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			row, ok := actual[testCase.name]
			require.True(t, ok, "missing route-state row %q", testCase.name)
			assertFieldSet(t, testCase.name+" required fields", row["required fields"], testCase.state.required)
			assertFieldSet(t, testCase.name+" forbidden fields", row["forbidden fields"], testCase.state.forbidden)
			assertCellContains(t, testCase.name+" evidence", row["evidence"], testCase.state.evidence)
			assertCellContains(t, testCase.name+" launch disposition", row["launch disposition"], testCase.state.disposition)
		})
	}
}

func TestB077RouteIdentityGlossaryAndWorkerSchemaStaySeparated(t *testing.T) {
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	resultFields := b077MarkdownTableRow(t,
		e36F04RawSection(t, contract, "## Result fields (adapter -> parent)", "## Host route attestation"),
		"session_id")
	assert.Contains(t, resultFields, "parent's own claim/session identity")
	assert.NotContains(t, strings.ToLower(resultFields), "provider identity")

	routeSection := e36F04RawSection(t, contract, "## Host route attestation", "## Lease lifetime")
	selectedProvider := b077MarkdownTableRow(t, routeSection, "selected_provider")
	assert.Contains(t, selectedProvider, "explicitly selected by the host route policy")
	selectedModel := b077MarkdownTableRow(t, routeSection, "selected_model")
	assert.Contains(t, selectedModel, "passed through its selector")
	for _, invariant := range []string{
		"host_session_id",
		"never use session_id as host identity or correlation",
		"route attestation itself is not a worker control envelope",
		"mismatch means either the independently observed provider or model differs from the selected provider or model",
		"selected_provider and selected_model record that selection separately",
		"does not claim semantic equivalence between the requested and selected models",
	} {
		assert.Contains(t, normalizeContractText(routeSection), normalizeContractText(invariant))
	}

	canonical := readF09RepositoryFile(t, "skills/shark-attack/context/worker-control-schema.yaml")
	embedded := readF09RepositoryFile(t, "internal/sharkdata/default_data/skills/shark-attack/context/worker-control-schema.yaml")
	require.Equal(t, canonical, embedded, "worker-control schema copies must remain identical")
	assert.Contains(t, canonical, "kind: final|question|needs_council|blocked_external|failed")
	assert.Contains(t, canonical, "example_blocked_external:\n  kind: blocked_external\n  evidence: []")
	assert.NotContains(t, canonical, "route_status")
	assert.NotContains(t, canonical, "pre-dispatch rejection")
}

func TestB077CodexModelSelectorEvidenceIsCLIOnlyAndCopiesMatch(t *testing.T) {
	canonical := readF09RepositoryFile(t, "skills/shark-attack/providers/codex.md")
	embedded := readF09RepositoryFile(t, "internal/sharkdata/default_data/skills/shark-attack/providers/codex.md")
	require.Equal(t, canonical, embedded, "canonical and embedded Codex provider references must match")
	supported := normalizeContractText(e36F04RawSection(t, canonical, "## Supported Operations", "## Unsupported Operations"))
	assert.Contains(t, canonical, "`codex exec -m, --model <MODEL>`")
	assert.Contains(t, canonical, "proves selector availability only")
	assert.Contains(t, canonical, "does not establish native host model selection")
	assert.Contains(t, canonical, "worker-correlated effective provider/model identity")
	assert.Contains(t, canonical, "session_meta.payload.model_provider=openai")
	assert.Contains(t, canonical, "turn_context.payload.model=gpt-6.2-astra")
	assert.Contains(t, supported, "thread.started event's thread_id as the codex worker identity")
	assert.Contains(t, supported, "requiring session_meta.payload.id to equal the event's thread id")
	assert.Contains(t, supported, "effective provider from session_meta.payload.model_provider")
	assert.Contains(t, supported, "effective model from each matching turn_context.payload.model")
	assert.Contains(t, supported, "missing or ambiguous records/fields are unverified")
	assert.Contains(t, supported, "if multiple records for the id disagree, classify the identity as unverified")
	assert.Contains(t, supported, "fail closed when the event, path, or record shape is unavailable")
}

type b077RouteState struct {
	required    []string
	forbidden   []string
	evidence    []string
	disposition []string
}

type b077RouteCase struct {
	name  string
	state b077RouteState
}

func b077ExpectedRouteStates(base, selected, worker, effective []string) []b077RouteCase {
	return []b077RouteCase{
		{name: "preflight/unsupported", state: b077RouteState{
			required:    base,
			forbidden:   []string{"selected_provider", "selected_model", "worker_id", "host_session_id", "effective_provider", "effective_model"},
			evidence:    []string{"no exact host selector", "documented compatible fallback exists"},
			disposition: []string{"reject before claim", "do not spawn", "no status advance"},
		}},
		{name: "preflight/unverified", state: b077RouteState{
			required:    append(append([]string{}, base...), selected...),
			forbidden:   []string{"worker_id", "host_session_id", "effective_provider", "effective_model"},
			evidence:    []string{"explicit selected route exists", "identity-capture evidence is missing or inconclusive"},
			disposition: []string{"reject before claim", "do not spawn", "no status advance"},
		}},
		{name: "post-dispatch/verified", state: b077RouteState{
			required:    append(append(append([]string{}, base...), selected...), append(worker, effective...)...),
			forbidden:   []string{},
			evidence:    []string{"requested route or documented compatible fallback", "worker-correlated observed provider and model match the selected provider and model"},
			disposition: []string{"launch permitted after preflight", "worker result may enter its existing envelope path"},
		}},
		{name: "post-dispatch/unverified", state: b077RouteState{
			required:    append(append(append([]string{}, base...), selected...), worker...),
			forbidden:   effective,
			evidence:    []string{"worker-correlated observation is missing or inconclusive"},
			disposition: []string{"reject the worker result", "stop the worker when supported", "do not advance"},
		}},
		{name: "post-dispatch/mismatch", state: b077RouteState{
			required:    append(append(append([]string{}, base...), selected...), append(worker, effective...)...),
			forbidden:   []string{},
			evidence:    []string{"observed provider or model differs from the selected provider or model"},
			disposition: []string{"reject the worker result", "stop the worker when supported", "do not advance"},
		}},
	}
}

func b077RouteMatrixRows(t *testing.T, contract string) []map[string]string {
	t.Helper()
	section := e36F04RawSection(t, contract, "### Phase/status matrix", "\n`unsupported` means")
	result := b077MarkdownTableRows(t, section, "| Phase | route_status |")
	for _, row := range result {
		require.Len(t, row, 6, "route-state matrix must keep its six named columns")
	}
	return result
}

func b077MarkdownTableRows(t *testing.T, content, headerContains string) []map[string]string {
	t.Helper()
	headers := b077MarkdownTableHeaders(t, content, headerContains)
	rows := f09ParseMarkdownTable(t, content, headerContains)
	result := make([]map[string]string, 0, len(rows))
	for _, cells := range rows {
		require.Len(t, cells, len(headers), "Markdown table row must match its header width")
		row := make(map[string]string, len(headers))
		for index, header := range headers {
			row[strings.ToLower(header)] = cells[index]
		}
		result = append(result, row)
	}
	return result
}

func b077MarkdownTableHeaders(t *testing.T, content, headerContains string) []string {
	t.Helper()
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") && strings.Contains(line, headerContains) {
			cells := strings.Split(strings.Trim(line, "|"), "|")
			for index := range cells {
				cells[index] = strings.TrimSpace(cells[index])
			}
			return cells
		}
	}
	require.FailNow(t, "Markdown table header not found", "header containing %q", headerContains)
	return nil
}

func b077MarkdownTableRow(t *testing.T, text, field string) string {
	t.Helper()
	for _, row := range b077MarkdownTableRows(t, text, "| Field | Meaning |") {
		if strings.Trim(row["field"], "`") == field {
			return row["meaning"]
		}
	}
	require.FailNow(t, "field table has no row", "field %q", field)
	return ""
}

func assertFieldSet(t *testing.T, label, cell string, expected []string) {
	t.Helper()
	actual := make([]string, 0)
	for _, item := range strings.Split(cell, ",") {
		item = strings.TrimSpace(item)
		if item == "" || strings.EqualFold(item, "none") {
			continue
		}
		actual = append(actual, strings.Trim(item, "`"))
	}
	assert.ElementsMatch(t, expected, actual, "B077: %s", label)
}

func assertCellContains(t *testing.T, label, cell string, expected []string) {
	t.Helper()
	cell = normalizeContractText(cell)
	for _, phrase := range expected {
		assert.Contains(t, cell, normalizeContractText(phrase), "B077: %s", label)
	}
}

func normalizeContractText(text string) string {
	text = strings.ToLower(strings.ReplaceAll(text, "`", ""))
	return f09NormalizeWhitespace(text)
}
