package sharkdata

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Sentinel errors for CheckInteractionMapCompleteness (T-E34-F08-011,
// TC-014). These are two distinct error classes per AC-8/AC-T2: an empty
// required field is a different problem from a non-empty shape-source link
// whose anchor doesn't resolve to any architecture.md heading, and a caller
// must be able to tell them apart with errors.Is.
var (
	// ErrInteractionRowFieldMissing means one of InteractionRow's five
	// required fields (producer, consumer(s), shape-source, payload, style)
	// is empty.
	ErrInteractionRowFieldMissing = errors.New("interaction row missing required field")
	// ErrInteractionRowAnchorUnresolved means a row's shape-source field is
	// non-empty but its link anchor does not match any "## I-0X ..." heading
	// in architecture.md.
	ErrInteractionRowAnchorUnresolved = errors.New("interaction row shape-source anchor does not resolve to an architecture.md heading")
)

// i0xHeadingPattern matches an architecture.md "## I-0X ..." heading line,
// capturing the heading text (without the leading "## ").
var i0xHeadingPattern = regexp.MustCompile(`(?m)^##\s+(I-\d+\s.+)$`)

// mdLinkTargetPattern extracts the "(target)" portion of a markdown link
// cell such as "[label](target)".
var mdLinkTargetPattern = regexp.MustCompile(`\(([^)]+)\)`)

// nonSlugCharPattern matches any character that isn't a lowercase letter,
// digit, or hyphen, for stripping while computing a GitHub-style heading
// anchor slug.
var nonSlugCharPattern = regexp.MustCompile(`[^a-z0-9\- ]`)

// headingAnchorSlug converts a markdown heading's text into the GitHub-style
// anchor fragment (without the leading "#") that a "[text](#anchor)" link
// would target: lowercase, strip characters other than letters/digits/
// hyphens/spaces, then replace spaces with hyphens.
func headingAnchorSlug(heading string) string {
	lower := strings.ToLower(strings.TrimSpace(heading))
	stripped := nonSlugCharPattern.ReplaceAllString(lower, "")
	return strings.Join(strings.Fields(stripped), "-")
}

// extractI0XHeadingAnchors returns the set of GitHub-style anchor slugs for
// every "## I-0X ..." heading in architectureMD.
func extractI0XHeadingAnchors(architectureMD []byte) map[string]bool {
	anchors := make(map[string]bool)
	for _, m := range i0xHeadingPattern.FindAllSubmatch(architectureMD, -1) {
		anchors[headingAnchorSlug(string(m[1]))] = true
	}
	return anchors
}

// extractLinkAnchor pulls the "#anchor" fragment out of a shape-source cell
// such as "[I-01 ReadinessEvidence v1](./architecture.md#i-01-readinessevidence-v1)".
// ok is false when cell contains no parenthesized link target or the link
// target has no "#" fragment.
func extractLinkAnchor(cell string) (anchor string, ok bool) {
	m := mdLinkTargetPattern.FindStringSubmatch(cell)
	if m == nil {
		return "", false
	}
	idx := strings.Index(m[1], "#")
	if idx == -1 {
		return "", false
	}
	return m[1][idx+1:], true
}

// CheckInteractionMapCompleteness proves every row in E34-interaction-map.md's
// Interaction Contracts table names all five required fields — producer,
// consumer(s), shape-source link, payload, and style — and that each row's
// shape-source link resolves to an actual "## I-0X ..." heading in
// architecture.md.
//
// Kept structurally separate from ParseInteractionMapTable (T-E34-F08-001):
// this function's fixture subtests construct []InteractionRow directly,
// bypassing the parser, so a parser bug cannot mask a checker bug or vice
// versa. architectureMD is a second, explicit document input — shape-source
// anchors are resolved against architecture.md's actual heading set rather
// than a hardcoded list, so an architecture.md edit is caught here too.
//
// Returns one error per problem found, distinguishing two classes:
//   - ErrInteractionRowFieldMissing — a required field is empty.
//   - ErrInteractionRowAnchorUnresolved — a non-empty shape-source link whose
//     anchor does not match any architecture.md "## I-0X ..." heading.
func CheckInteractionMapCompleteness(rows []InteractionRow, architectureMD []byte) []error {
	anchors := extractI0XHeadingAnchors(architectureMD)

	var errs []error
	for _, row := range rows {
		if strings.TrimSpace(row.Producer) == "" {
			errs = append(errs, fmt.Errorf("%w: row %s missing producer", ErrInteractionRowFieldMissing, row.ID))
		}
		if len(row.Consumers) == 0 {
			errs = append(errs, fmt.Errorf("%w: row %s missing consumer", ErrInteractionRowFieldMissing, row.ID))
		}
		if strings.TrimSpace(row.ShapeSource) == "" {
			errs = append(errs, fmt.Errorf("%w: row %s missing shape-source", ErrInteractionRowFieldMissing, row.ID))
		} else if anchor, ok := extractLinkAnchor(row.ShapeSource); !ok || !anchors[anchor] {
			errs = append(errs, fmt.Errorf("%w: row %s shape-source anchor %q", ErrInteractionRowAnchorUnresolved, row.ID, anchor))
		}
		if strings.TrimSpace(row.Payload) == "" {
			errs = append(errs, fmt.Errorf("%w: row %s missing payload", ErrInteractionRowFieldMissing, row.ID))
		}
		if strings.TrimSpace(row.Style) == "" {
			errs = append(errs, fmt.Errorf("%w: row %s missing style", ErrInteractionRowFieldMissing, row.ID))
		}
	}
	return errs
}
