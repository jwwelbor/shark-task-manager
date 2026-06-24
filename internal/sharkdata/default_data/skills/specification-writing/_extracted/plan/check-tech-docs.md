---
# Extracted scaffolding from workflows/plan/check-tech-docs.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $EPIC_KEY --field planning_template_path
shark related-docs list --epic=$EPIC_KEY --filter="plan|documentation|analysis"

# gate
Plan complete and resource-aware.
Timeline realistic and risk-assessed.
Dependencies and assumptions documented.

# mutate
shark context set $EPIC_KEY --field planning_status --value "complete"
shark note add $EPIC_KEY --type plan --content "Planning: [check-tech-docs complete]"

# advance
shark status advance $EPIC_KEY
