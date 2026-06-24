---
# Extracted scaffolding from workflows/qa-testing.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field test_plan_path
shark context get $TASK_KEY --field implementation_paths
shark related-docs list --task=$TASK_KEY --filter="test|acceptance|criteria"

# gate
All acceptance criteria verified.
Test results documented and reproducible.
No blockers identified in QA phase.

# mutate
shark context set $TASK_KEY --field qa_status --value "passed"
shark note add $TASK_KEY --type quality --content "QA testing: [all tests passed, ready for release]"

# advance
shark status advance $TASK_KEY
