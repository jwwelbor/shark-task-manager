---
# Extracted scaffolding from workflows/debug-frontend.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field browser_console_logs
shark context get $TASK_KEY --field ui_screenshot_path
shark related-docs list --task=$TASK_KEY --filter="frontend|ui|error"

# gate
UI bug isolated to specific component or page.
Console errors documented with stack traces.
Reproduction steps clearly defined.

# mutate
shark context set $TASK_KEY --field frontend_debug_status --value "in_progress"
shark note add $TASK_KEY --type blocker --content "Frontend bug: [affected component, root cause identified]"

# advance
shark status advance $TASK_KEY
