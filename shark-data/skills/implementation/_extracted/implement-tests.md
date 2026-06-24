---
# Extracted scaffolding from workflows/implement-tests.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field test_plan_path
shark context get $TASK_KEY --field implementation_paths
shark related-docs list --task=$TASK_KEY --filter="test|assertion|coverage"

# gate
All test plan scenarios implemented.
Code coverage meets minimum threshold.
Tests are deterministic and reliable.

# mutate
shark context set $TASK_KEY --field test_implementation_status --value "complete"
shark note add $TASK_KEY --type implementation --content "Test implementation: [unit, integration, e2e tests complete]"

# advance
shark status advance $TASK_KEY
