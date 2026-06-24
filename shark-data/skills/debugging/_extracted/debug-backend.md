---
# Extracted scaffolding from workflows/debug-backend.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field error_logs_path
shark context get $TASK_KEY --field last_known_working_state
shark related-docs list --task=$TASK_KEY --filter="error|logs|debug"

# gate
Error is reproducible with clear steps.
Root cause isolated to specific component.
Reproduction confirmed in controlled environment.

# mutate
shark context set $TASK_KEY --field debug_status --value "in_progress"
shark note add $TASK_KEY --type blocker --content "Backend error: [root cause identified]"
shark context set $TASK_KEY --field root_cause_analysis --value "complete"

# advance
shark status advance $TASK_KEY
