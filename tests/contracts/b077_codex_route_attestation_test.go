package contracts

import (
	"sort"
	"strings"
	"testing"
)

func TestB077CodexRouteContractKeepsPreflightOutsideWorkerResults(t *testing.T) {
	run := readF09RepositoryFile(t, "skills/shark-rider/verbs/run.md")
	preflight := sectionBetween(t, run,
		"### Route attestation pre-flight (B077)",
		"Claim the concrete entity returned by Shark:")
	preflight = normalizeContractText(preflight)

	claim := strings.Index(run, "Claim the concrete entity returned by Shark:")
	spawn := strings.Index(run, "Spawn the host worker using a host-safe adapter:")
	preflightStart := strings.Index(run, "### Route attestation pre-flight (B077)")
	if preflightStart < 0 || claim < 0 || spawn < 0 || preflightStart > claim || preflightStart > spawn {
		t.Fatal("B077: route attestation must precede claim and worker spawn")
	}
	for _, invariant := range []string{
		"parent/adapter pre-dispatch rejection",
		"no worker control envelope or worker result",
		"no claim",
		"no status advance",
		"blocked_external is reserved for a dispatched worker",
		"requested provider/model",
		"do not use the parent's model as an implicit fallback",
	} {
		if !strings.Contains(strings.ToLower(preflight), strings.ToLower(invariant)) {
			t.Errorf("B077: pre-flight must state %q", invariant)
		}
	}
	if strings.Contains(strings.ToLower(preflight), "pre-dispatch blocked_external result") {
		t.Fatal("B077: a pre-dispatch rejection cannot be a worker blocked_external result")
	}

	modelDescription := markdownTableRow(t, run, "response.model")
	modelDescription = strings.ToLower(modelDescription)
	if !strings.Contains(modelDescription, "requested model") || !strings.Contains(modelDescription, "regardless of provider") {
		t.Fatalf("B077: response.model must name the requested model for any provider; got %q", modelDescription)
	}
	if strings.Contains(modelDescription, "anthropic") {
		t.Fatalf("B077: response.model wire semantics must not be Anthropic-only: %q", modelDescription)
	}
	claudeAdapter := sectionBetween(t, run,
		"### Claude Code Agent tool adapter",
		"### Host-neutral handoff and lifecycle")
	if !strings.Contains(claudeAdapter, "response.provider") || !strings.Contains(claudeAdapter, "anthropic") || !strings.Contains(claudeAdapter, "Agent tool's `model` parameter") {
		t.Fatal("B077: Claude-specific model compatibility rules must remain under its adapter subsection")
	}
}

func TestB077RouteMatrixDefinesEveryPhaseAndFieldInvariant(t *testing.T) {
	contract := readF09RepositoryFile(t, "skills/shark-rider/context/host-adapter-contract.md")
	rows := markdownTable(t, contract, "### Phase/status matrix")
	if len(rows) != 5 {
		t.Fatalf("B077: route-state matrix must define all five valid phase/status rows, got %d", len(rows))
	}

	base := []string{"route_status", "requested_provider", "requested_model", "host_worker_class", "evidence_ref"}
	worker := []string{"worker_id"}
	effective := []string{"effective_provider", "effective_model"}
	expected := []struct {
		phase       string
		status      string
		required    []string
		forbidden   []string
		evidence    []string
		disposition []string
	}{
		{
			phase: "preflight", status: "unsupported", required: base,
			forbidden:   []string{"worker_id", "host_session_id", "effective_provider", "effective_model"},
			evidence:    []string{"evidence proves", "no explicit host model selector"},
			disposition: []string{"reject before claim", "do not spawn", "no status advance"},
		},
		{
			phase: "preflight", status: "unverified", required: base,
			forbidden:   []string{"worker_id", "host_session_id", "effective_provider", "effective_model"},
			evidence:    []string{"bounded reference", "missing or inconclusive"},
			disposition: []string{"reject before claim", "do not spawn", "no status advance"},
		},
		{
			phase: "post-dispatch", status: "verified", required: append(append([]string{}, base...), append(worker, effective...)...),
			forbidden:   []string{},
			evidence:    []string{"explicit model-selector mapping", "worker-correlated observed provider and model", "match the request"},
			disposition: []string{"launch permitted after preflight", "worker result may enter its existing envelope path"},
		},
		{
			phase: "post-dispatch", status: "unverified", required: append(append([]string{}, base...), worker...),
			forbidden:   effective,
			evidence:    []string{"worker-correlated observation is missing or inconclusive"},
			disposition: []string{"reject the worker result", "stop the worker when supported", "do not advance"},
		},
		{
			phase: "post-dispatch", status: "mismatch", required: append(append([]string{}, base...), append(worker, effective...)...),
			forbidden:   []string{},
			evidence:    []string{"worker-correlated observed provider and model", "differ from the request"},
			disposition: []string{"reject the worker result", "stop the worker when supported", "do not advance"},
		},
	}

	actual := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		key := strings.ToLower(row["phase"]) + "/" + strings.ToLower(row["route_status"])
		if _, exists := actual[key]; exists {
			t.Fatalf("B077: duplicate route-state row %q", key)
		}
		actual[key] = row
	}
	if len(actual) != len(expected) {
		t.Fatalf("B077: expected %d unique route-state rows, got %d", len(expected), len(actual))
	}

	for _, want := range expected {
		key := want.phase + "/" + want.status
		row, ok := actual[key]
		if !ok {
			t.Errorf("B077: missing route-state matrix row %q", key)
			continue
		}
		assertFieldSet(t, key+" required fields", row["required fields"], want.required)
		assertFieldSet(t, key+" forbidden fields", row["forbidden fields"], want.forbidden)
		assertCellContains(t, key+" evidence", row["evidence"], want.evidence)
		assertCellContains(t, key+" launch disposition", row["launch disposition"], want.disposition)
	}

	resultFields := markdownTableRow(t, sectionBetween(t, contract,
		"## Result fields (adapter -> parent)", "## Host route attestation"), "session_id")
	if !strings.Contains(resultFields, "parent's own claim/session identity") || strings.Contains(strings.ToLower(resultFields), "provider identity") {
		t.Fatalf("B077: session_id must remain exclusively the Shark claim identity; got %q", resultFields)
	}
	routeSection := sectionBetween(t, contract, "## Host route attestation", "## Lease lifetime")
	for _, invariant := range []string{
		"host_session_id",
		"never use session_id as host identity or correlation",
		"route attestation itself is not a worker control envelope",
	} {
		if !strings.Contains(normalizeContractText(routeSection), normalizeContractText(invariant)) {
			t.Errorf("B077: route identity contract must state %q", invariant)
		}
	}
}

