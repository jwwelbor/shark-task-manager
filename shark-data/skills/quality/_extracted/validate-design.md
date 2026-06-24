---
# Extracted scaffolding from workflows/validate-design.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field design_path
shark context get $TASK_KEY --field validation_criteria_path
shark related-docs list --task=$TASK_KEY --filter="design|validation|criteria"

# gate
Design meets acceptance criteria.
Design is feasible for implementation.
Risks identified and mitigation planned.

# mutate
shark context set $TASK_KEY --field design_validation_status --value "approved"
shark note add $TASK_KEY --type quality --content "Design validation: [approved, ready for implementation]"

# advance
shark status advance $TASK_KEY
