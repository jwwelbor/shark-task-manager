package sharkdata

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Sentinel errors for table_parser.go's three entrypoints. AC-T3 requires
// malformed or missing table input to return a typed error, never a panic
// or a silently empty slice.
var (
	// ErrTableNotFound means no markdown table with the expected header row
	// was found anywhere in the input.
	ErrTableNotFound = errors.New("table not found")
	// ErrTableEmpty means the expected header/separator row pair was found
	// but no data rows follow it.
	ErrTableEmpty = errors.New("table has no data rows")
	// ErrTableRowMalformed means a table row's column count or a cell's
	// value didn't match what the header/schema requires.
	ErrTableRowMalformed = errors.New("table row is malformed")
	// ErrSectionNotFound means an expected "## ..." heading was not found.
	ErrSectionNotFound = errors.New("section heading not found")
	// ErrTierGateBlank means one of tier-matrix.md's gate-determining
	// columns (Same-model gate / Separate QA / Final UAT) was blank for a
	// row. This is distinct from InteractionRow's blank-field tolerance:
	// here a blank cell would silently derive an incomplete RequiredGates
	// list, which AC-T4 requires to fail loud instead.
	ErrTierGateBlank = errors.New("tier matrix gate column is blank")
)

// ---------------------------------------------------------------------------
// Generic markdown table extraction
//
// All three parsers below share one strict-header table finder. A table is
// identified by an exact match (after per-cell trimming) on its header row,
// so callers never accidentally parse the wrong table out of a document that
// contains several (e.g. architecture.md repeats a "| Field | Type |
// Contract |" header once per I-0X section).
// ---------------------------------------------------------------------------

var separatorCellPattern = regexp.MustCompile(`^:?-+:?$`)

// splitTableRow splits a single markdown "| a | b | c |" line into trimmed
// cells. ok is false when the line is not a pipe-delimited table row.
func splitTableRow(line string) (cells []string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return nil, false
	}
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells = make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells, true
}

func isSeparatorRow(line string) bool {
	cells, ok := splitTableRow(line)
	if !ok {
		return false
	}
	for _, c := range cells {
		if !separatorCellPattern.MatchString(c) {
			return false
		}
	}
	return true
}

func headerMatches(cells, header []string) bool {
	if len(cells) != len(header) {
		return false
	}
	for i := range cells {
		if cells[i] != header[i] {
			return false
		}
	}
	return true
}

