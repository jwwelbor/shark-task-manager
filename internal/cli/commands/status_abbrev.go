package commands

import "strings"

// abbreviateStatus shortens a workflow status name to ≤4 characters for use
// in narrow list-view columns. Reduces e.g. "in_development" → "DEV",
// "ready_for_code_review" → "R-CR", "completed" → "DONE", so the Status
// column in `shark task list` does not consume 16+ chars on every row.
//
// Strategy:
//  1. Look up a hand-tuned abbreviation for known canonical statuses across
//     the shipped short and long workflows (predictable, collision-free,
//     reads naturally — "DEV", "QA", "DONE").
//  2. Fall back to algorithmic reduction for unknown statuses: strip the
//     "in_" / "ready_for_" prefix, take the first 4 chars of the payload
//     uppercased, and prepend "R-" for ready-for variants.
//
// The output is always uppercase so it visually distinguishes from titles
// and reads consistently across rows.
func abbreviateStatus(status string) string {
	if status == "" {
		return ""
	}
	if abbrev, ok := knownStatusAbbreviations[status]; ok {
		return abbrev
	}
	return abbreviateStatusFallback(status)
}

// knownStatusAbbreviations covers every canonical status used by the shipped
// short and long workflows (epic, feature, task, bug, change). Add to this
// map when introducing a new status to a workflow file.
var knownStatusAbbreviations = map[string]string{
	// Universal terminal / pause states
	"completed": "DONE",
	"cancelled": "CANC",
	"blocked":   "BLOK",
	"on_hold":   "HOLD",
	"draft":     "DRFT",
	"active":    "ACTV",
	"approved":  "APRV",
	"rejected":  "REJ",
	"reported":  "RPTD",
	"todo":      "TODO",

	// Task / bug / change "in_*" states
	"in_progress":        "PROG",
	"in_development":     "DEV",
	"in_code_review":     "CR",
	"in_qa":              "QA",
	"in_approval":        "APR", // shorter than terminal "approved" → APRV
	"in_feature_review":  "FRVW",
	"in_task_review":     "TRVW",
	"in_task_generation": "TGEN",
	"in_test_planning":   "TPLN",
	"in_specification":   "SPEC",
	"in_research":        "RSCH",
	"in_design":          "DSGN",
	"in_decomposition":   "DECO",
	"in_refinement":      "RFMT",
	"in_assessment":      "ASMT",

	// "ready_for_*" states — prefix with R- so they're visually distinct
	// from the corresponding "in_*" state.
	"ready_for_development":             "R-DEV",
	"ready_for_code_review":             "R-CR",
	"ready_for_qa":                      "R-QA",
	"ready_for_approval":                "R-APR",
	"ready_for_feature_review":          "R-FR",
	"ready_for_task_review":             "R-TR",
	"ready_for_task_generation":         "R-TG",
	"ready_for_test_planning":           "R-TP",
	"ready_for_specification":           "R-SP",
	"ready_for_research":                "R-RS",
	"ready_for_design":                  "R-DS",
	"ready_for_decomposition":           "R-DC",
	"ready_for_refinement":              "R-RF",
	"ready_for_assessment":              "R-AS",
	"ready_for_review":                  "R-RV",
	"ready_for_ba_check":                "R-BA",
	"ready_for_scope_validation":        "R-SV",
	"ready_for_refinement_ba":           "R-RFB",
	"ready_for_refinement_tech":         "R-RFT",
	"ready_for_feasibility_review_ba":   "R-FBA",
	"ready_for_feasibility_review_tech": "R-FTC",
}

// abbreviateStatusFallback handles statuses not present in
// knownStatusAbbreviations. Strips the "in_" / "ready_for_" prefix, then
// returns up to 4 uppercase chars of the payload (prepended with "R-" for
// ready_for variants).
func abbreviateStatusFallback(status string) string {
	const maxPayload = 4
	switch {
	case strings.HasPrefix(status, "ready_for_"):
		payload := abbreviatePayload(status[len("ready_for_"):], maxPayload)
		return "R-" + payload
	case strings.HasPrefix(status, "in_"):
		return abbreviatePayload(status[len("in_"):], maxPayload)
	default:
		return abbreviatePayload(status, maxPayload)
	}
}

// abbreviatePayload returns the first maxLen runes of payload uppercased,
// stripping underscores so multi-word payloads stay compact.
func abbreviatePayload(payload string, maxLen int) string {
	cleaned := strings.ReplaceAll(payload, "_", "")
	upper := strings.ToUpper(cleaned)
	r := []rune(upper)
	if len(r) <= maxLen {
		return upper
	}
	return string(r[:maxLen])
}
