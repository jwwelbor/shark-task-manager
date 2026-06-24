---
# Extracted scaffolding from workflows/tracing-knowledge-lineages.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field research_objective
shark related-docs list --feature=$FEATURE_KEY --filter="research|analysis|documentation"

# gate
Research objective clearly defined.
Findings documented with evidence.
Analysis complete and reviewed.

# mutate
shark context set $FEATURE_KEY --field research_status --value "complete"
shark note add $FEATURE_KEY --type research --content "Research: [tracing-knowledge-lineages complete]"

# advance
shark status advance $FEATURE_KEY