// findMarkdownTable locates the first table in md whose header row cells
// equal header exactly, and returns its data rows (each row has
// len(header) cells, in header order). It returns a typed error — never a
// panic or a silently empty slice — when the header can't be found, the
// header isn't followed by a valid separator row, a data row's column count
// doesn't match the header, or the table has zero data rows.
func findMarkdownTable(md []byte, header []string) ([][]string, error) {
	lines := strings.Split(string(md), "\n")

	headerIdx := -1
	for i, line := range lines {
		cells, ok := splitTableRow(line)
		if !ok {
			continue
		}
		if headerMatches(cells, header) {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return nil, fmt.Errorf("%w: header %v", ErrTableNotFound, header)
	}
	if headerIdx+1 >= len(lines) || !isSeparatorRow(lines[headerIdx+1]) {
		return nil, fmt.Errorf("%w: header %v not followed by a separator row", ErrTableRowMalformed, header)
	}

	var rows [][]string
	for i := headerIdx + 2; i < len(lines); i++ {
		cells, ok := splitTableRow(lines[i])
		if !ok {
			break
		}
		if len(cells) != len(header) {
			return nil, fmt.Errorf("%w: expected %d columns, got %d in row %q", ErrTableRowMalformed, len(header), len(cells), lines[i])
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: header %v", ErrTableEmpty, header)
	}
	return rows, nil
}

// extractHeadingSection returns the byte range of md starting at the first
// "## " heading whose text contains marker, up to (excluding) the next "## "
// heading or end of file. Used to scope a table lookup to one architecture.md
// section (I-01..I-05 share a "| Field | Type | Contract |" header shape, so
// an unscoped search could match the wrong section).
func extractHeadingSection(md []byte, marker string) ([]byte, error) {
	lines := strings.Split(string(md), "\n")

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") && strings.Contains(line, marker) {
			start = i
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("%w: %q", ErrSectionNotFound, marker)
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	return []byte(strings.Join(lines[start:end], "\n")), nil
}

// ---------------------------------------------------------------------------
// AC-T1: Interaction Contracts table parser
// ---------------------------------------------------------------------------

// InteractionRow is one row of E34-interaction-map.md's Interaction
// Contracts table (one row per I-## entry).
type InteractionRow struct {
	ID          string
	Producer    string
	Consumers   []string
	ShapeSource string
	Payload     string
	Style       string
}

var interactionMapHeader = []string{
	"ID", "Producer feature", "Consumer feature(s)", "Shape source", "Payload", "Style",
}

// ParseInteractionMapTable parses E34-interaction-map.md's Interaction
// Contracts table into one InteractionRow per I-## entry. This is the
// registry table (producer/consumer/shape-source/payload/style) — it is a
// different shape from architecture.md's per-interaction "## I-0X ..."
// sections, which ParseI05FieldList reads instead.
//
// A row with one or more blank cells is still returned: validating field
// completeness is CheckInteractionMapCompleteness's job (T-E34-F08-011), not
// this parser's. Only structural problems (no matching table, a data row
// with the wrong column count, or zero data rows) are errors.
func ParseInteractionMapTable(interactionMapMD []byte) ([]InteractionRow, error) {
	rows, err := findMarkdownTable(interactionMapMD, interactionMapHeader)
	if err != nil {
		return nil, fmt.Errorf("parse interaction contracts table: %w", err)
	}

	result := make([]InteractionRow, 0, len(rows))
	for _, cells := range rows {
		result = append(result, InteractionRow{
			ID:          cells[0],
			Producer:    cells[1],
			Consumers:   splitConsumerList(cells[2]),
			ShapeSource: cells[3],
			Payload:     cells[4],
			Style:       cells[5],
		})
	}
	return result, nil
}

// splitConsumerList splits a "Consumer feature(s)" cell such as
// "E34-F06, E34-F07, E34-F08" into its individual feature keys. A blank cell
// yields a nil (not empty-but-non-nil) slice, distinguishable from a
// single-consumer row.
func splitConsumerList(cell string) []string {
	if strings.TrimSpace(cell) == "" {
		return nil
	}
	parts := strings.Split(cell, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// AC-T2: I-05 field-list parser
// ---------------------------------------------------------------------------

var fieldTypeContractHeader = []string{"Field", "Type", "Contract"}

// ParseI05FieldList parses the I-05 CanonicalAdoptionManifest v1 field list
// out of architecture.md's "## I-05 CanonicalAdoptionManifest v1" section
// into an ordered field-name slice (backticks stripped). Scoping to the I-05
// heading matters: I-01 through I-04 each carry their own
// "| Field | Type | Contract |" table, so an unscoped table search could
// silently return the wrong section's fields.
func ParseI05FieldList(architectureMD []byte) ([]string, error) {
	section, err := extractHeadingSection(architectureMD, "I-05")
	if err != nil {
		return nil, fmt.Errorf("parse I-05 field list: %w", err)
	}

	rows, err := findMarkdownTable(section, fieldTypeContractHeader)
	if err != nil {
		return nil, fmt.Errorf("parse I-05 field list: %w", err)
	}

	fields := make([]string, 0, len(rows))
	for _, cells := range rows {
		fields = append(fields, strings.Trim(cells[0], "`"))
	}
	return fields, nil
}

// ---------------------------------------------------------------------------
// AC-T4: tier-matrix.md tier table parser
// ---------------------------------------------------------------------------

// TierRow is one row of tier-matrix.md's tier table: the tier name plus
// requirements derived from the matrix's Planning/Test-source and gate
// columns. tier-matrix.md does not exist yet (owned by T-E34-F08-002, which
// will carry feature.md's "## Tier contract" table verbatim per REQ-F-001);
// this parser is shape-driven off that exact, already-committed table, not
// off a guessed future format.
type TierRow struct {
	Tier              string
	RequiredArtifacts []string
	RequiredGates     []string
}

var tierMatrixHeader = []string{
	"Tier", "Planning source", "Test source", "Same-model gate", "Separate QA", "Final UAT",
}

// artifactFilenamePattern matches a backtick-quoted "*.md" filename, e.g.
// "`spec.md`".
var artifactFilenamePattern = regexp.MustCompile("`([\\w.-]+\\.md)`")

// ParseTierMatrixTable parses tier-matrix.md's tier table into one TierRow
// per tier.
//
// RequiredArtifacts is every backtick-quoted "*.md" filename named in a
// row's "Planning source" or "Test source" cells, in first-seen order
// (dedup'd) — e.g. SIMPLE's "`feature.md` and validated `research-report.md`"
// yields ["feature.md", "research-report.md"]; STANDARD/COMPLEX's
// "`spec.md` and `test-plan.md`" yields ["spec.md", "test-plan.md"].
//
// RequiredGates maps the row's gate columns onto this repo's canonical
// feature workflow step names
// (internal/sharkdata/default_data/workflow/feature.yaml):
//   - "Same-model gate" (any non-blank value, combined or craft) requires
//     the "code_review" step.
//   - "Separate QA" == "Yes" additionally requires the "qa" step
//     (COMPLEX-only deep verification); "No" requires nothing further.
//   - "Final UAT" == "Yes" additionally requires the "approval" step
//     (UAT red-team review); "No" requires nothing further.
//
// A blank cell in any of the three gate columns is a malformed row: it is
// reported as a typed error (wrapping ErrTierGateBlank), not silently
// folded into an incomplete RequiredGates slice — AC-T4's explicit
// requirement, since a silently-short gate list would be worse than an
// obvious parse failure.
func ParseTierMatrixTable(tierMatrixMD []byte) ([]TierRow, error) {
	rows, err := findMarkdownTable(tierMatrixMD, tierMatrixHeader)
	if err != nil {
		return nil, fmt.Errorf("parse tier matrix table: %w", err)
	}

	result := make([]TierRow, 0, len(rows))
	for _, cells := range rows {
		tier := cells[0]
		gates, err := deriveTierGates(tier, cells[3], cells[4], cells[5])
		if err != nil {
			return nil, err
		}
		result = append(result, TierRow{
			Tier:              tier,
			RequiredArtifacts: extractArtifactFilenames(cells[1], cells[2]),
			RequiredGates:     gates,
		})
	}
	return result, nil
}

func extractArtifactFilenames(cells ...string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, cell := range cells {
		for _, m := range artifactFilenamePattern.FindAllStringSubmatch(cell, -1) {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

func deriveTierGates(tier, sameModelGate, separateQA, finalUAT string) ([]string, error) {
	if strings.TrimSpace(sameModelGate) == "" {
		return nil, fmt.Errorf("%w: tier %q same-model gate column", ErrTierGateBlank, tier)
	}
	gates := []string{"code_review"}

	switch strings.TrimSpace(separateQA) {
	case "":
		return nil, fmt.Errorf("%w: tier %q separate QA column", ErrTierGateBlank, tier)
	case "Yes":
		gates = append(gates, "qa")
	case "No":
		// no separate QA gate for this tier
	default:
		return nil, fmt.Errorf("%w: tier %q separate QA column has unrecognized value %q", ErrTableRowMalformed, tier, separateQA)
	}

	switch strings.TrimSpace(finalUAT) {
	case "":
		return nil, fmt.Errorf("%w: tier %q final UAT column", ErrTierGateBlank, tier)
	case "Yes":
		gates = append(gates, "approval")
	case "No":
		// no UAT gate for this tier
	default:
		return nil, fmt.Errorf("%w: tier %q final UAT column has unrecognized value %q", ErrTableRowMalformed, tier, finalUAT)
	}

	return gates, nil
}
