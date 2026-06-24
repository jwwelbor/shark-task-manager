{{define "_resolve_spec_paths"}}# Resolve domain-specific artifact paths

Compute filesystem paths for any artifacts this prompt produces. Inherits `EPIC_ID` and `FEATURE_ID` from `_fetch_entity_context`.

```bash
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Per-domain artifact directories
QA_REPORTS_DIR="docs/plan/$EPIC_ID/$FEATURE_ID/qa_reports"
CODE_REVIEW_DIR="docs/plan/$EPIC_ID/$FEATURE_ID/code_review"
TEST_PLANS_DIR="docs/plan/$EPIC_ID/$FEATURE_ID/test_plans"
UAT_DIR="docs/uat/$EPIC_ID"
ARCH_DIR="docs/architecture"

# Create the relevant dir(s) for this prompt — host calls this with the right list
{{- if .domains }}
{{- range .domains }}
mkdir -p "${{ . }}_DIR"
{{- end }}
{{- end }}

# Standard artifact filename pattern (timestamp-entity-suffix)
# Use as: ${DOMAIN_DIR}/${TIMESTAMP}-${TASK_ID}-${SUFFIX}.md
```

`SUFFIX` is workflow-specific (`qa-results`, `qa-exploratory`, `code-review`, `test-plan`, `uat-results`, etc.).
{{end}}
