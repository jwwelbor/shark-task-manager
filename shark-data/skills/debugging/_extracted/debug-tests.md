---
# Extracted scaffolding from workflows/debug-tests.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field failing_test_logs
shark context get $TASK_KEY --field test_output_path
shark related-docs list --task=$TASK_KEY --filter="test|assertion|failure"

# gate
Test failure isolated and reproducible.
Assertion message clearly identifies what failed.
Test vs. code behavior divergence documented.

# mutate
shark context set $TASK_KEY --field test_debug_status --value "in_progress"
shark note add $TASK_KEY --type blocker --content "Test failure: [test name, root cause identified]"

# advance
shark status advance $TASK_KEY
