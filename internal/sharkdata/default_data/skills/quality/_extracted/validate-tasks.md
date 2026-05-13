---
# Extracted scaffolding from workflows/validate-tasks.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field task_spec_path
shark context get $TASK_KEY --field feature_spec_path
shark related-docs list --task=$TASK_KEY --filter="requirements|acceptance|criteria"

# gate
Tasks are well-formed and actionable.
Dependencies documented and valid.
Acceptance criteria clear and testable.

# mutate
shark context set $TASK_KEY --field task_validation_status --value "ready"
shark note add $TASK_KEY --type quality --content "Task validation: [complete, ready for development]"

# advance
shark status advance $TASK_KEY