func TestB077CodexReferenceAndWorkerSchemasKeepNativeRouteUnverified(t *testing.T) {
	codex := readF09RepositoryFile(t, "skills/shark-attack/providers/codex.md")
	if !strings.Contains(codex, "Scope: OpenAI Codex CLI only") ||
		!strings.Contains(codex, "does not establish native host model selection") ||
		!strings.Contains(codex, "effective provider/model identity for a native host worker") {
		t.Fatal("B077: Codex CLI reference must not claim native host selector or identity evidence")
	}

	canonical := readF09RepositoryFile(t, "skills/shark-attack/context/worker-control-schema.yaml")
	embedded := readF09RepositoryFile(t, "internal/sharkdata/default_data/skills/shark-attack/context/worker-control-schema.yaml")
	if canonical != embedded {
		t.Fatal("B077: canonical and embedded worker-control schema copies must remain identical")
	}
	if !strings.Contains(canonical, "kind: final|question|needs_council|blocked_external|failed") ||
		!strings.Contains(canonical, "example_blocked_external:\n  kind: blocked_external\n  evidence: []") {
		t.Fatal("B077: blocked_external must remain a dispatched worker control-envelope variant")
	}
	if strings.Contains(canonical, "route_status") || strings.Contains(canonical, "pre-dispatch rejection") {
		t.Fatal("B077: host route attestation must not be added to the Shark worker envelope schema")
	}
}

func markdownTableRow(t *testing.T, text, field string) string {
	t.Helper()
	for _, row := range markdownTable(t, text, "") {
		if strings.Trim(row["field"], "`") == field {
			return row["meaning"]
		}
	}
	t.Fatalf("markdown table has no row for %q", field)
	return ""
}

func markdownTable(t *testing.T, text, heading string) []map[string]string {
	t.Helper()
	if heading != "" {
		start := strings.Index(text, heading)
		if start < 0 {
			t.Fatalf("missing contract table heading %q", heading)
		}
		text = text[start+len(heading):]
	}
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			start = i
			break
		}
	}
	if start < 0 || start+1 >= len(lines) {
		t.Fatalf("no markdown table found after %q", heading)
	}
	headers := splitMarkdownCells(lines[start])
	rows := make([]map[string]string, 0)
	for _, line := range lines[start+2:] {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			break
		}
		cells := splitMarkdownCells(line)
		if len(cells) != len(headers) {
			t.Fatalf("markdown table row has %d cells, expected %d: %s", len(cells), len(headers), line)
		}
		row := make(map[string]string, len(headers))
		for i, header := range headers {
			row[strings.ToLower(header)] = cells[i]
		}
		rows = append(rows, row)
	}
	return rows
}

func splitMarkdownCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
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
	sort.Strings(actual)
	want := append([]string(nil), expected...)
	sort.Strings(want)
	if strings.Join(actual, ",") != strings.Join(want, ",") {
		t.Errorf("B077: %s expected fields %v, got %v", label, want, actual)
	}
}

func assertCellContains(t *testing.T, label, cell string, expected []string) {
	t.Helper()
	cell = normalizeContractText(cell)
	for _, phrase := range expected {
		if !strings.Contains(cell, normalizeContractText(phrase)) {
			t.Errorf("B077: %s must state %q; got %q", label, phrase, cell)
		}
	}
}

func sectionBetween(t *testing.T, text, start, end string) string {
	t.Helper()
	startIndex := strings.Index(text, start)
	if startIndex < 0 {
		t.Fatalf("missing contract section marker %q", start)
	}
	endIndex := strings.Index(text[startIndex+len(start):], end)
	if endIndex < 0 {
		t.Fatalf("missing contract section end marker %q after %q", end, start)
	}
	return text[startIndex+len(start) : startIndex+len(start)+endIndex]
}

func normalizeContractText(text string) string {
	text = strings.ToLower(strings.ReplaceAll(text, "`", ""))
	return strings.Join(strings.Fields(text), " ")
}
