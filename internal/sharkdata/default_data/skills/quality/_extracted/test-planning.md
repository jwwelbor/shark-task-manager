---
# Extracted scaffolding from workflows/test-planning.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field spec_path
shark context get $TASK_KEY --field acceptance_criteria
shark related-docs list --task=$TASK_KEY --filter="test|plan|scenario"

# gate
Test plan covers all acceptance criteria.
Test scenarios are concrete and executable.
Resource estimates provided for test execution.

# mutate
shark context set $TASK_KEY --field test_plan_status --value "finalized"
shark note add $TASK_KEY --type quality --content "Test plan: [scenarios defined, ready for implementation]"

# advance
shark status advance $TASK_KEY
